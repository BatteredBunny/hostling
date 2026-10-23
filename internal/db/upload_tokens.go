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
)

type UploadTokens struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time

	LastUsed *time.Time
	Nickname string

	TokenHash string `gorm:"uniqueIndex;not null"`

	AccountID uint     `gorm:"index"`
	Account   Accounts `gorm:"foreignKey:AccountID;constraint:OnDelete:CASCADE"`
}

func (db *Database) DeleteUploadTokensFromAccount(accountID uint) (err error) {
	return db.Where("account_id = ?", accountID).
		Delete(&UploadTokens{}).Error
}

func (db *Database) GetUploadTokensCount(accountID uint) (count int64, err error) {
	err = db.Model(&UploadTokens{}).
		Where("account_id = ?", accountID).
		Count(&count).Error

	return
}

type UiUploadToken struct {
	ID        uint       `json:"id"`
	Nickname  string     `json:"name"`
	CreatedAt time.Time  `json:"created_at"`
	LastUsed  *time.Time `json:"last_used"`
}
type CreatedUploadToken struct {
	UiUploadToken
	Token string `json:"token"`
}

// HashUploadToken canonicalizes legacy UUID credentials before hashing.
func HashUploadToken(token string) (string, error) {
	if strings.HasPrefix(token, "hl_upload_") {
		decoded, err := base64.RawURLEncoding.Strict().DecodeString(strings.TrimPrefix(token, "hl_upload_"))
		if err != nil || len(decoded) != 32 {
			return "", errors.New("invalid upload token")
		}
	} else {
		parsed, err := uuid.Parse(token)
		if err != nil {
			return "", errors.New("invalid upload token")
		}
		token = parsed.String()
	}
	hash := sha256.Sum256([]byte(token))

	return hex.EncodeToString(hash[:]), nil
}
func (db *Database) GetUploadTokens(accountID uint) (uploadTokens []UiUploadToken, err error) {
	err = db.Model(&UploadTokens{}).Where("account_id = ?", accountID).
		Select("id, nickname, created_at, last_used").Order("id DESC").Scan(&uploadTokens).Error

	return
}
func (db *Database) CreateUploadToken(accountID uint, nickname string) (created CreatedUploadToken, err error) {
	secret := make([]byte, 32)
	if _, err = rand.Read(secret); err != nil {
		return
	}
	token := "hl_upload_" + base64.RawURLEncoding.EncodeToString(secret)
	hash, err := HashUploadToken(token)
	if err != nil {
		return
	}
	record := UploadTokens{AccountID: accountID, TokenHash: hash, Nickname: nickname}
	if err = db.Create(&record).Error; err != nil {
		return
	}
	created = CreatedUploadToken{
		UiUploadToken: UiUploadToken{ID: record.ID, Nickname: nickname, CreatedAt: record.CreatedAt},
		Token:         token,
	}

	return
}
func (db *Database) DeleteUploadToken(accountID uint, id uint) error {
	return db.Where("account_id = ? AND id = ?", accountID, id).Delete(&UploadTokens{}).Error
}
