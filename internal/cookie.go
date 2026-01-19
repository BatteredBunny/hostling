package internal

import (
	"errors"

	"github.com/BatteredBunny/hostling/internal/db"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	AUTH_COOKIE    = "auth"
	LINKING_COOKIE = "linking"
)

func (app *Application) setLinkingCookie(c *gin.Context) {
	c.SetCookie("linking", "true", 500, "/", app.config.PublicUrl, gin.Mode() == gin.ReleaseMode, true)
}

func (app *Application) clearLinkingCookie(c *gin.Context) {
	c.SetCookie("linking", "", -1, "/", app.config.PublicUrl, gin.Mode() == gin.ReleaseMode, true)
}

func (app *Application) setAuthCookie(sessionToken uuid.UUID, c *gin.Context) {
	// TODO: use actual max age
	c.SetCookie(AUTH_COOKIE, sessionToken.String(), 86400*7, "/", app.config.PublicUrl, gin.Mode() == gin.ReleaseMode, true)
}

func (app *Application) clearAuthCookie(c *gin.Context) {
	c.SetCookie(AUTH_COOKIE, "", -1, "/", app.config.PublicUrl, gin.Mode() == gin.ReleaseMode, true)
}

var ErrInvalidAuthCookie = errors.New("Invalid session token")

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

func (app *Application) validateAuthCookie(c *gin.Context) (sessionToken uuid.UUID, account db.Accounts, loggedIn bool, err error) {
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
