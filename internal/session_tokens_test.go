package internal

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/BatteredBunny/hostling/internal/db"
)

func TestSessionSecretsAreHashedAndDomainSeparated(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.registerAdmin()
	secret := oauthCookie(admin, AUTH_COOKIE)
	if !strings.HasPrefix(secret, "hl_session_") || len(secret) != 54 {
		t.Fatalf("unexpected session format: %q", secret)
	}
	hash, err := db.HashSessionToken(secret)
	if err != nil {
		t.Fatal(err)
	}
	var record db.SessionTokens
	if err := ts.app.db.Where("token_hash = ?", hash).First(&record).Error; err != nil {
		t.Fatal(err)
	}
	if len(record.TokenHash) != 64 || record.TokenHash == secret {
		t.Fatal("session secret was not hashed")
	}
	remaining := time.Until(record.ExpiryDate)
	if remaining < db.SessionTokenDuration-time.Minute || remaining > db.SessionTokenDuration {
		t.Fatalf("unexpected session lifetime: %s", remaining)
	}
	if _, err := db.HashUploadToken(secret); err == nil {
		t.Fatal("session secret accepted as upload token")
	}
	upload := admin.newUploadToken("separate namespace")
	for _, credential := range []string{hash, upload, "hl_session_invalid"} {
		client := ts.newClient()
		oauthSetCookie(client, AUTH_COOKIE, credential)
		if status, body := client.get("/api/account/files"); status != http.StatusUnauthorized {
			t.Errorf("credential %q authenticated: %d %s", credential, status, body)
		}
	}
	if status, _ := ts.newClient().uploadStatus(formField{"upload_token", secret}); status != http.StatusUnauthorized {
		t.Fatalf("session secret accepted as upload token: %d", status)
	}
}

func TestSessionRevokedBetweenAuthenticationAndFileCreation(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.registerAdmin()
	secret := oauthCookie(admin, AUTH_COOKIE)
	if _, err := ts.app.db.GetAccountBySessionToken(secret); err != nil {
		t.Fatal(err)
	}
	hash, _ := db.HashSessionToken(secret)
	if err := ts.app.db.DeleteSession(secret); err != nil {
		t.Fatal(err)
	}
	if err := ts.app.db.CreateFileEntry(
		db.CreateFileEntryInput{SessionTokenHash: hash, Files: db.Files{FileName: "revoked-session.png"}},
	); err == nil {
		t.Fatal("revoked session created a file")
	}
	var count int64
	if err := ts.app.db.Model(&db.Files{}).
		Where("file_name = ?", "revoked-session.png").
		Count(&count).
		Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("revoked session left file record")
	}
}

func TestSessionExpiryRecheckedDuringFileCreation(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.registerAdmin()
	secret := oauthCookie(admin, AUTH_COOKIE)
	if _, err := ts.app.db.GetAccountBySessionToken(secret); err != nil {
		t.Fatal(err)
	}
	hash, _ := db.HashSessionToken(secret)
	if err := ts.app.db.Model(&db.SessionTokens{}).
		Where("token_hash = ?", hash).
		Update("expiry_date", time.Now().Add(-time.Second)).
		Error; err != nil {
		t.Fatal(err)
	}
	if err := ts.app.db.CreateFileEntry(
		db.CreateFileEntryInput{SessionTokenHash: hash, Files: db.Files{FileName: "expired-session.png"}},
	); err == nil {
		t.Fatal("expired session created a file")
	}
}

func TestLegacySessionCookieStillAuthenticates(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.registerAdmin()
	legacy := "123e4567-e89b-12d3-a456-426614174000"
	hash, err := db.HashSessionToken(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if err := ts.app.db.Create(
		&db.SessionTokens{
			AccountID:  ts.adminID(),
			TokenHash:  hash,
			ExpiryDate: time.Now().Add(time.Hour),
			LastUsed:   time.Now(),
		},
	).Error; err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{legacy, strings.ToUpper(legacy), strings.ReplaceAll(legacy, "-", "")} {
		client := ts.newClient()
		oauthSetCookie(client, AUTH_COOKIE, raw)
		if status, body := client.get("/api/account/files"); status != http.StatusOK {
			t.Fatalf("legacy cookie rejected: %d %s", status, body)
		}
		if status, body := client.uploadStatus(); status != http.StatusOK {
			t.Fatalf("legacy session upload rejected: %d %s", status, body)
		}
	}
	if err := ts.app.db.DeleteSessionForAccount(legacy, ts.adminID()); err != nil {
		t.Fatal(err)
	}
	stale := ts.newClient()
	oauthSetCookie(stale, AUTH_COOKIE, legacy)
	if status, _ := stale.get("/api/account/files"); status != http.StatusUnauthorized {
		t.Fatalf("revoked legacy cookie accepted: %d", status)
	}
	if status, _ := admin.get("/api/account/files"); status != http.StatusOK {
		t.Fatal("legacy revocation removed unrelated modern session")
	}
}
