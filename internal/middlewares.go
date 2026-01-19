package internal

import (
	"errors"
	"net/http"

	"github.com/didip/tollbooth/v8"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

func (app *Application) ratelimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		httpError := tollbooth.LimitByRequest(app.RateLimiter, c.Writer, c.Request)
		if httpError != nil {
			c.Data(httpError.StatusCode, app.RateLimiter.GetMessageContentType(), []byte(httpError.Message))
			c.Abort()
		} else {
			c.Next()
		}
	}
}

// limits request body size
func (app *Application) bodySizeMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, app.config.MaxUploadSize)
		c.Next()
	}
}

// parses form
func (app *Application) apiMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > 0 {
			if err := c.Request.ParseMultipartForm(app.config.MaxUploadSize); err != nil {
				c.String(http.StatusRequestEntityTooLarge, "Too big file")
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

type SessionTokenVerification struct {
	SessionToken string `form:"token"`
}

func (app *Application) parseSessionTokenFromForm(c *gin.Context) (sessionToken uuid.UUID, err error) {
	var form SessionTokenVerification
	if err = c.ShouldBindWith(&form, binding.FormPost); err != nil {
		return
	}

	sessionToken, err = uuid.Parse(form.SessionToken)

	return
}

func (app *Application) parseSessionTokenFromCookieOrForm(c *gin.Context) (sessionToken uuid.UUID, err error) {
	sessionToken, err = app.parseAuthCookie(c)
	if err != nil {
		sessionToken, err = app.parseSessionTokenFromForm(c)
	}

	return
}

// Makes sure request has session token and a valid one
func (app *Application) verifySessionAuthentication() gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionToken, err := app.parseSessionTokenFromForm(c)
		if err != nil {
			// Fallback to checking cookie
			log.Debug().Msg("Validating cookie")
			var loggedIn bool
			sessionToken, _, loggedIn, _ = app.validateAuthCookie(c)
			if loggedIn {
			} else {
				c.AbortWithError(http.StatusUnauthorized, err)
				return
			}
		}

		c.Set("sessionToken", sessionToken)
		c.Next()
	}
}

// Makes sure the authenticated account is admin
func (app *Application) isAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Verify the field exists
		sessionToken, exists := c.Get("sessionToken")
		if !exists {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		if account, err := app.db.GetAccountBySessionToken(sessionToken.(uuid.UUID)); errors.Is(err, gorm.ErrRecordNotFound) {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		} else if err != nil {
			log.Err(err).Msg("Failed to find account by session token")
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		} else if account.AccountType != "ADMIN" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		c.Next()
	}
}

// Makes sure the account token provided is valid
func (app *Application) isSessionAuthenticated() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Verify the field exists
		sessionToken, exists := c.Get("sessionToken")
		if !exists {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		if _, err := app.db.GetAccountBySessionToken(sessionToken.(uuid.UUID)); errors.Is(err, gorm.ErrRecordNotFound) {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		} else if err != nil {
			log.Err(err).Msg("Failed to find account by session token")
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
				c.AbortWithError(http.StatusUnauthorized, err)
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
			sessionToken, err := app.parseSessionTokenFromCookieOrForm(c)
			if err != nil {
				c.AbortWithError(http.StatusUnauthorized, err)
				return
			}

			c.Set("sessionToken", sessionToken)
		}

		c.Next()
	}
}
