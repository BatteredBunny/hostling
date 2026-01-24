package internal

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"time"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/postgres"
	"ariga.io/atlas/sql/sqlite"
	embed "github.com/BatteredBunny/hostling"
	"github.com/BatteredBunny/hostling/internal/db"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

// Hacky way to get the migration files from virtual fs
func loadEmbeddedMigrations(databaseType string) (dir *migrate.MemDir, err error) {
	dir = migrate.OpenMemDir(databaseType)

	migrationsPath := "migrations/" + databaseType
	err = fs.WalkDir(embed.MigrationFiles, migrationsPath, func(path string, d fs.DirEntry, err error) (outerr error) {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return
		}

		file, err := embed.MigrationFiles.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open embedded file %s: %w", path, err)
		}
		defer file.Close()

		data, err := io.ReadAll(file)
		if err != nil {
			return fmt.Errorf("failed to read embedded file %s: %w", path, err)
		}

		name := d.Name()

		if err := dir.WriteFile(name, data); err != nil {
			return fmt.Errorf("failed to write file %s to MemDir: %w", name, err)
		}

		return
	})

	return
}

func RunMigrations(database *gorm.DB, databaseType string, databaseConnectionUrl string) (err error) {
	log.Info().Msg("Running migrations")

	if err = database.AutoMigrate(&db.AtlasSchemaRevision{}); err != nil {
		err = fmt.Errorf("failed to create atlas_schema_revisions table: %w", err)
		return
	}

	sqlDB, err := database.DB()
	if err != nil {
		err = fmt.Errorf("failed to get database connection: %w", err)
		return
	}

	dir, err := loadEmbeddedMigrations(databaseType)
	if err != nil {
		err = fmt.Errorf("failed to load embedded migrations: %w", err)
		return
	}

	var driver migrate.Driver
	switch databaseType {
	case "postgresql":
		driver, err = postgres.Open(sqlDB)
		if err != nil {
			err = fmt.Errorf("failed to open postgres driver: %w", err)
			return
		}
	case "sqlite":
		driver, err = sqlite.Open(sqlDB)
		if err != nil {
			err = fmt.Errorf("failed to open sqlite driver: %w", err)
			return
		}
	}

	migrator := db.NewRevisionReaderWriter(database)
	executor, err := migrate.NewExecutor(driver, dir, migrator)
	if err != nil {
		err = fmt.Errorf("failed to create migration executor: %w", err)
		return
	}

	ctx := context.Background()
	if err = MigrateExistingDatabase(
		ctx,
		database,
		databaseType,
		databaseConnectionUrl,
		migrator,
		sqlDB,
		dir,
	); err != nil {
		return err
	}

	// Apply all pending migrations
	if err = executor.ExecuteN(ctx, 0); err != nil {
		if !errors.Is(err, migrate.ErrNoPendingFiles) {
			err = fmt.Errorf("failed to execute migrations: %w", err)
			return
		}
	} else {
		log.Info().Msg("Migrations applied successfully")
	}

	return
}

// Temporary code to migrate from existing database from before migration system
// Can be dumped in the future
func MigrateExistingDatabase(
	ctx context.Context,
	database *gorm.DB,
	databaseType string,
	databaseConnectionUrl string,
	migrator migrate.RevisionReadWriter,
	sqlDB *sql.DB,
	dir *migrate.MemDir,
) (err error) {
	var revisions []*migrate.Revision
	if revisions, err = migrator.ReadRevisions(ctx); err != nil {
		return fmt.Errorf("failed to read existing revisions: %w", err)
	} else if len(revisions) != 0 {
		return
	}

	var tableCount int
	switch databaseType {
	case "postgresql":
		err = sqlDB.QueryRow("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_type = 'BASE TABLE'").Scan(&tableCount)
	case "sqlite":
		err = sqlDB.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'").Scan(&tableCount)
	}
	if err != nil {
		return fmt.Errorf("failed to check existing tables: %w", err)
	}

	// If tables exist (excluding atlas_schema_revisions since that gets created automatically before),
	// Mark the first migration as done since that adds the tables
	if tableCount > 1 {
		files, err := dir.Files()
		if err != nil {
			return fmt.Errorf("failed to read migration files: %w", err)
		}

		firstFile := files[0]
		baseline := &migrate.Revision{
			Version:         firstFile.Version(),
			Description:     firstFile.Desc(),
			Type:            migrate.RevisionTypeBaseline,
			Applied:         0,
			Total:           0,
			ExecutedAt:      time.Now(),
			ExecutionTime:   0,
			OperatorVersion: "",
		}

		if err := migrator.WriteRevision(ctx, baseline); err != nil {
			return fmt.Errorf("failed to write baseline revision: %w", err)
		}
		log.Info().Str("version", firstFile.Version()).Msg("Baseline revision created")
	}

	return
}
