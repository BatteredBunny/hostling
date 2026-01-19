package db

import (
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type SessionTokens struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time

	LastUsed   time.Time
	ExpiryDate time.Time
	Token      uuid.UUID `gorm:"uniqueIndex"`

	AccountID uint
	Account   Accounts `gorm:"foreignKey:AccountID"`
}

func (db *Database) DeleteSession(sessionToken uuid.UUID) (err error) {
	return db.Model(&SessionTokens{}).
		Where(&SessionTokens{Token: sessionToken}).
		Delete(&SessionTokens{}).Error
}

func (db *Database) DeleteSessionsFromAccount(accountID uint) (err error) {
	return db.Model(&SessionTokens{}).
		Where(&SessionTokens{AccountID: accountID}).
		Delete(&SessionTokens{}).Error
}

func (db *Database) GetSessionsCount(accountID uint) (count int64, err error) {
	err = db.Model(&SessionTokens{}).
		Where(&SessionTokens{AccountID: accountID}).
		Where("expiry_date > ?", time.Now()).
		Count(&count).Error

	return
}

func (db *Database) CreateSessionToken(accountID uint) (sessionToken uuid.UUID, err error) {
	log.Debug().Msgf("Creating session token for account %d", accountID)

	session := SessionTokens{
		AccountID:  accountID,
		Token:      uuid.New(),
		ExpiryDate: time.Now().Add(time.Hour * 24 * 7), // A week from now
		LastUsed:   time.Now(),
	}

	if err = db.Model(&SessionTokens{}).Create(&session).Error; err != nil {
		return
	}

	sessionToken = session.Token

	return
}

func (db *Database) DeleteExpiredSessionTokens() (err error) {
	return db.Model(&SessionTokens{}).
		Where("expiry_date is not null AND expiry_date < ?", time.Now()).
		Delete(&SessionTokens{}).Error
}
