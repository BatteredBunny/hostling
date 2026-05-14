package internal

import (
	"errors"
	"net/http"

	"github.com/BatteredBunny/hostling/internal/db"
	"github.com/didip/tollbooth/v8"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

func (app *Application) ratelimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		httpError := tollbooth.LimitByKeys(app.RateLimiter, []string{c.ClientIP()})
		if httpError != nil {
			c.Data(httpError.StatusCode, app.RateLimiter.GetMessageContentType(), []byte(httpError.Message))
			c.Abort()

			return
		}
		c.Next()
	}
}

// limits request body size
func (app *Application) bodySizeMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, app.config.MaxUploadSize)
		c.Next()
	}
}

// Max size before using disk instead of ram
const multipartMaxMemory = 32 << 20 // 32 MiB

// parses form
func (app *Application) apiMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > 0 {
			if err := c.Request.ParseMultipartForm(
				multipartMaxMemory,
			); err != nil &&
				!errors.Is(err, http.ErrNotMultipart) {
				c.String(http.StatusRequestEntityTooLarge, "Too big file")
				c.Abort()

				return
			}
		}

		c.Next()
	}
}

func getSessionToken(c *gin.Context) (uuid.UUID, bool) {
	v, ok := c.Get("sessionToken")
	if !ok {
		return uuid.Nil, false
	}
	id, ok := v.(uuid.UUID)

	return id, ok
}

func getUploadToken(c *gin.Context) (uuid.UUID, bool) {
	v, ok := c.Get("uploadToken")
	if !ok {
		return uuid.Nil, false
	}
	id, ok := v.(uuid.UUID)

	return id, ok
}

func getAccount(c *gin.Context) (db.Accounts, bool) {
	v, ok := c.Get("account")
	if !ok {
		return db.Accounts{}, false
	}
	account, ok := v.(db.Accounts)

	return account, ok
}

// Makes sure request has session token and a valid one, attaches the
// account to the context so future handlers can read it without
func (app *Application) verifySessionAuthentication() gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionToken, account, loggedIn, err := app.validateAuthCookie(c)
		if err != nil && !errors.Is(err, ErrInvalidAuthCookie) {
			log.Err(err).Msg("validateAuthCookie failed")
			c.AbortWithStatus(http.StatusInternalServerError)

			return
		}
		if !loggedIn {
			c.AbortWithStatus(http.StatusUnauthorized)

			return
		}

		c.Set("sessionToken", sessionToken)
		c.Set("account", account)
		c.Next()
	}
}

// Must run after verifySessionAuthentication.
func (app *Application) isAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		account, ok := getAccount(c)
		if !ok || account.AccountType != "ADMIN" {
			c.AbortWithStatus(http.StatusUnauthorized)

			return
		}
		c.Next()
	}
}

// Makes sure request has a valid upload or session token
func (app *Application) hasUploadOrSessionTokenMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		rawUploadToken, uploadTokenExists := c.GetPostForm("upload_token")

		if uploadTokenExists && rawUploadToken != "" {
			var uploadToken uuid.UUID
			var err error
			if uploadToken, err = uuid.Parse(rawUploadToken); err != nil {
				_ = c.AbortWithError(http.StatusUnauthorized, err)

				return
			}

			valid, err := app.isValidUploadToken(uploadToken)
			if err != nil { // Could be a database error
				log.Err(err).Msg("Failed to check if upload token is valid")
				c.AbortWithStatus(http.StatusInternalServerError)

				return
			} else if !valid { // Wrong or expired token given
				c.AbortWithStatus(http.StatusUnauthorized)

				return
			}

			c.Set("uploadToken", uploadToken)
		} else {
			sessionToken, account, loggedIn, err := app.validateAuthCookie(c)
			if err != nil && !errors.Is(err, ErrInvalidAuthCookie) {
				_ = c.AbortWithError(http.StatusInternalServerError, err)

				return
			}
			if !loggedIn {
				c.AbortWithStatus(http.StatusUnauthorized)

				return
			}

			c.Set("sessionToken", sessionToken)
			c.Set("account", account)
		}

		c.Next()
	}
}
