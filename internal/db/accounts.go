package db

import (
	"database/sql"
	"errors"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

type Accounts struct {
	ID        uint `gorm:"primaryKey"` // Internal numeric account ID
	CreatedAt time.Time
	UpdatedAt time.Time

	// GithubID is 0 when unlinked, otherwise the account's GitHub user id.
	// Partial unique index prevents two accounts from linking the same GitHub identity.
	GithubID       uint `gorm:"uniqueIndex:idx_accounts_github_id,where:github_id <> 0"`
	GithubUsername string

	// OIDCID is empty when unlinked, same uniqueness guarantee as above.
	OIDCID       string `gorm:"column:oidc_id;uniqueIndex:idx_accounts_oidc_id,where:oidc_id <> ''"`
	OIDCUsername string `gorm:"column:oidc_username"`

	InvitedBy uint // Account ID of the user who invited this account

	AccountType string // Either "USER" or "ADMIN"
}

// func (db *Database) fileAdd
// Returns number of accounts in the database
func (db *Database) AccountAmount() (count int64, err error) {
	err = db.Model(&Accounts{}).
		Count(&count).Error

	return
}

func (db *Database) FindAccountByGithubID(rawID string) (account Accounts, err error) {
	id, err := strconv.ParseUint(rawID, 10, 0)
	if err != nil {
		return
	}

	if err = db.Model(&Accounts{}).
		Where(&Accounts{GithubID: uint(id)}).
		First(&account).Error; err != nil {
		return
	}

	return
}

func (db *Database) UpdateGithubUsername(accountID uint, username string) (err error) {
	return db.Model(&Accounts{}).
		Where(&Accounts{ID: accountID}).
		Update("github_username", username).Error
}

var ErrProviderAlreadyLinked = errors.New("provider identity already linked to another account")

func (db *Database) LinkGithub(accountID uint, username string, rawGithubID string) (err error) {
	githubID, err := strconv.ParseUint(rawGithubID, 10, 0)
	if err != nil {
		return
	}

	return db.Transaction(func(tx *gorm.DB) error {
		var existingID uint
		lookupErr := tx.Model(&Accounts{}).
			Where("github_id = ?", uint(githubID)).
			Select("id").
			First(&existingID).Error
		if lookupErr == nil && existingID != accountID {
			return ErrProviderAlreadyLinked
		}
		if lookupErr != nil && !errors.Is(lookupErr, gorm.ErrRecordNotFound) {
			return lookupErr
		}

		return tx.Model(&Accounts{}).
			Where(&Accounts{ID: accountID}).
			Updates(map[string]interface{}{
				"github_username": username,
				"github_id":       uint(githubID),
			}).Error
	})
}

func (db *Database) FindAccountByOIDCID(oidcID string) (account Accounts, err error) {
	if err = db.Model(&Accounts{}).
		Where(&Accounts{OIDCID: oidcID}).
		First(&account).Error; err != nil {
		return
	}

	return
}

func (db *Database) UpdateOIDCUsername(accountID uint, username string) (err error) {
	return db.Model(&Accounts{}).
		Where(&Accounts{ID: accountID}).
		Update("oidc_username", username).Error
}

func (db *Database) UnlinkGithub(accountID uint) error {
	return db.Model(&Accounts{}).
		Where(&Accounts{ID: accountID}).
		Updates(map[string]interface{}{
			"github_id":       0,
			"github_username": "",
		}).Error
}

func (db *Database) UnlinkOIDC(accountID uint) error {
	return db.Model(&Accounts{}).
		Where(&Accounts{ID: accountID}).
		Updates(map[string]interface{}{
			"oidc_id":       "",
			"oidc_username": "",
		}).Error
}

func (db *Database) LinkOIDC(accountID uint, username string, oidcID string) (err error) {
	return db.Transaction(func(tx *gorm.DB) error {
		var existingID uint
		lookupErr := tx.Model(&Accounts{}).
			Where("oidc_id = ?", oidcID).
			Select("id").
			First(&existingID).Error
		if lookupErr == nil && existingID != accountID {
			return ErrProviderAlreadyLinked
		}
		if lookupErr != nil && !errors.Is(lookupErr, gorm.ErrRecordNotFound) {
			return lookupErr
		}

		return tx.Model(&Accounts{}).
			Where(&Accounts{ID: accountID}).
			Updates(map[string]interface{}{
				"oidc_username": username,
				"oidc_id":       oidcID,
			}).Error
	})
}

// Returns the latest time a session token or an upload token was used
func (db *Database) LastAccountActivity(accountID uint) (time.Time, error) {
	var sessionLastUsed, uploadLastUsed sql.NullTime

	err := db.Model(&SessionTokens{}).
		Where(&SessionTokens{AccountID: accountID}).
		Select("last_used").
		Order("last_used DESC").
		First(&sessionLastUsed).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return time.Time{}, err
	}

	err = db.Model(&UploadTokens{}).
		Where(&UploadTokens{AccountID: accountID}).
		Select("last_used").
		Order("last_used DESC").
		First(&uploadLastUsed).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return time.Time{}, err
	}

	switch {
	case !sessionLastUsed.Valid && !uploadLastUsed.Valid:
		return time.Time{}, nil
	case !sessionLastUsed.Valid:
		return uploadLastUsed.Time, nil
	case !uploadLastUsed.Valid:
		return sessionLastUsed.Time, nil
	case uploadLastUsed.Time.After(sessionLastUsed.Time):
		return uploadLastUsed.Time, nil
	default:
		return sessionLastUsed.Time, nil
	}
}

func (db *Database) GetAccountBySessionToken(sessionToken uuid.UUID) (account Accounts, err error) {
	var accountID uint
	if err = db.Model(&SessionTokens{}).
		Where(&SessionTokens{Token: sessionToken}).
		Where("expiry_date > ?", time.Now()).
		Select("account_id").
		First(&accountID).Error; err != nil {
		return
	}

	if err = db.Model(&Accounts{}).
		Where(&Accounts{ID: accountID}).
		First(&account).Error; err != nil {
		return
	}

	if updErr := db.Model(&SessionTokens{}).
		Where(&SessionTokens{Token: sessionToken}).
		Update("last_used", time.Now()).Error; updErr != nil {
		log.Err(updErr).Msg("Failed to update last used time for session token")
	}

	return
}

func (db *Database) GetAccountByUploadToken(uploadToken uuid.UUID) (account Accounts, err error) {
	var accountID uint
	if err = db.Model(&UploadTokens{}).
		Where(&UploadTokens{Token: uploadToken}).
		Select("account_id").
		First(&accountID).Error; err != nil {
		return
	}

	if err = db.Model(&UploadTokens{}).
		Where(&UploadTokens{Token: uploadToken}).
		Update("last_used", time.Now()).Error; err != nil {
		return
	}

	err = db.Model(&Accounts{}).
		Where(&Accounts{ID: accountID}).
		First(&account).Error

	return
}

func (db *Database) DeleteAccount(accountID uint) error {
	return db.Delete(&Accounts{}, accountID).Error
}

func (db *Database) GetAccounts() (accounts []Accounts, err error) {
	err = db.Model(&Accounts{}).
		Scan(&accounts).Error

	return
}

func (db *Database) GetAccountByID(accountID uint) (account Accounts, err error) {
	err = db.Model(&Accounts{}).
		Where(&Accounts{ID: accountID}).
		First(&account).Error

	return
}

var ErrInvalidAccountType = errors.New("invalid account type specified")

func (db *Database) CreateAccount(accountType string, invitedBy uint) (account Accounts, err error) {
	if accountType == "ADMIN" || accountType == "USER" {
		account = Accounts{
			AccountType: accountType,
			InvitedBy:   invitedBy,
		}

		err = db.Model(&Accounts{}).Create(&account).Error
	} else {
		err = ErrInvalidAccountType
	}

	return
}

func (db *Database) RegisterWithInviteCode(code string) (account Accounts, sessionToken uuid.UUID, err error) {
	err = db.Transaction(func(tx *gorm.DB) error {
		txDB := &Database{DB: tx}

		accountType, invitedBy, useErr := txDB.UseCode(code)
		if useErr != nil {
			return useErr
		}

		acc, createErr := txDB.CreateAccount(accountType, invitedBy)
		if createErr != nil {
			return createErr
		}

		token, tokenErr := txDB.CreateSessionToken(acc.ID)
		if tokenErr != nil {
			return tokenErr
		}

		account = acc
		sessionToken = token
		return nil
	})
	return
}
