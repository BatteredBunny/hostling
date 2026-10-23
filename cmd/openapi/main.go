package main

import (
	"fmt"
	"github.com/BatteredBunny/hostling/internal"
	"github.com/gin-gonic/gin"
	"os"
)

func main() {
	gin.SetMode(gin.ReleaseMode)
	schema, err := internal.OpenAPISchema()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if len(os.Args) == 2 {
		err = os.WriteFile(os.Args[1], append(schema, '\n'), 0644)
	} else {
		_, err = os.Stdout.Write(append(schema, '\n'))
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
