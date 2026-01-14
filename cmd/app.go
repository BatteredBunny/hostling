package cmd

import (
	"embed"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"net/http"
	"os"

	"github.com/BatteredBunny/hostling/cmd/db"
	"github.com/BatteredBunny/hostling/cmd/tags"
	"github.com/BurntSushi/toml"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/rs/zerolog/log"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var ErrUnknownStorageMethod = errors.New("unknown file storage method")

//go:embed public/*
var publicFiles embed.FS

func PublicFiles() http.FileSystem {
	var files fs.FS = publicFiles

	if tags.DevMode {
		files = os.DirFS("cmd")
	}

	sub, err := fs.Sub(files, "public")
	if err != nil {
		panic(err)
	}

	return http.FS(sub)
}

func prepareStorage(c Config) (s3client *s3.S3) {
	switch c.FileStorageMethod {
	case fileStorageS3:
		log.Info().Msg("Storing files in s3 bucket")

		if s3session, err := session.NewSession(&aws.Config{
			Credentials:      credentials.NewStaticCredentials(c.S3.AccessKeyID, c.S3.SecretAccessKey, ""),
			Endpoint:         aws.String(c.S3.Endpoint),
			Region:           aws.String(c.S3.Region),
			S3ForcePathStyle: aws.Bool(true),
		}); err != nil {
			log.Fatal().Err(err).Msg("Failed to create s3 session")
		} else {
			s3client = s3.New(s3session)
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

	if c.S3 != (s3Config{}) {
		c.FileStorageMethod = fileStorageS3
	} else {
		c.FileStorageMethod = fileStorageLocal
	}

	if c.MaxUploadSize <= 0 {
		log.Warn().Msgf("Max upload size of %d is not allowed", c.MaxUploadSize)
		c.MaxUploadSize = 100 * 1024 * 1024 // 100 MB
	}

	if c.PublicUrl == "" {
		log.Warn().Msg("Warning no public_url option set in toml, github login might not work")
		c.PublicUrl = fmt.Sprintf("http://localhost:%s", c.Port)
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

var ErrInvalidDatabaseType = errors.New("Invalid database type")

func prepareDB(c Config) (database db.Database) {
	log.Info().Msg("Setting up database")

	var gormConnection gorm.Dialector
	switch c.DatabaseType {
	case "postgresql":
		gormConnection = postgres.Open(c.DatabaseConnectionUrl)
	case "sqlite":
		gormConnection = sqlite.Open(c.DatabaseConnectionUrl)
	default:
		log.Fatal().Err(ErrInvalidDatabaseType).Msg("Invalid database chosehn")
	}

	var err error
	database.DB, err = gorm.Open(gormConnection, &gorm.Config{})
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to open database connection")
	}

	if err := database.DB.AutoMigrate(
		&db.Accounts{},
		&db.Files{},
		&db.FileViews{},
		&db.InviteCodes{},
		&db.SessionTokens{},
		&db.UploadTokens{},
		&db.Tag{},
	); err != nil {
		log.Fatal().Err(err).Msg("Migration failed")
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
		inviteCode, err := database.CreateInviteCode(1, "ADMIN", 0)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to create initial invite")
		}

		log.Warn().Msgf("No accounts found, please create your account via this registration token: %s", inviteCode.Code)
	}

	return
}
