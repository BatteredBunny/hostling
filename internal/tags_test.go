package internal

import (
	"net/http"
	"net/url"
	"slices"
	"testing"
)

func TestDuplicateTagRejected(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.registerAdmin()
	file := admin.upload()
	admin.addTag(file, "alpha")

	status, body := admin.postForm("/api/account/file/tag", url.Values{"file_name": {file}, "tag": {"alpha"}})
	if status != http.StatusBadRequest {
		t.Errorf("expected 400 for duplicate tag, got %d: %s", status, body)
	}
}

func TestTagDedupOnUpload(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.registerAdmin()
	file := admin.upload(formField{"tag", "alpha"}, formField{"tag", "alpha"}, formField{"tag", "beta"})

	var tags []string
	for _, tag := range admin.fileRecord(file).Tags {
		tags = append(tags, tag.Name)
	}
	slices.Sort(tags)
	expectNames(t, "tags", tags, []string{"alpha", "beta"})
}

func TestAddTagIsTrimmedAndFilterable(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.registerAdmin()
	file := admin.upload()
	admin.addTag(file, "  Alpha ")

	expectNames(t, "files tagged alpha", fileNames(admin.listFiles("tag=alpha").Files), []string{file})

	status, body := admin.postForm("/api/account/file/tag", url.Values{"file_name": {file}, "tag": {"alpha"}})
	if status != http.StatusBadRequest {
		t.Errorf("expected 400 for duplicate of trimmed tag, got %d: %s", status, body)
	}
}

func TestAddTagRejectsBlank(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.registerAdmin()
	file := admin.upload()

	for _, tag := range []string{"", "   "} {
		status, body := admin.postForm("/api/account/file/tag", url.Values{"file_name": {file}, "tag": {tag}})
		if status != http.StatusBadRequest {
			t.Errorf("tag %q: expected 400, got %d: %s", tag, status, body)
		}
	}
	if tags := admin.fileRecord(file).Tags; len(tags) != 0 {
		t.Errorf("expected no tags, got %v", tags)
	}
}

func TestRemoveTagFromMissingFile(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.registerAdmin()

	status, body := admin.formRequest(
		http.MethodDelete,
		"/api/account/file/tag",
		url.Values{"file_name": {"missing.png"}, "tag": {"alpha"}},
	)
	if status != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", status, body)
	}
}
