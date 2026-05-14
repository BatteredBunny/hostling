package internal

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/BatteredBunny/hostling/internal/db"
	"github.com/gabriel-vasile/mimetype"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

func (app *Application) validateOrAbort(c *gin.Context) (account db.Accounts, loggedIn, ok bool) {
	_, account, loggedIn, err := app.validateAuthCookie(c)
	if errors.Is(err, ErrInvalidAuthCookie) {
		app.clearAuthCookie(c)
	} else if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)

		return
	}
	ok = true

	return
}

// requireAuth is validateOrAbort + redirect to /login when not logged in.
func (app *Application) requireAuth(c *gin.Context) (account db.Accounts, ok bool) {
	account, loggedIn, ok := app.validateOrAbort(c)
	if !ok {
		return
	}
	if !loggedIn {
		c.Redirect(http.StatusTemporaryRedirect, "/login")
		ok = false

		return
	}

	return
}

func writeLocalFile(path string, body io.Reader) (written int64, err error) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()
	if written, err = io.Copy(f, body); err != nil {
		return
	}
	err = f.Sync()

	return
}

// btw a missing file is treated as success.
func (app *Application) deleteFile(ctx context.Context, fileName string) (err error) {
	switch app.config.FileStorageMethod {
	case fileStorageLocal:
		err = os.Remove(filepath.Join(app.config.DataFolder, fileName))
		if errors.Is(err, os.ErrNotExist) {
			err = nil
		}
	case fileStorageS3:
		err = app.deleteFileS3(ctx, fileName)
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			log.Warn().Str("file", fileName).Msg("S3 delete returned NoSuchKey, blob was missing")
			err = nil
		}
	default:
		err = ErrUnknownStorageMethod
	}

	return
}

func randomString() string {
	return rand.Text()
}

const appSecretSize = 32

func writeFileExclusive(path string, data []byte, mode os.FileMode) (err error) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()
	if _, err = f.Write(data); err != nil {
		return
	}
	err = f.Sync()

	return
}

func readAppSecret(path string) (secret []byte, err error) {
	if secret, err = os.ReadFile(path); err != nil {
		return
	}
	if len(secret) != appSecretSize {
		err = fmt.Errorf("app secret at %s has unexpected size %d (want %d)", path, len(secret), appSecretSize)
	}

	return
}

func loadOrCreateAppSecret(dataFolder string) (secret []byte, err error) {
	if dataFolder == "" {
		err = errors.New("data_folder must be set to persist the app secret")

		return
	}
	if err = os.MkdirAll(dataFolder, 0o770); err != nil {
		return
	}

	path := filepath.Join(dataFolder, "secret.key")
	if secret, err = readAppSecret(path); err == nil || !errors.Is(err, os.ErrNotExist) {
		return
	}

	secret = make([]byte, appSecretSize)
	if _, err = rand.Read(secret); err != nil {
		return
	}

	if err = writeFileExclusive(path, secret, 0o600); errors.Is(err, os.ErrExist) {
		secret, err = readAppSecret(path)
	}

	return
}

func (app *Application) generateFullFileName(mime *mimetype.MIME) string {
	ext := mime.Extension()
	if ext == "" {
		ext = ".bin"
	}

	return randomString() + ext
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
