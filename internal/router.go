package internal

import (
	"context"
	"math"
	"net/http"
	"time"

	embed "github.com/BatteredBunny/hostling"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/sethvargo/go-limiter"
	"github.com/sethvargo/go-limiter/memorystore"
)

func setupRatelimiting(c Config) limiter.Store {
	rate := c.RateLimit
	if rate <= 0 {
		rate = 10
	}

	// requests/second -> interval
	tokens := max(1, uint64(math.Round(rate)))
	interval := time.Duration(float64(tokens) / rate * float64(time.Second))

	store, err := memorystore.New(&memorystore.Config{
		Tokens:        tokens,
		Interval:      interval,
		SweepInterval: 10 * time.Minute,
		SweepMinTTL:   time.Hour,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to setup rate limiter")
	}

	return store
}

func setupRouter(uninitializedApp *uninitializedApplication, c Config) (app *Application) {
	app = (*Application)(uninitializedApp)
	log.Info().Msg("Setting up router")

	app.shutdownCtx, app.shutdownCancel = context.WithCancel(context.Background())

	secret, err := loadOrCreateAppSecret(c.DataFolder)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load app secret")
	}
	app.appSecret = secret

	app.Router = gin.Default()
	app.Router.ForwardedByClientIP = c.BehindReverseProxy
	if c.BehindReverseProxy {
		if err := app.Router.SetTrustedProxies([]string{c.TrustedProxy}); err != nil {
			log.Fatal().Err(err).Msg("Failed to set trusted proxies")
		}
	} else {
		if err := app.Router.SetTrustedProxies(nil); err != nil {
			log.Fatal().Err(err).Msg("Failed to clear trusted proxies")
		}
	}

	app.Router.Use(
		app.bodySizeMiddleware(),
	)

	app.Router.StaticFS("/public/", embed.PublicFiles())
	app.Router.StaticFS("/assets/", embed.BuildFiles())

	api := app.Router.Group("/api")

	app.setupAuth(api)
	humaAPI := app.registerAPIs(app.Router)
	api.GET("/openapi.json", func(c *gin.Context) { c.JSON(http.StatusOK, humaAPI.OpenAPI()) })

	// Pages
	app.Router.GET("/login", app.loginPage)
	app.Router.GET("/register", app.registerPage)
	app.Router.POST("/logout", app.logoutHandler)
	app.Router.GET("/gallery", app.galleryPage)
	app.Router.GET("/settings", app.settingsPage)
	app.Router.GET("/tokens", app.tokensPage)
	app.Router.GET("/admin", app.adminPage)
	app.Router.GET("/", app.indexPage)

	app.Router.NoRoute(app.ratelimitMiddleware(), app.indexFiles)

	return
}
