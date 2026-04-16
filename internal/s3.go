package internal

import (
	"context"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
)

// Timeout for short S3 control-plane calls (stat, delete, presign). Upload
// and streaming read are bounded by the request context instead.
const s3Timeout = 30 * time.Second

func (app *Application) uploadFileS3(ctx context.Context, r io.Reader, size int64, fileName string) (err error) {
	_, err = app.s3client.PutObject(
		ctx,
		app.config.S3.Bucket,
		fileName,
		r,
		size,
		minio.PutObjectOptions{},
	)

	return
}

func (app *Application) deleteFileS3(fileName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), s3Timeout)
	defer cancel()
	return app.s3client.RemoveObject(ctx, app.config.S3.Bucket, fileName, minio.RemoveObjectOptions{})
}

func (app *Application) streamS3File(ctx context.Context, fileName string) (*minio.Object, error) {
	return app.s3client.GetObject(ctx, app.config.S3.Bucket, fileName, minio.GetObjectOptions{})
}
