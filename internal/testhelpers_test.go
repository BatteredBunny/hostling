package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BatteredBunny/hostling/internal/db"
	"github.com/gin-gonic/gin"
)

const (
	testInitialToken  = "test-token"
	testMaxUploadSize = 8192
	testDummyContent  = "123dummydata"
)

type testServer struct {
	t   *testing.T
	app *Application
	srv *httptest.Server
}

// Spins up the full app against a fresh sqlite database and data folder
func newTestServer(t *testing.T) *testServer {
	t.Helper()
	gin.SetMode(gin.TestMode)
	t.Setenv("INITIAL_REGISTER_TOKEN", testInitialToken)

	dir := t.TempDir()
	c := Config{
		DatabaseType:          "sqlite",
		DatabaseConnectionUrl: filepath.Join(dir, "hostling.db") + "?_sync=OFF&_journal=MEMORY", // no fsync, test DBs are throwaway
		DataFolder:            dir,
		FileStorageMethod:     fileStorageLocal,
		MaxUploadSize:         testMaxUploadSize,
		RateLimit:             1000,
		PublicUrl:             "http://localhost",
		Branding:              "Hostling",
	}

	app := setupRouter(&uninitializedApplication{
		config:      c,
		db:          prepareDB(c),
		s3client:    prepareStorage(c),
		RateLimiter: setupRatelimiting(c),
	}, c)

	srv := httptest.NewServer(app.Router)
	t.Cleanup(func() {
		srv.Close()
		app.Shutdown()
		if sqlDB, err := app.db.DB.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})

	return &testServer{t: t, app: app, srv: srv}
}

type testClient struct {
	t    *testing.T
	base string
	http *http.Client
}

// Client with its own cookie jar that doesn't follow redirects, so tests can assert on 3xx
func (ts *testServer) newClient() *testClient {
	jar, err := cookiejar.New(nil)
	if err != nil {
		ts.t.Fatal(err)
	}

	return &testClient{
		t:    ts.t,
		base: ts.srv.URL,
		http: &http.Client{
			Jar: jar,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

// Registers a new account with the invite code, client keeps the session cookie
func (ts *testServer) registerClient(code string) *testClient {
	ts.t.Helper()
	client := ts.newClient()
	if status, body := client.postForm("/api/auth/register", url.Values{"code": {code}}); status != http.StatusSeeOther {
		ts.t.Fatalf("register: expected 303, got %d: %s", status, body)
	}

	return client
}

// Registers the initial admin account
func (ts *testServer) registerAdmin() *testClient {
	ts.t.Helper()

	return ts.registerClient(testInitialToken)
}

func (tc *testClient) do(req *http.Request) (int, string) {
	tc.t.Helper()
	resp, err := tc.http.Do(req)
	if err != nil {
		tc.t.Fatalf("%s %s: %v", req.Method, req.URL.Path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		tc.t.Fatalf("%s %s: reading body: %v", req.Method, req.URL.Path, err)
	}

	return resp.StatusCode, string(body)
}

func (tc *testClient) get(path string) (int, string) {
	tc.t.Helper()
	req, err := http.NewRequest(http.MethodGet, tc.base+path, nil)
	if err != nil {
		tc.t.Fatal(err)
	}

	return tc.do(req)
}

func (tc *testClient) postForm(path string, values url.Values) (int, string) {
	tc.t.Helper()

	return tc.formRequest(http.MethodPost, path, values)
}

func (tc *testClient) formRequest(method, path string, values url.Values) (int, string) {
	tc.t.Helper()

	// Go only parses urlencoded bodies for POST/PUT/PATCH, so other methods (DELETE) use
	// multipart like the frontend's FormData does
	if method != http.MethodPost && method != http.MethodPut && method != http.MethodPatch {
		var fields []formField
		for k, vs := range values {
			for _, v := range vs {
				fields = append(fields, formField{k, v})
			}
		}

		return tc.multipartRequest(method, path, nil, fields...)
	}

	req, err := http.NewRequest(method, tc.base+path, strings.NewReader(values.Encode()))
	if err != nil {
		tc.t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	return tc.do(req)
}

type formField struct {
	key, value string
}

// Sends a multipart form like curl -F, fields are ordered so keys can repeat
func (tc *testClient) postMultipart(path string, fileContent []byte, fields ...formField) (int, string) {
	tc.t.Helper()

	return tc.multipartRequest(http.MethodPost, path, fileContent, fields...)
}

func (tc *testClient) multipartRequest(method, path string, fileContent []byte, fields ...formField) (int, string) {
	tc.t.Helper()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for _, f := range fields {
		if err := w.WriteField(f.key, f.value); err != nil {
			tc.t.Fatal(err)
		}
	}
	if fileContent != nil {
		part, err := w.CreateFormFile("file", "dummy.txt")
		if err != nil {
			tc.t.Fatal(err)
		}
		if _, err = part.Write(fileContent); err != nil {
			tc.t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		tc.t.Fatal(err)
	}

	req, err := http.NewRequest(method, tc.base+path, &buf)
	if err != nil {
		tc.t.Fatal(err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	return tc.do(req)
}

// Uploads the dummy file, returns status and body
func (tc *testClient) uploadStatus(fields ...formField) (int, string) {
	tc.t.Helper()

	return tc.postMultipart("/api/file/upload", []byte(testDummyContent), append([]formField{{"plain", "true"}}, fields...)...)
}

// Uploads the dummy file and returns its name, fails the test on non-200
func (tc *testClient) upload(fields ...formField) string {
	tc.t.Helper()
	status, body := tc.uploadStatus(fields...)
	if status != http.StatusOK {
		tc.t.Fatalf("upload: expected 200, got %d: %s", status, body)
	}

	return strings.TrimPrefix(body, "/")
}

// Uploads arbitrary content, returns status and body
func (tc *testClient) uploadBytesStatus(content []byte, fields ...formField) (int, string) {
	tc.t.Helper()

	return tc.postMultipart("/api/file/upload", content, append([]formField{{"plain", "true"}}, fields...)...)
}

// Uploads content of the given size, returns the file name
func (tc *testClient) uploadSized(size int, fields ...formField) string {
	tc.t.Helper()
	status, body := tc.uploadBytesStatus(bytes.Repeat([]byte("a"), size), fields...)
	if status != http.StatusOK {
		tc.t.Fatalf("upload: expected 200, got %d: %s", status, body)
	}

	return strings.TrimPrefix(body, "/")
}

// Sends a request with arbitrary method and headers, returns status and body
func (tc *testClient) request(method, path string, headers map[string]string) (int, string) {
	tc.t.Helper()
	req, err := http.NewRequest(method, tc.base+path, nil)
	if err != nil {
		tc.t.Fatal(err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return tc.do(req)
}

// Requests path and fails the test unless the status matches
func (tc *testClient) expectStatus(want int, method, path string, values url.Values) string {
	tc.t.Helper()
	status, body := tc.formRequest(method, path, values)
	if status != want {
		tc.t.Fatalf("%s %s: expected %d, got %d: %s", method, path, want, status, body)
	}

	return body
}

// GET a JSON endpoint and decode it into out, fails the test on non-200
func (tc *testClient) getJSON(path string, out any) {
	tc.t.Helper()
	status, body := tc.get(path)
	if status != http.StatusOK {
		tc.t.Fatalf("GET %s: expected 200, got %d: %s", path, status, body)
	}
	if err := json.Unmarshal([]byte(body), out); err != nil {
		tc.t.Fatalf("GET %s: decoding %q: %v", path, body, err)
	}
}

type fileListing struct {
	Files []db.Files `json:"files"`
	Count int64      `json:"count"`
}

type fileStats struct {
	Count     uint     `json:"count"`
	SizeTotal uint     `json:"size_total"`
	Tags      []string `json:"tags"`
}

func (tc *testClient) listFiles(query string) fileListing {
	tc.t.Helper()
	path := "/api/account/files"
	if query != "" {
		path += "?" + query
	}
	var out fileListing
	tc.getJSON(path, &out)

	return out
}

func (tc *testClient) stats() fileStats {
	tc.t.Helper()
	var out fileStats
	tc.getJSON("/api/account/files/stats", &out)

	return out
}

func fileNames(files []db.Files) []string {
	names := make([]string, len(files))
	for i, f := range files {
		names[i] = f.FileName
	}

	return names
}

// Finds a file in the account listing, fails the test if it's missing
func (tc *testClient) fileRecord(name string) db.Files {
	tc.t.Helper()
	for _, f := range tc.listFiles("").Files {
		if f.FileName == name {
			return f
		}
	}
	tc.t.Fatalf("file %s not in listing", name)

	return db.Files{}
}

func (tc *testClient) addTag(file, tag string) {
	tc.t.Helper()
	tc.expectStatus(http.StatusOK, http.MethodPost, "/api/account/file/tag", url.Values{"file_name": {file}, "tag": {tag}})
}

func (tc *testClient) removeTag(file, tag string) {
	tc.t.Helper()
	tc.expectStatus(http.StatusOK, http.MethodDelete, "/api/account/file/tag", url.Values{"file_name": {file}, "tag": {tag}})
}

func (tc *testClient) togglePublic(file string) {
	tc.t.Helper()
	tc.expectStatus(http.StatusOK, http.MethodPost, "/api/account/file/public", url.Values{"file_name": {file}})
}

func (tc *testClient) deleteFile(file string) {
	tc.t.Helper()
	tc.expectStatus(http.StatusOK, http.MethodDelete, "/api/account/file", url.Values{"file_name": {file}})
}

// Creates an upload token, returns it
func (tc *testClient) newUploadToken(nickname string) string {
	tc.t.Helper()

	return tc.expectStatus(http.StatusOK, http.MethodPost, "/api/account/upload_token", url.Values{"nickname": {nickname}})
}

// Admin gives an invite code to the account, returns the code
func (tc *testClient) giveInviteCode(accountID uint, uses int) string {
	tc.t.Helper()

	return tc.expectStatus(http.StatusOK, http.MethodPost, "/api/admin/give_invite_code", url.Values{
		"id":   {fmt.Sprint(accountID)},
		"uses": {fmt.Sprint(uses)},
	})
}

// Admin registers a new user through an invite code, returns the client and the new account ID
func (ts *testServer) registerUser(admin *testClient) (*testClient, uint) {
	ts.t.Helper()
	code := admin.giveInviteCode(ts.adminID(), 1)
	user := ts.registerClient(code)

	var acc db.Accounts
	if err := ts.app.db.Order("id DESC").First(&acc).Error; err != nil {
		ts.t.Fatal(err)
	}

	return user, acc.ID
}

// ID of the (only) admin account
func (ts *testServer) adminID() uint {
	ts.t.Helper()
	var acc db.Accounts
	if err := ts.app.db.Where("account_type = ?", "ADMIN").First(&acc).Error; err != nil {
		ts.t.Fatal(err)
	}

	return acc.ID
}

func (ts *testServer) blobPath(name string) string {
	return filepath.Join(ts.app.config.DataFolder, name)
}

func (ts *testServer) blobExists(name string) bool {
	ts.t.Helper()
	_, err := os.Stat(ts.blobPath(name))
	if err != nil && !os.IsNotExist(err) {
		ts.t.Fatal(err)
	}

	return err == nil
}

func (ts *testServer) fileRowCount(name string) int64 {
	ts.t.Helper()
	var n int64
	if err := ts.app.db.Model(&db.Files{}).Where("file_name = ?", name).Count(&n).Error; err != nil {
		ts.t.Fatal(err)
	}

	return n
}
