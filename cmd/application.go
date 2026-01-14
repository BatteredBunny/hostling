package cmd

import (
	"net/http"

	"github.com/BatteredBunny/hostling/cmd/db"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/didip/tollbooth/v8/limiter"
	"github.com/gin-gonic/gin"
	"github.com/go-co-op/gocron/v2"
	"github.com/rs/zerolog/log"
)

type Application struct {
	config      Config
	db          db.Database
	s3client    *s3.S3
	RateLimiter *limiter.Limiter
	cron        gocron.Scheduler

	Router *gin.Engine
}

type fileStorageMethod string

const (
	fileStorageLocal fileStorageMethod = "LOCAL"
	fileStorageS3    fileStorageMethod = "S3"
)

type Config struct {
	DataFolder            string `toml:"data_folder"`
	MaxUploadSize         int64  `toml:"max_upload_size"`
	DatabaseType          string `toml:"database_type"`
	DatabaseConnectionUrl string `toml:"database_connection_url"`
	Port                  string `toml:"port"`

	BehindReverseProxy bool   `toml:"behind_reverse_proxy"`
	TrustedProxy       string `toml:"trusted_proxy"`
	PublicUrl          string `toml:"public_url"` // URL to use for github callback and cookies
	Branding           string `toml:"branding"`   // Branding text for toolbar (max 20 characters)
	Tagline            string `toml:"tagline"`    // Used for meta description and text on index page (max 100 characters)

	FileStorageMethod fileStorageMethod
	S3                s3Config `toml:"s3"`
}

type s3Config struct {
	AccessKeyID     string `toml:"access_key_id"`
	SecretAccessKey string `toml:"secret_access_key"`
	Bucket          string `toml:"bucket"`
	Region          string `toml:"region"`
	Endpoint        string `toml:"endpoint"`
	CdnDomain       string `toml:"cdn_domain"`
}

func (app *Application) Run() {
	log.Info().Msgf("Starting server at http://localhost:%s", app.config.Port)
	log.Fatal().Err(http.ListenAndServe(":"+app.config.Port, app.Router)).Msg("HTTP server failed")
}
