package internal

import (
	"net/http"
	"net/url"
	"slices"
	"testing"
)

func expectNames(t *testing.T, label string, got, want []string) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Errorf("%s: expected %v, got %v", label, want, got)
	}
}

func TestStatsAndListing(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.registerAdmin()
	admin.upload()
	admin.uploadSized(1024)

	stats := admin.stats()
	if stats.Count != 2 {
		t.Errorf("expected 2 files in stats, got %d", stats.Count)
	}
	if stats.SizeTotal != uint(len(testDummyContent)+1024) {
		t.Errorf("expected total size %d, got %d", len(testDummyContent)+1024, stats.SizeTotal)
	}

	listing := admin.listFiles("")
	if listing.Count != 2 || len(listing.Files) != 2 {
		t.Errorf("expected count=2 and 2 files, got count=%d files=%d", listing.Count, len(listing.Files))
	}
}

func TestListingSorting(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.registerAdmin()
	small := admin.upload()
	large := admin.uploadSized(1024)

	tests := []struct {
		query string
		want  []string
	}{
		{"sort=file_size&desc=true", []string{large, small}},
		{"sort=file_size&desc=false", []string{small, large}},
		{"sort=created_at&desc=true", []string{large, small}},
		{"sort=created_at&desc=false", []string{small, large}},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			expectNames(t, tt.query, fileNames(admin.listFiles(tt.query).Files), tt.want)
		})
	}

	t.Run("invalid sort", func(t *testing.T) {
		if status, body := admin.get("/api/account/files?sort=nope"); status != http.StatusBadRequest {
			t.Errorf("expected 400, got %d: %s", status, body)
		}
	})

	t.Run("invalid filter", func(t *testing.T) {
		if status, body := admin.get("/api/account/files?filter=nope"); status != http.StatusBadRequest {
			t.Errorf("expected 400, got %d: %s", status, body)
		}
	})
}

func TestListingFilters(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.registerAdmin()
	small := admin.upload()
	large := admin.uploadSized(1024)

	admin.addTag(small, "alpha")
	admin.addTag(small, "beta")
	admin.addTag(large, "alpha")

	t.Run("tags in stats", func(t *testing.T) {
		tags := admin.stats().Tags
		slices.Sort(tags)
		expectNames(t, "tags", tags, []string{"alpha", "beta"})
	})

	t.Run("tag filter", func(t *testing.T) {
		if got := admin.listFiles("tag=alpha").Count; got != 2 {
			t.Errorf("expected 2 files with tag alpha, got %d", got)
		}
		beta := admin.listFiles("tag=beta")
		if beta.Count != 1 {
			t.Fatalf("expected 1 file with tag beta, got %d", beta.Count)
		}
		expectNames(t, "beta", fileNames(beta.Files), []string{small})
	})

	t.Run("untagged filter with all tagged", func(t *testing.T) {
		if got := admin.listFiles("filter=untagged").Count; got != 0 {
			t.Errorf("expected 0 untagged files, got %d", got)
		}
	})

	t.Run("public and private filter", func(t *testing.T) {
		if got := admin.listFiles("filter=public").Count; got != 2 {
			t.Errorf("expected 2 public files, got %d", got)
		}

		admin.togglePublic(small)

		private := admin.listFiles("filter=private")
		expectNames(t, "private", fileNames(private.Files), []string{small})
		public := admin.listFiles("filter=public")
		expectNames(t, "public", fileNames(public.Files), []string{large})
	})

	t.Run("tag removal", func(t *testing.T) {
		admin.removeTag(small, "beta")
		expectNames(t, "tags", admin.stats().Tags, []string{"alpha"})

		if got := admin.listFiles("filter=untagged").Count; got != 0 {
			t.Errorf("expected 0 untagged files, got %d", got)
		}

		admin.removeTag(large, "alpha")
		untagged := admin.listFiles("filter=untagged")
		expectNames(t, "untagged", fileNames(untagged.Files), []string{large})
	})
}

func TestListingPagination(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.registerAdmin()

	// The API returns 8 files per page, upload 10 so the second page has something
	uploaded := make([]string, 10)
	for i := range uploaded {
		uploaded[i] = admin.upload()
	}

	page1 := admin.listFiles("skip=0")
	if page1.Count != 10 || len(page1.Files) != 8 {
		t.Errorf("page 1: expected count=10 and 8 files, got count=%d files=%d", page1.Count, len(page1.Files))
	}
	page2 := admin.listFiles("skip=8")
	if page2.Count != 10 || len(page2.Files) != 2 {
		t.Errorf("page 2: expected count=10 and 2 files, got count=%d files=%d", page2.Count, len(page2.Files))
	}

	all := append(fileNames(page1.Files), fileNames(page2.Files)...)
	slices.Sort(all)
	slices.Sort(uploaded)
	expectNames(t, "union of pages", slices.Compact(all), uploaded)
	if len(all) != 10 {
		t.Errorf("pages overlap: got %d entries across pages, expected 10", len(all))
	}
}

func TestListingCombinedQueryParams(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.registerAdmin()
	small := admin.upload()
	large := admin.uploadSized(2048)

	admin.addTag(small, "combo")
	admin.addTag(large, "combo")
	admin.togglePublic(small) // small is private, large stays public

	t.Run("tag and filter compose", func(t *testing.T) {
		listing := admin.listFiles("tag=combo&filter=public")
		expectNames(t, "tag=combo&filter=public", fileNames(listing.Files), []string{large})
		if listing.Count != 1 {
			t.Errorf("expected count 1, got %d", listing.Count)
		}
	})

	t.Run("tag and sort compose", func(t *testing.T) {
		listing := admin.listFiles("tag=combo&sort=file_size&desc=true")
		expectNames(t, "tag=combo sorted by size desc", fileNames(listing.Files), []string{large, small})
	})
}

func TestListingTagFilterComposesWithVisibility(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.registerAdmin()
	tagged := admin.upload()
	untagged := admin.upload()
	privateTagged := admin.upload()

	admin.addTag(tagged, "combo")
	admin.addTag(privateTagged, "combo")
	admin.togglePublic(privateTagged)

	tests := []struct {
		query string
		want  []string
	}{
		{"tag=combo&filter=public", []string{tagged}},
		{"tag=combo&filter=private", []string{privateTagged}},
		{"tag=combo&filter=untagged", []string{}},
		{"filter=public", []string{tagged, untagged}},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			got := fileNames(admin.listFiles(tt.query + "&sort=created_at&desc=false").Files)
			expectNames(t, tt.query, got, tt.want)
		})
	}
}

func TestListingTagFilterIsCaseInsensitive(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.registerAdmin()
	file := admin.upload()
	admin.addTag(file, "Alpha")

	for _, tag := range []string{"alpha", "ALPHA", " Alpha "} {
		t.Run(tag, func(t *testing.T) {
			expectNames(t, "tag="+tag, fileNames(admin.listFiles("tag="+url.QueryEscape(tag)).Files), []string{file})
		})
	}
}
