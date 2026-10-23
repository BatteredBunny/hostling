package internal

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ariga.io/atlas/sql/migrate"
	atlaspostgres "ariga.io/atlas/sql/postgres"
	atlassqlite "ariga.io/atlas/sql/sqlite"
	"github.com/BatteredBunny/hostling/internal/db"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Set HOSTLING_TEST_POSTGRES_DSN to an isolated PostgreSQL instance to exercise
// the same upgrade cases there. Every case creates and removes its own schema.
func migrationTestDatabase(t *testing.T, dialect string) (*gorm.DB, string) {
	t.Helper()
	config := &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)}
	if dialect == "sqlite" {
		dsn := filepath.Join(t.TempDir(), "migration.db") + "?_foreign_keys=on"
		database, err := gorm.Open(sqlite.Open(dsn), config)
		if err != nil {
			t.Fatal(err)
		}
		sqlDB, err := database.DB()
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = sqlDB.Close() })

		return database, dsn
	}
	dsn := os.Getenv("HOSTLING_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set HOSTLING_TEST_POSTGRES_DSN to test PostgreSQL migrations")
	}
	admin, err := gorm.Open(postgres.Open(dsn), config)
	if err != nil {
		t.Fatal(err)
	}
	schema := "token_migration_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if err := admin.Exec(`CREATE SCHEMA "` + schema + `"`).Error; err != nil {
		t.Fatal(err)
	}
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		parsed, err := url.Parse(dsn)
		if err != nil {
			t.Fatal(err)
		}
		query := parsed.Query()
		query.Set("search_path", schema)
		parsed.RawQuery = query.Encode()
		dsn = parsed.String()
	} else {
		dsn += " search_path=" + schema
	}
	database, err := gorm.Open(postgres.Open(dsn), config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		sqlDB, _ := database.DB()
		_ = sqlDB.Close()
		_ = admin.Exec(`DROP SCHEMA "` + schema + `" CASCADE`).Error
		adminDB, _ := admin.DB()
		_ = adminDB.Close()
	})

	return database, dsn
}

func migrationTestExecutor(t *testing.T, database *gorm.DB, dialect string) *migrate.Executor {
	t.Helper()
	if err := database.AutoMigrate(&db.AtlasSchemaRevision{}); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := database.DB()
	if err != nil {
		t.Fatal(err)
	}
	var driver migrate.Driver
	if dialect == "sqlite" {
		driver, err = atlassqlite.Open(sqlDB)
	} else {
		driver, err = atlaspostgres.Open(sqlDB)
	}
	if err != nil {
		t.Fatal(err)
	}
	dir, err := loadEmbeddedMigrations(dialect)
	if err != nil {
		t.Fatal(err)
	}
	executor, err := migrate.NewExecutor(
		driver,
		dir,
		db.NewRevisionReaderWriter(database),
		migrate.WithAllowDirty(true),
	)
	if err != nil {
		t.Fatal(err)
	}

	return executor
}

func prepareLegacyUploadToken(t *testing.T, database *gorm.DB, dialect string) (string, time.Time) {
	t.Helper()
	executor := migrationTestExecutor(t, database, dialect)
	if err := executor.ExecuteTo(context.Background(), "20260513000000"); err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&db.Accounts{ID: 1, AccountType: "NORMAL"}).Error; err != nil {
		t.Fatal(err)
	}
	token := uuid.NewString()
	used := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	if err := database.Table("upload_tokens").Create(map[string]any{
		"id": 17, "account_id": 1, "nickname": "legacy uploader", "token": token,
		"created_at": used, "updated_at": used, "last_used": used,
	}).Error; err != nil {
		t.Fatal(err)
	}

	return token, used
}

func TestUploadTokenMigration(t *testing.T) {
	for _, dialect := range []string{"sqlite", "postgresql"} {
		t.Run(dialect, func(t *testing.T) {
			for _, phase := range []string{"legacy", "hash_column_added", "backfilled"} {
				t.Run(phase, func(t *testing.T) {
					database, dsn := migrationTestDatabase(t, dialect)
					token, used := prepareLegacyUploadToken(t, database, dialect)
					if phase != "legacy" {
						if err := migrationTestExecutor(
							t,
							database,
							dialect,
						).ExecuteTo(context.Background(), "20261023000000"); err != nil {
							t.Fatal(err)
						}
					}
					if phase == "backfilled" {
						if err := db.BackfillUploadTokenHashes(database); err != nil {
							t.Fatal(err)
						}
					}
					if err := RunMigrations(database, dialect, dsn); err != nil {
						t.Fatal(err)
					}
					// Restarting a successfully upgraded application is also safe.
					if err := RunMigrations(database, dialect, dsn); err != nil {
						t.Fatal(err)
					}
					if database.Migrator().HasColumn("upload_tokens", "token") {
						t.Fatal("plaintext token column still exists")
					}
					var record db.UploadTokens
					if err := database.First(&record, 17).Error; err != nil {
						t.Fatal(err)
					}
					hash, err := db.HashUploadToken(token)
					if err != nil {
						t.Fatal(err)
					}
					if record.TokenHash != hash || record.AccountID != 1 || record.Nickname != "legacy uploader" ||
						record.LastUsed == nil ||
						!record.LastUsed.Equal(used) ||
						!record.CreatedAt.Equal(used) {
						t.Fatal("migration changed a credential or its metadata")
					}
					account, err := (&db.Database{DB: database}).GetAccountByUploadToken(token)
					if err != nil || account.ID != 1 {
						t.Fatalf("legacy credential cannot authenticate: %v", err)
					}
					if err := database.Table("upload_tokens").
						Create(map[string]any{"account_id": 1, "nickname": "missing hash"}).
						Error; err == nil {
						t.Fatal("final schema permits a null hash")
					}
					if err := database.Create(
						&db.UploadTokens{AccountID: 1, Nickname: "duplicate", TokenHash: hash},
					).Error; err == nil {
						t.Fatal("final schema permits duplicate credentials")
					}
					// IDs remain usable after SQLite's table rebuild and PG upgrades.
					created, err := (&db.Database{DB: database}).CreateUploadToken(1, "new uploader")
					if err != nil || created.ID == 17 {
						t.Fatalf("new token creation failed after migration: %v", err)
					}
					if err := database.Delete(&db.Accounts{}, 1).Error; err != nil {
						t.Fatal(err)
					}
					var count int64
					if err := database.Model(&db.UploadTokens{}).Count(&count).Error; err != nil || count != 0 {
						t.Fatalf("account cascade changed: %d, %v", count, err)
					}
				})
			}
		})
	}
}

func TestUploadTokenMigrationFailureIsResumable(t *testing.T) {
	for _, dialect := range []string{"sqlite", "postgresql"} {
		t.Run(dialect, func(t *testing.T) {
			database, dsn := migrationTestDatabase(t, dialect)
			token, _ := prepareLegacyUploadToken(t, database, dialect)
			if err := database.Table("upload_tokens").
				Create(map[string]any{"id": 18, "account_id": 1, "nickname": "broken", "token": "invalid legacy credential"}).
				Error; err != nil {
				t.Fatal(err)
			}
			if err := RunMigrations(database, dialect, dsn); err == nil {
				t.Fatal("invalid legacy secret should stop the upgrade")
			} else if strings.Contains(err.Error(), token) || strings.Contains(err.Error(), "invalid legacy credential") {
				t.Fatal("migration error disclosed a secret")
			}
			if !database.Migrator().HasColumn("upload_tokens", "token") {
				t.Fatal("failed upgrade removed plaintext before preserving credentials")
			}
			var hash *string
			if err := database.Table("upload_tokens").
				Select("token_hash").
				Where("id = ?", 17).
				Row().
				Scan(&hash); err != nil {
				t.Fatal(err)
			}
			if hash != nil {
				t.Fatal("failed backfill committed partial data")
			}
			if database.Migrator().HasTable("new_upload_tokens") {
				t.Fatal("failed upgrade left a temporary table")
			}
			if err := database.Table("upload_tokens").
				Where("id = ?", 18).
				Update("token", uuid.NewString()).
				Error; err != nil {
				t.Fatal(err)
			}
			if err := RunMigrations(database, dialect, dsn); err != nil {
				t.Fatal(err)
			}
			if database.Migrator().HasColumn("upload_tokens", "token") {
				t.Fatal("retry did not finish")
			}
		})
	}
}

func TestUploadTokenMigrationFinalizationRollback(t *testing.T) {
	for _, dialect := range []string{"sqlite", "postgresql"} {
		t.Run(dialect, func(t *testing.T) {
			database, dsn := migrationTestDatabase(t, dialect)
			_, _ = prepareLegacyUploadToken(t, database, dialect)
			if err := migrationTestExecutor(
				t,
				database,
				dialect,
			).ExecuteTo(context.Background(), "20261023000000"); err != nil {
				t.Fatal(err)
			}
			injected := errors.New("simulated interrupted upgrade")
			// Atlas writes its revision inside the same transaction as the SQL
			// finalization. Fail that final revision update after schema writes.
			callback := "test:interrupt_token_finalization"
			err := database.Callback().Update().Before("gorm:update").Register(callback, func(tx *gorm.DB) {
				if tx.Statement.Table != "atlas_schema_revisions" {
					return
				}
				revision, ok := tx.Statement.Dest.(*db.AtlasSchemaRevision)
				if ok && revision.Version == uploadTokenFinalizeVersion && revision.Applied == revision.Total {
					_ = tx.AddError(injected)
				}
			})
			if err != nil {
				t.Fatal(err)
			}
			err = RunMigrations(database, dialect, dsn)
			_ = database.Callback().Update().Remove(callback)
			if !errors.Is(err, injected) {
				t.Fatalf("expected injected failure, got %v", err)
			}
			if !database.Migrator().HasColumn("upload_tokens", "token") ||
				database.Migrator().HasTable("new_upload_tokens") {
				t.Fatal("failed finalization was not rolled back")
			}
			if err := RunMigrations(database, dialect, dsn); err != nil {
				t.Fatal(fmt.Errorf("retry failed: %w", err))
			}
			if database.Migrator().HasColumn("upload_tokens", "token") {
				t.Fatal("retry left plaintext column")
			}
		})
	}
}
