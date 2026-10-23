package internal

import (
	"bufio"
	"context"
	"errors"
	"github.com/danielgtaylor/huma/v2"
	"io"
	"net/http"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/BatteredBunny/hostling/internal/db"
	"github.com/gabriel-vasile/mimetype"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

// Bytes sniffed from the front of an upload to detect its MIME type
// Same as mimetype's internal read size
const mimeSniffSize = 3072

const maxPaginationSkip = 1_000_000
const maxExpiryDuration = 100 * 365 * 24 * time.Hour // insane expiry length

// Api for deleting your own account
func (app *Application) accountDeleteAPI(_ context.Context, request *EmptyInput) (*TextOutput, error) {
	c := request.Gin

	account, ok := getAccount(c)
	if !ok {
		return nil, huma.NewError(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
	}

	if err := app.deleteAccount(c.Request.Context(), account.ID); err != nil {
		log.Err(err).Msg("Failed to delete own account")

		return nil, huma.NewError(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}

	return textOutput("Account deleted successfully"), nil
}

var (
	ErrStorageDeleteFailed = errors.New("storage delete failed")
	ErrPartialDeleteFailed = errors.New("one or more files failed to delete")
)

func (app *Application) deleteAccount(ctx context.Context, accountID uint) error {
	if err := app.deleteFilesFromAccount(ctx, accountID); err != nil {
		return err
	}

	return app.db.WithContext(ctx).DeleteAccount(accountID)
}

// Api for deleting a file from your account
type deleteFileAPIInput struct {
	FileName string `form:"file_name" json:"file_name" required:"true"`
}

func (app *Application) deleteFileAPI(_ context.Context, request *FormInput[deleteFileAPIInput]) (*TextOutput, error) {
	c := request.Gin
	input := request.RawBody.Data()

	var err error

	account, ok := getAccount(c)
	if !ok {
		return nil, huma.NewError(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
	}

	database := app.db.WithContext(c.Request.Context())
	owned, err := database.AccountOwnsFile(input.FileName, account.ID)
	if err != nil {
		log.Err(err).Msg("Failed to check file ownership")

		return nil, huma.NewError(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}
	if !owned {
		return nil, huma.NewError(http.StatusNotFound, http.StatusText(http.StatusNotFound))
	}

	if err = app.deleteFile(c.Request.Context(), input.FileName); err != nil {
		log.Err(err).Str("file", input.FileName).Msg("Failed to delete file from storage; keeping DB row for retry")

		return nil, huma.NewError(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}

	if err = database.DeleteFileEntry(input.FileName, account.ID); err != nil {
		log.Err(err).Msg("Failed to delete file entry")

		return nil, huma.NewError(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}

	return textOutput("Successfully deleted the file"), nil
}

// Api for toggling file public/private status
type toggleFilePublicAPIInput struct {
	FileName string `form:"file_name" required:"true"`
}

// TODO: merge into a generic image properties modification api
func (app *Application) toggleFilePublicAPI(
	_ context.Context,
	request *FormInput[toggleFilePublicAPIInput],
) (*TextOutput, error) {
	c := request.Gin
	input := request.RawBody.Data()

	var err error

	account, ok := getAccount(c)
	if !ok {
		return nil, huma.NewError(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
	}

	newPublicStatus, err := app.db.WithContext(c.Request.Context()).ToggleFilePublic(input.FileName, account.ID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, huma.NewError(http.StatusNotFound, "File not found or you don't own this file")
	} else if err != nil {
		log.Err(err).Msg("Failed to toggle file public status")

		return nil, huma.NewError(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}

	if newPublicStatus {
		return textOutput("File is now public"), nil
	} else {
		return textOutput("File is now private"), nil
	}
}

/*
Api for uploading file
curl -F "upload_token=$HOSTLING_UPLOAD_TOKEN" -F 'file=@yourfile.png' https://your-hostling/api/file/upload

Additional inputs:
expiry_timestamp: unix timestamp in seconds
expiry_date: YYYY-MM-DD in string, expiry_timestamp gets priority
plain: if set to true, api will return plain url instead of redirecting
tag: tags to add to the file
*/
func (app *Application) uploadFileAPI(_ context.Context, request *UploadInput) (*TextOutput, error) {
	c := request.Gin
	input := request.RawBody.Data()

	var expiryDate time.Time

	plainRedirect := input.Plain

	rawTags := input.Tag
	tags_exists := len(rawTags) > 0

	tags := make([]string, 0, len(rawTags))
	for _, t := range rawTags {
		t = strings.ToLower(strings.TrimSpace(t))
		if t == "" {
			continue
		}
		if utf8.RuneCountInString(t) > db.TagMaxLength {
			return nil, huma.NewError(http.StatusBadRequest, db.ErrTagTooLong.Error())
		}
		tags = append(tags, t)
	}
	slices.Sort(tags)
	tags = slices.Compact(tags)
	tags_exists = tags_exists && len(tags) > 0

	if len(tags) > db.MaxTagsPerFile {
		return nil, huma.NewError(http.StatusBadRequest, db.ErrTooManyTags.Error())
	}

	if date := input.ExpiryDate; date != "" {
		parsed, parseErr := time.Parse("2006-01-02", date)
		if parseErr != nil {
			return nil, huma.NewError(http.StatusBadRequest, "Invalid expiry_date (want YYYY-MM-DD)")
		}
		expiryDate = parsed.Add(24*time.Hour - time.Second)
	}

	if timestamp := input.ExpiryTimestamp; timestamp != "" {
		unixSecs, parseErr := strconv.ParseInt(timestamp, 10, 64)
		if parseErr != nil {
			return nil, huma.NewError(http.StatusBadRequest, "Invalid expiry_timestamp (want unix seconds)")
		}
		expiryDate = time.Unix(unixSecs, 0)
	}

	if !expiryDate.IsZero() {
		now := time.Now()
		if expiryDate.Before(now) {
			return nil, huma.NewError(http.StatusBadRequest, "Can't specify expiry in the past, sorry.")
		}
		if expiryDate.After(now.Add(maxExpiryDuration)) {
			return nil, huma.NewError(http.StatusBadRequest, "Expiry too far in the future.")
		}
	}

	fileHeader := request.FileHeader
	fileRaw, err := fileHeader.Open()
	if err != nil {
		return nil, huma.Error500InternalServerError("Could not open uploaded file")
	}
	defer fileRaw.Close()

	originalFileName := fileHeader.Filename

	// Peek at the first few KB for MIME detection without consuming the
	// stream, then hand the buffered reader straight to storage.
	body := bufio.NewReaderSize(fileRaw, mimeSniffSize)
	header, err := body.Peek(mimeSniffSize)
	if err != nil && !errors.Is(err, io.EOF) {
		if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
			return nil, huma.NewError(http.StatusRequestEntityTooLarge, "Too big file")
		}
		log.Err(err).Msg("Failed to read uploaded file header")

		return nil, huma.NewError(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}

	mime := mimetype.Detect(header)
	fullFileName := app.generateFullFileName(mime)

	var written int64
	switch app.config.FileStorageMethod {
	case fileStorageS3:
		written, err = app.uploadFileS3(c.Request.Context(), body, fileHeader.Size, fullFileName)
	case fileStorageLocal:
		written, err = writeLocalUploadFile(
			c.Request.Context(), filepath.Join(app.config.DataFolder, fullFileName), body, fileRaw, fileHeader.Size,
		)
	default:
		err = ErrUnknownStorageMethod
	}

	if err != nil {
		log.Err(err).Msg("Upload issue")

		return nil, huma.NewError(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}

	entry := db.CreateFileEntryInput{
		Files: db.Files{
			FileName:         fullFileName,
			OriginalFileName: originalFileName,
			FileSize:         uint(written),
			MimeType:         mime.String(),
			ExpiryDate:       expiryDate,
			Public:           true,
		},
	}

	if tags_exists {
		for _, tag := range tags {
			entry.Files.Tags = append(entry.Files.Tags, db.Tag{Name: tag})
		}
	}

	if sid, ok := getSessionToken(c); ok {
		entry.SessionTokenHash = sid
	} else if uid, ok := getUploadToken(c); ok {
		entry.UploadTokenHash = uid
	} else {
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(c.Request.Context()), 30*time.Second)
		defer cancel()
		if deleteErr := app.deleteFile(cleanupCtx, fullFileName); deleteErr != nil {
			log.Err(deleteErr).Str("file", fullFileName).Msg("Failed to clean up blob after auth check failed")
		}

		return nil, huma.NewError(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
	}

	if err = app.db.WithContext(c.Request.Context()).CreateFileEntry(entry); err != nil {
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(c.Request.Context()), 30*time.Second)
		defer cancel()
		if deleteErr := app.deleteFile(cleanupCtx, fullFileName); deleteErr != nil {
			log.Err(deleteErr).Str("file", fullFileName).Msg("Failed to clean up blob after DB insert failed")
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, huma.NewError(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
		}
		log.Err(err).Msg("Failed to create file entry")

		return nil, huma.NewError(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}

	if plainRedirect {
		return textOutput("/" + fullFileName), nil
	} else {
		return &TextOutput{Status: http.StatusTemporaryRedirect, Location: "/" + fullFileName}, nil
	}
}

type TagInput struct {
	FileName string  `form:"file_name" required:"true"`
	Tag      TagName `form:"tag"       required:"true" json:"tag"`
}

func (app *Application) addTagAPI(_ context.Context, request *FormInput[TagInput]) (*TextOutput, error) {
	c := request.Gin
	input := request.RawBody.Data()

	var err error

	if utf8.RuneCountInString(string(input.Tag)) > db.TagMaxLength {
		return nil, huma.NewError(http.StatusBadRequest, db.ErrTagTooLong.Error())
	}

	account, ok := getAccount(c)
	if !ok {
		return nil, huma.NewError(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
	}

	err = app.db.WithContext(c.Request.Context()).AddTagToFile(input.FileName, string(input.Tag), account.ID)
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return nil, huma.NewError(http.StatusNotFound, "File not found or you don't own this file")
	case errors.Is(err, db.ErrTagAlreadyOnFile):
		return nil, huma.NewError(http.StatusBadRequest, db.ErrTagAlreadyOnFile.Error())
	case errors.Is(err, db.ErrTooManyTags):
		return nil, huma.NewError(http.StatusBadRequest, db.ErrTooManyTags.Error())
	case err != nil:
		log.Err(err).Msg("Failed to add tag to file")

		return nil, huma.NewError(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}

	return textOutput("Tag added successfully"), nil
}

func (app *Application) deleteTagAPI(_ context.Context, request *FormInput[TagInput]) (*TextOutput, error) {
	c := request.Gin
	input := request.RawBody.Data()

	var err error

	account, ok := getAccount(c)
	if !ok {
		return nil, huma.NewError(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
	}

	database := app.db.WithContext(c.Request.Context())
	hasTag, err := database.FileHasTag(input.FileName, string(input.Tag), account.ID)
	if err != nil {
		return nil, huma.NewError(http.StatusInternalServerError, err.Error())
	}

	if !hasTag {
		return nil, huma.NewError(http.StatusBadRequest, "File does not have this tag")
	}

	if err := database.RemoveTagFromFile(input.FileName, string(input.Tag), account.ID); err != nil {
		log.Err(err).Msg("Failed to remove tag from file")

		return nil, huma.NewError(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}

	return textOutput("Tag removed successfully"), nil
}
