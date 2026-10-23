package embed

import (
	"io/fs"
	"testing"
)

func TestAstroAssetsEmbedded(t *testing.T) {
	entries, err := fs.Glob(publicFiles, "public/dist/_astro/*.js")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Fatal("Astro JavaScript missing from embedded public files; build the frontend first")
	}
	for _, page := range []string{"gallery", "tokens", "settings", "admin", "login", "register"} {
		if _, err := publicFiles.ReadFile("public/dist/" + page + "/index.html"); err != nil {
			t.Fatalf("missing generated %s page: %v", page, err)
		}
	}
	if _, err := publicFiles.ReadFile("public/dist/index.html"); err != nil {
		t.Fatal(err)
	}
	if _, err := publicFiles.ReadFile("public/dist/public/favicon.ico"); err != nil {
		t.Fatal(err)
	}
}
