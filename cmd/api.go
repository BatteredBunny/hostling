package cmd

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/BatteredBunny/hostling/cmd/db"
	"github.com/gabriel-vasile/mimetype"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

// Api for deleting your own account
func (app *Application) accountDeleteAPI(c *gin.Context) {
	sessionToken, exists := c.Get("sessionToken")
	if !exists {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	account, err := app.db.GetAccountBySessionToken(sessionToken.(uuid.UUID))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	} else if err != nil {
		log.Err(err).Msg("Failed to fetch user by session token")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	if err = app.deleteAccount(account.ID); err != nil {
		log.Err(err).Msg("Failed to delete own account")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.String(http.StatusOK, "Account deleted successfully")
}

func (app *Application) deleteAccount(userID uint) (err error) {
	if err = app.db.DeleteSessionsFromAccount(userID); err != nil {
		return
	}

	if err = app.db.DeleteUploadTokensFromAccount(userID); err != nil {
		return
	}

	if err = app.db.DeleteInviteCodesFromAccount(userID); err != nil {
		return
	}

	files, err := app.db.GetAllFilesFromAccount(userID)
	if err != nil {
		return
	}

	for _, file := range files {
		if err = app.deleteFile(file.FileName); err != nil {
			log.Err(err).Msg("Failed to delete file")
		}
	}

	if err = app.db.DeleteFilesFromAccount(userID); err != nil {
		return
	}

	if err = app.db.DeleteAccount(userID); err != nil {
		return
	}

	return
}

// Api for deleting a file from your account
type deleteFileAPIInput struct {
	FileName string `form:"file_name"`
}

func (app *Application) deleteFileAPI(c *gin.Context) {
	var input deleteFileAPIInput
	var err error

	if err = c.MustBindWith(&input, binding.FormPost); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	if input.FileName == "" {
		c.String(http.StatusBadRequest, "File name is required")
		c.Abort()
		return
	}

	rawSessionToken, sessionTokenExists := c.Get("sessionToken")
	rawUploadToken, uploadTokenExists := c.Get("uploadToken")

	var sessionToken uuid.NullUUID
	var uploadToken uuid.NullUUID
	if sessionTokenExists {
		sessionToken = uuid.NullUUID{UUID: rawSessionToken.(uuid.UUID), Valid: true}
	} else if uploadTokenExists {
		uploadToken = uuid.NullUUID{UUID: rawUploadToken.(uuid.UUID), Valid: true}
	} else {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	// Makes sure the file exists
	var exists bool
	if exists, err = app.db.FileExists(input.FileName); err != nil {
		log.Err(err).Msg("Failed to check if file exists")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	} else if !exists {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	// Deletes file
	if err = app.deleteFile(input.FileName); err != nil {
		log.Err(err).Msg("Failed to delete file")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	if err = app.db.DeleteFileEntry(input.FileName, uploadToken, sessionToken); err != nil { // Deletes file entry from database
		log.Err(err).Msg("Failed to delete file entry")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.String(http.StatusOK, "Successfully deleted the file")
}

// Api for toggling file public/private status
type toggleFilePublicAPIInput struct {
	FileName string `form:"file_name"`
}

// TODO: merge into a generic image properties modification api
func (app *Application) toggleFilePublicAPI(c *gin.Context) {
	var (
		input toggleFilePublicAPIInput
		err   error
	)
	if err = c.MustBindWith(&input, binding.FormPost); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	if input.FileName == "" {
		c.String(http.StatusBadRequest, "File name is required")
		c.Abort()
		return
	}

	sessionToken, exists := c.Get("sessionToken")
	if !exists {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	account, err := app.db.GetAccountBySessionToken(sessionToken.(uuid.UUID))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	} else if err != nil {
		log.Err(err).Msg("Failed to fetch user by session token")
		c.AbortWithStatus(http.StatusInternalServerError)
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
*/
func (app *Application) uploadFileAPI(c *gin.Context) {
	var expiryDate time.Time

	date, exists := c.GetPostForm("expiry_date")
	if exists {
		expiryDate, _ = time.Parse("2006-01-02", date)
	}

	timestamp, exists := c.GetPostForm("expiry_timestamp")
	if exists {
		log.Info().Any("expiry_date", timestamp).Msg("Expiry date provided")
		unixSecs, err := strconv.Atoi(timestamp)
		if err == nil {
			expiryDate = time.Unix(int64(unixSecs), 0)
		}
	}

	if !expiryDate.IsZero() && expiryDate.Before(time.Now()) {
		c.String(http.StatusBadRequest, "Can't specify expiry in the past, sorry.")
		return
	}

	fileRaw, fileHeader, err := c.Request.FormFile("file")
	if err != nil {
		c.String(http.StatusBadRequest, "No file provided")
		c.Abort()
		return
	}
	defer fileRaw.Close()

	originalFileName := fileHeader.Filename

	file, err := io.ReadAll(fileRaw)
	if err != nil {
		log.Err(err).Msg("Failed to read uploaded file")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	mime := mimetype.Detect(file)
	fullFileName := app.generateFullFileName(mime)

	switch app.config.FileStorageMethod {
	case fileStorageS3:
		err = app.uploadFileS3(file, fullFileName)
	case fileStorageLocal:
		err = os.WriteFile(filepath.Join(app.config.DataFolder, fullFileName), file, 0o600)
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
			FileSize:         uint(len(file)),
			MimeType:         mime.String(),
			ExpiryDate:       expiryDate,
			Public:           true,
		},
	}

	sessionToken, sessionTokenExists := c.Get("sessionToken")
	uploadToken, uploadTokenExists := c.Get("uploadToken")

	if sessionTokenExists {
		input.SessionToken = uuid.NullUUID{
			UUID:  sessionToken.(uuid.UUID),
			Valid: true,
		}
	} else if uploadTokenExists {
		input.UploadToken = uuid.NullUUID{
			UUID:  uploadToken.(uuid.UUID),
			Valid: true,
		}
	} else {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	if err = app.db.CreateFileEntry(input); errors.Is(err, gorm.ErrRecordNotFound) {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	} else if err != nil {
		log.Err(err).Msg("Failed to create file entry")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, "/"+fullFileName)
}
