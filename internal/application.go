package internal

import (
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/BatteredBunny/hostling/internal/db"
	"github.com/didip/tollbooth/v8/limiter"
	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"github.com/rs/zerolog/log"
)

type Application struct {
	config      Config
	db          db.Database
	s3client    *minio.Client
	RateLimiter *limiter.Limiter
	Router              *gin.Engine
	configuredProviders []string // provider names that are configured (env vars set), even if not yet initialized or configured wrong

	failedProvidersMutex sync.RWMutex
	failedProviders      []string // provider names that are configured but failed to initialize
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

	Port       int    `toml:"port"`
	UnixSocket string `toml:"unix_socket"`

	BehindReverseProxy bool   `toml:"behind_reverse_proxy"`
	TrustedProxy       string `toml:"trusted_proxy"`
	PublicUrl          string `toml:"public_url"` // URL to use for github callback and cookies, e.g http://cdn.example.com
	CookieDomain       string // hostname extracted from PublicUrl

	Branding string `toml:"branding"` // Branding text for toolbar (max 20 characters)
	Tagline  string `toml:"tagline"`  // Used for meta description and text on index page (max 100 characters)

	FileStorageMethod fileStorageMethod
	S3                s3Config `toml:"s3"`
}

type s3Config struct {
	AccessKeyID     string `toml:"access_key_id"`
	SecretAccessKey string `toml:"secret_access_key"`
	Bucket          string `toml:"bucket"`
	Region          string `toml:"region"`
	Endpoint        string `toml:"endpoint"`
	ProxyFiles      bool   `toml:"proxyfiles"`
}

func (app *Application) Run() {
	app.StartJobScheduler()

	listener, err := app.listen()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to start listener")
	}

	log.Fatal().Err(http.Serve(listener, app.Router)).Msg("HTTP server failed")
}

func (app *Application) verifySocketUsable() (err error) {
	socketDir := filepath.Dir(app.config.UnixSocket)
	if err := os.MkdirAll(socketDir, 0o750); err != nil {
		return err
	}

	if _, err := os.Stat(app.config.UnixSocket); err == nil {
		if err := os.Remove(app.config.UnixSocket); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	return
}

func (app *Application) listen() (listener net.Listener, err error) {
	if app.config.UnixSocket != "" {
		if err = app.verifySocketUsable(); err != nil {
			return
		}

		listener, err = net.Listen("unix", app.config.UnixSocket)
		if err != nil {
			return
		}

		log.Info().Msgf("Starting server on unix socket %s", app.config.UnixSocket)
	} else {
		// Maybe verify port taken first?

		listener, err = net.Listen("tcp", ":"+strconv.Itoa(app.config.Port))
		if err != nil {
			return
		}

		log.Info().Msgf("Starting server at http://localhost:%d", app.config.Port)
	}

	return
}
