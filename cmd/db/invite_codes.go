package db

import (
	"crypto/rand"
	"time"

	"gorm.io/gorm"
)

type InviteCodes struct {
	gorm.Model

	ID          uint `gorm:"primaryKey"`
	Code        string
	Uses        uint // How many usages of this code is left
	ExpiryDate  time.Time
	AccountType string // Either registers normal or admin users

	InviteCreatorID uint     `gorm:"default:null"`
	InviteCreator   Accounts `gorm:"foreignKey:InviteCreatorID"`
}

func (db *Database) InviteCodeAmount() (count int64, err error) {
	err = db.Model(&InviteCodes{}).
		Where("expiry_date > ?", time.Now()).
		Where("uses > 0").
		Count(&count).Error

	return
}

func (db *Database) CreateInviteCode(uses uint, accountType string, inviteCreatorID uint) (inviteCode InviteCodes, err error) {
	inviteCode = InviteCodes{
		Code:            rand.Text(),
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
	if err = db.Model(&InviteCodes{}).
		Where(&InviteCodes{Code: code}).
		Where("expiry_date > ?", time.Now()).
		Where("uses > 0").
		First(&inviteCode).Error; err != nil {
		return
	}

	if err = db.Model(&InviteCodes{}).
		Where(&InviteCodes{ID: inviteCode.ID}).
		Update("uses", gorm.Expr("uses - 1")).Error; err != nil {
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
