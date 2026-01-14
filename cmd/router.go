package cmd

import (
	"embed"
	"html/template"
	"time"

	"github.com/didip/tollbooth/v8"
	"github.com/didip/tollbooth/v8/limiter"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

func setupRatelimiting(c Config) *limiter.Limiter {
	rateLimiter := tollbooth.NewLimiter(2, &limiter.ExpirableOptions{DefaultExpirationTTL: time.Hour})

	if c.BehindReverseProxy {
		rateLimiter.SetIPLookup(limiter.IPLookup{
			Name: "X-Forwarded-For",
		})
	} else {
		rateLimiter.SetIPLookup(limiter.IPLookup{
			Name: "RemoteAddr",
		})
	}

	return rateLimiter
}

//go:embed templates
var TemplateFiles embed.FS

func setupRouter(uninitializedApp *uninitializedApplication, c Config) (app *Application) {
	app = (*Application)(uninitializedApp)
	log.Info().Msg("Setting up router")

	app.Router = gin.Default()
	app.Router.ForwardedByClientIP = c.BehindReverseProxy
	app.Router.SetTrustedProxies([]string{c.TrustedProxy})

	app.Router.SetFuncMap(template.FuncMap{
		"formatTimeDate": formatTimeDate,
		"relativeTime":   relativeTime,
		"humanizeBytes":  humanizeBytes,
		"mimeIsImage":    mimeIsImage,
		"mimeIsVideo":    mimeIsVideo,
		"mimeIsAudio":    mimeIsAudio,
	})

	app.Router.SetHTMLTemplate(template.Must(template.
		New("templates").
		Funcs(app.Router.FuncMap).
		ParseFS(TemplateFiles, "templates/*.gohtml", "templates/components/*.gohtml"),
	))

	app.Router.Use(
		app.bodySizeMiddleware(),
	)

	api := app.Router.Group("/api")
	api.Use(app.apiMiddleware())

	app.setupAuth(api)

	// Upload token should only have access to upload endpoint!
	fileAPI := api.Group("/file")
	fileAPI.Use(
		app.hasUploadOrSessionTokenMiddleware(),
	)

	fileAPI.POST("/upload", app.uploadFileAPI)
	// ---

	// Accounts for managing your user
	accountAPI := api.Group("/account")
	accountAPI.Use(
		app.verifySessionAuthentication(),
		app.isSessionAuthenticated(),
	)

	accountAPI.DELETE("/", app.accountDeleteAPI)

	accountAPI.POST("/upload_token", app.newUploadTokenApi)
	accountAPI.DELETE("/upload_token", app.deleteUploadTokenAPI)

	accountAPI.DELETE("/invite_code", app.deleteInviteCodeAPI)

	accountAPI.DELETE("/files/delete", app.deleteFilesAPI)
	accountAPI.GET("/files", app.filesAPI)
	accountAPI.GET("/files/stats", app.fileStatsAPI)

	accountAPI.DELETE("/file", app.deleteFileAPI)
	accountAPI.POST("/file/public", app.toggleFilePublicAPI)
	accountAPI.POST("/file/tag", app.addTagAPI)
	accountAPI.DELETE("/file/tag", app.deleteTagAPI)
	// ---

	// Admin apis
	adminAPI := api.Group("/admin")
	adminAPI.Use(
		app.verifySessionAuthentication(),
		app.isAdmin(),
	)

	adminAPI.DELETE("/user", app.adminDeleteUser)
	adminAPI.DELETE("/files", app.adminDeleteFiles)
	adminAPI.DELETE("/sessions", app.adminDeleteSessions)
	adminAPI.DELETE("/upload_tokens", app.adminDeleteUploadTokens)
	adminAPI.POST("/give_invite_code", app.adminGiveInviteCode)

	app.Router.StaticFS("/public/", PublicFiles())

	app.Router.GET("/login", app.loginPage)
	app.Router.GET("/register", app.registerPage)
	app.Router.GET("/logout", app.logoutHandler)
	app.Router.GET("/user", app.userPage)
	app.Router.GET("/admin", app.adminPage)
	app.Router.GET("/", app.indexPage)
	app.Router.Use(app.ratelimitMiddleware())
	app.Router.NoRoute(app.indexFiles)

	return
}
