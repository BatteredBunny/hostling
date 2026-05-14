package db

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"gorm.io/gorm/clause"
)

type FileViews struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time

	// Each IP counts once as a view
	IpHash  string `gorm:"index:,unique,composite:hash_collision"`
	FilesID uint   `gorm:"index:,unique,composite:hash_collision;index"`
}

func (db *Database) BumpFileViews(fileID uint, ip string, hmacKey []byte) (err error) {
	mac := hmac.New(sha256.New, hmacKey)
	mac.Write([]byte(ip))
	ipHash := hex.EncodeToString(mac.Sum(nil))

	return db.Model(&FileViews{}).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&FileViews{
			IpHash:  ipHash,
			FilesID: fileID,
		}).Error
}
