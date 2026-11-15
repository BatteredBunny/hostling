package internal

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/johannesboyne/gofakes3"
	"github.com/johannesboyne/gofakes3/backend/s3mem"
	"github.com/minio/minio-go/v7"
)

const testBucket = "hostling"

// Runs the app against an in-process S3 server instead of local disk.
func newS3TestServer(t *testing.T, proxyFiles bool) *testServer {
	t.Helper()

	backend := s3mem.New()
	if err := backend.CreateBucket(testBucket); err != nil {
		t.Fatal(err)
	}
	s3srv := httptest.NewServer(gofakes3.New(backend).Server())
	t.Cleanup(s3srv.Close)

	endpoint, err := url.Parse(s3srv.URL)
	if err != nil {
		t.Fatal(err)
	}

	return newTestServer(t, func(c *Config) {
		c.FileStorageMethod = fileStorageS3
		c.S3 = s3Config{
			AccessKeyID:     "test-access-key",
			SecretAccessKey: "test-secret-key",
			Bucket:          testBucket,
			Region:          "us-east-1",
			Endpoint:        endpoint.Host,
			Insecure:        true,
			ProxyFiles:      proxyFiles,
		}
	})
}

func (ts *testServer) s3Objects() []string {
	ts.t.Helper()

	var names []string
	for object := range ts.app.s3client.ListObjects(
		context.Background(),
		testBucket,
		minio.ListObjectsOptions{Recursive: true},
	) {
		if object.Err != nil {
			ts.t.Fatal(object.Err)
		}
		names = append(names, object.Key)
	}

	return names
}

func TestS3ProxiedRoundTrip(t *testing.T) {
	ts := newS3TestServer(t, true)
	admin := ts.registerAdmin()
	anon := ts.newClient()

	file := admin.upload()
	expectNames(t, "bucket objects", ts.s3Objects(), []string{file})

	status, body := anon.get("/" + file)
	if status != http.StatusOK || body != testDummyContent {
		t.Fatalf("expected proxied content %q, got %d: %q", testDummyContent, status, body)
	}
	if got := anon.lastHeader.Get("Content-Security-Policy"); !strings.Contains(got, "sandbox") {
		t.Errorf("proxied file missing sandbox CSP, got %q", got)
	}

	admin.deleteFile(file)
	if objects := ts.s3Objects(); len(objects) != 0 {
		t.Errorf("deleted file left objects in bucket: %v", objects)
	}
	if status, body := anon.get("/" + file); status != http.StatusTemporaryRedirect {
		t.Errorf("expected 307 for deleted file, got %d: %s", status, body)
	}
}

func TestS3PresignedRedirect(t *testing.T) {
	ts := newS3TestServer(t, false)
	admin := ts.registerAdmin()
	anon := ts.newClient()

	file := admin.upload()
	if status, body := anon.get("/" + file); status != http.StatusTemporaryRedirect {
		t.Fatalf("expected 307 to presigned URL, got %d: %s", status, body)
	}
	location := anon.lastHeader.Get("Location")
	presigned, err := url.Parse(location)
	if err != nil {
		t.Fatal(err)
	}
	if presigned.Query().Get("X-Amz-Signature") == "" ||
		!strings.HasPrefix(presigned.Query().Get("response-content-disposition"), "inline") {
		t.Errorf("unexpected presigned URL: %s", location)
	}

	resp, err := http.Get(location)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK || string(content) != testDummyContent {
		t.Errorf("expected presigned content %q, got %d: %q", testDummyContent, resp.StatusCode, content)
	}
}

func TestS3AccountDeletionPurgesBucket(t *testing.T) {
	ts := newS3TestServer(t, true)
	admin := ts.registerAdmin()
	user, _ := ts.registerUser(admin)

	kept := admin.upload()
	user.upload()
	user.uploadSized(512)
	if got := len(ts.s3Objects()); got != 3 {
		t.Fatalf("expected 3 objects after uploads, got %d", got)
	}

	user.expectStatus(http.StatusOK, http.MethodDelete, "/api/account/", nil)

	expectNames(t, "bucket objects after account deletion", ts.s3Objects(), []string{kept})
}
