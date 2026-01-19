package main

import (
	"github.com/BatteredBunny/hostling/internal"
)

func main() {
	app := internal.InitializeApplication()

	app.StartJobScheudler()

	app.Run()
}
