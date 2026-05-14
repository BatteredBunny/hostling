package internal

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

// Admin api for deleting accounts and their data
type adminDeleteAccountInput struct {
	ID uint `form:"id" binding:"required"`
}

var ErrCantDeleteSelf = fmt.Errorf("you can't delete yourself")

func (app *Application) adminDeleteAccount(c *gin.Context) {
	var input adminDeleteAccountInput
	if err := c.MustBindWith(&input, binding.FormPost); err != nil {
		_ = c.AbortWithError(http.StatusBadRequest, err)

		return
	}

	account, _ := getAccount(c)
	if account.ID == input.ID {
		_ = c.AbortWithError(http.StatusBadRequest, ErrCantDeleteSelf)

		return
	}

	if err := app.deleteAccount(c.Request.Context(), input.ID); err != nil {
		log.Err(err).Msg("Failed to delete account")
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	c.String(http.StatusOK, fmt.Sprintf("Account %d deleted", input.ID))
}

type adminDeleteAccountFilesInput struct {
	ID uint `form:"id" binding:"required"`
}

func (app *Application) adminDeleteFiles(c *gin.Context) {
	var (
		input adminDeleteAccountFilesInput
		err   error
	)

	if err = c.MustBindWith(&input, binding.FormPost); err != nil {
		_ = c.AbortWithError(http.StatusBadRequest, err)

		return
	}

	if err = app.deleteFilesFromAccount(c.Request.Context(), input.ID); err != nil {
		log.Err(err).Uint("account_id", input.ID).Msg("Failed to delete files from account")
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	c.String(http.StatusOK, "Files deleted")
}

type adminDeleteAccountSessionsInput struct {
	ID uint `form:"id" binding:"required"`
}

func (app *Application) adminDeleteSessions(c *gin.Context) {
	var (
		input adminDeleteAccountSessionsInput
		err   error
	)

	if err = c.MustBindWith(&input, binding.FormPost); err != nil {
		_ = c.AbortWithError(http.StatusBadRequest, err)

		return
	}

	if err = app.db.DeleteSessionsFromAccount(input.ID); err != nil {
		log.Err(err).Uint("account_id", input.ID).Msg("Failed to delete sessions from account")
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	c.String(http.StatusOK, "Sessions deleted")
}

type adminDeleteAccountUploadTokensInput struct {
	ID uint `form:"id" binding:"required"`
}

func (app *Application) adminDeleteUploadTokens(c *gin.Context) {
	var (
		input adminDeleteAccountUploadTokensInput
		err   error
	)

	if err = c.MustBindWith(&input, binding.FormPost); err != nil {
		_ = c.AbortWithError(http.StatusBadRequest, err)

		return
	}

	if err = app.db.DeleteUploadTokensFromAccount(input.ID); err != nil {
		log.Err(err).Uint("account_id", input.ID).Msg("Failed to delete upload tokens from account")
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	c.String(http.StatusOK, "Upload tokens deleted")
}

type adminGiveInviteCodeInput struct {
	ID   uint `form:"id"             binding:"required"`
	Uses uint `form:"uses,default=5"` // How many uses the invite code has
}

// TODO: allow giving admin account invites
func (app *Application) adminGiveInviteCode(c *gin.Context) {
	var (
		input adminGiveInviteCodeInput
		err   error
	)

	if err = c.MustBindWith(&input, binding.FormPost); err != nil {
		_ = c.AbortWithError(http.StatusBadRequest, err)

		return
	}

	if _, err = app.db.GetAccountByID(input.ID); errors.Is(err, gorm.ErrRecordNotFound) {
		c.String(http.StatusNotFound, "Account not found")

		return
	} else if err != nil {
		log.Err(err).Uint("account_id", input.ID).Msg("Failed to look up account")
		c.AbortWithStatus(http.StatusInternalServerError)

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
