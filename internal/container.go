package internal

type uninitializedApplication Application

func InitializeApplication() *Application {
	config := initializeConfig()
	app := &uninitializedApplication{
		config:      config,
		db:          prepareDB(config),
		s3client:    prepareStorage(config),
		RateLimiter: setupRatelimiting(config),
	}

	return setupRouter(app, config)
}
