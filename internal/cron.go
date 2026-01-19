package internal

import (
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/rs/zerolog/log"
)

func (app *Application) StartJobScheudler() (err error) {
	app.cron, err = gocron.NewScheduler()
	if err != nil {
		return
	}

	if _, err = app.cron.NewJob(
		gocron.DurationJob(time.Minute*10),
		gocron.NewTask(app.CleanUpJob),
	); err != nil {
		return
	}

	log.Info().Msg("Successfully setup job scheudler")
	app.cron.Start()

	go app.CleanUpJob()

	return
}

func (app *Application) CleanUpJob() {
	log.Info().Msg("Starting clean up job")

	log.Info().Msg("Starting clean up expired sessions")
	if err := app.db.DeleteExpiredSessionTokens(); err != nil {
		log.Err(err).Msg("Failed to delete expired session tokens")
	}

	log.Info().Msg("Starting clean up of expired invite codes")
	if err := app.db.DeleteExpiredInviteCodes(); err != nil {
		log.Err(err).Msg("Failed to delete expired invite codes")
	}

	log.Info().Msg("Starting clean up of orphaned tags")
	if count, err := app.db.CleanupOrphanedTags(); err != nil {
		log.Err(err).Msg("Failed to clean up orphaned tags")
	} else {
		log.Info().Msgf("Cleaned up %d orphaned tags", count)
	}

	files, err := app.db.FindExpiredFiles()
	if err != nil {
		log.Err(err).Msg("Failed to find expired files")
		return
	}

	if len(files) == 0 {
		return
	}

	log.Info().Msgf("Found %d expired files", len(files))

	for _, file := range files {
		if err = app.deleteFile(file.FileName); err != nil {
			log.Err(err).Msg("Failed to delete file")
		}
	}

	if err = app.db.DeleteExpiredFiles(); err != nil {
		log.Err(err).Msg("Failed to delete file entries in database")
		return
	}
}
