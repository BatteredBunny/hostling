package internal

import (
	"bytes"
	"context"

	"github.com/minio/minio-go/v7"
)

func (app *Application) uploadFileS3(file []byte, fileName string) (err error) {
	_, err = app.s3client.PutObject(
		context.Background(),
		app.config.S3.Bucket,
		fileName,
		bytes.NewReader(file),
		int64(len(file)),
		minio.PutObjectOptions{},
	)

	return
}

func (app *Application) deleteFileS3(fileName string) (err error) {
	err = app.s3client.RemoveObject(
		context.Background(),
		app.config.S3.Bucket,
		fileName,
		minio.RemoveObjectOptions{},
	)

	return
}
