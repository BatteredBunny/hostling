package internal

import (
	"errors"
	"net/http"

	"github.com/BatteredBunny/hostling/internal/db"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	AUTH_COOKIE    = "auth"
	LINKING_COOKIE = "linking"
)

var authCookieMaxAge = int(db.SessionTokenDuration.Seconds())

func (app *Application) setCookie(c *gin.Context, name, value string, maxAge int) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(name, value, maxAge, "/", app.config.CookieDomain, app.config.CookieSecure, true)
}

func (app *Application) setLinkingCookie(c *gin.Context) {
	app.setCookie(c, LINKING_COOKIE, "true", 500)
}

func (app *Application) clearLinkingCookie(c *gin.Context) {
	app.setCookie(c, LINKING_COOKIE, "", -1)
}

func (app *Application) setAuthCookie(sessionToken uuid.UUID, c *gin.Context) {
	app.setCookie(c, AUTH_COOKIE, sessionToken.String(), authCookieMaxAge)
}

func (app *Application) clearAuthCookie(c *gin.Context) {
	app.setCookie(c, AUTH_COOKIE, "", -1)
}

var ErrInvalidAuthCookie = errors.New("invalid session token")

func (app *Application) parseAuthCookie(c *gin.Context) (sessionToken uuid.UUID, err error) {
	rawSessionToken, err := c.Cookie(AUTH_COOKIE)
	if err != nil {
		return
	}

	sessionToken, err = parseToken(rawSessionToken)
	if err != nil {
		return
	}

	return
}

func (app *Application) validateAuthCookie(
	c *gin.Context,
) (sessionToken uuid.UUID, account db.Accounts, loggedIn bool, err error) {
	sessionToken, err = app.parseAuthCookie(c)
	if err != nil {
		err = ErrInvalidAuthCookie
		app.clearAuthCookie(c)
		return
	}

	if account, err = app.db.GetAccountBySessionToken(sessionToken); errors.Is(err, gorm.ErrRecordNotFound) {
		err = ErrInvalidAuthCookie
		app.clearAuthCookie(c)
		return
	} else if err != nil {
		return
	}

	loggedIn = true

	return
}
