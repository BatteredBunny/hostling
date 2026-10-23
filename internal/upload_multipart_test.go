package internal

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func uploadMultipartBody(t *testing.T, contents ...string) (*bytes.Buffer, string) {
	t.Helper()
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	for _, content := range contents {
		part, err := writer.CreateFormFile("file", "upload.txt")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(part, content); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	return body, writer.FormDataContentType()
}

func TestUploadMultipartMemoryBudget(t *testing.T) {
	for _, tc := range []struct {
		name     string
		contents []string
		disk     bool
	}{
		{"small file spools directly", []string{"small file"}, true},
		{"larger file spools directly", []string{strings.Repeat("x", 128<<10)}, true},
		{"empty file remains valid", []string{""}, false},
		{"shared temporary file preserves first file", []string{"first", "second"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()
			t.Setenv("TMPDIR", tempDir)
			body, contentType := uploadMultipartBody(t, tc.contents...)
			request := httptest.NewRequest(http.MethodPost, "/", body)
			request.Header.Set("Content-Type", contentType)
			router := gin.New()
			router.Use((&Application{}).multipartFormMiddleware(0))
			router.POST("/", func(c *gin.Context) {
				file, header, err := c.Request.FormFile("file")
				if err != nil {
					t.Fatal(err)
				}
				defer file.Close()
				_, disk := file.(*os.File)
				if disk != tc.disk {
					t.Errorf("disk-backed=%v, want %v", disk, tc.disk)
				}
				got, err := io.ReadAll(file)
				if err != nil || string(got) != tc.contents[0] || header.Size != int64(len(tc.contents[0])) {
					t.Fatalf("wrong file: %q %v", got, err)
				}
				c.Status(http.StatusOK)
			})
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != http.StatusOK {
				t.Fatalf("status=%d: %s", response.Code, response.Body.String())
			}
			// Direct ServeHTTP bypasses net/http's request cleanup; exercise the same
			// standard-library cleanup explicitly and verify all spool files disappear.
			if err := request.MultipartForm.RemoveAll(); err != nil {
				t.Fatal(err)
			}
			entries, err := os.ReadDir(tempDir)
			if err != nil || len(entries) != 0 {
				t.Fatalf("temporary files remain: %v %v", entries, err)
			}
		})
	}
}

func TestUploadDefaultMultipartMemoryBudgetUnchanged(t *testing.T) {
	body, contentType := uploadMultipartBody(t, "small file")
	request := httptest.NewRequest(http.MethodPost, "/", body)
	request.Header.Set("Content-Type", contentType)
	router := gin.New()
	router.Use((&Application{}).apiMiddleware())
	router.POST("/", func(c *gin.Context) {
		file, _, err := c.Request.FormFile("file")
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()
		if _, disk := file.(*os.File); disk {
			t.Fatal("ordinary API parser unexpectedly spools small files")
		}
	})
	router.ServeHTTP(httptest.NewRecorder(), request)
}

func TestUploadChunkedMultipartPreservesLateFieldsAndLimit(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.registerAdmin()
	token := admin.newUploadToken("chunked uploader")
	for _, tc := range []struct {
		name    string
		content string
		status  int
	}{
		{"file then authentication and metadata", "chunked data", http.StatusOK},
		{"empty file", "", http.StatusOK},
		{"whole chunked body limit", strings.Repeat("x", 2*testMaxUploadSize), http.StatusRequestEntityTooLarge},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := new(bytes.Buffer)
			writer := multipart.NewWriter(body)
			file, err := writer.CreateFormFile("file", "chunked.txt")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := io.WriteString(file, tc.content); err != nil {
				t.Fatal(err)
			}
			for _, field := range []formField{{"upload_token", token}, {"tag", "late-tag"}, {"plain", "true"}} {
				if err := writer.WriteField(field.key, field.value); err != nil {
					t.Fatal(err)
				}
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			request, err := http.NewRequest(http.MethodPost, ts.srv.URL+"/api/file/upload", io.NopCloser(body))
			if err != nil {
				t.Fatal(err)
			}
			request.ContentLength = -1
			request.Header.Set("Content-Type", writer.FormDataContentType())
			response, err := http.DefaultClient.Do(request)
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			responseBody, err := io.ReadAll(response.Body)
			if err != nil {
				t.Fatal(err)
			}
			if response.StatusCode != tc.status {
				t.Fatalf("status=%d want=%d: %s", response.StatusCode, tc.status, responseBody)
			}
		})
	}
	listing := admin.listFiles("tag=late-tag")
	if listing.Count != 2 {
		t.Fatalf("late tags missing: %+v", listing)
	}
}

func TestUploadMultipartLimitRemovesPartialSpool(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("TMPDIR", tempDir)
	body, contentType := uploadMultipartBody(t, strings.Repeat("x", 128<<10))
	request := httptest.NewRequest(http.MethodPost, "/", io.NopCloser(body))
	request.ContentLength = -1
	request.Header.Set("Content-Type", contentType)
	app := &Application{config: Config{MaxUploadSize: 64 << 10}}
	router := gin.New()
	router.Use(app.bodySizeMiddleware(), app.multipartFormMiddleware(0))
	router.POST("/", func(c *gin.Context) { t.Fatal("oversized request reached handler") })
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status=%d: %s", response.Code, response.Body.String())
	}
	entries, err := os.ReadDir(tempDir)
	if err != nil || len(entries) != 0 {
		t.Fatalf("partial spool leaked: %v %v", entries, err)
	}
}
