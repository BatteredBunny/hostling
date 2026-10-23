package internal

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
)

func (app *Application) StartJobScheduler() {
	app.backgroundWg.Go(func() {
		runJob := func() {
			defer func() {
				if r := recover(); r != nil {
					log.Error().Interface("panic", r).Msg("Cleanup job panicked; scheduler recovering")
				}
			}()
			app.CleanUpJob(app.shutdownCtx)
		}

		runJob()

		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-app.shutdownCtx.Done():
				log.Info().Msg("Job scheduler shutdown")

				return
			case <-ticker.C:
				runJob()
			}
		}
	})

	log.Info().Msg("Successfully setup job scheduler")
}

func (app *Application) CleanUpJob(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}
	database := app.db.WithContext(ctx)
	if err := database.DeleteExpiredSessionTokens(); err != nil {
		log.Err(err).Msg("Failed to delete expired session tokens")
	}
	if ctx.Err() != nil {
		return
	}

	if err := database.DeleteExpiredInviteCodes(); err != nil {
		log.Err(err).Msg("Failed to delete expired invite codes")
	}
	if ctx.Err() != nil {
		return
	}

	if count, err := database.CleanupOrphanedTags(); err != nil {
		log.Err(err).Msg("Failed to clean up orphaned tags")
	} else if count > 0 {
		log.Info().Msgf("Cleaned up %d orphaned tags", count)
	}
	if ctx.Err() != nil {
		return
	}

	for {
		if ctx.Err() != nil {
			return
		}

		files, err := database.FindExpiredFiles()
		if err != nil {
			log.Err(err).Msg("Failed to find expired files")

			return
		}
		if len(files) == 0 {
			return
		}

		log.Info().Msgf("Found %d expired files", len(files))

		var deleted int
		for _, file := range files {
			if ctx.Err() != nil {
				return
			}

			if err := app.deleteFile(ctx, file.FileName); err != nil {
				log.Err(err).Str("file", file.FileName).Msg("Failed to delete blob; retaining DB row for retry")

				continue
			}

			if err := database.DeleteFileEntry(file.FileName, file.UploaderID); err != nil {
				log.Err(err).Str("file", file.FileName).Msg("Failed to delete file entry from database")

				continue
			}
			deleted++
		}

		// If we couldn't make progress (all blob deletes failed), bail
		// rather than spinning forever in this tick.
		if deleted == 0 {
			return
		}
	}
}
