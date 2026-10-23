package db

import (
	"fmt"

	"gorm.io/gorm"
)

// BackfillUploadTokenHashes runs between the schema migrations that add the
// verifier column and remove plaintext secrets. It is transactional and safe to
// repeat after a completed backfill or an interrupted application upgrade.
func BackfillUploadTokenHashes(database *gorm.DB) error {
	return backfillTokenHashes(database, "upload_tokens", HashUploadToken)
}

// BackfillSessionTokenHashes preserves existing browser credentials while
// replacing their plaintext storage with one-way verifiers.
func BackfillSessionTokenHashes(database *gorm.DB) error {
	return backfillTokenHashes(database, "session_tokens", HashSessionToken)
}

func backfillTokenHashes(database *gorm.DB, table string, hashToken func(string) (string, error)) error {
	// Once the final schema migration has run, there is nothing left to backfill.
	if !database.Migrator().HasColumn(table, "token") {

		return nil
	}
	if !database.Migrator().HasColumn(table, "token_hash") {
		return fmt.Errorf("%s hash column is missing", table)
	}

	return database.Transaction(func(tx *gorm.DB) error {
		var rows []struct {
			ID        uint
			Token     string
			TokenHash *string
		}
		if err := tx.Table(table).Select("id, token, token_hash").Find(&rows).Error; err != nil {
			return fmt.Errorf("failed to read legacy %s", table)
		}
		for _, row := range rows {
			hash, err := hashToken(row.Token)
			if err != nil {
				// Do not include the secret, SQL parameters, or parser error.
				return fmt.Errorf("legacy %s record %d is invalid", table, row.ID)
			}
			if row.TokenHash != nil && *row.TokenHash != "" {
				if *row.TokenHash != hash {
					return fmt.Errorf("legacy %s record %d has an inconsistent hash", table, row.ID)
				}

				continue
			}
			if err := tx.Table(table).Where("id = ?", row.ID).Update("token_hash", hash).Error; err != nil {
				return fmt.Errorf("failed to hash legacy %s record %d", table, row.ID)
			}
		}

		return nil
	})
}
