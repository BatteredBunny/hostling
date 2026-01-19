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

	GithubID       uint
	GithubUsername string

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

func (db *Database) LinkGithub(accountID uint, username string, rawGithubID string) (err error) {
	githubID, err := strconv.ParseUint(rawGithubID, 10, 0)
	if err != nil {
		return
	}

	return db.Model(&Accounts{}).
		Where(&Accounts{ID: accountID}).
		Updates(map[string]interface{}{
			"github_username": username,
			"github_id":       uint(githubID),
		}).Error
}

// Returns the latest time a session token or an upload token was used
func (db *Database) LastAccountActivity(accountID uint) (lastActivity time.Time, err error) {
	var (
		sessionLastUsed sql.NullTime
		uploadLastUsed  sql.NullTime
	)

	if err = db.Model(&SessionTokens{}).
		Where(&SessionTokens{AccountID: accountID}).
		Select("last_used").
		Order("last_used DESC").
		First(&sessionLastUsed).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return
	}

	// TODO: somehow hide the error in logs if no upload tokens exist
	if err = db.Model(&UploadTokens{}).
		Where(&UploadTokens{AccountID: accountID}).
		Select("last_used").
		Order("last_used DESC").
		First(&uploadLastUsed).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		err = nil
	} else if err != nil {
		return
	}

	if !sessionLastUsed.Valid && !uploadLastUsed.Valid {
		return time.Time{}, nil // No activity found
	} else if uploadLastUsed.Valid && uploadLastUsed.Time.After(sessionLastUsed.Time) {
		lastActivity = uploadLastUsed.Time
	} else {
		lastActivity = sessionLastUsed.Time
	}

	return
}

func (db *Database) GetAccountBySessionToken(sessionToken uuid.UUID) (account Accounts, err error) {
	if err = db.Model(&SessionTokens{}).
		Where(&SessionTokens{Token: sessionToken}).
		Where("expiry_date > ?", time.Now()).
		Update("last_used", time.Now()).Error; err != nil {
		log.Err(err).Msg("Failed to update last used time for session token")
	}

	var accountID uint
	if err = db.Model(&SessionTokens{}).
		Where(&SessionTokens{Token: sessionToken}).
		Where("expiry_date > ?", time.Now()).
		Select("account_id").
		First(&accountID).Error; err != nil {
		return
	}

	err = db.Model(&Accounts{}).
		Where(&Accounts{ID: accountID}).
		First(&account).Error

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

// Deletes account entry only
func (db *Database) DeleteAccount(accountID uint) (err error) {
	return db.Model(&Accounts{}).
		Delete(&Accounts{}, accountID).Error
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

var ErrInvalidAccountType = errors.New("Invalid account type specified")

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
