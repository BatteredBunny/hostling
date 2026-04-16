package internal

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gabriel-vasile/mimetype"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"gorm.io/gorm"
)

func (app *Application) fileExistsInStorage(fileName string) (bool, error) {
	switch app.config.FileStorageMethod {
	case fileStorageLocal:
		_, err := os.Stat(filepath.Join(app.config.DataFolder, fileName))
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return err == nil, err
	case fileStorageS3:
		_, err := app.s3client.StatObject(
			context.Background(),
			app.config.S3.Bucket,
			fileName,
			minio.StatObjectOptions{},
		)
		if err != nil {
			errResp := minio.ToErrorResponse(err)
			if errResp.Code == "NoSuchKey" {
				return false, nil
			}
			return false, err
		}
		return true, nil
	default:
		return false, ErrUnknownStorageMethod
	}
}

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
