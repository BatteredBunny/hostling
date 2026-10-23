package internal

import (
	"net/http"
	"net/url"
	"testing"
)

func TestUnauthenticatedAPIRejected(t *testing.T) {
	ts := newTestServer(t)
	anon := ts.newClient()

	if status, body := anon.get("/api/account/files"); status != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d: %s", status, body)
	}
}

func TestPrivateFileEnumerationSafe(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.registerAdmin()
	anon := ts.newClient()

	// Private, deleted and nonexistent files must all look the same so
	// attackers can't probe filename existence by status code.
	private := admin.upload()
	admin.togglePublic(private)
	deleted := admin.upload()
	admin.deleteFile(deleted)

	tests := []struct {
		name string
		path string
	}{
		{"private", "/" + private},
		{"deleted", "/" + deleted},
		{"nonexistent", "/doesnotexist.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if status, body := anon.get(tt.path); status != http.StatusTemporaryRedirect {
				t.Errorf("expected 307, got %d: %s", status, body)
			}
		})
	}

	t.Run("owner can still read private file", func(t *testing.T) {
		if status, body := admin.get("/" + private); status != http.StatusOK || body != testDummyContent {
			t.Errorf("expected 200 with content, got %d: %s", status, body)
		}
	})
}

func TestCrossUserIsolation(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.registerAdmin()
	other, _ := ts.registerUser(admin)
	anon := ts.newClient()

	otherFile := other.upload()
	other.togglePublic(otherFile) // private

	tests := []struct {
		name   string
		method string
		path   string
		values url.Values
	}{
		{"delete", http.MethodDelete, "/api/account/file", url.Values{"file_name": {otherFile}}},
		{"tag", http.MethodPost, "/api/account/file/tag", url.Values{"file_name": {otherFile}, "tag": {"x"}}},
		{"toggle public", http.MethodPost, "/api/account/file/public", url.Values{"file_name": {otherFile}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if status, body := admin.formRequest(tt.method, tt.path, tt.values); status != http.StatusNotFound {
				t.Errorf("expected 404, got %d: %s", status, body)
			}
		})
	}

	t.Run("unauthenticated fetch of private file", func(t *testing.T) {
		if status, body := anon.get("/" + otherFile); status != http.StatusTemporaryRedirect {
			t.Errorf("expected 307, got %d: %s", status, body)
		}
	})

	t.Run("other user's file is untouched", func(t *testing.T) {
		if got := other.listFiles("").Count; got != 1 {
			t.Errorf("expected 1 file, got %d", got)
		}
		if got := len(other.fileRecord(otherFile).Tags); got != 0 {
			t.Errorf("expected no tags, got %d", got)
		}
		if admin.listFiles("").Count != 0 {
			t.Errorf("admin should not see other user's files")
		}
	})
}

func TestHTMLPages(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.registerAdmin()
	anon := ts.newClient()

	tests := []struct {
		page   string
		authed int
		anon   int
	}{
		{"/", http.StatusOK, http.StatusOK},
		{"/login", http.StatusTemporaryRedirect, http.StatusOK},
		{"/register", http.StatusTemporaryRedirect, http.StatusOK},
		{"/gallery", http.StatusOK, http.StatusTemporaryRedirect},
		{"/settings", http.StatusOK, http.StatusTemporaryRedirect},
		{"/tokens", http.StatusOK, http.StatusTemporaryRedirect},
		{"/admin", http.StatusOK, http.StatusTemporaryRedirect},
	}

	for _, tt := range tests {
		t.Run("authed "+tt.page, func(t *testing.T) {
			if status, _ := admin.get(tt.page); status != tt.authed {
				t.Errorf("expected %d, got %d", tt.authed, status)
			}
		})
		t.Run("unauthed "+tt.page, func(t *testing.T) {
			if status, _ := anon.get(tt.page); status != tt.anon {
				t.Errorf("expected %d, got %d", tt.anon, status)
			}
		})
	}
}

func TestLogoutInvalidatesSession(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.registerAdmin()

	if status, body := admin.postForm("/logout", nil); status != http.StatusSeeOther {
		t.Fatalf("expected 303 on logout, got %d: %s", status, body)
	}

	if status, body := admin.get("/api/account/files"); status != http.StatusUnauthorized {
		t.Errorf("expected 401 after logout, got %d: %s", status, body)
	}

	// Second logout hits the already-logged-out branch
	if status, body := admin.postForm("/logout", nil); status != http.StatusSeeOther {
		t.Errorf("expected 303 for unauthenticated logout, got %d: %s", status, body)
	}
}
