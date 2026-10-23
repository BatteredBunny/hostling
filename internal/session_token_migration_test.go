package internal

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/BatteredBunny/hostling/internal/db"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func prepareLegacySessionTokens(
	t *testing.T,
	database *gorm.DB,
	dialect string,
) (string, string, time.Time, time.Time) {
	t.Helper()
	// The preceding upload-token finalization has no rows to backfill here.
	if err := migrationTestExecutor(
		t,
		database,
		dialect,
	).ExecuteTo(context.Background(), uploadTokenFinalizeVersion); err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&db.Accounts{ID: 1, AccountType: "NORMAL"}).Error; err != nil {
		t.Fatal(err)
	}
	active, expired := uuid.NewString(), uuid.NewString()
	used := time.Now().Truncate(time.Second).Add(-time.Hour)
	expiry := used.Add(3 * time.Hour)
	for i, token := range []string{active, expired} {
		expires := expiry
		if i == 1 {
			expires = used
		}
		if err := database.Table("session_tokens").Create(map[string]any{
			"id": 17 + i, "account_id": 1, "token": token, "created_at": used,
			"updated_at": used, "last_used": used, "expiry_date": expires,
		}).Error; err != nil {
			t.Fatal(err)
		}
	}

	return active, expired, used, expiry
}

func TestSessionTokenMigration(t *testing.T) {
	for _, dialect := range []string{"sqlite", "postgresql"} {
		t.Run(dialect, func(t *testing.T) {
			for _, phase := range []string{"legacy", "hash_column_added", "backfilled"} {
				t.Run(phase, func(t *testing.T) {
					database, dsn := migrationTestDatabase(t, dialect)
					active, expired, used, expiry := prepareLegacySessionTokens(t, database, dialect)
					if phase != "legacy" {
						if err := migrationTestExecutor(
							t,
							database,
							dialect,
						).ExecuteTo(context.Background(), "20261023000002"); err != nil {
							t.Fatal(err)
						}
					}
					if phase == "backfilled" {
						if err := db.BackfillSessionTokenHashes(database); err != nil {
							t.Fatal(err)
						}
					}
					if err := RunMigrations(database, dialect, dsn); err != nil {
						t.Fatal(err)
					}
					if err := RunMigrations(database, dialect, dsn); err != nil {
						t.Fatal(err)
					}
					if database.Migrator().HasColumn("session_tokens", "token") {
						t.Fatal("plaintext session column remains")
					}
					var record db.SessionTokens
					if err := database.First(&record, 17).Error; err != nil {
						t.Fatal(err)
					}
					hash, err := db.HashSessionToken(active)
					if err != nil {
						t.Fatal(err)
					}
					if record.TokenHash != hash || record.AccountID != 1 || !record.LastUsed.Equal(used) ||
						!record.CreatedAt.Equal(used) ||
						!record.ExpiryDate.Equal(expiry) {
						t.Fatal("migration changed session credentials, timestamps, or expiry")
					}
					store := &db.Database{DB: database}
					account, err := store.GetAccountBySessionToken(active)
					if err != nil || account.ID != 1 {
						t.Fatalf("existing session cannot authenticate: %v", err)
					}
					if _, err := store.GetAccountBySessionToken(expired); !errors.Is(err, gorm.ErrRecordNotFound) {
						t.Fatalf("expired session became usable: %v", err)
					}
					if _, err := store.GetAccountBySessionToken(hash); !errors.Is(err, gorm.ErrRecordNotFound) {
						t.Fatalf("stored hash authenticated: %v", err)
					}
					if err := database.Table("session_tokens").
						Create(map[string]any{"account_id": 1}).
						Error; err == nil {
						t.Fatal("final schema allows missing hashes")
					}
					if err := database.Create(
						&db.SessionTokens{AccountID: 1, TokenHash: hash, ExpiryDate: expiry},
					).Error; err == nil {
						t.Fatal("final schema allows duplicate session hashes")
					}
					fresh, err := store.CreateSessionToken(1)
					if err != nil || fresh == "" {
						t.Fatalf("new session creation failed: %v", err)
					}
					if err := store.DeleteSession(active); err != nil {
						t.Fatal(err)
					}
					if _, err := store.GetAccountBySessionToken(active); !errors.Is(err, gorm.ErrRecordNotFound) {
						t.Fatal("migrated session cannot be revoked")
					}
					if err := database.Delete(&db.Accounts{}, 1).Error; err != nil {
						t.Fatal(err)
					}
					var count int64
					if err := database.Model(&db.SessionTokens{}).Count(&count).Error; err != nil || count != 0 {
						t.Fatalf("session account cascade failed: %d, %v", count, err)
					}
				})
			}
		})
	}
}

func TestSessionTokenMigrationFailureIsResumable(t *testing.T) {
	for _, dialect := range []string{"sqlite", "postgresql"} {
		t.Run(dialect, func(t *testing.T) {
			database, dsn := migrationTestDatabase(t, dialect)
			active, _, _, _ := prepareLegacySessionTokens(t, database, dialect)
			invalid := "invalid legacy session credential"
			if err := database.Table("session_tokens").
				Create(map[string]any{"id": 19, "account_id": 1, "token": invalid}).
				Error; err != nil {
				t.Fatal(err)
			}
			if err := RunMigrations(database, dialect, dsn); err == nil {
				t.Fatal("invalid legacy credential should stop migration")
			} else if strings.Contains(err.Error(), active) || strings.Contains(err.Error(), invalid) {
				t.Fatal("migration disclosed a session credential")
			}
			if !database.Migrator().HasColumn("session_tokens", "token") {
				t.Fatal("failed migration removed plaintext prematurely")
			}
			var hash *string
			if err := database.Table("session_tokens").
				Select("token_hash").
				Where("id = ?", 17).
				Row().
				Scan(&hash); err != nil {
				t.Fatal(err)
			}
			if hash != nil {
				t.Fatal("failed migration committed partial backfill")
			}
			if database.Migrator().HasTable("new_session_tokens") {
				t.Fatal("failed migration left a temporary table")
			}
			if err := database.Table("session_tokens").
				Where("id = ?", 19).
				Update("token", uuid.NewString()).
				Error; err != nil {
				t.Fatal(err)
			}
			if err := RunMigrations(database, dialect, dsn); err != nil {
				t.Fatal(err)
			}
			if database.Migrator().HasColumn("session_tokens", "token") {
				t.Fatal("retry left plaintext session storage")
			}
		})
	}
}

func TestSessionTokenMigrationFinalizationRollback(t *testing.T) {
	for _, dialect := range []string{"sqlite", "postgresql"} {
		t.Run(dialect, func(t *testing.T) {
			database, dsn := migrationTestDatabase(t, dialect)
			_, _, _, _ = prepareLegacySessionTokens(t, database, dialect)
			if err := migrationTestExecutor(
				t,
				database,
				dialect,
			).ExecuteTo(context.Background(), "20261023000002"); err != nil {
				t.Fatal(err)
			}
			injected := errors.New("simulated interrupted session upgrade")
			callback := "test:interrupt_session_finalization"
			if err := database.Callback().Update().Before("gorm:update").Register(callback, func(tx *gorm.DB) {
				if tx.Statement.Table != "atlas_schema_revisions" {
					return
				}
				revision, ok := tx.Statement.Dest.(*db.AtlasSchemaRevision)
				if ok && revision.Version == sessionTokenFinalizeVersion && revision.Applied == revision.Total {
					_ = tx.AddError(injected)
				}
			}); err != nil {
				t.Fatal(err)
			}
			err := RunMigrations(database, dialect, dsn)
			_ = database.Callback().Update().Remove(callback)
			if !errors.Is(err, injected) {
				t.Fatalf("expected injected interruption, got %v", err)
			}
			if !database.Migrator().HasColumn("session_tokens", "token") ||
				database.Migrator().HasTable("new_session_tokens") {
				t.Fatal("interrupted finalization was not rolled back")
			}
			if err := RunMigrations(database, dialect, dsn); err != nil {
				t.Fatal(err)
			}
			if database.Migrator().HasColumn("session_tokens", "token") {
				t.Fatal("retry did not remove plaintext")
			}
		})
	}
}
