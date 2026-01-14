package db

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UploadTokens struct {
	gorm.Model

	ID uint `gorm:"primaryKey"`

	LastUsed *time.Time
	Nickname string

	Token uuid.UUID `gorm:"uniqueIndex"`

	AccountID uint
	Account   Accounts `gorm:"foreignKey:AccountID"`
}

func (db *Database) DeleteUploadTokensFromAccount(accountID uint) (err error) {
	return db.Model(&UploadTokens{}).
		Where(&UploadTokens{AccountID: accountID}).
		Delete(&UploadTokens{}).Error
}

func (db *Database) GetUploadTokensCount(accountID uint) (count int64, err error) {
	err = db.Model(&UploadTokens{}).
		Where(&UploadTokens{AccountID: accountID}).
		Count(&count).Error

	return
}

type UiUploadToken struct {
	Token    uuid.UUID
	Nickname string
	LastUsed *time.Time
}

func (db *Database) GetUploadTokens(accountID uint) (uploadTokens []UiUploadToken, err error) {
	err = db.Model(&UploadTokens{}).
		Where(&UploadTokens{AccountID: accountID}).
		Select("token, nickname, last_used").
		Scan(&uploadTokens).Error

	return
}

func (db *Database) CreateUploadToken(accountID uint, nickname string) (uploadToken uuid.UUID, err error) {
	uploadToken = uuid.New()

	err = db.Model(&UploadTokens{}).
		Create(&UploadTokens{
			AccountID: accountID,
			Token:     uploadToken,
			LastUsed:  nil,
			Nickname:  nickname,
		}).Error

	return
}

func (db *Database) DeleteUploadToken(accountID uint, uploadToken uuid.UUID) (err error) {
	return db.Model(&UploadTokens{}).
		Where(&UploadTokens{
			AccountID: accountID,
			Token:     uploadToken,
		}).
		Delete(&UploadTokens{}).Error
}
