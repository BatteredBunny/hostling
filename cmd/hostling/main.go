package main

import (
	"time"

	"github.com/BatteredBunny/hostling/internal"
)

func main() {
	time.Local = time.UTC

	app := internal.InitializeApplication()
	app.Run()
}
