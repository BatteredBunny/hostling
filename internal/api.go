package internal

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/BatteredBunny/hostling/internal/db"
	"github.com/gabriel-vasile/mimetype"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

// Bytes sniffed from the front of an upload to detect its MIME type
// Same as mimetype's internal read size
const mimeSniffSize = 3072

const maxPaginationSkip = 1_000_000
const maxExpiryDuration = 100 * 365 * 24 * time.Hour // insane expiry length

// Api for deleting your own account
func (app *Application) accountDeleteAPI(c *gin.Context) {
	account, ok := getAccount(c)
	if !ok {
		c.AbortWithStatus(http.StatusUnauthorized)

		return
	}

	if err := app.deleteAccount(c.Request.Context(), account.ID); err != nil {
		log.Err(err).Msg("Failed to delete own account")
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	c.String(http.StatusOK, "Account deleted successfully")
}

var (
	ErrStorageDeleteFailed = errors.New("storage delete failed")
	ErrPartialDeleteFailed = errors.New("one or more files failed to delete")
)

func (app *Application) deleteAccount(ctx context.Context, accountID uint) error {
	if err := app.deleteFilesFromAccount(ctx, accountID); err != nil {
		return err
	}

	return app.db.DeleteAccount(accountID)
}

// Api for deleting a file from your account
type deleteFileAPIInput struct {
	FileName string `form:"file_name" binding:"required"`
}

func (app *Application) deleteFileAPI(c *gin.Context) {
	var input deleteFileAPIInput
	var err error

	if err = c.MustBindWith(&input, binding.FormPost); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		c.Abort()

		return
	}

	account, ok := getAccount(c)
	if !ok {
		c.AbortWithStatus(http.StatusUnauthorized)

		return
	}

	owned, err := app.db.AccountOwnsFile(input.FileName, account.ID)
	if err != nil {
		log.Err(err).Msg("Failed to check file ownership")
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}
	if !owned {
		c.AbortWithStatus(http.StatusNotFound)

		return
	}

	if err = app.deleteFile(c.Request.Context(), input.FileName); err != nil {
		log.Err(err).Str("file", input.FileName).Msg("Failed to delete file from storage; keeping DB row for retry")
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	if err = app.db.DeleteFileEntry(input.FileName, account.ID); err != nil {
		log.Err(err).Msg("Failed to delete file entry")
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	c.String(http.StatusOK, "Successfully deleted the file")
}

// Api for toggling file public/private status
type toggleFilePublicAPIInput struct {
	FileName string `form:"file_name" binding:"required"`
}

// TODO: merge into a generic image properties modification api
func (app *Application) toggleFilePublicAPI(c *gin.Context) {
	var (
		input toggleFilePublicAPIInput
		err   error
	)
	if err = c.MustBindWith(&input, binding.FormPost); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		c.Abort()

		return
	}

	account, ok := getAccount(c)
	if !ok {
		c.AbortWithStatus(http.StatusUnauthorized)

		return
	}

	newPublicStatus, err := app.db.ToggleFilePublic(input.FileName, account.ID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.String(http.StatusNotFound, "File not found or you don't own this file")

		return
	} else if err != nil {
		log.Err(err).Msg("Failed to toggle file public status")
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	if newPublicStatus {
		c.String(http.StatusOK, "File is now public")
	} else {
		c.String(http.StatusOK, "File is now private")
	}
}

/*
Api for uploading file
curl -F 'upload_token=1234567890' -F 'file=@yourfile.png'

Additional inputs:
expiry_timestamp: unix timestamp in seconds
expiry_date: YYYY-MM-DD in string, expiry_timestamp gets priority
plain: if set to true, api will return plain url instead of redirecting
tag: tags to add to the file
*/
func (app *Application) uploadFileAPI(c *gin.Context) {
	var expiryDate time.Time

	plainRedirect := c.PostForm("plain") == "true"

	rawTags, tags_exists := c.GetPostFormArray("tag")

	tags := make([]string, 0, len(rawTags))
	for _, t := range rawTags {
		t = strings.ToLower(strings.TrimSpace(t))
		if t == "" {
			continue
		}
		if len(t) > db.TagMaxLength {
			c.String(http.StatusBadRequest, db.ErrTagTooLong.Error())
			c.Abort()

			return
		}
		tags = append(tags, t)
	}
	slices.Sort(tags)
	tags = slices.Compact(tags)
	tags_exists = tags_exists && len(tags) > 0

	if len(tags) > db.MaxTagsPerFile {
		c.String(http.StatusBadRequest, db.ErrTooManyTags.Error())
		c.Abort()

		return
	}

	if date, exists := c.GetPostForm("expiry_date"); exists && date != "" {
		parsed, parseErr := time.Parse("2006-01-02", date)
		if parseErr != nil {
			c.String(http.StatusBadRequest, "Invalid expiry_date (want YYYY-MM-DD)")

			return
		}
		expiryDate = parsed.Add(24*time.Hour - time.Second)
	}

	if timestamp, exists := c.GetPostForm("expiry_timestamp"); exists && timestamp != "" {
		unixSecs, parseErr := strconv.ParseInt(timestamp, 10, 64)
		if parseErr != nil {
			c.String(http.StatusBadRequest, "Invalid expiry_timestamp (want unix seconds)")

			return
		}
		expiryDate = time.Unix(unixSecs, 0)
	}

	if !expiryDate.IsZero() {
		now := time.Now()
		if expiryDate.Before(now) {
			c.String(http.StatusBadRequest, "Can't specify expiry in the past, sorry.")

			return
		}
		if expiryDate.After(now.Add(maxExpiryDuration)) {
			c.String(http.StatusBadRequest, "Expiry too far in the future.")

			return
		}
	}

	fileRaw, fileHeader, err := c.Request.FormFile("file")
	if err != nil {
		c.String(http.StatusBadRequest, "No file provided")
		c.Abort()

		return
	}
	defer fileRaw.Close()

	originalFileName := fileHeader.Filename

	// Peek at the first few KB for MIME detection without consuming the
	// stream, then hand the buffered reader straight to storage.
	body := bufio.NewReaderSize(fileRaw, mimeSniffSize)
	header, err := body.Peek(mimeSniffSize)
	if err != nil && !errors.Is(err, io.EOF) {
		if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
			c.String(http.StatusRequestEntityTooLarge, "Too big file")
			c.Abort()

			return
		}
		log.Err(err).Msg("Failed to read uploaded file header")
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	mime := mimetype.Detect(header)
	fullFileName := app.generateFullFileName(mime)

	var written int64
	switch app.config.FileStorageMethod {
	case fileStorageS3:
		written, err = app.uploadFileS3(c.Request.Context(), body, fileHeader.Size, fullFileName)
	case fileStorageLocal:
		written, err = writeLocalFile(filepath.Join(app.config.DataFolder, fullFileName), body)
	default:
		err = ErrUnknownStorageMethod
	}

	if err != nil {
		log.Err(err).Msg("Upload issue")
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	input := db.CreateFileEntryInput{
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
			input.Files.Tags = append(input.Files.Tags, db.Tag{Name: tag})
		}
	}

	if sid, ok := getSessionToken(c); ok {
		input.SessionToken = uuid.NullUUID{UUID: sid, Valid: true}
	} else if uid, ok := getUploadToken(c); ok {
		input.UploadToken = uuid.NullUUID{UUID: uid, Valid: true}
	} else {
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(c.Request.Context()), 30*time.Second)
		defer cancel()
		if deleteErr := app.deleteFile(cleanupCtx, fullFileName); deleteErr != nil {
			log.Err(deleteErr).Str("file", fullFileName).Msg("Failed to clean up blob after auth check failed")
		}
		c.AbortWithStatus(http.StatusUnauthorized)

		return
	}

	if err = app.db.CreateFileEntry(input); err != nil {
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(c.Request.Context()), 30*time.Second)
		defer cancel()
		if deleteErr := app.deleteFile(cleanupCtx, fullFileName); deleteErr != nil {
			log.Err(deleteErr).Str("file", fullFileName).Msg("Failed to clean up blob after DB insert failed")
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.AbortWithStatus(http.StatusUnauthorized)

			return
		}
		log.Err(err).Msg("Failed to create file entry")
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	if plainRedirect {
		c.String(http.StatusOK, "/"+fullFileName)
	} else {
		c.Redirect(http.StatusTemporaryRedirect, "/"+fullFileName)
	}
}

type TagInput struct {
	FileName string `form:"file_name" binding:"required"`
	Tag      string `form:"tag"       binding:"required"`
}

func (app *Application) addTagAPI(c *gin.Context) {
	var (
		input TagInput
		err   error
	)
	if err = c.MustBindWith(&input, binding.FormPost); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		c.Abort()

		return
	}

	if len(input.Tag) > db.TagMaxLength {
		c.String(http.StatusBadRequest, db.ErrTagTooLong.Error())
		c.Abort()

		return
	}

	account, ok := getAccount(c)
	if !ok {
		c.AbortWithStatus(http.StatusUnauthorized)

		return
	}

	err = app.db.AddTagToFile(input.FileName, input.Tag, account.ID)
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		c.String(http.StatusNotFound, "File not found or you don't own this file")

		return
	case errors.Is(err, db.ErrTagAlreadyOnFile):
		c.String(http.StatusBadRequest, db.ErrTagAlreadyOnFile.Error())

		return
	case errors.Is(err, db.ErrTooManyTags):
		c.String(http.StatusBadRequest, db.ErrTooManyTags.Error())

		return
	case err != nil:
		log.Err(err).Msg("Failed to add tag to file")
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	c.String(http.StatusOK, "Tag added successfully")
}

func (app *Application) deleteTagAPI(c *gin.Context) {
	var (
		input TagInput
		err   error
	)
	if err = c.MustBindWith(&input, binding.FormPost); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		c.Abort()

		return
	}

	account, ok := getAccount(c)
	if !ok {
		c.AbortWithStatus(http.StatusUnauthorized)

		return
	}

	hasTag, err := app.db.FileHasTag(input.FileName, input.Tag, account.ID)
	if err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, err)

		return
	}

	if !hasTag {
		c.String(http.StatusBadRequest, "File does not have this tag")

		return
	}

	if err := app.db.RemoveTagFromFile(input.FileName, input.Tag, account.ID); err != nil {
		log.Err(err).Msg("Failed to remove tag from file")
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	c.String(http.StatusOK, "Tag removed successfully")
}
