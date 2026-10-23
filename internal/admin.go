package internal

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

// Admin api for deleting accounts and their data
type adminDeleteAccountInput struct {
	ID uint `form:"id" minimum:"1" required:"true"`
}

var ErrCantDeleteSelf = fmt.Errorf("you can't delete yourself")

func (app *Application) adminDeleteAccount(
	_ context.Context,
	request *FormInput[adminDeleteAccountInput],
) (*TextOutput, error) {
	c := request.Gin
	input := request.RawBody.Data()

	account, _ := getAccount(c)
	if account.ID == input.ID {
		return nil, huma.NewError(http.StatusBadRequest, ErrCantDeleteSelf.Error())
	}

	if err := app.deleteAccount(c.Request.Context(), input.ID); err != nil {
		log.Err(err).Msg("Failed to delete account")

		return nil, huma.NewError(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}

	return textOutput(fmt.Sprintf("Account %d deleted", input.ID)), nil
}

type adminDeleteAccountFilesInput struct {
	ID uint `form:"id" minimum:"1" required:"true"`
}

func (app *Application) adminDeleteFiles(
	_ context.Context,
	request *FormInput[adminDeleteAccountFilesInput],
) (*TextOutput, error) {
	c := request.Gin
	input := request.RawBody.Data()

	var err error

	if err = app.deleteFilesFromAccount(c.Request.Context(), input.ID); err != nil {
		log.Err(err).Uint("account_id", input.ID).Msg("Failed to delete files from account")

		return nil, huma.NewError(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}

	return textOutput("Files deleted"), nil
}

type adminDeleteAccountSessionsInput struct {
	ID uint `form:"id" minimum:"1" required:"true"`
}

func (app *Application) adminDeleteSessions(
	ctx context.Context,
	request *FormInput[adminDeleteAccountSessionsInput],
) (*TextOutput, error) {
	input := request.RawBody.Data()

	var err error

	if err = app.db.WithContext(ctx).DeleteSessionsFromAccount(input.ID); err != nil {
		log.Err(err).Uint("account_id", input.ID).Msg("Failed to delete sessions from account")

		return nil, huma.NewError(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}

	return textOutput("Sessions deleted"), nil
}

type adminDeleteAccountUploadTokensInput struct {
	ID uint `form:"id" minimum:"1" required:"true"`
}

func (app *Application) adminDeleteUploadTokens(
	ctx context.Context,
	request *FormInput[adminDeleteAccountUploadTokensInput],
) (*TextOutput, error) {
	input := request.RawBody.Data()

	var err error

	if err = app.db.WithContext(ctx).DeleteUploadTokensFromAccount(input.ID); err != nil {
		log.Err(err).Uint("account_id", input.ID).Msg("Failed to delete upload tokens from account")

		return nil, huma.NewError(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}

	return textOutput("Upload tokens deleted"), nil
}

type adminGiveInviteCodeInput struct {
	ID   uint `form:"id"   minimum:"1" required:"true"`
	Uses uint `form:"uses"             required:"false" default:"5"` // How many uses the invite code has
}

// TODO: allow giving admin account invites
func (app *Application) adminGiveInviteCode(
	ctx context.Context,
	request *FormInput[adminGiveInviteCodeInput],
) (*TextOutput, error) {
	input := request.RawBody.Data()

	var err error

	database := app.db.WithContext(ctx)
	if _, err = database.GetAccountByID(input.ID); errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, huma.NewError(http.StatusNotFound, "Account not found")
	} else if err != nil {
		log.Err(err).Uint("account_id", input.ID).Msg("Failed to look up account")

		return nil, huma.NewError(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}

	inviteCode, err := database.CreateInviteCode(input.Uses, "USER", input.ID)
	if err != nil {
		log.Err(err).Msg("Failed to create invite code")

		return nil, huma.NewError(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}

	return textOutput(inviteCode.Code), nil
}
