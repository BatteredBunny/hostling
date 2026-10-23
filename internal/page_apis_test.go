package internal

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/BatteredBunny/hostling/internal/db"
)

func TestPageAPIAccess(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.registerAdmin()
	user, _ := ts.registerUser(admin)
	for _, path := range []string{"/api/account/settings", "/api/admin/overview"} {
		if status, _ := ts.newClient().get(path); status != http.StatusUnauthorized {
			t.Errorf("%s: anonymous status %d", path, status)
		}
	}
	if status, _ := user.get("/api/admin/overview"); status != http.StatusUnauthorized {
		t.Fatalf("regular user accessed admin data: %d", status)
	}
	for _, client := range []*testClient{admin, user} {
		if status, body := client.get("/api/account/settings"); status != http.StatusOK {
			t.Fatalf("settings status %d: %s", status, body)
		}
		if client.lastHeader.Get("Cache-Control") != "no-store" {
			t.Fatal("settings must not be cached")
		}
	}
	if status, _ := user.get("/admin"); status != http.StatusTemporaryRedirect {
		t.Fatalf("regular user accessed admin page: %d", status)
	}
}

func TestSettingsAPIProviders(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.registerAdmin()
	var data struct {
		Providers   []ProviderInfo `json:"providers"`
		NoProviders bool           `json:"no_providers_configured"`
		Unlinked    bool           `json:"unlinked_account"`
	}
	admin.getJSON("/api/account/settings", &data)
	if !data.NoProviders || !data.Unlinked || data.Providers == nil || len(data.Providers) != 0 {
		t.Fatalf("unexpected empty settings: %+v", data)
	}
	ts.app.providersMutex.Lock()
	ts.app.configuredProviders = []string{"github", "openid-connect"}
	ts.app.providersMutex.Unlock()
	if err := ts.app.db.Model(&db.Accounts{}).
		Where("id = ?", ts.adminID()).
		Updates(map[string]any{"github_id": 42, "github_username": "example", "oidc_id": "private-subject", "oidc_username": "Example OIDC"}).
		Error; err != nil {
		t.Fatal(err)
	}
	admin.getJSON("/api/account/settings", &data)
	if data.NoProviders || data.Unlinked || len(data.Providers) != 2 {
		t.Fatalf("unexpected linked settings: %+v", data)
	}
	if !data.Providers[0].IsLinked ||
		data.Providers[0].ProfileURL != "https://github.com/example" ||
		data.Providers[1].Username != "Example OIDC" {
		t.Fatalf("unexpected providers: %+v", data.Providers)
	}
	_, body := admin.get("/api/account/settings")
	if strings.Contains(body, "private-subject") {
		t.Fatal("settings exposes provider subject")
	}
	if strings.Contains(body, `"Icon"`) || strings.Contains(body, `"LinkingText"`) {
		t.Fatal("settings includes frontend presentation fields")
	}
	user, _ := ts.registerUser(admin)
	user.getJSON("/api/account/settings", &data)
	if !data.Unlinked || data.Providers[0].IsLinked || data.Providers[1].IsLinked {
		t.Fatal("provider state leaked across accounts")
	}
}

func TestAdminOverviewAPI(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.registerAdmin()
	user, userID := ts.registerUser(admin)
	user.upload()
	var data struct {
		Accounts      []AccountStats `json:"accounts"`
		MaxUploadSize int64          `json:"max_upload_size"`
		Storage       string         `json:"file_storage_method"`
		NoProviders   bool           `json:"no_providers_configured"`
		Failed        []string       `json:"failed_providers"`
	}
	ts.app.providersMutex.Lock()
	ts.app.failedProviders = []string{"openid-connect"}
	ts.app.providersMutex.Unlock()
	status, body := admin.get("/api/admin/overview")
	if status != http.StatusOK {
		t.Fatalf("status %d: %s", status, body)
	}
	if err := json.Unmarshal([]byte(body), &data); err != nil {
		t.Fatal(err)
	}
	if len(data.Accounts) != 2 || data.MaxUploadSize != testMaxUploadSize ||
		data.Storage != string(fileStorageLocal) ||
		!data.NoProviders ||
		len(data.Failed) != 1 {
		t.Fatalf("unexpected overview: %+v", data)
	}
	for _, account := range data.Accounts {
		if account.You != (account.ID == ts.adminID()) {
			t.Fatal("incorrect self flag")
		}
		if account.ID == userID &&
			(account.FilesUploaded != 1 || account.SpaceUsed != uint(len(testDummyContent)) || account.SessionsCount != 1 || account.InvitedBy == "system") {
			t.Fatalf("incorrect aggregation: %+v", account)
		}
	}
	if admin.lastHeader.Get("Cache-Control") != "no-store" {
		t.Fatal("overview must not be cached")
	}
	for _, key := range []string{"GithubID", "OIDCID", "TokenHash", "DatabaseConnectionUrl"} {
		if strings.Contains(body, key) {
			t.Fatalf("overview exposes %s", key)
		}
	}
}
