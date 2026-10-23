package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/BatteredBunny/hostling/internal/db"
)

func TestUploadRejectsInvalidExpiry(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.registerAdmin()

	farFuture := time.Now().Add(200 * 365 * 24 * time.Hour).Unix()

	tests := []struct {
		name  string
		field formField
	}{
		{"past timestamp", formField{"expiry_timestamp", "1"}},
		{"non-numeric timestamp", formField{"expiry_timestamp", "soon"}},
		{"timestamp too far in the future", formField{"expiry_timestamp", strconv.FormatInt(farFuture, 10)}},
		{"date with invalid month and day", formField{"expiry_date", "2099-13-99"}},
		{"date not a date", formField{"expiry_date", "not-a-date"}},
		{"date wrong format", formField{"expiry_date", "99/12/31"}},
		{"date missing day", formField{"expiry_date", "2099-12"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if status, body := admin.uploadStatus(tt.field); status != http.StatusBadRequest {
				t.Errorf("expected 400, got %d: %s", status, body)
			}
		})
	}
}

func TestUploadTagLimits(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.registerAdmin()

	tagFields := func(n int) []formField {
		fields := make([]formField, n)
		for i := range fields {
			fields[i] = formField{"tag", fmt.Sprintf("t%d", i)}
		}

		return fields
	}

	t.Run("max tags allowed", func(t *testing.T) {
		admin.upload(tagFields(db.MaxTagsPerFile)...)
	})

	t.Run("too many tags", func(t *testing.T) {
		if status, body := admin.uploadStatus(tagFields(db.MaxTagsPerFile + 1)...); status != http.StatusBadRequest {
			t.Errorf("expected 400, got %d: %s", status, body)
		}
	})

	longTag := strings.Repeat("x", db.TagMaxLength+1)

	t.Run("tag too long on upload", func(t *testing.T) {
		if status, body := admin.uploadStatus(formField{"tag", longTag}); status != http.StatusBadRequest {
			t.Errorf("expected 400, got %d: %s", status, body)
		}
	})

	t.Run("tag too long on existing file", func(t *testing.T) {
		file := admin.upload()
		status, body := admin.postForm("/api/account/file/tag", url.Values{"file_name": {file}, "tag": {longTag}})
		if status != http.StatusBadRequest {
			t.Errorf("expected 400, got %d: %s", status, body)
		}
	})
}

func TestUploadRejectsOversizedFile(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.registerAdmin()

	status, body := admin.postMultipart(
		"/api/file/upload",
		bytes.Repeat([]byte("a"), 2*testMaxUploadSize),
		formField{"plain", "true"},
	)
	if status != http.StatusRequestEntityTooLarge {
		t.Errorf("expected 413, got %d: %s", status, body)
	}
}

func TestUploadRejectsMalformedUploadToken(t *testing.T) {
	ts := newTestServer(t)
	ts.registerAdmin()

	if status, body := ts.newClient().
		uploadStatus(formField{"upload_token", "not-a-uuid"}); status != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d: %s", status, body)
	}
}

func TestUploadTokenNicknameLength(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.registerAdmin()

	status, token := admin.postForm(
		"/api/account/upload_tokens",
		url.Values{"nickname": {strings.Repeat("x", maxUploadTokenNickname)}},
	)
	var created db.UiUploadToken
	if err := json.Unmarshal([]byte(token), &created); err != nil {
		t.Fatal(err)
	}
	if status != http.StatusCreated || len(created.Token.String()) != 36 {
		t.Errorf("max length nickname: expected 201 with UUID, got %d: %q", status, token)
	}

	status, body := admin.postForm(
		"/api/account/upload_tokens",
		url.Values{"nickname": {strings.Repeat("x", maxUploadTokenNickname+1)}},
	)
	if status != http.StatusBadRequest {
		t.Errorf("too long nickname: expected 400, got %d: %s", status, body)
	}
}
