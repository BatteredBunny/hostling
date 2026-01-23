package internal

import (
	"bytes"
	"context"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/encrypt"
)

func (app *Application) uploadFileS3(file []byte, fileName string) (err error) {
	opts := minio.PutObjectOptions{}
	if app.config.S3.ServerSideEncryption != "" {
		opts.ServerSideEncryption = encrypt.NewSSE()
	}

	_, err = app.s3client.PutObject(
		context.Background(),
		app.config.S3.Bucket,
		fileName,
		bytes.NewReader(file),
		int64(len(file)),
		opts,
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
