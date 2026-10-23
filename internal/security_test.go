package internal

import (
	"mime"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/BatteredBunny/hostling/internal/db"
)

func TestAdminEndpointsRejectNonAdmins(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.registerAdmin()
	user, _ := ts.registerUser(admin)
	adminFile := admin.upload()
	adminID := idValues(ts.adminID())

	giveInvite := idValues(ts.adminID())
	giveInvite.Set("uses", "1")

	tests := []struct {
		method string
		path   string
		values url.Values
	}{
		{http.MethodDelete, "/api/admin/user", adminID},
		{http.MethodDelete, "/api/admin/files", adminID},
		{http.MethodDelete, "/api/admin/sessions", adminID},
		{http.MethodDelete, "/api/admin/upload_tokens", adminID},
		{http.MethodPost, "/api/admin/give_invite_code", giveInvite},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			if status, body := user.formRequest(tt.method, tt.path, tt.values); status != http.StatusUnauthorized {
				t.Errorf("expected 401, got %d: %s", status, body)
			}
		})
	}

	// None of the attempts above may have had an effect on the admin
	if status, body := admin.get("/api/account/files"); status != http.StatusOK {
		t.Fatalf("admin session should still work, got %d: %s", status, body)
	}
	if ts.fileRowCount(adminFile) != 1 || !ts.blobExists(adminFile) {
		t.Errorf("admin's file should be untouched")
	}
}

func TestPrivateFileHiddenFromOtherUsers(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.registerAdmin()
	other, _ := ts.registerUser(admin)

	private := admin.upload()
	admin.togglePublic(private)

	if status, body := other.get("/" + private); status != http.StatusTemporaryRedirect {
		t.Errorf("other logged in user: expected 307, got %d: %s", status, body)
	}
	if status, body := admin.get("/" + private); status != http.StatusOK || body != testDummyContent {
		t.Errorf("owner: expected 200 with content, got %d: %s", status, body)
	}
}

func TestServedUploadHeaders(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.registerAdmin()
	anon := ts.newClient()

	tests := []struct {
		name        string
		content     string
		disposition string
	}{
		{"plain text", testDummyContent, "inline"},
		{"html", "<!DOCTYPE html><html><body><script>alert(1)</script></body></html>", "attachment"},
		{"svg", `<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`, "attachment"},
		{"xml", `<?xml version="1.0"?><root/>`, "attachment"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, body := admin.uploadBytesStatus([]byte(tt.content))
			if status != http.StatusOK {
				t.Fatalf("upload: expected 200, got %d: %s", status, body)
			}
			file := strings.TrimPrefix(body, "/")

			if status, _ := anon.get("/" + file); status != http.StatusOK {
				t.Fatalf("expected 200, got %d", status)
			}
			h := anon.lastHeader

			if got := h.Get("X-Content-Type-Options"); got != "nosniff" {
				t.Errorf("X-Content-Type-Options: expected nosniff, got %q", got)
			}
			if got := h.Get("Content-Security-Policy"); !strings.HasPrefix(got, "sandbox;") || !strings.Contains(got, "default-src 'none'") {
				t.Errorf("Content-Security-Policy: expected sandbox with default-src 'none', got %q", got)
			}
			if got := h.Get("Referrer-Policy"); got != "no-referrer" {
				t.Errorf("Referrer-Policy: expected no-referrer, got %q", got)
			}

			disposition, _, err := mime.ParseMediaType(h.Get("Content-Disposition"))
			if err != nil {
				t.Fatalf("invalid Content-Disposition %q: %v", h.Get("Content-Disposition"), err)
			}
			if disposition != tt.disposition {
				t.Errorf("expected %s disposition, got %s", tt.disposition, disposition)
			}
		})
	}
}

func FuzzFormatContentDisposition(f *testing.F) {
	for _, s := range []string{
		"file.txt",
		`quo"te.txt`,
		`back\slash.txt`,
		"crlf\r\nSet-Cookie: x=y.txt",
		"null\x00byte",
		"ünïcödé ファイル.png",
		"semi;colon.txt",
		"x;filename=evil.html",
		"percent%20.txt",
		"\xff\xfe invalid utf8",
		"",
	} {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, filename string) {
		header := formatContentDisposition("inline", filename)

		if strings.ContainsAny(header, "\r\n\x00") {
			t.Fatalf("header contains control characters: %q", header)
		}
		for i := 0; i < len(header); i++ {
			if header[i] < ' ' || header[i] > '~' {
				t.Fatalf("header contains non-printable-ASCII byte %q at %d: %q", header[i], i, header)
			}
		}

		disposition, params, err := mime.ParseMediaType(header)
		if err != nil {
			t.Fatalf("header %q doesn't parse: %v", header, err)
		}
		if disposition != "inline" {
			t.Fatalf("expected inline disposition, got %q from %q", disposition, header)
		}
		// filename* carries the exact original name, the parser prefers it over filename
		if utf8.ValidString(filename) && params["filename"] != filename {
			t.Fatalf("filename round trip: expected %q, got %q from %q", filename, params["filename"], header)
		}
	})
}

func TestReverseProxyClientIP(t *testing.T) {
	// httptest clients connect from 127.0.0.1, the rate limiter allows 1 request per second per client IP
	tests := []struct {
		name          string
		behindProxy   bool
		trustedProxy  string
		separateLimit bool // whether a different X-Forwarded-For gets its own rate limit bucket
	}{
		{"not behind proxy ignores forwarded headers", false, "", false},
		{"untrusted proxy ignores forwarded headers", true, "10.0.0.1", false},
		{"trusted proxy honours forwarded headers", true, "127.0.0.1", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := newTestServer(t, func(c *Config) {
				c.RateLimit = 1
				c.BehindReverseProxy = tt.behindProxy
				c.TrustedProxy = tt.trustedProxy
			})
			client := ts.newClient()

			get := func(forwardedFor string) int {
				status, _ := client.request(http.MethodGet, "/doesnotexist", map[string]string{
					"X-Forwarded-For": forwardedFor,
					"X-Real-IP":       forwardedFor,
				})

				return status
			}

			if status := get("203.0.113.1"); status == http.StatusTooManyRequests {
				t.Fatalf("first request should not be rate limited")
			}

			status := get("203.0.113.2")
			if tt.separateLimit && status == http.StatusTooManyRequests {
				t.Errorf("forwarded client IP should get its own rate limit bucket, got 429")
			}
			if !tt.separateLimit && status != http.StatusTooManyRequests {
				t.Errorf("spoofed X-Forwarded-For must not bypass the rate limit, got %d", status)
			}
		})
	}
}

func TestViewCounterIgnoresSpoofedForwardedFor(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.registerAdmin()
	anon := ts.newClient()
	file := admin.upload()

	for _, ip := range []string{"203.0.113.1", "203.0.113.2", "203.0.113.3"} {
		if status, _ := anon.request(http.MethodGet, "/"+file, map[string]string{"X-Forwarded-For": ip}); status != http.StatusOK {
			t.Fatalf("expected 200, got %d", status)
		}
	}

	if got := admin.fileRecord(file).ViewsCount; got != 1 {
		t.Errorf("spoofed X-Forwarded-For must not inflate views: expected 1, got %d", got)
	}
}

func TestAuthCookieAttributes(t *testing.T) {
	for _, secure := range []bool{false, true} {
		t.Run(map[bool]string{false: "http", true: "https"}[secure], func(t *testing.T) {
			ts := newTestServer(t, func(c *Config) { c.CookieSecure = secure })
			client := ts.newClient()

			if status, body := client.postForm("/api/auth/register", url.Values{"code": {testInitialToken}}); status != http.StatusSeeOther {
				t.Fatalf("register: expected 303, got %d: %s", status, body)
			}

			var auth *http.Cookie
			for _, c := range (&http.Response{Header: client.lastHeader}).Cookies() {
				if c.Name == AUTH_COOKIE {
					auth = c
				}
			}
			if auth == nil {
				t.Fatalf("no %s cookie set, got headers %v", AUTH_COOKIE, client.lastHeader.Values("Set-Cookie"))
			}

			if !auth.HttpOnly {
				t.Errorf("auth cookie must be HttpOnly")
			}
			if auth.SameSite != http.SameSiteLaxMode {
				t.Errorf("auth cookie must be SameSite=Lax, got %v", auth.SameSite)
			}
			if auth.Secure != secure {
				t.Errorf("auth cookie Secure: expected %v, got %v", secure, auth.Secure)
			}
			if auth.Path != "/" {
				t.Errorf("auth cookie path: expected /, got %q", auth.Path)
			}
		})
	}
}

func TestExpiredSessionRejected(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.registerAdmin()
	private := admin.upload()
	admin.togglePublic(private)

	if err := ts.app.db.Model(&db.SessionTokens{}).
		Where("1 = 1").
		Update("expiry_date", time.Now().Add(-time.Minute)).Error; err != nil {
		t.Fatal(err)
	}

	if status, body := admin.get("/api/account/files"); status != http.StatusUnauthorized {
		t.Errorf("API: expected 401, got %d: %s", status, body)
	}
	if status, body := admin.uploadStatus(); status != http.StatusUnauthorized {
		t.Errorf("upload: expected 401, got %d: %s", status, body)
	}
	if status, body := admin.get("/" + private); status != http.StatusTemporaryRedirect {
		t.Errorf("own private file: expected 307, got %d: %s", status, body)
	}
	for _, page := range []string{"/gallery", "/settings", "/tokens", "/admin"} {
		if status, _ := admin.get(page); status != http.StatusTemporaryRedirect {
			t.Errorf("%s: expected 307, got %d", page, status)
		}
	}
}

func TestUploadTokenOnlyGrantsUpload(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.registerAdmin()
	adminFile := admin.upload()
	token := admin.newUploadToken("scoped")
	tokenOnly := ts.newClient()

	giveInvite := idValues(ts.adminID())
	giveInvite.Set("uses", "1")
	giveInvite.Set("upload_token", token)

	tests := []struct {
		method string
		path   string
		values url.Values
	}{
		{http.MethodGet, "/api/account/files?upload_token=" + token, nil},
		{http.MethodDelete, "/api/account/files", url.Values{"upload_token": {token}}},
		{http.MethodPost, "/api/account/upload_token", url.Values{"upload_token": {token}, "nickname": {"x"}}},
		{http.MethodDelete, "/api/account/", url.Values{"upload_token": {token}}},
		{http.MethodPost, "/api/admin/give_invite_code", giveInvite},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			if status, body := tokenOnly.formRequest(tt.method, tt.path, tt.values); status != http.StatusUnauthorized {
				t.Errorf("expected 401, got %d: %s", status, body)
			}
		})
	}

	if ts.fileRowCount(adminFile) != 1 {
		t.Errorf("admin's file should be untouched")
	}
	if status, body := tokenOnly.uploadStatus(formField{"upload_token", token}); status != http.StatusOK {
		t.Errorf("token should still allow uploads, got %d: %s", status, body)
	}
}

func TestCannotRevokeOtherUsersCredentials(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.registerAdmin()
	other, _ := ts.registerUser(admin)

	token := admin.newUploadToken("admin-token")
	invite := admin.giveInviteCode(ts.adminID(), 1)

	other.formRequest(http.MethodDelete, "/api/account/upload_token", url.Values{"upload_token": {token}})
	other.formRequest(http.MethodDelete, "/api/account/invite_code", url.Values{"invite_code": {invite}})

	if status, body := ts.newClient().uploadStatus(formField{"upload_token", token}); status != http.StatusOK {
		t.Errorf("admin's upload token should still work, got %d: %s", status, body)
	}
	if status, body := ts.newClient().postForm("/api/auth/register", url.Values{"code": {invite}}); status != http.StatusSeeOther {
		t.Errorf("admin's invite code should still work, got %d: %s", status, body)
	}
}

func TestExpiredInviteCodeRejected(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.registerAdmin()
	code := admin.giveInviteCode(ts.adminID(), 5)

	if err := ts.app.db.Model(&db.InviteCodes{}).
		Where("code = ?", code).
		Update("expiry_date", time.Now().Add(-time.Minute)).Error; err != nil {
		t.Fatal(err)
	}

	if status, body := ts.newClient().postForm("/api/auth/register", url.Values{"code": {code}}); status != http.StatusBadRequest {
		t.Errorf("expected 400 for expired invite, got %d: %s", status, body)
	}
}

func TestSingleUseInviteCodeRace(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.registerAdmin()
	code := admin.giveInviteCode(ts.adminID(), 1)

	var before int64
	if err := ts.app.db.Model(&db.Accounts{}).Count(&before).Error; err != nil {
		t.Fatal(err)
	}

	const attempts = 10
	statuses := make([]int, attempts)
	var wg sync.WaitGroup
	for i := range attempts {
		client := ts.newClient()
		wg.Go(func() {
			statuses[i], _ = client.postForm("/api/auth/register", url.Values{"code": {code}})
		})
	}
	wg.Wait()

	var succeeded int
	for _, s := range statuses {
		if s == http.StatusSeeOther {
			succeeded++
		}
	}
	if succeeded != 1 {
		t.Errorf("expected exactly 1 successful registration, got %d (statuses %v)", succeeded, statuses)
	}

	var after int64
	if err := ts.app.db.Model(&db.Accounts{}).Count(&after).Error; err != nil {
		t.Fatal(err)
	}
	if after-before != 1 {
		t.Errorf("expected exactly 1 new account, got %d", after-before)
	}
}
