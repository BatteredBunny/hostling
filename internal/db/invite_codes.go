package db

import (
	"crypto/rand"
	"time"

	"gorm.io/gorm"
)

type InviteCodes struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time

	Code        string
	Uses        uint // How many usages of this code is left
	ExpiryDate  time.Time
	AccountType string // Either registers normal or admin users

	InviteCreatorID uint     `gorm:"default:null;index"`
	InviteCreator   Accounts `gorm:"foreignKey:InviteCreatorID"`
}

func (db *Database) InviteCodeAmount() (count int64, err error) {
	err = db.Model(&InviteCodes{}).
		Where("expiry_date > ?", time.Now()).
		Where("uses > 0").
		Count(&count).Error

	return
}

func (db *Database) CreateInviteCode(
	uses uint,
	accountType string,
	inviteCreatorID uint,
) (inviteCode InviteCodes, err error) {
	return db.CreateInviteCodeWithCode(rand.Text(), uses, accountType, inviteCreatorID)
}

func (db *Database) CreateInviteCodeWithCode(
	code string,
	uses uint,
	accountType string,
	inviteCreatorID uint,
) (inviteCode InviteCodes, err error) {
	inviteCode = InviteCodes{
		Code:            code,
		Uses:            uses,
		AccountType:     accountType,
		InviteCreatorID: inviteCreatorID,
		ExpiryDate:      time.Now().Add(time.Hour * 24 * 7), // A week from now
	}

	err = db.Create(&inviteCode).Error

	return
}

func (db *Database) DeleteInviteCode(code string, accountID uint) (err error) {
	return db.Model(&InviteCodes{}).
		Where(&InviteCodes{
			Code:            code,
			InviteCreatorID: accountID,
		}).
		Delete(&InviteCodes{}).Error
}

func (db *Database) UseCode(code string) (accountType string, invitedBy uint, err error) {
	var inviteCode InviteCodes

	err = db.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&InviteCodes{}).
			Where("code = ? AND uses > 0 AND expiry_date > ?", code, time.Now()).
			Update("uses", gorm.Expr("uses - 1"))
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}

		return tx.Model(&InviteCodes{}).
			Where(&InviteCodes{Code: code}).
			First(&inviteCode).Error
	})
	if err != nil {
		return
	}

	accountType = inviteCode.AccountType
	invitedBy = inviteCode.InviteCreatorID

	return
}

func (db *Database) DeleteInviteCodesFromAccount(accountID uint) (err error) {
	return db.Model(&InviteCodes{}).
		Where(&InviteCodes{InviteCreatorID: accountID}).
		Delete(&InviteCodes{}).Error
}

func (db *Database) InviteCodesByAccount(accountID uint) (inviteCodes []InviteCodes, err error) {
	err = db.Model(&InviteCodes{}).
		Where("expiry_date > ?", time.Now()).
		Where("uses > 0").
		Where(&InviteCodes{InviteCreatorID: accountID}).
		Scan(&inviteCodes).Error

	return
}

func (db *Database) DeleteExpiredInviteCodes() (err error) {
	return db.Model(&InviteCodes{}).
		Where("expiry_date is not null AND expiry_date < ?", time.Now()).
		Delete(&InviteCodes{}).Error
}
