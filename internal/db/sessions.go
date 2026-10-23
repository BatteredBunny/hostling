package db

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

const SessionTokenDuration = 7 * 24 * time.Hour

type SessionTokens struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time

	LastUsed   time.Time
	ExpiryDate time.Time
	TokenHash  string `gorm:"uniqueIndex;not null"`

	AccountID uint     `gorm:"index"`
	Account   Accounts `gorm:"foreignKey:AccountID;constraint:OnDelete:CASCADE"`
}

func (db *Database) DeleteSession(sessionToken string) (err error) {
	hash, err := HashSessionToken(sessionToken)
	if err != nil {
		return err
	}

	_, err = gorm.G[SessionTokens](db.DB).Where("token_hash = ?", hash).Delete(db.Statement.Context)

	return
}

func (db *Database) DeleteSessionForAccount(sessionToken string, accountID uint) (err error) {
	hash, err := HashSessionToken(sessionToken)
	if err != nil {
		return err
	}

	_, err = gorm.G[SessionTokens](db.DB).
		Where("token_hash = ? AND account_id = ?", hash, accountID).
		Delete(db.Statement.Context)

	return
}

func (db *Database) DeleteSessionsFromAccount(accountID uint) (err error) {
	_, err = gorm.G[SessionTokens](db.DB).Where("account_id = ?", accountID).Delete(db.Statement.Context)

	return
}

func (db *Database) GetSessionsCount(accountID uint) (int64, error) {
	return gorm.G[SessionTokens](db.DB).
		Where("account_id = ?", accountID).
		Where("expiry_date > ?", time.Now()).
		Count(db.Statement.Context, "*")
}

// HashSessionToken canonicalizes legacy UUID cookies before hashing. Session
// secrets use a distinct namespace from upload credentials.
func HashSessionToken(token string) (string, error) {
	if strings.HasPrefix(token, "hl_session_") {
		decoded, err := base64.RawURLEncoding.Strict().DecodeString(strings.TrimPrefix(token, "hl_session_"))
		if err != nil || len(decoded) != 32 {
			return "", errors.New("invalid session token")
		}
	} else {
		parsed, err := uuid.Parse(token)
		if err != nil {
			return "", errors.New("invalid session token")
		}
		token = parsed.String()
	}
	hash := sha256.Sum256([]byte(token))

	return hex.EncodeToString(hash[:]), nil
}

func (db *Database) CreateSessionToken(accountID uint) (sessionToken string, err error) {
	log.Debug().Msgf("Creating session token for account %d", accountID)
	secret := make([]byte, 32)
	if _, err = rand.Read(secret); err != nil {
		return
	}
	sessionToken = "hl_session_" + base64.RawURLEncoding.EncodeToString(secret)
	hash, err := HashSessionToken(sessionToken)
	if err != nil {
		return
	}
	now := time.Now()
	session := SessionTokens{
		AccountID:  accountID,
		TokenHash:  hash,
		ExpiryDate: now.Add(SessionTokenDuration),
		LastUsed:   now,
	}
	err = gorm.G[SessionTokens](db.DB).Create(db.Statement.Context, &session)

	return
}

func (db *Database) DeleteExpiredSessionTokens() (err error) {
	_, err = gorm.G[SessionTokens](db.DB).Where("expiry_date < ?", time.Now()).Delete(db.Statement.Context)

	return
}
