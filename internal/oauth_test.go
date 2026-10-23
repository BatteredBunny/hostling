package internal

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/BatteredBunny/hostling/internal/db"
	"github.com/markbates/goth"
	"github.com/markbates/goth/providers/faux"
)

// oauthFakeProvider wraps goth's faux provider so it can be registered under
// the names the app looks up and so each login can pretend to be a different
// remote identity (the stock faux provider always returns the user id "id").
type oauthFakeProvider struct {
	*faux.Provider
	name string

	mu sync.Mutex
	id string
}

func (p *oauthFakeProvider) Name() string { return p.name }

func (p *oauthFakeProvider) setIdentity(id string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.id = id
}

func (p *oauthFakeProvider) BeginAuth(state string) (goth.Session, error) {
	sess, err := p.Provider.BeginAuth(state)
	if err != nil {
		return nil, err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	sess.(*faux.Session).ID = p.id

	return sess, nil
}

type oauthEnv struct {
	*testServer
	github *oauthFakeProvider
	oidc   *oauthFakeProvider
}

func newOauthEnv(t *testing.T) *oauthEnv {
	t.Helper()
	// Make sure env based init can't enable/replace the fake providers
	for _, k := range []string{
		"GITHUB_CLIENT_ID", "GITHUB_SECRET",
		"OPENID_CONNECT_CLIENT_ID", "OPENID_CONNECT_CLIENT_SECRET", "OPENID_CONNECT_DISCOVERY_URL",
	} {
		t.Setenv(k, "")
	}

	ts := newTestServer(t)
	env := &oauthEnv{
		testServer: ts,
		github:     &oauthFakeProvider{Provider: &faux.Provider{}, name: "github"},
		oidc:       &oauthFakeProvider{Provider: &faux.Provider{}, name: "openid-connect"},
	}
	goth.UseProviders(env.github, env.oidc)
	t.Cleanup(goth.ClearProviders)

	return env
}

func (e *oauthEnv) fake(provider string) *oauthFakeProvider {
	if provider == "github" {
		return e.github
	}

	return e.oidc
}

type oauthResp struct {
	status   int
	location string
	body     string
}

func oauthDo(tc *testClient, method, path string) oauthResp {
	tc.t.Helper()
	req, err := http.NewRequest(method, tc.base+path, nil)
	if err != nil {
		tc.t.Fatal(err)
	}
	resp, err := tc.http.Do(req)
	if err != nil {
		tc.t.Fatal(err)
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		tc.t.Fatal(err)
	}

	return oauthResp{status: resp.StatusCode, location: resp.Header.Get("Location"), body: string(b)}
}

func oauthCookie(tc *testClient, name string) string {
	u, _ := url.Parse(tc.base)
	for _, c := range tc.http.Jar.Cookies(u) {
		if c.Name == name {
			return c.Value
		}
	}

	return ""
}

func oauthSetCookie(tc *testClient, name, value string) {
	u, _ := url.Parse(tc.base)
	tc.http.Jar.SetCookies(u, []*http.Cookie{{Name: name, Value: value, Path: "/"}})
}

// Runs the begin step (login redirect, or link POST) and returns the state sent to the provider
func oauthBegin(t *testing.T, tc *testClient, link bool, provider string) (oauthResp, string) {
	t.Helper()
	method, path := http.MethodGet, "/api/auth/login/"+provider
	if link {
		method, path = http.MethodPost, "/api/auth/link/"+provider
	}
	r := oauthDo(tc, method, path)
	u, err := url.Parse(r.location)
	if err != nil {
		t.Fatal(err)
	}

	return r, u.Query().Get("state")
}

func oauthCallback(tc *testClient, provider, state string) oauthResp {
	tc.t.Helper()

	return oauthDo(tc, http.MethodGet, "/api/auth/login/"+provider+"/callback?state="+url.QueryEscape(state))
}

// Full provider round trip as the given remote identity
func (e *oauthEnv) flow(tc *testClient, link bool, provider, identity string) oauthResp {
	e.t.Helper()
	e.fake(provider).setIdentity(identity)
	begin, state := oauthBegin(e.t, tc, link, provider)
	if state == "" {
		e.t.Fatalf("begin did not redirect to provider: %d %q", begin.status, begin.location)
	}

	return oauthCallback(tc, provider, state)
}

func (e *oauthEnv) login(tc *testClient, provider, identity string) oauthResp {
	e.t.Helper()

	return e.flow(tc, false, provider, identity)
}

// Links identity and asserts the redirect to settings
func (e *oauthEnv) link(tc *testClient, provider, identity string) {
	e.t.Helper()
	r := e.flow(tc, true, provider, identity)
	if r.status != http.StatusSeeOther || r.location != "/settings" {
		e.t.Fatalf("link %s: expected 303 /settings, got %d %q: %s", provider, r.status, r.location, r.body)
	}
}

func (e *oauthEnv) account(id uint) db.Accounts {
	e.t.Helper()
	acc, err := e.app.db.GetAccountByID(id)
	if err != nil {
		e.t.Fatal(err)
	}

	return acc
}

func oauthExpectLoggedIn(t *testing.T, tc *testClient) {
	t.Helper()
	if status, body := tc.get("/api/account/files"); status != http.StatusOK {
		t.Fatalf("expected logged in (200), got %d: %s", status, body)
	}
}

func oauthExpectLoggedOut(t *testing.T, tc *testClient) {
	t.Helper()
	if status, body := tc.get("/api/account/files"); status != http.StatusUnauthorized {
		t.Fatalf("expected logged out (401), got %d: %s", status, body)
	}
}

func oauthExpectRedirect(t *testing.T, r oauthResp, status int, location string) {
	t.Helper()
	if r.status != status || r.location != location {
		t.Fatalf("expected %d -> %q, got %d -> %q: %s", status, location, r.status, r.location, r.body)
	}
}

func TestOAuthLoginUnlinkedIdentityDoesNotLogIn(t *testing.T) {
	for _, provider := range []string{"github", "openid-connect"} {
		t.Run(provider, func(t *testing.T) {
			env := newOauthEnv(t)
			env.registerAdmin()

			anon := env.newClient()
			r := env.login(anon, provider, "4242")
			oauthExpectRedirect(t, r, http.StatusSeeOther, "/login")
			if v := oauthCookie(anon, AUTH_COOKIE); v != "" {
				t.Fatalf("auth cookie must not be set, got %q", v)
			}
			oauthExpectLoggedOut(t, anon)
		})
	}
}

func TestOAuthLinkThenLoginGivesSameAccount(t *testing.T) {
	for _, provider := range []string{"github", "openid-connect"} {
		t.Run(provider, func(t *testing.T) {
			env := newOauthEnv(t)
			admin := env.registerAdmin()
			user, userID := env.registerUser(admin)

			env.link(user, provider, "5151")
			acc := env.account(userID)
			if provider == "github" && acc.GithubID != 5151 {
				t.Fatalf("github id not linked: %+v", acc)
			}
			if provider == "openid-connect" && (acc.OIDCID != "5151" || acc.OIDCUsername != "5151") {
				t.Fatalf("oidc id not linked: %+v", acc)
			}
			if v := oauthCookie(user, LINKING_COOKIE); v != "" {
				t.Fatalf("linking cookie should be cleared after link, got %q", v)
			}

			// Upload so we can prove the new session belongs to the same account
			name := user.upload()

			user.expectStatus(http.StatusSeeOther, http.MethodPost, "/logout", nil)
			oauthExpectLoggedOut(t, user)

			r := env.login(user, provider, "5151")
			oauthExpectRedirect(t, r, http.StatusSeeOther, "/gallery")
			oauthExpectLoggedIn(t, user)
			if got := fileNames(user.listFiles("").Files); len(got) != 1 || got[0] != name {
				t.Fatalf("expected same account's file %q, got %v", name, got)
			}

			// A different (fresh) client logging in as the identity also lands on the same account
			other := env.newClient()
			env.login(other, provider, "5151")
			if got := fileNames(other.listFiles("").Files); len(got) != 1 || got[0] != name {
				t.Fatalf("second client: expected %q, got %v", name, got)
			}
		})
	}
}

func TestOAuthReLoginDeletesPriorSession(t *testing.T) {
	env := newOauthEnv(t)
	admin := env.registerAdmin()
	user, _ := env.registerUser(admin)
	env.link(user, "github", "6001")

	oldToken := oauthCookie(user, AUTH_COOKIE)
	if oldToken == "" {
		t.Fatal("expected a session cookie")
	}

	// Log in again while still holding the old session
	r := env.login(user, "github", "6001")
	oauthExpectRedirect(t, r, http.StatusSeeOther, "/gallery")
	newToken := oauthCookie(user, AUTH_COOKIE)
	if newToken == "" || newToken == oldToken {
		t.Fatalf("expected a fresh session token, old=%q new=%q", oldToken, newToken)
	}
	oauthExpectLoggedIn(t, user)

	stale := env.newClient()
	oauthSetCookie(stale, AUTH_COOKIE, oldToken)
	oauthExpectLoggedOut(t, stale)
}

func TestOAuthReLoginKeepsOtherAccountsSession(t *testing.T) {
	env := newOauthEnv(t)
	admin := env.registerAdmin()
	user, _ := env.registerUser(admin)
	env.link(user, "github", "6002")

	// Client holds admin's session but logs in as the user: admin's session must survive
	adminToken := oauthCookie(admin, AUTH_COOKIE)
	env.login(admin, "github", "6002")
	if oauthCookie(admin, AUTH_COOKIE) == adminToken {
		t.Fatal("expected the cookie to be replaced")
	}
	stale := env.newClient()
	oauthSetCookie(stale, AUTH_COOKIE, adminToken)
	oauthExpectLoggedIn(t, stale)
}

func TestOAuthLinkAlreadyLinkedToOtherAccountConflicts(t *testing.T) {
	for _, provider := range []string{"github", "openid-connect"} {
		t.Run(provider, func(t *testing.T) {
			env := newOauthEnv(t)
			admin := env.registerAdmin()
			user, userID := env.registerUser(admin)

			env.link(admin, provider, "7001")

			r := env.flow(user, true, provider, "7001")
			if r.status != http.StatusConflict {
				t.Fatalf("expected 409, got %d: %s", r.status, r.body)
			}
			acc := env.account(userID)
			if acc.GithubID != 0 || acc.OIDCID != "" {
				t.Fatalf("identity must not be linked: %+v", acc)
			}
			if v := oauthCookie(user, LINKING_COOKIE); v != "" {
				t.Fatalf("linking cookie should be cleared, got %q", v)
			}
		})
	}
}

func TestOAuthLinkSameIdentityTwiceIsNoop(t *testing.T) {
	env := newOauthEnv(t)
	admin := env.registerAdmin()
	env.link(admin, "github", "7101")

	// linkApi short-circuits to settings when the account already has that provider
	r, _ := oauthBegin(t, admin, true, "github")
	oauthExpectRedirect(t, r, http.StatusSeeOther, "/settings")
	if v := oauthCookie(admin, LINKING_COOKIE); v != "" {
		t.Fatalf("no linking cookie expected, got %q", v)
	}
}

func TestOAuthLinkingCookieForOtherAccountRejected(t *testing.T) {
	env := newOauthEnv(t)
	admin := env.registerAdmin()
	user, userID := env.registerUser(admin)

	// user is logged in, but the linking cookie was minted for the admin
	env.github.setIdentity("8001")
	_, state := oauthBegin(t, user, false, "github")
	oauthSetCookie(user, LINKING_COOKIE, env.app.signLinkingValue(env.adminID()))

	r := oauthCallback(user, "github", state)
	if r.status != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", r.status, r.body)
	}
	if acc := env.account(userID); acc.GithubID != 0 {
		t.Fatalf("must not link: %+v", acc)
	}
	if acc := env.account(env.adminID()); acc.GithubID != 0 {
		t.Fatalf("must not link admin either: %+v", acc)
	}
	if v := oauthCookie(user, LINKING_COOKIE); v != "" {
		t.Fatalf("linking cookie should be cleared, got %q", v)
	}
	oauthExpectLoggedIn(t, user)
}

func TestOAuthValidLinkingCookieWithoutSessionRedirectsToLogin(t *testing.T) {
	env := newOauthEnv(t)
	admin := env.registerAdmin()

	anon := env.newClient()
	env.github.setIdentity("8101")
	_, state := oauthBegin(t, anon, false, "github")
	oauthSetCookie(anon, LINKING_COOKIE, env.app.signLinkingValue(env.adminID()))

	r := oauthCallback(anon, "github", state)
	oauthExpectRedirect(t, r, http.StatusSeeOther, "/login")
	if acc := env.account(env.adminID()); acc.GithubID != 0 {
		t.Fatalf("must not link: %+v", acc)
	}
	if v := oauthCookie(anon, LINKING_COOKIE); v != "" {
		t.Fatalf("linking cookie should be cleared, got %q", v)
	}
	oauthExpectLoggedOut(t, anon)
	oauthExpectLoggedIn(t, admin)
}

func TestOAuthBadLinkingCookieIsClearedAndFallsThroughToLogin(t *testing.T) {
	key := func(e *oauthEnv) []byte { return deriveKey(e.app.appSecret, "link-cookie") }

	cases := map[string]func(e *oauthEnv, accountID uint) string{
		"garbage": func(*oauthEnv, uint) string { return "not-a-cookie" },
		"tampered": func(e *oauthEnv, id uint) string {
			v := e.app.signLinkingValue(id)
			parts := strings.SplitN(v, ".", 3)
			parts[0] = "1" + parts[0] // different account id, same signature

			return strings.Join(parts, ".")
		},
		"bad signature": func(e *oauthEnv, id uint) string {
			return signLinkingValue(id, []byte("some other key"), time.Now().Add(time.Hour))
		},
		"expired": func(e *oauthEnv, id uint) string {
			return signLinkingValue(id, key(e), time.Now().Add(-time.Minute))
		},
	}

	for name, mk := range cases {
		t.Run(name, func(t *testing.T) {
			env := newOauthEnv(t)
			admin := env.registerAdmin()
			user, userID := env.registerUser(admin)

			// The identity is not linked anywhere: a bad linking cookie must not link it,
			// the flow is treated as a plain login and bounces to /login
			anon := env.newClient()
			env.github.setIdentity("9001")
			_, state := oauthBegin(t, anon, false, "github")
			oauthSetCookie(anon, LINKING_COOKIE, mk(env, userID))
			r := oauthCallback(anon, "github", state)
			oauthExpectRedirect(t, r, http.StatusSeeOther, "/login")
			if v := oauthCookie(anon, LINKING_COOKIE); v != "" {
				t.Fatalf("bad linking cookie should be cleared, got %q", v)
			}
			if acc := env.account(userID); acc.GithubID != 0 {
				t.Fatalf("must not link: %+v", acc)
			}
			oauthExpectLoggedOut(t, anon)

			// Same with a logged in session: still no link, and the user stays logged in
			env.github.setIdentity("9002")
			_, state = oauthBegin(t, user, false, "github")
			oauthSetCookie(user, LINKING_COOKIE, mk(env, userID))
			r = oauthCallback(user, "github", state)
			oauthExpectRedirect(t, r, http.StatusSeeOther, "/login")
			if acc := env.account(userID); acc.GithubID != 0 {
				t.Fatalf("logged in: must not link: %+v", acc)
			}
			if v := oauthCookie(user, LINKING_COOKIE); v != "" {
				t.Fatalf("bad linking cookie should be cleared, got %q", v)
			}

			// And when the identity is linked, the bad cookie is cleared and login proceeds
			env.link(user, "github", "9003")
			fresh := env.newClient()
			env.github.setIdentity("9003")
			_, state = oauthBegin(t, fresh, false, "github")
			oauthSetCookie(fresh, LINKING_COOKIE, mk(env, env.adminID()))
			r = oauthCallback(fresh, "github", state)
			oauthExpectRedirect(t, r, http.StatusSeeOther, "/gallery")
			if v := oauthCookie(fresh, LINKING_COOKIE); v != "" {
				t.Fatalf("bad linking cookie should be cleared, got %q", v)
			}
			oauthExpectLoggedIn(t, fresh)
		})
	}
}

func TestOAuthUnlink(t *testing.T) {
	env := newOauthEnv(t)
	admin := env.registerAdmin()
	user, userID := env.registerUser(admin)

	// Provider not linked: nothing to do, back to settings
	r := oauthDo(user, http.MethodPost, "/api/auth/unlink/github")
	oauthExpectRedirect(t, r, http.StatusSeeOther, "/settings")
	r = oauthDo(user, http.MethodPost, "/api/auth/unlink/openid-connect")
	oauthExpectRedirect(t, r, http.StatusSeeOther, "/settings")

	env.link(user, "github", "1111")

	// Only identity: blocked
	r = oauthDo(user, http.MethodPost, "/api/auth/unlink/github")
	if r.status != http.StatusConflict || r.body != ErrLastLinkedIdent.Error() {
		t.Fatalf("expected 409 %q, got %d: %s", ErrLastLinkedIdent, r.status, r.body)
	}
	if acc := env.account(userID); acc.GithubID != 1111 {
		t.Fatalf("github must still be linked: %+v", acc)
	}

	// With a second identity, either can be removed
	env.link(user, "openid-connect", "oidc-1")
	r = oauthDo(user, http.MethodPost, "/api/auth/unlink/github")
	oauthExpectRedirect(t, r, http.StatusSeeOther, "/settings")
	acc := env.account(userID)
	if acc.GithubID != 0 || acc.GithubUsername != "" || acc.OIDCID != "oidc-1" {
		t.Fatalf("unexpected account after unlink: %+v", acc)
	}

	// The remaining one is now the last
	r = oauthDo(user, http.MethodPost, "/api/auth/unlink/openid-connect")
	if r.status != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", r.status, r.body)
	}
	if acc := env.account(userID); acc.OIDCID != "oidc-1" {
		t.Fatalf("oidc must still be linked: %+v", acc)
	}

	// Unlinked identity can no longer log in, the remaining one can
	anon := env.newClient()
	oauthExpectRedirect(t, env.login(anon, "github", "1111"), http.StatusSeeOther, "/login")
	oauthExpectLoggedOut(t, anon)
	oauthExpectRedirect(t, env.login(anon, "openid-connect", "oidc-1"), http.StatusSeeOther, "/gallery")
	oauthExpectLoggedIn(t, anon)

	// Unlinking oidc via the other path works once github is back
	env.link(user, "github", "1111")
	r = oauthDo(user, http.MethodPost, "/api/auth/unlink/openid-connect")
	oauthExpectRedirect(t, r, http.StatusSeeOther, "/settings")
	if acc := env.account(userID); acc.OIDCID != "" || acc.OIDCUsername != "" || acc.GithubID != 1111 {
		t.Fatalf("unexpected account after oidc unlink: %+v", acc)
	}
}

func TestOAuthUnknownProvider(t *testing.T) {
	env := newOauthEnv(t)
	admin := env.registerAdmin()

	t.Run("login", func(t *testing.T) {
		r := oauthDo(env.newClient(), http.MethodGet, "/api/auth/login/nope")
		if r.status != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", r.status, r.body)
		}
	})
	t.Run("callback", func(t *testing.T) {
		r := oauthDo(env.newClient(), http.MethodGet, "/api/auth/login/nope/callback?state=x")
		if r.status != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", r.status, r.body)
		}
	})
	t.Run("unlink", func(t *testing.T) {
		r := oauthDo(admin, http.MethodPost, "/api/auth/unlink/nope")
		if r.status != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", r.status, r.body)
		}
	})
	t.Run("link", func(t *testing.T) {
		r := oauthDo(admin, http.MethodPost, "/api/auth/link/nope")
		if r.status != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", r.status, r.body)
		}
		if v := oauthCookie(admin, LINKING_COOKIE); v != "" {
			t.Fatalf("no linking cookie expected for unknown provider, got %q", v)
		}
	})
}

func TestOAuthUnauthenticatedLinkUnlink(t *testing.T) {
	env := newOauthEnv(t)
	env.registerAdmin()

	for _, provider := range []string{"github", "openid-connect", "nope"} {
		anon := env.newClient()
		r := oauthDo(anon, http.MethodPost, "/api/auth/link/"+provider)
		oauthExpectRedirect(t, r, http.StatusSeeOther, "/login")
		if v := oauthCookie(anon, LINKING_COOKIE); v != "" {
			t.Fatalf("no linking cookie expected, got %q", v)
		}

		r = oauthDo(anon, http.MethodPost, "/api/auth/unlink/"+provider)
		oauthExpectRedirect(t, r, http.StatusSeeOther, "/login")
	}

	// A stale (deleted) session is treated the same way
	stale := env.newClient()
	oauthSetCookie(stale, AUTH_COOKIE, "11111111-1111-4111-8111-111111111111")
	oauthExpectRedirect(t, oauthDo(stale, http.MethodPost, "/api/auth/link/github"), http.StatusSeeOther, "/login")
	oauthExpectRedirect(t, oauthDo(stale, http.MethodPost, "/api/auth/unlink/github"), http.StatusSeeOther, "/login")
}

func TestOAuthLinkSetsSignedLinkingCookie(t *testing.T) {
	env := newOauthEnv(t)
	admin := env.registerAdmin()

	env.github.setIdentity("3001")
	r, state := oauthBegin(t, admin, true, "github")
	if r.status != http.StatusSeeOther || state == "" || !strings.HasPrefix(r.location, "http://example.com/auth") {
		t.Fatalf("unexpected link begin response: %d %q", r.status, r.location)
	}
	raw := oauthCookie(admin, LINKING_COOKIE)
	id, err := parseLinkingValue(raw, deriveKey(env.app.appSecret, "link-cookie"), time.Now())
	if err != nil || id != env.adminID() {
		t.Fatalf("linking cookie id=%d err=%v raw=%q", id, err, raw)
	}
}

func TestOAuthCallbackRequiresValidState(t *testing.T) {
	env := newOauthEnv(t)
	adminClient := env.registerAdmin()

	t.Run("no gothic session", func(t *testing.T) {
		r := oauthCallback(env.newClient(), "github", "whatever")
		if r.status != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", r.status, r.body)
		}
	})
	t.Run("state mismatch", func(t *testing.T) {
		anon := env.newClient()
		env.github.setIdentity("1234")
		_, state := oauthBegin(t, anon, false, "github")
		r := oauthCallback(anon, "github", state+"x")
		if r.status != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", r.status, r.body)
		}
		oauthExpectLoggedOut(t, anon)
	})
	t.Run("state is single use", func(t *testing.T) {
		admin := env.newClient()
		env.link(adminClient, "github", "1235")
		env.github.setIdentity("1235")
		_, state := oauthBegin(t, admin, false, "github")
		oauthExpectRedirect(t, oauthCallback(admin, "github", state), http.StatusSeeOther, "/gallery")
		if r := oauthCallback(admin, "github", state); r.status != http.StatusBadRequest {
			t.Fatalf("replayed callback: expected 400, got %d: %s", r.status, r.body)
		}
	})
}
