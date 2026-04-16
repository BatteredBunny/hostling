package internal

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strings"
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

type providerContextKey struct{}

func contextWithProviderName(c *gin.Context, provider string) *http.Request {
	return c.Request.WithContext(context.WithValue(c.Request.Context(), providerContextKey{}, provider))
}

func generateSecureKey(length int) []byte {
	key := make([]byte, length)
	if _, err := rand.Read(key); err != nil {
		panic(err)
	}
	return key
}

type providerInit struct {
	gothName string // the name goth uses for this provider (e.g. "openid-connect")
	init     func() (bool, error)
}

func (app *Application) setupSocialLogin() {
	// Goth creates its own cookie for the auth flow
	gothic.Store = sessions.NewCookieStore(generateSecureKey(32))

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
			go app.retryProviderInit(app.shutdownCtx, p.gothName, p.init)
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
	const maxRetries = 10
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
		delay *= 2
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

	if _, err = goth.GetProvider("openid-connect"); err == nil {
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

	api.GET("/auth/login/:provider/callback", app.loginCallback)
	api.GET("/auth/login/:provider", app.loginApi)

	api.POST("/auth/register", app.registerApi)

	api.GET("/auth/link/:provider", app.linkApi)
}

func (app *Application) loginApi(c *gin.Context) {
	provider := c.Param("provider")
	c.Request = contextWithProviderName(c, provider)

	// TODO: refactor retry logic if needed by any further providers
	if provider == "openid-connect" {
		if _, err := app.initOIDCProvider(); err != nil {
			log.Error().Err(err).Msg("OpenID Connect provider unavailable")
			c.String(http.StatusServiceUnavailable, "OpenID Connect provider is temporarily unavailable")
			return
		}
	}

	if _, err := gothic.CompleteUserAuth(c.Writer, c.Request); err == nil {
		c.JSON(http.StatusOK, "logged in")
	} else {
		gothic.BeginAuthHandler(c.Writer, c.Request)
	}
}

func (app *Application) loginCallback(c *gin.Context) {
	provider := c.Param("provider")
	c.Request = contextWithProviderName(c, provider)

	user, err := gothic.CompleteUserAuth(c.Writer, c.Request)
	if err != nil {
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	if _, err := c.Cookie("linking"); err == nil {
		_, account, loggedIn, err := app.validateAuthCookie(c)
		if errors.Is(err, ErrInvalidAuthCookie) {
			app.clearAuthCookie(c)
		} else if err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		app.clearLinkingCookie(c)

		if loggedIn {
			switch provider {
			case "github":
				if account.GithubID == 0 {
					if err := app.db.LinkGithub(account.ID, user.NickName, user.UserID); err != nil {
						c.String(http.StatusInternalServerError, "Failed to link github")
						return
					}
				}
				c.Redirect(http.StatusTemporaryRedirect, "/settings")
				return
			case "openid-connect":
				if account.OIDCID == "" {
					if err := app.db.LinkOIDC(account.ID, oidcUsername(user), user.UserID); err != nil {
						c.String(http.StatusInternalServerError, "Failed to link OpenID Connect")
						return
					}
				}
				c.Redirect(http.StatusTemporaryRedirect, "/settings")
				return
			}
		}
		c.Redirect(http.StatusTemporaryRedirect, "/login")
	} else {
		var account db.Accounts
		var err error

		switch provider {
		case "github":
			account, err = app.db.FindAccountByGithubID(user.UserID)
			if err != nil {
				c.Redirect(http.StatusTemporaryRedirect, "/login")
				return
			}

			if err := app.db.UpdateGithubUsername(account.ID, user.NickName); err != nil {
				log.Warn().Err(err).Msg("Failed to update github username")
			}
		case "openid-connect":
			account, err = app.db.FindAccountByOIDCID(user.UserID)
			if err != nil {
				c.Redirect(http.StatusTemporaryRedirect, "/login")
				return
			}

			if err := app.db.UpdateOIDCUsername(account.ID, oidcUsername(user)); err != nil {
				log.Warn().Err(err).Msg("Failed to update OIDC username")
			}
		default:
			c.Redirect(http.StatusTemporaryRedirect, "/login")
			return
		}

		sessionToken, err := app.db.CreateSessionToken(account.ID)
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		app.setAuthCookie(sessionToken, c)
		c.Redirect(http.StatusTemporaryRedirect, "/gallery")
	}
}

func (app *Application) linkApi(c *gin.Context) {
	provider := c.Param("provider")
	c.Request = contextWithProviderName(c, provider)

	_, account, loggedIn, err := app.validateAuthCookie(c)
	if errors.Is(err, ErrInvalidAuthCookie) {
		app.clearAuthCookie(c)
		return
	} else if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	alreadyLinked := false
	switch provider {
	case "github":
		alreadyLinked = account.GithubID > 0
	case "openid-connect":
		alreadyLinked = account.OIDCID != ""
	}

	if !loggedIn || alreadyLinked {
		c.Redirect(http.StatusTemporaryRedirect, "/")
		return
	}

	// TODO: refactor retry logic if needed by any further providers
	if provider == "openid-connect" {
		if _, err := app.initOIDCProvider(); err != nil {
			log.Error().Err(err).Msg("OpenID Connect provider unavailable")
			c.String(http.StatusServiceUnavailable, "OpenID Connect provider is temporarily unavailable")
			return
		}
	}

	if _, err := gothic.CompleteUserAuth(c.Writer, c.Request); err == nil {
		c.JSON(http.StatusOK, "linked")
	} else {
		app.setLinkingCookie(c)

		gothic.BeginAuthHandler(c.Writer, c.Request)
	}
}

type registerApiInput struct {
	Code string `form:"code"`
}

func (app *Application) registerApi(c *gin.Context) {
	var input registerApiInput
	var err error

	if err = c.MustBindWith(&input, binding.FormPost); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	accountType, invitedBy, err := app.db.UseCode(input.Code)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.String(http.StatusBadRequest, "Invalid code")
		return
	} else if err != nil {
		c.String(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
		return
	}

	acc, err := app.db.CreateAccount(accountType, invitedBy)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to create account")
		return
	}

	token, err := app.db.CreateSessionToken(acc.ID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to create account")
		return
	}

	app.setAuthCookie(token, c)
	c.Redirect(http.StatusTemporaryRedirect, "/gallery")
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
		c.Redirect(http.StatusTemporaryRedirect, "/")
		app.clearAuthCookie(c)
		return
	} else if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	if !loggedIn {
		c.Redirect(http.StatusTemporaryRedirect, "/")
		return
	}

	if err = app.db.DeleteSession(sessionToken); err != nil {
		log.Err(err).Msg("Failed to delete session from db")
	}

	app.clearAuthCookie(c)

	if err := gothic.Logout(c.Writer, c.Request); err != nil {
		log.Warn().Err(err).Msg("gothic logout failed")
	}

	c.Redirect(http.StatusTemporaryRedirect, "/")
}
