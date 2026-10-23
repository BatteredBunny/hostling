package internal

import (
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/BatteredBunny/hostling/internal/db"
)

func idValues(id uint) url.Values {
	return url.Values{"id": {strconv.FormatUint(uint64(id), 10)}}
}

func TestAdminSelfDeleteBlocked(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.registerAdmin()

	if status, body := admin.formRequest(http.MethodDelete, "/api/admin/user", idValues(ts.adminID())); status != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", status, body)
	}
	if status, body := admin.get("/api/account/files"); status != http.StatusOK {
		t.Errorf("admin should still exist, got %d: %s", status, body)
	}
}

func TestAdminEndpointsRequireAdmin(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.registerAdmin()
	user, _ := ts.registerUser(admin)

	values := idValues(ts.adminID())
	values.Set("uses", "1")
	if status, body := user.postForm("/api/admin/give_invite_code", values); status != http.StatusUnauthorized {
		t.Errorf("non-admin creating invite codes: expected 401, got %d: %s", status, body)
	}
	if status, body := user.get("/admin"); status != http.StatusTemporaryRedirect {
		t.Errorf("non-admin on /admin page: expected 307, got %d: %s", status, body)
	}
}

func TestInviteCodeDeletion(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.registerAdmin()

	code := admin.giveInviteCode(ts.adminID(), 1)
	admin.expectStatus(http.StatusOK, http.MethodDelete, "/api/account/invite_code", url.Values{"invite_code": {code}})

	if status, body := ts.newClient().postForm("/api/auth/register", url.Values{"code": {code}}); status != http.StatusBadRequest {
		t.Errorf("expected 400 for deleted invite, got %d: %s", status, body)
	}
}

func TestInviteUseCounter(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.registerAdmin()

	code := admin.giveInviteCode(ts.adminID(), 2)
	ts.registerClient(code)
	ts.registerClient(code)

	if status, body := ts.newClient().postForm("/api/auth/register", url.Values{"code": {code}}); status != http.StatusBadRequest {
		t.Errorf("expected 400 for exhausted invite, got %d: %s", status, body)
	}
}

func TestAdminForceDeleteFiles(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.registerAdmin()
	user, userID := ts.registerUser(admin)

	file := user.upload()
	user.togglePublic(file)
	if !ts.blobExists(file) {
		t.Fatal("expected blob on disk after upload")
	}

	admin.expectStatus(http.StatusOK, http.MethodDelete, "/api/admin/files", idValues(userID))

	if got := user.listFiles("").Count; got != 0 {
		t.Errorf("expected 0 files after admin wipe, got %d", got)
	}
	if status, body := ts.newClient().get("/" + file); status != http.StatusTemporaryRedirect {
		t.Errorf("expected 307 for admin-deleted file, got %d: %s", status, body)
	}
	if ts.blobExists(file) {
		t.Errorf("blob should be removed from disk")
	}
	if ts.fileRowCount(file) != 0 {
		t.Errorf("file row should be removed")
	}
}

func TestAdminRevokesUploadTokens(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.registerAdmin()
	user, userID := ts.registerUser(admin)

	token := user.newUploadToken("user-token")
	anon := ts.newClient()
	if status, body := anon.uploadStatus(formField{"upload_token", token}); status != http.StatusOK {
		t.Fatalf("expected upload with token to work, got %d: %s", status, body)
	}

	admin.expectStatus(http.StatusOK, http.MethodDelete, "/api/admin/upload_tokens", idValues(userID))

	if status, body := anon.uploadStatus(formField{"upload_token", token}); status != http.StatusUnauthorized {
		t.Errorf("expected 401 for admin-revoked upload token, got %d: %s", status, body)
	}
}

func TestAdminRevokesSessions(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.registerAdmin()
	user, userID := ts.registerUser(admin)

	admin.expectStatus(http.StatusOK, http.MethodDelete, "/api/admin/sessions", idValues(userID))

	if status, body := user.get("/api/account/files"); status != http.StatusUnauthorized {
		t.Errorf("expected 401 after admin session wipe, got %d: %s", status, body)
	}
	if status, body := admin.get("/api/account/files"); status != http.StatusOK {
		t.Errorf("admin session should be unaffected, got %d: %s", status, body)
	}
}

func TestAdminDeleteUser(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.registerAdmin()
	user, userID := ts.registerUser(admin)
	file := user.upload()

	admin.expectStatus(http.StatusOK, http.MethodDelete, "/api/admin/user", idValues(userID))

	// Account is gone, admin operations targeting it return 404
	values := idValues(userID)
	values.Set("uses", "1")
	if status, body := admin.postForm("/api/admin/give_invite_code", values); status != http.StatusNotFound {
		t.Errorf("expected 404 for admin op on deleted user, got %d: %s", status, body)
	}
	if status, body := user.get("/api/account/files"); status != http.StatusUnauthorized {
		t.Errorf("expected 401 for deleted user's session, got %d: %s", status, body)
	}
	if ts.blobExists(file) {
		t.Errorf("blob should be removed from disk")
	}
}

func TestAccountSelfDelete(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.registerAdmin()
	user, _ := ts.registerUser(admin)
	file := user.upload()

	user.expectStatus(http.StatusOK, http.MethodDelete, "/api/account/", nil)

	if status, body := user.get("/api/account/files"); status != http.StatusUnauthorized {
		t.Errorf("expected 401 after self-delete, got %d: %s", status, body)
	}
	if status, body := ts.newClient().get("/" + file); status != http.StatusTemporaryRedirect {
		t.Errorf("expected 307 for deleted user's file, got %d: %s", status, body)
	}
	if ts.blobExists(file) || ts.fileRowCount(file) != 0 {
		t.Errorf("file blob and row should be removed")
	}
}

// Expired files that cleanup hasn't swept yet must not be orphaned on disk when the account goes away
func TestAccountDeletionRemovesExpiredBlobs(t *testing.T) {
	tests := []struct {
		name   string
		delete func(ts *testServer, admin, user *testClient, userID uint)
	}{
		{"self delete", func(_ *testServer, _, user *testClient, _ uint) {
			user.expectStatus(http.StatusOK, http.MethodDelete, "/api/account/", nil)
		}},
		{"admin delete", func(_ *testServer, admin, _ *testClient, userID uint) {
			admin.expectStatus(http.StatusOK, http.MethodDelete, "/api/admin/user", idValues(userID))
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := newTestServer(t)
			admin := ts.registerAdmin()
			user, userID := ts.registerUser(admin)

			expired := user.upload()
			if err := ts.app.db.Model(&db.Files{}).
				Where("file_name = ?", expired).
				Update("expiry_date", time.Now().Add(-time.Minute)).Error; err != nil {
				t.Fatal(err)
			}

			tt.delete(ts, admin, user, userID)

			if ts.blobExists(expired) {
				t.Errorf("expired file blob was orphaned on disk")
			}
			if ts.fileRowCount(expired) != 0 {
				t.Errorf("expired file row should be removed")
			}
		})
	}
}
