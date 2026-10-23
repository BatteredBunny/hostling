package db

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// BackfillUploadTokenHashes runs between the schema migrations that add the
// verifier column and remove plaintext secrets. It is transactional and safe to
// repeat after a completed backfill or an interrupted application upgrade.
func BackfillUploadTokenHashes(database *gorm.DB) error {
	// Once the final schema migration has run, there is nothing left to backfill.
	if !database.Migrator().HasColumn("upload_tokens", "token") {

		return nil
	}
	if !database.Migrator().HasColumn("upload_tokens", "token_hash") {
		return errors.New("upload token hash column is missing")
	}

	return database.Transaction(func(tx *gorm.DB) error {
		var rows []struct {
			ID        uint
			Token     string
			TokenHash *string
		}
		if err := tx.Table("upload_tokens").Select("id, token, token_hash").Find(&rows).Error; err != nil {
			return errors.New("failed to read legacy upload tokens")
		}
		for _, row := range rows {
			hash, err := HashUploadToken(row.Token)
			if err != nil {
				// Do not include the secret, SQL parameters, or parser error.
				return fmt.Errorf("legacy upload token %d is invalid", row.ID)
			}
			if row.TokenHash != nil && *row.TokenHash != "" {
				if *row.TokenHash != hash {
					return fmt.Errorf("legacy upload token %d has an inconsistent hash", row.ID)
				}

				continue
			}
			if err := tx.Table("upload_tokens").Where("id = ?", row.ID).Update("token_hash", hash).Error; err != nil {
				return fmt.Errorf("failed to hash legacy upload token %d", row.ID)
			}
		}

		return nil
	})
}
