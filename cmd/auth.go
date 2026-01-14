package cmd

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/gorilla/sessions"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/github"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

func contextWithProviderName(c *gin.Context, provider string) *http.Request {
	return c.Request.WithContext(context.WithValue(c.Request.Context(), "provider", provider))
}

func generateSecureKey(length int) []byte {
	key := make([]byte, length)
	if _, err := rand.Read(key); err != nil {
		panic(err)
	}
	return key
}

func (app *Application) setupSocialLogin() {
	githubApiKey := os.Getenv("GITHUB_CLIENT_ID")
	githubSecret := os.Getenv("GITHUB_SECRET")

	// Goth creates its own cookie for the auth flow
	gothic.Store = sessions.NewCookieStore(generateSecureKey(32))

	goth.UseProviders(
		github.New(githubApiKey, githubSecret, fmt.Sprintf("%s/api/auth/login/github/callback", app.config.PublicUrl)),
	)
}

func (app *Application) setupAuth(api *gin.RouterGroup) {
	app.setupSocialLogin()

	api.GET("/auth/login/:provider/callback", app.loginCallback)
	api.GET("/auth/login/:provider", app.loginApi)

	api.GET("/auth/register", app.registerApi)

	api.GET("/auth/link/:provider", app.linkApi)
}

func (app *Application) loginApi(c *gin.Context) {
	provider := c.Param("provider")
	c.Request = contextWithProviderName(c, provider)

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
		c.AbortWithError(http.StatusBadRequest, err)
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

		if loggedIn && account.GithubID == 0 {
			if err := app.db.LinkGithub(account.ID, user.NickName, user.UserID); err != nil {
				c.String(http.StatusInternalServerError, "Failed to link github")
				return
			}

			c.Redirect(http.StatusTemporaryRedirect, "/user")
		} else {
			c.Redirect(http.StatusTemporaryRedirect, "/login")
		}
	} else {
		account, err := app.db.FindAccountByGithubID(user.UserID)
		if err != nil {
			c.Redirect(http.StatusTemporaryRedirect, "/login")
			return
		}

		if err := app.db.UpdateGithubUsername(account.ID, user.NickName); err != nil {
			log.Warn().Err(err).Msg("Failed to update github username")
		}

		sessionToken, err := app.db.CreateSessionToken(account.ID)
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		app.setAuthCookie(sessionToken, c)
		c.Redirect(http.StatusTemporaryRedirect, "/user")
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

	if !loggedIn || account.GithubID > 0 {
		c.Redirect(http.StatusTemporaryRedirect, "/")
		return
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
	c.Redirect(http.StatusTemporaryRedirect, "/user")
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

	gothic.Logout(c.Writer, c.Request)

	c.Redirect(http.StatusTemporaryRedirect, "/")
}
