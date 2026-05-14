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
	if id == 0 {
		err = gorm.ErrRecordNotFound

		return
	}

	if err = db.Model(&Accounts{}).
		Where("github_id = ?", uint(id)).
		First(&account).Error; err != nil {
		return
	}

	return
}

func (db *Database) UpdateGithubUsername(accountID uint, username string) (err error) {
	return db.Model(&Accounts{}).
		Where("id = ?", accountID).
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
			Where("id = ?", accountID).
			Updates(map[string]interface{}{
				"github_username": username,
				"github_id":       uint(githubID),
			}).Error
	})
}

func (db *Database) FindAccountByOIDCID(oidcID string) (account Accounts, err error) {
	if oidcID == "" {
		err = gorm.ErrRecordNotFound

		return
	}

	if err = db.Model(&Accounts{}).
		Where("oidc_id = ?", oidcID).
		First(&account).Error; err != nil {
		return
	}

	return
}

func (db *Database) UpdateOIDCUsername(accountID uint, username string) (err error) {
	return db.Model(&Accounts{}).
		Where("id = ?", accountID).
		Update("oidc_username", username).Error
}

func (db *Database) UnlinkGithub(accountID uint) error {
	return db.Model(&Accounts{}).
		Where("id = ?", accountID).
		Updates(map[string]interface{}{
			"github_id":       0,
			"github_username": "",
		}).Error
}

func (db *Database) UnlinkOIDC(accountID uint) error {
	return db.Model(&Accounts{}).
		Where("id = ?", accountID).
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
			Where("id = ?", accountID).
			Updates(map[string]interface{}{
				"oidc_username": username,
				"oidc_id":       oidcID,
			}).Error
	})
}

type AccountStats struct {
	FilesUploaded     int64
	SpaceUsed         uint
	SessionsCount     int64
	UploadTokensCount int64
	LastActivity      time.Time
}

func parseDBTime(s sql.NullString) (time.Time, bool) {
	if !s.Valid || s.String == "" {
		return time.Time{}, false
	}
	for _, layout := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
	} {
		if t, err := time.Parse(layout, s.String); err == nil {
			return t, true
		}
	}

	return time.Time{}, false
}

func (db *Database) AllAccountStats() (stats map[uint]AccountStats, err error) {
	now := time.Now()
	stats = map[uint]AccountStats{}

	var fileRows []struct {
		UploaderID uint
		Files      int64
		Storage    uint
	}
	if err = db.Model(&Files{}).
		Select("uploader_id, COUNT(*) AS files, COALESCE(SUM(file_size), 0) AS storage").
		Where("expiry_date IS NULL OR expiry_date > ?", now).
		Group("uploader_id").
		Scan(&fileRows).Error; err != nil {
		return
	}
	for _, r := range fileRows {
		a := stats[r.UploaderID]
		a.FilesUploaded = r.Files
		a.SpaceUsed = r.Storage
		stats[r.UploaderID] = a
	}

	var sessionCounts []struct {
		AccountID uint
		Count     int64
	}
	if err = db.Model(&SessionTokens{}).
		Select("account_id, COUNT(*) AS count").
		Where("expiry_date > ?", now).
		Group("account_id").
		Scan(&sessionCounts).Error; err != nil {
		return
	}
	for _, r := range sessionCounts {
		a := stats[r.AccountID]
		a.SessionsCount = r.Count
		stats[r.AccountID] = a
	}

	var sessionLast []struct {
		AccountID uint
		LastUsed  sql.NullString
	}
	if err = db.Model(&SessionTokens{}).
		Select("account_id, MAX(last_used) AS last_used").
		Group("account_id").
		Scan(&sessionLast).Error; err != nil {
		return
	}
	for _, r := range sessionLast {
		t, ok := parseDBTime(r.LastUsed)
		if !ok {
			continue
		}
		a := stats[r.AccountID]
		if t.After(a.LastActivity) {
			a.LastActivity = t
		}
		stats[r.AccountID] = a
	}

	var uploadCounts []struct {
		AccountID uint
		Count     int64
	}
	if err = db.Model(&UploadTokens{}).
		Select("account_id, COUNT(*) AS count").
		Group("account_id").
		Scan(&uploadCounts).Error; err != nil {
		return
	}
	for _, r := range uploadCounts {
		a := stats[r.AccountID]
		a.UploadTokensCount = r.Count
		stats[r.AccountID] = a
	}

	var uploadLast []struct {
		AccountID uint
		LastUsed  sql.NullString
	}
	if err = db.Model(&UploadTokens{}).
		Select("account_id, MAX(last_used) AS last_used").
		Where("last_used IS NOT NULL").
		Group("account_id").
		Scan(&uploadLast).Error; err != nil {
		return
	}
	for _, r := range uploadLast {
		t, ok := parseDBTime(r.LastUsed)
		if !ok {
			continue
		}
		a := stats[r.AccountID]
		if t.After(a.LastActivity) {
			a.LastActivity = t
		}
		stats[r.AccountID] = a
	}

	return
}

// Returns the latest time a session token or an upload token was used
func (db *Database) LastAccountActivity(accountID uint) (time.Time, error) {
	var sessionLastUsed, uploadLastUsed sql.NullTime

	err := db.Model(&SessionTokens{}).
		Where("account_id = ?", accountID).
		Select("last_used").
		Order("last_used DESC").
		First(&sessionLastUsed).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return time.Time{}, err
	}

	err = db.Model(&UploadTokens{}).
		Where("account_id = ?", accountID).
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

// Only refresh last_used at most once per debounce window to avoid a
// write on every authenticated request.
const lastUsedDebounce = time.Minute

func (db *Database) GetAccountBySessionToken(sessionToken uuid.UUID) (account Accounts, err error) {
	now := time.Now()

	var accountID uint
	if err = db.Model(&SessionTokens{}).
		Where("token = ?", sessionToken).
		Where("expiry_date > ?", now).
		Select("account_id").
		First(&accountID).Error; err != nil {
		return
	}

	if err = db.Model(&Accounts{}).
		Where("id = ?", accountID).
		First(&account).Error; err != nil {
		return
	}

	if updErr := db.Model(&SessionTokens{}).
		Where("token = ?", sessionToken).
		Where("last_used < ?", now.Add(-lastUsedDebounce)).
		Update("last_used", now).Error; updErr != nil {
		log.Err(updErr).Msg("Failed to update last used time for session token")
	}

	return
}

func (db *Database) GetAccountByUploadToken(uploadToken uuid.UUID) (account Accounts, err error) {
	now := time.Now()

	var accountID uint
	if err = db.Model(&UploadTokens{}).
		Where("token = ?", uploadToken).
		Select("account_id").
		First(&accountID).Error; err != nil {
		return
	}

	if err = db.Model(&Accounts{}).
		Where("id = ?", accountID).
		First(&account).Error; err != nil {
		return
	}

	if updErr := db.Model(&UploadTokens{}).
		Where("token = ?", uploadToken).
		Where("last_used IS NULL OR last_used < ?", now.Add(-lastUsedDebounce)).
		Update("last_used", now).Error; updErr != nil {
		log.Err(updErr).Msg("Failed to update last used time for upload token")
	}

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
		Where("id = ?", accountID).
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
