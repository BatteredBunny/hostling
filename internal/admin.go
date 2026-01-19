package internal

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// Admin api for deleting accounts and their data
type adminDeleteAccountInput struct {
	ID uint `form:"id"`
}

var ErrCantDeleteSelf = fmt.Errorf("you can't delete yourself")

func (app *Application) adminDeleteAccount(c *gin.Context) {
	var (
		input adminDeleteAccountInput
		err   error
	)

	if err = c.MustBindWith(&input, binding.FormPost); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	sessionToken, exists := c.Get("sessionToken")
	if !exists {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	// You can't delete yourself
	if account, err := app.db.GetAccountBySessionToken(sessionToken.(uuid.UUID)); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	} else if account.ID == input.ID {
		c.AbortWithError(http.StatusBadRequest, ErrCantDeleteSelf)
		return
	}

	if err = app.deleteAccount(input.ID); err != nil {
		log.Err(err).Msg("Failed to delete account")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.String(http.StatusOK, fmt.Sprintf("Account %d deleted", input.ID))
}

type adminDeleteAccountFilesInput struct {
	ID uint `form:"id"`
}

func (app *Application) adminDeleteFiles(c *gin.Context) {
	var (
		input adminDeleteAccountFilesInput
		err   error
	)

	if err = c.MustBindWith(&input, binding.FormPost); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	if err = app.deleteFilesFromAccount(input.ID); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.String(http.StatusOK, "Files deleted")
}

type adminDeleteAccountSessionsInput struct {
	ID uint `form:"id"`
}

func (app *Application) adminDeleteSessions(c *gin.Context) {
	var (
		input adminDeleteAccountSessionsInput
		err   error
	)

	if err = c.MustBindWith(&input, binding.FormPost); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	if err = app.db.DeleteSessionsFromAccount(input.ID); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.String(http.StatusOK, "Sessions deleted")
}

type adminDeleteAccountUploadTokensInput struct {
	ID uint `form:"id"`
}

func (app *Application) adminDeleteUploadTokens(c *gin.Context) {
	var (
		input adminDeleteAccountUploadTokensInput
		err   error
	)

	if err = c.MustBindWith(&input, binding.FormPost); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	if err = app.db.DeleteUploadTokensFromAccount(input.ID); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.String(http.StatusOK, "Upload tokens deleted")
}

type adminGiveInviteCodeInput struct {
	ID   uint `form:"id"`
	Uses uint `form:"uses,default=5"` // How many uses the invite code has
}

// TODO: allow giving admin account invites
func (app *Application) adminGiveInviteCode(c *gin.Context) {
	var (
		input adminGiveInviteCodeInput
		err   error
	)

	if err = c.MustBindWith(&input, binding.FormPost); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	inviteCode, err := app.db.CreateInviteCode(input.Uses, "USER", input.ID)
	if err != nil {
		log.Err(err).Msg("Failed to create invite code")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.String(http.StatusOK, inviteCode.Code)
}
