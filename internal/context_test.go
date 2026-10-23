package internal

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/BatteredBunny/hostling/internal/db"
	"gorm.io/gorm"
)

func TestDatabaseCancellationIsScoped(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.registerAdmin()
	token := admin.newUploadToken("Cancellation test")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	database := ts.app.db.WithContext(ctx)
	if _, _, err := database.GetFilesPaginatedFromAccount(
		ts.adminID(),
		0,
		8,
		"created_at",
		true,
		nil,
		nil,
	); !errors.Is(
		err,
		context.Canceled,
	) {
		t.Fatalf("listing ignored cancellation: %v", err)
	}
	if _, err := ts.app.isValidUploadToken(ctx, token); !errors.Is(err, context.Canceled) {
		t.Fatalf("authentication ignored cancellation: %v", err)
	}
	if _, err := ts.app.db.GetAccounts(); err != nil {
		t.Fatalf("scoped cancellation affected shared DB: %v", err)
	}
}

func TestBackgroundWaitHonorsDeadline(t *testing.T) {
	app := &Application{}
	release := make(chan struct{})
	app.backgroundWg.Go(func() { <-release })
	defer close(release)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if err := app.waitBackground(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected shutdown deadline, got %v", err)
	}
}

func TestTypedCRUDHonorsScopedCancellation(t *testing.T) {
	ts := newTestServer(t)
	ts.registerAdmin()
	accountID := ts.adminID()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	database := ts.app.db.WithContext(ctx)
	for _, operation := range []struct {
		name string
		run  func() error
	}{
		{"lookup", func() error {
			_, err := database.GetAccountByID(accountID)

			return err
		}},
		{"count", func() error {
			_, err := database.GetSessionsCount(accountID)

			return err
		}},
		{"create", func() error {
			_, err := database.CreateAccount("USER", accountID)

			return err
		}},
		{"update", func() error { return database.UpdateGithubUsername(accountID, "cancelled") }},
		{"delete", func() error { return database.DeleteAccount(accountID) }},
	} {
		t.Run(operation.name, func(t *testing.T) {
			if err := operation.run(); !errors.Is(err, context.Canceled) {
				t.Fatalf("expected cancellation, got %v", err)
			}
		})
	}
	accounts, err := ts.app.db.GetAccounts()
	if err != nil || len(accounts) != 1 || accounts[0].ID != accountID || accounts[0].GithubUsername != "" {
		t.Fatalf("cancelled operations changed accounts: %+v, %v", accounts, err)
	}
}

func TestTypedCreatesStayInsideTransaction(t *testing.T) {
	ts := newTestServer(t)
	ts.registerAdmin()
	rollback := errors.New("roll back typed creates")
	err := ts.app.db.Transaction(func(tx *gorm.DB) error {
		database := db.Database{DB: tx}
		account, err := database.CreateAccount("USER", ts.adminID())
		if err != nil {
			return err
		}
		if _, err := database.CreateSessionToken(account.ID); err != nil {
			return err
		}
		if _, err := database.CreateUploadToken(account.ID, "rolled back"); err != nil {
			return err
		}
		if _, err := database.GetAccountByID(account.ID); err != nil {
			return err
		}

		return rollback
	})
	if !errors.Is(err, rollback) {
		t.Fatalf("unexpected transaction error: %v", err)
	}
	for _, model := range []any{&db.Accounts{}, &db.SessionTokens{}, &db.UploadTokens{}} {
		var count int64
		if err := ts.app.db.Model(model).Count(&count).Error; err != nil {
			t.Fatal(err)
		}
		want := int64(1)
		if _, ok := model.(*db.UploadTokens); ok {
			want = 0
		}
		if count != want {
			t.Fatalf("transaction leaked %T rows: got %d, want %d", model, count, want)
		}
	}
}
