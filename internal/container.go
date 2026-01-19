//go:build wireinject
// +build wireinject

package internal

import (
	"github.com/google/wire"
)

type uninitializedApplication Application

func InitializeApplication() *Application {
	panic(wire.Build(wire.NewSet(
		initializeConfig,
		setupRatelimiting,
		prepareDB,
		prepareStorage,

		wire.Struct(
			new(uninitializedApplication),
			"config",
			"db",
			"s3client",
			"RateLimiter",
		),

		setupRouter, // Finishes the setup
	)))
}
