package internal

import (
	"errors"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/BatteredBunny/hostling/internal/db"
	"github.com/danielgtaylor/huma/v2"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

func (app *Application) ratelimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		limit, remaining, reset, ok, err := app.RateLimiter.Take(c.Request.Context(), c.ClientIP())
		if err != nil {
			log.Err(err).Msg("Rate limiter failed")
			abortAPI(c, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))

			return
		}

		resetTime := time.Unix(0, int64(reset))
		c.Header("X-RateLimit-Limit", strconv.FormatUint(limit, 10))
		c.Header("X-RateLimit-Remaining", strconv.FormatUint(remaining, 10))
		c.Header("X-RateLimit-Reset", resetTime.UTC().Format(http.TimeFormat))

		if !ok {
			retryAfter := max(1, int(math.Ceil(time.Until(resetTime).Seconds())))
			c.Header("Retry-After", strconv.Itoa(retryAfter))
			abortAPI(c, http.StatusTooManyRequests, "You have reached maximum request limit.")

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
	return app.multipartFormMiddleware(multipartMaxMemory)
}

// multipartFormMiddleware keeps standard multipart parsing and its field-order
// semantics. A zero file-memory budget spools upload bodies directly to disk.
func (app *Application) multipartFormMiddleware(maxFileMemory int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength != 0 {
			if err := c.Request.ParseMultipartForm(
				maxFileMemory,
			); err != nil &&
				!errors.Is(err, http.ErrNotMultipart) {
				abortAPI(c, http.StatusRequestEntityTooLarge, "Too big file")

				return
			}
		}

		c.Next()
	}
}

func getSessionToken(c *gin.Context) (string, bool) {
	v, ok := c.Get("sessionToken")
	if !ok {
		return "", false
	}
	hash, ok := v.(string)

	return hash, ok
}

func getUploadToken(c *gin.Context) (string, bool) {
	v, ok := c.Get("uploadToken")
	if !ok {
		return "", false
	}
	hash, ok := v.(string)

	return hash, ok
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
			abortAPI(c, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))

			return
		}
		if !loggedIn {
			abortAPI(c, http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))

			return
		}

		sessionHash, _ := db.HashSessionToken(sessionToken) // Cookie validation already checked its format.
		c.Set("sessionToken", sessionHash)
		c.Set("account", account)
		c.Next()
	}
}

// Must run after verifySessionAuthentication.
func (app *Application) isAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		account, ok := getAccount(c)
		if !ok || account.AccountType != "ADMIN" {
			abortAPI(c, http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))

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
			uploadToken := rawUploadToken
			hash, err := db.HashUploadToken(uploadToken)
			if err != nil {
				abortAPI(c, http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))

				return
			}

			valid, err := app.isValidUploadToken(c.Request.Context(), uploadToken)
			if err != nil { // Could be a database error
				log.Err(err).Msg("Failed to check if upload token is valid")
				abortAPI(c, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))

				return
			} else if !valid { // Wrong or expired token given
				abortAPI(c, http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))

				return
			}

			c.Set("uploadToken", hash)
		} else {
			sessionToken, account, loggedIn, err := app.validateAuthCookie(c)
			if err != nil && !errors.Is(err, ErrInvalidAuthCookie) {
				log.Err(err).Msg("Authentication failed")
				abortAPI(c, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))

				return
			}
			if !loggedIn {
				abortAPI(c, http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))

				return
			}

			sessionHash, _ := db.HashSessionToken(sessionToken) // Cookie validation already checked its format.
			c.Set("sessionToken", sessionHash)
			c.Set("account", account)
		}

		c.Next()
	}
}

// Middleware and typed handlers share the same documented API error format.
func abortAPI(c *gin.Context, status int, message string) {
	c.Header("Content-Type", "application/problem+json")
	c.AbortWithStatusJSON(status, huma.NewError(status, message))
}
