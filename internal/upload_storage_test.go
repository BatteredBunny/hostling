package internal

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"testing"
)

func multipartStorageFixture(t *testing.T, contents ...[]byte) (*multipart.Form, multipart.File) {
	t.Helper()
	var encoded bytes.Buffer
	writer := multipart.NewWriter(&encoded)
	for _, content := range contents {
		part, err := writer.CreateFormFile("file", "upload.bin")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write(content); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	// A small memory budget exercises the same disk-backed representation as
	// a >32 MiB production upload without allocating a large test payload.
	form, err := multipart.NewReader(bytes.NewReader(encoded.Bytes()), writer.Boundary()).ReadForm(1)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = form.RemoveAll() })
	file, err := form.File["file"][0].Open()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = file.Close() })

	return form, file
}

func TestLocalUploadReusesWholeMultipartFile(t *testing.T) {
	contents := bytes.Repeat([]byte("test upload content"), 1000)
	form, file := multipartStorageFixture(t, contents)
	disk, ok := file.(*os.File)
	if !ok {
		t.Fatal("expected a whole disk-backed multipart file")
	}
	sourceInfo, err := disk.Stat()
	if err != nil {
		t.Fatal(err)
	}
	// Put the destination beside the temp file to guarantee the same filesystem.
	dir, err := os.MkdirTemp(filepath.Dir(disk.Name()), "hostling-upload-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	destination := filepath.Join(dir, "stored.bin")
	reader := bufio.NewReader(file)
	if _, err := reader.Peek(100); err != nil {
		t.Fatal(err)
	}
	written, err := writeLocalUploadFile(context.Background(), destination, reader, file, int64(len(contents)))
	if err != nil || written != int64(len(contents)) {
		t.Fatalf("store: %d, %v", written, err)
	}
	targetInfo, err := os.Stat(destination)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(sourceInfo, targetInfo) {
		t.Fatal("whole multipart upload was copied instead of reused")
	}
	if targetInfo.Mode().Perm() != 0o600 {
		t.Fatal("stored file permissions changed")
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if err := form.RemoveAll(); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(destination)
	if err != nil || !bytes.Equal(got, contents) {
		t.Fatal("request cleanup damaged the published upload")
	}
}

func TestLocalUploadSharedMultipartFileCopiesOnlyFirstSection(t *testing.T) {
	first := bytes.Repeat([]byte("first"), 1000)
	second := bytes.Repeat([]byte("second"), 1000)
	_, file := multipartStorageFixture(t, first, second)
	if _, ok := file.(*os.File); ok {
		t.Skip("multipart file combining is disabled in this environment")
	}
	destination := filepath.Join(t.TempDir(), "stored.bin")
	reader := bufio.NewReader(file)
	if _, err := reader.Peek(100); err != nil {
		t.Fatal(err)
	}
	written, err := writeLocalUploadFile(context.Background(), destination, reader, file, int64(len(first)))
	if err != nil || written != int64(len(first)) {
		t.Fatalf("store: %d, %v", written, err)
	}
	got, err := os.ReadFile(destination)
	if err != nil || !bytes.Equal(got, first) {
		t.Fatal("shared multipart upload included another file or lost its MIME prefix")
	}
}

func TestLocalUploadNeverOverwritesExistingFile(t *testing.T) {
	contents := bytes.Repeat([]byte("upload"), 1000)
	_, file := multipartStorageFixture(t, contents)
	destination := filepath.Join(t.TempDir(), "existing.bin")
	original := []byte("keep me")
	if err := os.WriteFile(destination, original, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := writeLocalUploadFile(
		context.Background(),
		destination,
		file,
		file,
		int64(len(contents)),
	); !errors.Is(
		err,
		os.ErrExist,
	) {
		t.Fatalf("expected exclusive creation failure, got %v", err)
	}
	got, err := os.ReadFile(destination)
	if err != nil || !bytes.Equal(got, original) {
		t.Fatal("upload overwrote the existing file")
	}
}

func TestLocalUploadSizeMismatchFallsBackToBufferedCopy(t *testing.T) {
	contents := bytes.Repeat([]byte("upload"), 1000)
	_, file := multipartStorageFixture(t, contents)
	// A bounded section must not publish the whole underlying file.
	destination := filepath.Join(t.TempDir(), "stored.bin")
	written, err := writeLocalUploadFile(context.Background(), destination, io.LimitReader(file, 100), file, 100)
	if err != nil || written != 100 {
		t.Fatalf("store: %d, %v", written, err)
	}
	got, err := os.ReadFile(destination)
	if err != nil || !bytes.Equal(got, contents[:100]) {
		t.Fatal("size mismatch bypassed the bounded reader")
	}
}

type cancelAfterRead struct {
	io.Reader
	cancel context.CancelFunc
}

func (reader cancelAfterRead) Read(buffer []byte) (int, error) {
	n, err := reader.Reader.Read(buffer)
	reader.cancel()

	return n, err
}

func TestCancelledLocalUploadRemovesPartialFile(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	body := cancelAfterRead{Reader: bytes.NewReader([]byte("partial upload")), cancel: cancel}
	destination := filepath.Join(t.TempDir(), "cancelled.bin")
	if _, err := writeLocalUploadFile(ctx, destination, body, nil, 14); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
	if _, err := os.Stat(destination); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("partial file remains after cancellation: %v", err)
	}
}
