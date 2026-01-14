package cmd

import (
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gabriel-vasile/mimetype"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func (app *Application) deleteFile(fileName string) (err error) {
	switch app.config.FileStorageMethod {
	case fileStorageLocal:
		err = os.Remove(filepath.Join(app.config.DataFolder, fileName))
	case fileStorageS3:
		err = app.deleteFileS3(fileName)
	default:
		err = ErrUnknownStorageMethod
	}

	return
}

func randomString() string {
	return rand.Text()
}

func (app *Application) generateFullFileName(mime *mimetype.MIME) string {
	return fmt.Sprintf("%s%s", randomString(), mime.Extension())
}

func (app *Application) isValidUploadToken(uploadToken uuid.UUID) (bool, error) {
	_, err := app.db.GetAccountByUploadToken(uploadToken)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}
