package embed

import (
	"embed"
	"io/fs"
	"net/http"
	"os"

	"github.com/BatteredBunny/hostling/internal/tags"
)

//go:embed public/*
var publicFiles embed.FS

//go:embed templates
var TemplateFiles embed.FS

func PublicFiles() http.FileSystem {
	var files fs.FS = publicFiles

	if tags.DevMode {
		files = os.DirFS(".")
	}

	sub, err := fs.Sub(files, "public")
	if err != nil {
		panic(err)
	}

	return http.FS(sub)
}
