package internal

import (
	"errors"
	"io"
	"mime/multipart"
	"os"
)

// writeLocalUploadFile reuses a complete, disk-backed multipart upload when its
// temporary file and final destination share a filesystem. Linking avoids a
// second file copy, preserves exclusive creation, and lets net/http own cleanup
// of the temporary name. Shared multipart sections and memory-backed uploads
// continue through the normal buffered-reader copy path.
func writeLocalUploadFile(path string, body io.Reader, upload multipart.File, size int64) (int64, error) {
	if file, ok := upload.(*os.File); ok {
		info, err := file.Stat()
		if err == nil && info.Mode().IsRegular() && info.Size() == size {
			if err := os.Link(file.Name(), path); err == nil {
				if err := file.Sync(); err != nil {
					return 0, errors.Join(err, os.Remove(path))
				}

				return size, nil
			} else if errors.Is(err, os.ErrExist) {
				return 0, err
			}
			// Cross-device links, unsupported filesystems, and permission
			// restrictions retain the existing portable copy behavior.
		}
	}

	return writeLocalFile(path, body)
}
