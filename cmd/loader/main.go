package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"ariga.io/atlas-provider-gorm/gormschema"

	"github.com/BatteredBunny/hostling/internal/db"
)

func main() {
	dialect := flag.String("dialect", "sqlite", "Database dialect (sqlite or postgres)")
	flag.Parse()

	stmts, err := gormschema.New(*dialect).Load(
		&db.Accounts{},
		&db.Files{},
		&db.FileViews{},
		&db.Tag{},
		&db.InviteCodes{},
		&db.SessionTokens{},
		&db.UploadTokens{},
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load gorm schema: %v\n", err)
		os.Exit(1)
	}
	io.WriteString(os.Stdout, stmts)
}
