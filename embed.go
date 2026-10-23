package embed

import (
	"embed"
	"io/fs"
	"net/http"
	"os"

	"github.com/BatteredBunny/hostling/internal/tags"
)

//go:embed all:public
var publicFiles embed.FS

//go:embed migrations
var MigrationFiles embed.FS

func frontendFiles(subdir string) http.FileSystem {
	if tags.DevMode {
		return http.FS(os.DirFS("public/dist" + subdir))
	}
	sub, err := fs.Sub(publicFiles, "public/dist"+subdir)
	if err != nil {
		panic(err)
	}

	return http.FS(sub)
}

func BuildFiles() http.FileSystem  { return frontendFiles("") }
func PublicFiles() http.FileSystem { return frontendFiles("/public") }
