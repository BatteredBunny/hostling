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
