package internal

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/BatteredBunny/hostling/internal/db"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	AUTH_COOKIE         = "auth"
	LINKING_COOKIE      = "linking"
	linkingCookieMaxAge = 500 // seconds
)

var authCookieMaxAge = int(db.SessionTokenDuration.Seconds())

func (app *Application) setCookie(c *gin.Context, name, value string, maxAge int) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(name, value, maxAge, "/", "", app.config.CookieSecure, true)
}

var ErrInvalidLinkingCookie = errors.New("invalid linking cookie")

func signLinkingValue(accountID uint, key []byte, expiry time.Time) string {
	id := strconv.FormatUint(uint64(accountID), 10)
	exp := strconv.FormatInt(expiry.Unix(), 10)
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(id + ":" + exp))

	return id + "." + exp + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// parseLinkingValue is the inverse of signLinkingValue. Returns
// ErrInvalidLinkingCookie for any malformed, tampered, or expired input.
// `now` is taken as an argument so tests can control expiry comparisons.
func parseLinkingValue(raw string, key []byte, now time.Time) (accountID uint, err error) {
	parts := strings.SplitN(raw, ".", 3)
	if len(parts) != 3 {
		err = ErrInvalidLinkingCookie

		return
	}
	idPart, expPart, sigPart := parts[0], parts[1], parts[2]

	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(idPart + ":" + expPart))
	sig, decodeErr := base64.RawURLEncoding.DecodeString(sigPart)
	if decodeErr != nil || !hmac.Equal(sig, mac.Sum(nil)) {
		err = ErrInvalidLinkingCookie

		return
	}

	expUnix, parseErr := strconv.ParseInt(expPart, 10, 64)
	if parseErr != nil || now.Unix() >= expUnix {
		err = ErrInvalidLinkingCookie

		return
	}

	parsed, parseErr := strconv.ParseUint(idPart, 10, 64)
	if parseErr != nil {
		err = ErrInvalidLinkingCookie

		return
	}
	accountID = uint(parsed)

	return
}

func (app *Application) signLinkingValue(accountID uint) string {
	return signLinkingValue(
		accountID,
		deriveKey(app.appSecret, "link-cookie"),
		time.Now().Add(linkingCookieMaxAge*time.Second),
	)
}

func (app *Application) parseLinkingCookie(c *gin.Context) (accountID uint, err error) {
	raw, err := c.Cookie(LINKING_COOKIE)
	if err != nil {
		return
	}

	return parseLinkingValue(raw, deriveKey(app.appSecret, "link-cookie"), time.Now())
}

func (app *Application) setLinkingCookie(c *gin.Context, accountID uint) {
	app.setCookie(c, LINKING_COOKIE, app.signLinkingValue(accountID), linkingCookieMaxAge)
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

	return uuid.Parse(rawSessionToken)
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
