package internal

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/BatteredBunny/hostling/internal/db"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/gorilla/sessions"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/github"
	"github.com/markbates/goth/providers/openidConnect"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

var gothRegistryMutex sync.Mutex

func contextWithProviderName(c *gin.Context, provider string) *http.Request {
	return c.Request.WithContext(context.WithValue(c.Request.Context(), gothic.ProviderParamKey, provider))
}

func deriveKey(secret []byte, label string) []byte {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(label))

	return mac.Sum(nil)
}

type providerInit struct {
	gothName string // the name goth uses for this provider (e.g. "openid-connect")
	init     func() (bool, error)
}

func (app *Application) setupSocialLogin() {
	// Goth creates its own cookie for the auth flow
	store := sessions.NewCookieStore(deriveKey(app.appSecret, "gothic-session-key"))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   600,
		HttpOnly: true,
		Secure:   app.config.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	}
	gothic.Store = store

	providers := []providerInit{
		{gothName: "github", init: app.initGithubProvider},
		{gothName: "openid-connect", init: app.initOIDCProvider},
	}

	hasEnabledProvider := false
	for _, p := range providers {
		enabled, err := p.init()
		if enabled {
			app.addConfiguredProvider(p.gothName)
			hasEnabledProvider = true
		}
		if err != nil {
			log.Warn().Err(err).Str("provider", p.gothName).Msg("Failed to initialize provider, retrying in background")
			app.addFailedProvider(p.gothName)
			app.backgroundWg.Go(func() {
				app.retryProviderInit(app.shutdownCtx, p.gothName, p.init)
			})
		}
	}

	if !hasEnabledProvider {
		names := make([]string, len(providers))
		for i, p := range providers {
			names[i] = p.gothName
		}
		log.Warn().Msgf("No authentication providers enabled, configure at least one (%s)", strings.Join(names, ", "))
	}
}

func (app *Application) addConfiguredProvider(name string) {
	app.providersMutex.Lock()
	defer app.providersMutex.Unlock()
	if slices.Contains(app.configuredProviders, name) {
		return
	}
	app.configuredProviders = append(app.configuredProviders, name)
}

func (app *Application) addFailedProvider(name string) {
	app.providersMutex.Lock()
	defer app.providersMutex.Unlock()
	if slices.Contains(app.failedProviders, name) {
		return
	}
	app.failedProviders = append(app.failedProviders, name)
}

func (app *Application) removeFailedProvider(name string) {
	app.providersMutex.Lock()
	defer app.providersMutex.Unlock()
	for i, n := range app.failedProviders {
		if n == name {
			app.failedProviders = append(app.failedProviders[:i], app.failedProviders[i+1:]...)

			return
		}
	}
}

func (app *Application) retryProviderInit(ctx context.Context, name string, init func() (bool, error)) {
	const (
		maxRetries = 10
		maxDelay   = 5 * time.Minute
	)
	delay := 5 * time.Second
	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.Warn().
			Str("provider", name).
			Int("attempt", attempt).
			Dur("retry_in", delay).
			Msg("Provider initialization failed, retrying")

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			log.Info().Str("provider", name).Msg("Provider retry cancelled")

			return
		case <-timer.C:
		}

		enabled, err := init()
		if err == nil {
			log.Info().Str("provider", name).Msg("Provider initialized successfully after retry")
			app.removeFailedProvider(name)
			if enabled {
				app.addConfiguredProvider(name)
			}

			return
		}
		log.Warn().Err(err).Str("provider", name).Int("attempt", attempt).Msg("Provider retry failed")
		delay = min(delay*2, maxDelay)
	}
	log.Error().Str("provider", name).Msg("Provider initialization failed after all retries, will retry on login")
}

func (app *Application) initGithubProvider() (enabled bool, err error) {
	clientID := os.Getenv("GITHUB_CLIENT_ID")
	secret := os.Getenv("GITHUB_SECRET")
	enabled = clientID != "" && secret != ""
	if !enabled {
		return
	}

	gothRegistryMutex.Lock()
	defer gothRegistryMutex.Unlock()

	if _, getErr := goth.GetProvider("github"); getErr == nil {
		return
	}

	goth.UseProviders(github.New(
		clientID,
		secret,
		fmt.Sprintf("%s/api/auth/login/github/callback", app.config.PublicUrl),
	))
	log.Info().Msg("GitHub authentication enabled")

	return
}

func (app *Application) initOIDCProvider() (enabled bool, err error) {
	clientID := os.Getenv("OPENID_CONNECT_CLIENT_ID")
	clientSecret := os.Getenv("OPENID_CONNECT_CLIENT_SECRET")
	discoveryURL := os.Getenv("OPENID_CONNECT_DISCOVERY_URL")
	enabled = clientID != "" && clientSecret != "" && discoveryURL != ""
	if !enabled {
		return
	}

	gothRegistryMutex.Lock()
	defer gothRegistryMutex.Unlock()

	if _, getErr := goth.GetProvider("openid-connect"); getErr == nil {
		return
	}

	callback := fmt.Sprintf("%s/api/auth/login/openid-connect/callback", app.config.PublicUrl)
	oidcProvider, err := openidConnect.New(clientID, clientSecret, callback, discoveryURL)
	if err != nil {
		return
	}

	goth.UseProviders(oidcProvider)
	log.Info().Msg("OpenID Connect authentication enabled")

	return
}

func (app *Application) setupAuth(api *gin.RouterGroup) {
	app.setupSocialLogin()

	auth := api.Group("/auth")
	auth.Use(app.ratelimitMiddleware())

	auth.GET("/login/:provider/callback", app.loginCallback)
	auth.GET("/login/:provider", app.loginApi)

	auth.POST("/register", app.registerApi)

	auth.POST("/link/:provider", app.linkApi)
	auth.POST("/unlink/:provider", app.unlinkApi)
}

func (app *Application) unlinkApi(c *gin.Context) {
	provider := c.Param("provider")

	_, account, loggedIn, err := app.validateAuthCookie(c)
	if errors.Is(err, ErrInvalidAuthCookie) || !loggedIn {
		c.Redirect(http.StatusSeeOther, "/login")

		return
	} else if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)

		return
	}

	var unlinkErr error
	switch provider {
	case "github":
		if account.GithubID == 0 {
			c.Redirect(http.StatusSeeOther, "/settings")

			return
		}
		if linkedIdentityCount(account) <= 1 {
			c.String(http.StatusConflict, ErrLastLinkedIdent.Error())

			return
		}
		unlinkErr = app.db.UnlinkGithub(account.ID)
	case "openid-connect":
		if account.OIDCID == "" {
			c.Redirect(http.StatusSeeOther, "/settings")

			return
		}
		if linkedIdentityCount(account) <= 1 {
			c.String(http.StatusConflict, ErrLastLinkedIdent.Error())

			return
		}
		unlinkErr = app.db.UnlinkOIDC(account.ID)
	default:
		c.AbortWithStatus(http.StatusBadRequest)

		return
	}
	if unlinkErr != nil {
		log.Err(unlinkErr).Str("provider", provider).Msg("Failed to unlink provider")
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	c.Redirect(http.StatusSeeOther, "/settings")
}

func (app *Application) loginApi(c *gin.Context) {
	provider := c.Param("provider")
	c.Request = contextWithProviderName(c, provider)

	if err := app.ensureProvider(provider); err != nil {
		log.Error().Err(err).Str("provider", provider).Msg("Provider unavailable")
		c.String(http.StatusServiceUnavailable, "Login provider is temporarily unavailable")

		return
	}

	gothic.BeginAuthHandler(c.Writer, c.Request)
}

func (app *Application) ensureProvider(provider string) error {
	var init func() (bool, error)
	switch provider {
	case "github":
		init = app.initGithubProvider
	case "openid-connect":
		init = app.initOIDCProvider
	default:
		return nil
	}

	enabled, err := init()
	if err != nil {
		return err
	}
	if enabled {
		app.addConfiguredProvider(provider)
		app.removeFailedProvider(provider)
	}

	return nil
}

var (
	ErrUnknownProvider = errors.New("unknown provider")
	ErrLastLinkedIdent = errors.New("can't unlink the only remaining login provider")
)

// reports how many auth identities the account has linked.
func linkedIdentityCount(account db.Accounts) (n int) {
	if account.GithubID != 0 {
		n++
	}
	if account.OIDCID != "" {
		n++
	}

	return
}

func (app *Application) linkProvider(
	provider string,
	account db.Accounts,
	user goth.User,
) (alreadyTaken bool, err error) {
	switch provider {
	case "github":
		if account.GithubID != 0 {
			return
		}
		err = app.db.LinkGithub(account.ID, user.NickName, user.UserID)
	case "openid-connect":
		if account.OIDCID != "" {
			return
		}
		err = app.db.LinkOIDC(account.ID, oidcUsername(user), user.UserID)
	default:
		err = ErrUnknownProvider

		return
	}
	if errors.Is(err, db.ErrProviderAlreadyLinked) {
		alreadyTaken = true
		err = nil
	}

	return
}

// find account for provider & refresh username
func (app *Application) findAccountForProvider(provider string, user goth.User) (account db.Accounts, err error) {
	switch provider {
	case "github":
		if account, err = app.db.FindAccountByGithubID(user.UserID); err != nil {
			return
		}
		if updateErr := app.db.UpdateGithubUsername(account.ID, user.NickName); updateErr != nil {
			log.Warn().Err(updateErr).Msg("Failed to update github username")
		}
	case "openid-connect":
		if account, err = app.db.FindAccountByOIDCID(user.UserID); err != nil {
			return
		}
		if updateErr := app.db.UpdateOIDCUsername(account.ID, oidcUsername(user)); updateErr != nil {
			log.Warn().Err(updateErr).Msg("Failed to update OIDC username")
		}
	default:
		err = gorm.ErrRecordNotFound
	}

	return
}

func (app *Application) loginCallback(c *gin.Context) {
	provider := c.Param("provider")
	c.Request = contextWithProviderName(c, provider)

	user, err := gothic.CompleteUserAuth(c.Writer, c.Request)
	if err != nil {
		_ = c.AbortWithError(http.StatusBadRequest, err)

		return
	}

	if linkingAccountID, linkErr := app.parseLinkingCookie(c); linkErr == nil {
		app.handleLinkCallback(c, provider, user, linkingAccountID)

		return
	} else if errors.Is(linkErr, ErrInvalidLinkingCookie) {
		app.clearLinkingCookie(c)
	}

	account, err := app.findAccountForProvider(provider, user)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.Redirect(http.StatusSeeOther, "/login")

		return
	} else if err != nil {
		log.Err(err).Str("provider", provider).Msg("Failed to find account by provider id")
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	sessionToken, err := app.db.CreateSessionToken(account.ID)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	if prev, parseErr := app.parseAuthCookie(c); parseErr == nil {
		if delErr := app.db.DeleteSessionForAccount(prev, account.ID); delErr != nil {
			log.Warn().Err(delErr).Msg("Failed to delete prior session on re-login")
		}
	}

	app.setAuthCookie(sessionToken, c)
	c.Redirect(http.StatusSeeOther, "/gallery")
}

func (app *Application) handleLinkCallback(c *gin.Context, provider string, user goth.User, linkingAccountID uint) {
	_, account, loggedIn, err := app.validateAuthCookie(c)
	if errors.Is(err, ErrInvalidAuthCookie) {
		app.clearAuthCookie(c)
	} else if err != nil {
		app.clearLinkingCookie(c)
		log.Err(err).Msg("Failed to validate auth cookie during link callback")
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	app.clearLinkingCookie(c)

	if !loggedIn {
		c.Redirect(http.StatusSeeOther, "/login")

		return
	}

	if account.ID != linkingAccountID {
		c.String(http.StatusBadRequest, http.StatusText(http.StatusBadRequest))

		return
	}

	alreadyTaken, err := app.linkProvider(provider, account, user)
	if alreadyTaken {
		c.String(http.StatusConflict, "This identity is already linked to another user")

		return
	}
	if err != nil {
		log.Err(err).Str("provider", provider).Msg("Failed to link provider")
		c.String(http.StatusInternalServerError, "Failed to link provider")

		return
	}

	c.Redirect(http.StatusSeeOther, "/settings")
}

func (app *Application) linkApi(c *gin.Context) {
	provider := c.Param("provider")
	c.Request = contextWithProviderName(c, provider)

	_, account, loggedIn, err := app.validateAuthCookie(c)
	if errors.Is(err, ErrInvalidAuthCookie) {
		app.clearAuthCookie(c)
		c.Redirect(http.StatusSeeOther, "/login")

		return
	} else if err != nil {
		log.Err(err).Msg("Failed to validate auth cookie during link")
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	alreadyLinked := false
	switch provider {
	case "github":
		alreadyLinked = account.GithubID > 0
	case "openid-connect":
		alreadyLinked = account.OIDCID != ""
	}

	if !loggedIn {
		c.Redirect(http.StatusSeeOther, "/login")

		return
	}
	if alreadyLinked {
		c.Redirect(http.StatusSeeOther, "/settings")

		return
	}

	if err := app.ensureProvider(provider); err != nil {
		log.Error().Err(err).Str("provider", provider).Msg("Provider unavailable")
		c.String(http.StatusServiceUnavailable, "Login provider is temporarily unavailable")

		return
	}

	app.setLinkingCookie(c, account.ID)

	url, err := gothic.GetAuthURL(c.Writer, c.Request)
	if err != nil {
		log.Err(err).Msg("Failed to build OAuth authorize URL")
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}
	c.Redirect(http.StatusSeeOther, url)
}

type registerApiInput struct {
	Code string `form:"code"`
}

func (app *Application) registerApi(c *gin.Context) {
	var input registerApiInput
	var err error

	if err = c.MustBindWith(&input, binding.FormPost); err != nil {
		c.String(http.StatusBadRequest, "Invalid request")

		return
	}

	_, token, err := app.db.RegisterWithInviteCode(input.Code)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.String(http.StatusBadRequest, "Invalid code")

		return
	} else if err != nil {
		log.Err(err).Msg("Failed to register with invite code")
		c.String(http.StatusInternalServerError, "Failed to create account")

		return
	}

	app.setAuthCookie(token, c)
	c.Redirect(http.StatusSeeOther, "/gallery")
}

func oidcUsername(user goth.User) string {
	for _, v := range []string{user.NickName, user.Name, user.Email, user.UserID} {
		v = strings.TrimSpace(v)
		if v != "" {
			return v
		}
	}

	return ""
}

func (app *Application) logoutHandler(c *gin.Context) {
	sessionToken, _, loggedIn, err := app.validateAuthCookie(c)
	if errors.Is(err, ErrInvalidAuthCookie) {
		c.Redirect(http.StatusSeeOther, "/")
		app.clearAuthCookie(c)

		return
	} else if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)

		return
	}

	if !loggedIn {
		c.Redirect(http.StatusSeeOther, "/")

		return
	}

	if err = app.db.DeleteSession(sessionToken); err != nil {
		log.Err(err).Msg("Failed to delete session from db")
	}

	app.clearAuthCookie(c)

	if err := gothic.Logout(c.Writer, c.Request); err != nil {
		log.Warn().Err(err).Msg("gothic logout failed")
	}

	c.Redirect(http.StatusSeeOther, "/")
}
