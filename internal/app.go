package internal

import (
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"

	"github.com/BatteredBunny/hostling/internal/db"
	"github.com/BurntSushi/toml"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/rs/zerolog/log"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var ErrUnknownStorageMethod = errors.New("unknown file storage method")

func prepareStorage(c Config) (s3client *minio.Client) {
	switch c.FileStorageMethod {
	case fileStorageS3:
		log.Info().Msg("Storing files in s3 bucket")

		var err error
		s3client, err = minio.New(c.S3.Endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(c.S3.AccessKeyID, c.S3.SecretAccessKey, ""),
			Secure: true,
			Region: c.S3.Region,
		})
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to create s3 client")
		}
	case fileStorageLocal:
		log.Info().Msgf("Storing files in %s", c.DataFolder)

		if file, _ := os.Stat(c.DataFolder); file == nil {
			log.Info().Msg("Creating data folder")

			if err := os.Mkdir(c.DataFolder, 0o770); err != nil {
				log.Fatal().Err(err).Msg("Failed to create data folder")
			}
		}
	default:
		log.Fatal().Err(ErrUnknownStorageMethod).Msg("Can't setup storage, none selected")
	}

	return
}

func initializeConfig() (c Config) {
	var configLocation string
	flag.StringVar(&configLocation, "c", "config.toml", "Location of config file")
	flag.Parse()

	rawConfig, err := os.ReadFile(configLocation)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to open config file")
	}

	if _, err = toml.Decode(string(rawConfig), &c); err != nil {
		log.Fatal().Err(err).Msg("Can't parse config file")
	}

	if envAccessKeyID := os.Getenv("S3_ACCESS_KEY_ID"); envAccessKeyID != "" {
		c.S3.AccessKeyID = envAccessKeyID
	}
	if envSecretAccessKey := os.Getenv("S3_SECRET_ACCESS_KEY"); envSecretAccessKey != "" {
		c.S3.SecretAccessKey = envSecretAccessKey
	}

	if c.S3 != (s3Config{}) {
		var missing []string
		if c.S3.Bucket == "" {
			missing = append(missing, "bucket")
		}
		if c.S3.AccessKeyID == "" {
			missing = append(missing, "access_key_id")
		}
		if c.S3.SecretAccessKey == "" {
			missing = append(missing, "secret_access_key")
		}
		if c.S3.Endpoint == "" {
			missing = append(missing, "endpoint")
		}
		if c.S3.Region == "" {
			missing = append(missing, "region")
		}
		if len(missing) > 0 {
			log.Fatal().
				Strs("missing", missing).
				Msg("S3 config is incomplete; refusing to start with a half-configured bucket")
		}

		c.FileStorageMethod = fileStorageS3
	} else {
		c.FileStorageMethod = fileStorageLocal
	}

	if c.MaxUploadSize <= 0 {
		log.Warn().Msgf("Max upload size of %d is not allowed", c.MaxUploadSize)
		c.MaxUploadSize = 100 * 1024 * 1024 // 100 MB
	}

	if c.BehindReverseProxy && c.TrustedProxy == "" {
		log.Fatal().
			Msg("behind_reverse_proxy is enabled but trusted_proxy is not set; refusing to start to avoid X-Forwarded-For spoofing")
	}

	if c.PublicUrl == "" {
		log.Warn().Msg("Warning no public_url option set in toml, github login might not work")
		if c.UnixSocket != "" {
			c.PublicUrl = "http://localhost"
		} else {
			c.PublicUrl = fmt.Sprintf("http://localhost:%d", c.Port)
		}
	}

	if parsed, err := url.Parse(c.PublicUrl); err == nil {
		c.CookieDomain = parsed.Hostname()
		c.CookieSecure = parsed.Scheme == "https"
	} else {
		log.Fatal().Err(err).Msg("Failed to parse public_url")
	}

	if c.Branding == "" {
		c.Branding = "Hostling"
	} else if len(c.Branding) > 20 {
		log.Fatal().Msgf("Branding text exceeds maximum length of 20 characters (got %d)", len(c.Branding))
	}

	if c.Tagline == "" {
		c.Tagline = "Simple file hosting service"
	} else if len(c.Tagline) > 100 {
		log.Fatal().Msgf("Tagline text exceeds maximum length of 100 characters (got %d)", len(c.Tagline))
	}

	return
}

var ErrInvalidDatabaseType = errors.New("invalid database type")

func prepareDB(c Config) (database db.Database) {
	log.Info().Msg("Setting up database")

	var gormConnection gorm.Dialector
	switch c.DatabaseType {
	case "postgresql":
		gormConnection = postgres.Open(c.DatabaseConnectionUrl)
	case "sqlite":
		gormConnection = sqlite.Open(c.DatabaseConnectionUrl)
	default:
		log.Fatal().Err(ErrInvalidDatabaseType).Msg("Invalid database chosen")
	}

	var err error
	database.DB, err = gorm.Open(gormConnection, &gorm.Config{})
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to open database connection")
	}

	if err := RunMigrations(database.DB, c.DatabaseType, c.DatabaseConnectionUrl); err != nil {
		log.Fatal().Err(err).Msg("Failed to run migrations")
	}

	// Create the first admin account if no account with ID 1 exists
	accountAmount, err := database.AccountAmount()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get account amount")
	}
	inviteCodeAmount, err := database.InviteCodeAmount()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get invite amount")
	}

	if accountAmount == 0 && inviteCodeAmount == 0 {
		var inviteCode db.InviteCodes
		if initialToken := os.Getenv("INITIAL_REGISTER_TOKEN"); initialToken != "" {
			inviteCode, err = database.CreateInviteCodeWithCode(initialToken, 1, "ADMIN", 0)
		} else {
			inviteCode, err = database.CreateInviteCode(1, "ADMIN", 0)
		}
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to create initial invite")
		}

		log.Warn().
			Msgf("No accounts found, please create your account via this registration token: %s", inviteCode.Code)
	}

	return
}
