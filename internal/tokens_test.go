package internal

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/BatteredBunny/hostling/internal/db"
	"github.com/google/uuid"
)

func TestTokenListsRequireSession(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.registerAdmin()
	token := admin.newUploadToken("uploader")
	anon := ts.newClient()
	for _, path := range []string{"/api/account/upload_tokens", "/api/account/invite_code"} {
		for _, query := range []string{"", "?upload_token=" + token} {
			if status, body := anon.get(path + query); status != http.StatusUnauthorized {
				t.Errorf("%s%s: expected 401, got %d: %s", path, query, status, body)
			}
		}
	}
}

func TestTokenListsEmptyAndUncached(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.registerAdmin()
	for _, path := range []string{"/api/account/upload_tokens", "/api/account/invite_code"} {
		status, body := admin.get(path)
		if status != http.StatusOK || body != "[]" {
			t.Errorf("%s: expected 200 and [], got %d: %s", path, status, body)
		}
		if got := admin.lastHeader.Get("Cache-Control"); got != "no-store" {
			t.Errorf("%s: expected no-store, got %q", path, got)
		}
	}
}

func TestUploadTokenManagementAPI(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.registerAdmin()
	other, _ := ts.registerUser(admin)
	token := admin.newUploadToken("  Screenshot uploader  ")
	otherToken := other.newUploadToken("Other account")

	var tokens []db.UiUploadToken
	admin.getJSON("/api/account/upload_tokens", &tokens)
	if len(tokens) != 1 || tokens[0].Token.String() != token || tokens[0].Nickname != "Screenshot uploader" ||
		tokens[0].LastUsed != nil {
		t.Fatalf("unexpected token list: %+v", tokens)
	}
	other.getJSON("/api/account/upload_tokens", &tokens)
	if len(tokens) != 1 || tokens[0].Token.String() != otherToken {
		t.Fatalf("other account's list contains the wrong tokens: %+v", tokens)
	}

	// A different account cannot revoke this token.
	other.formRequest(http.MethodDelete, "/api/account/upload_tokens", url.Values{"upload_token": {token}})
	admin.getJSON("/api/account/upload_tokens", &tokens)
	if len(tokens) != 1 || tokens[0].Token.String() != token {
		t.Fatal("cross-account deletion removed a token")
	}

	ts.newClient().upload(formField{"upload_token", token})
	admin.getJSON("/api/account/upload_tokens", &tokens)
	if len(tokens) != 1 || tokens[0].LastUsed == nil {
		t.Fatal("token usage was not reflected in listing")
	}

	admin.expectStatus(
		http.StatusOK,
		http.MethodDelete,
		"/api/account/upload_tokens",
		url.Values{"upload_token": {token}},
	)
	admin.getJSON("/api/account/upload_tokens", &tokens)
	if len(tokens) != 0 {
		t.Fatalf("deleted token is still listed: %+v", tokens)
	}
	if status, body := ts.newClient().
		uploadStatus(formField{"upload_token", token}); status != http.StatusUnauthorized {
		t.Errorf("deleted token should not authorize uploads, got %d: %s", status, body)
	}
}

func TestUploadTokenRequiresName(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.registerAdmin()
	for _, name := range []string{"", " ", "\t\n", strings.Repeat("x", maxUploadTokenNickname+1)} {
		if status, body := admin.postForm(
			"/api/account/upload_tokens",
			url.Values{"nickname": {name}},
		); status != http.StatusBadRequest {
			t.Errorf("name %q: expected 400, got %d: %s", name, status, body)
		}
	}
	if status, body := admin.postForm("/api/account/upload_tokens", nil); status != http.StatusBadRequest {
		t.Errorf("missing name: expected 400, got %d: %s", status, body)
	}
	var tokens []db.UiUploadToken
	admin.getJSON("/api/account/upload_tokens", &tokens)
	if len(tokens) != 0 {
		t.Fatal("invalid requests created tokens")
	}
	admin.newUploadToken(" <script>alert(1)</script> ")
	admin.getJSON("/api/account/upload_tokens", &tokens)
	if len(tokens) != 1 || tokens[0].Nickname != "<script>alert(1)</script>" {
		t.Fatalf("name was not preserved: %+v", tokens)
	}
}

func TestUploadTokenLimit(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.registerAdmin()
	tokens := make([]db.UploadTokens, maxUploadTokensPerAccount)
	for i := range tokens {
		tokens[i] = db.UploadTokens{AccountID: ts.adminID(), Token: uuid.New(), Nickname: "existing"}
	}
	if err := ts.app.db.Create(&tokens).Error; err != nil {
		t.Fatal(err)
	}
	if status, body := admin.postForm(
		"/api/account/upload_tokens",
		url.Values{"nickname": {"one too many"}},
	); status != http.StatusBadRequest {
		t.Errorf("expected 400 at limit, got %d: %s", status, body)
	}
}

func TestUploadTokenDeleteValidation(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.registerAdmin()
	if status, body := admin.formRequest(
		http.MethodDelete,
		"/api/account/upload_tokens",
		url.Values{"upload_token": {"not-a-uuid"}},
	); status != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", status, body)
	}
}

func TestInviteCodeListingIsScopedAndActive(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.registerAdmin()
	other, otherID := ts.registerUser(admin)
	own := admin.giveInviteCode(ts.adminID(), 3)
	otherCode := admin.giveInviteCode(otherID, 2)
	expired := admin.giveInviteCode(ts.adminID(), 1)
	exhausted := admin.giveInviteCode(ts.adminID(), 1)
	if err := ts.app.db.Model(&db.InviteCodes{}).
		Where("code = ?", expired).
		Update("expiry_date", time.Now().Add(-time.Hour)).
		Error; err != nil {
		t.Fatal(err)
	}
	if err := ts.app.db.Model(&db.InviteCodes{}).Where("code = ?", exhausted).Update("uses", 0).Error; err != nil {
		t.Fatal(err)
	}
	var invites []InviteCodeAPIOutput
	admin.getJSON("/api/account/invite_code", &invites)
	if len(invites) != 1 || invites[0].Code != own || invites[0].Uses != 3 || !invites[0].ExpiryDate.After(time.Now()) {
		t.Fatalf("unexpected invite list: %+v", invites)
	}
	other.getJSON("/api/account/invite_code", &invites)
	if len(invites) != 1 || invites[0].Code != otherCode {
		t.Fatalf("other account's invite list is wrong: %+v", invites)
	}
	admin.expectStatus(http.StatusOK, http.MethodDelete, "/api/account/invite_code", url.Values{"invite_code": {own}})
	admin.getJSON("/api/account/invite_code", &invites)
	if len(invites) != 0 {
		t.Fatal("deleted invite is still listed")
	}
}
