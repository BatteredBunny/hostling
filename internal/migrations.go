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
	default:
		err = fmt.Errorf("unsupported database type %q", databaseType)

		return
	}

	migrator := db.NewRevisionReaderWriter(database)
	executor, err := migrate.NewExecutor(driver, dir, migrator, migrate.WithAllowDirty(true))
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

	if err = executePendingMigrations(ctx, executor, database, databaseType, dir); err != nil {
		return err
	}
	log.Info().Msg("Migrations applied successfully")

	return
}

const (
	uploadTokenFinalizeVersion  = "20261023000001"
	sessionTokenFinalizeVersion = "20261023000003"
)

func executePendingMigrations(
	ctx context.Context,
	executor *migrate.Executor,
	database *gorm.DB,
	databaseType string,
	dir migrate.Dir,
) error {
	for {
		pending, err := executor.Pending(ctx)
		if errors.Is(err, migrate.ErrNoPendingFiles) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("failed to read pending migrations: %w", err)
		}
		if len(pending) == 0 {
			return nil
		}
		var backfill func(*gorm.DB) error
		switch pending[0].Version() {
		case uploadTokenFinalizeVersion:
			backfill = db.BackfillUploadTokenHashes
		case sessionTokenFinalizeVersion:
			backfill = db.BackfillSessionTokenHashes
		}
		if backfill != nil {
			if err := finalizeTokenHashMigration(ctx, database, databaseType, dir, backfill); err != nil {
				return fmt.Errorf("failed to migrate token secrets in %s: %w", pending[0].Version(), err)
			}

			continue
		}
		if err := executor.ExecuteN(ctx, 1); err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", pending[0].Version(), err)
		}
	}
}

// Backfill, schema changes, and Atlas revision tracking commit together. A
// stopped upgrade can therefore retry without a half-rebuilt SQLite table or a
// plaintext column removed before its verifier was committed.
func finalizeTokenHashMigration(
	ctx context.Context,
	database *gorm.DB,
	databaseType string,
	dir migrate.Dir,
	backfill func(*gorm.DB) error,
) error {
	return database.Transaction(func(tx *gorm.DB) error {
		if err := backfill(tx); err != nil {
			return err
		}
		var driver migrate.Driver
		var err error
		switch databaseType {
		case "postgresql":
			driver, err = postgres.Open(tx.Statement.ConnPool)
		case "sqlite":
			driver, err = sqlite.Open(tx.Statement.ConnPool)
		default:
			return fmt.Errorf("unsupported database type %q", databaseType)
		}
		if err != nil {
			return err
		}
		executor, err := migrate.NewExecutor(driver, dir, db.NewRevisionReaderWriter(tx), migrate.WithAllowDirty(true))
		if err != nil {
			return err
		}

		return executor.ExecuteN(ctx, 1)
	})
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

	var accountsExists bool
	switch databaseType {
	case "postgresql":
		err = sqlDB.QueryRow("SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'accounts')").
			Scan(&accountsExists)
	case "sqlite":
		err = sqlDB.QueryRow("SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name = 'accounts')").
			Scan(&accountsExists)
	}
	if err != nil {
		return fmt.Errorf("failed to check existing tables: %w", err)
	}

	if accountsExists {
		var files []migrate.File
		files, err = dir.Files()
		if err != nil {
			return fmt.Errorf("failed to read migration files: %w", err)
		} else if len(files) == 0 {
			return
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
