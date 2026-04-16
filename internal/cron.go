package internal

import (
	"time"

	"github.com/rs/zerolog/log"
)

func (app *Application) StartJobScheduler() {
	go func() {
		app.CleanUpJob()

		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-app.shutdownCtx.Done():
				log.Info().Msg("Job scheduler shutdown")
				return
			case <-ticker.C:
				app.CleanUpJob()
			}
		}
	}()

	log.Info().Msg("Successfully setup job scheduler")
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
