package internal

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/BatteredBunny/hostling/internal/db"
)

func TestFileDeletionRemovesBlob(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.registerAdmin()
	anon := ts.newClient()
	small := admin.upload()
	large := admin.uploadSized(1024)

	if !ts.blobExists(large) {
		t.Fatal("expected blob on disk after upload")
	}
	admin.deleteFile(large)

	expectNames(t, "listing", fileNames(admin.listFiles("").Files), []string{small})
	if got := admin.stats().Count; got != 1 {
		t.Errorf("expected 1 file in stats, got %d", got)
	}
	// Deleted file must redirect (must look the same as a private file)
	if status, body := anon.get("/" + large); status != http.StatusTemporaryRedirect {
		t.Errorf("expected 307 for deleted file, got %d: %s", status, body)
	}
	if ts.blobExists(large) {
		t.Errorf("blob should be removed from disk")
	}

	admin.deleteFile(small)
	if got := admin.listFiles("").Count; got != 0 {
		t.Errorf("expected 0 files, got %d", got)
	}
}

func TestBulkDelete(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.registerAdmin()
	files := []string{admin.upload(), admin.upload(), admin.upload()}

	admin.expectStatus(http.StatusOK, http.MethodDelete, "/api/account/files", nil)

	if got := admin.listFiles("").Count; got != 0 {
		t.Errorf("expected 0 files after bulk delete, got %d", got)
	}
	for _, f := range files {
		if ts.blobExists(f) {
			t.Errorf("blob %s should be removed from disk", f)
		}
	}
}

func TestViewCounter(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.registerAdmin()
	anon := ts.newClient()

	t.Run("GET counts once per IP", func(t *testing.T) {
		f := admin.upload()
		if got := admin.fileRecord(f).ViewsCount; got != 0 {
			t.Fatalf("expected 0 views on fresh upload, got %d", got)
		}

		for i, want := range []uint{1, 1} {
			if status, body := anon.get("/" + f); status != http.StatusOK || body != testDummyContent {
				t.Fatalf("expected 200 with content, got %d: %s", status, body)
			}
			if got := admin.fileRecord(f).ViewsCount; got != want {
				t.Errorf("after GET #%d: expected %d views, got %d", i+1, want, got)
			}
		}
	})

	t.Run("HEAD does not count", func(t *testing.T) {
		f := admin.upload()
		if status, body := anon.request(http.MethodHead, "/"+f, nil); status != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", status, body)
		}
		if got := admin.fileRecord(f).ViewsCount; got != 0 {
			t.Errorf("HEAD must not bump views, got %d", got)
		}
	})

	t.Run("Range does not count", func(t *testing.T) {
		f := admin.upload()
		status, body := anon.request(http.MethodGet, "/"+f, map[string]string{"Range": "bytes=0-5"})
		if status != http.StatusPartialContent {
			t.Fatalf("expected 206, got %d: %s", status, body)
		}
		if body != testDummyContent[:6] {
			t.Errorf("expected partial content %q, got %q", testDummyContent[:6], body)
		}
		if got := admin.fileRecord(f).ViewsCount; got != 0 {
			t.Errorf("Range request must not bump views, got %d", got)
		}
	})
}

func TestFutureExpiryHappyPath(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.registerAdmin()
	anon := ts.newClient()

	future := time.Now().Add(24 * time.Hour).Unix()
	f := admin.upload(formField{"expiry_timestamp", strconv.FormatInt(future, 10)})

	record := admin.fileRecord(f)
	if record.ExpiryDate.IsZero() {
		t.Fatalf("expected non-zero ExpiryDate")
	}
	if record.ExpiryDate.Unix() != future {
		t.Errorf("expected ExpiryDate %d, got %d", future, record.ExpiryDate.Unix())
	}

	// File is reachable until it expires
	if status, body := anon.get("/" + f); status != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", status, body)
	}
}

func TestExpiryDateFormField(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.registerAdmin()

	f := admin.upload(formField{"expiry_date", "2099-12-31"})
	if got := admin.fileRecord(f).ExpiryDate.UTC().Format("2006-01-02"); got != "2099-12-31" {
		t.Errorf("expected ExpiryDate on 2099-12-31, got %s", got)
	}
}

func TestUploadTokenLifecycle(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.registerAdmin()
	anon := ts.newClient()

	token := admin.newUploadToken("test-token")
	if len(token) != 53 {
		t.Fatalf("expected random upload token, got %q", token)
	}

	file := anon.upload(formField{"upload_token", token})
	if file == "" {
		t.Fatal("expected upload path")
	}
	if got := admin.listFiles("").Count; got != 1 {
		t.Errorf("expected token upload to land in the account, got %d files", got)
	}

	admin.expectStatus(
		http.StatusOK,
		http.MethodDelete,
		"/api/account/upload_tokens",
		url.Values{"id": {admin.uploadTokenID(token)}},
	)

	if status, body := anon.uploadStatus(formField{"upload_token", token}); status != http.StatusUnauthorized {
		t.Errorf("expected 401 after token deletion, got %d: %s", status, body)
	}
}

func TestFileExpiry(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.registerAdmin()
	anon := ts.newClient()

	f := admin.upload(formField{"expiry_timestamp", strconv.FormatInt(time.Now().Add(time.Hour).Unix(), 10)})
	keep := admin.upload()

	if status, body := anon.get("/" + f); status != http.StatusOK {
		t.Fatalf("expected file reachable before expiry, got %d: %s", status, body)
	}

	// Move the expiry into the past instead of sleeping
	if err := ts.app.db.Model(&db.Files{}).
		Where("file_name = ?", f).
		Update("expiry_date", time.Now().Add(-time.Minute)).Error; err != nil {
		t.Fatal(err)
	}

	// Expired files are invisible to reads even before the cleanup sweep runs
	if status, body := anon.get("/" + f); status != http.StatusTemporaryRedirect {
		t.Errorf("expected 307 for expired file, got %d: %s", status, body)
	}
	expectNames(t, "listing", fileNames(admin.listFiles("").Files), []string{keep})
	if !ts.blobExists(f) || ts.fileRowCount(f) != 1 {
		t.Fatal("expected row and blob to still exist before the cleanup sweep")
	}

	ts.app.CleanUpJob(context.Background())

	if ts.fileRowCount(f) != 0 {
		t.Errorf("expired file row should be removed by cleanup")
	}
	if ts.blobExists(f) {
		t.Errorf("expired file blob should be removed by cleanup")
	}
	if !ts.blobExists(keep) || ts.fileRowCount(keep) != 1 {
		t.Errorf("unexpired file must survive cleanup")
	}
}
