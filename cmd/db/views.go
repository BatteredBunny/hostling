package db

import (
	"crypto/sha1"
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
	FilesID uint   `gorm:"index:,unique,composite:hash_collision"`
}

func (db *Database) BumpFileViews(fileName string, ip string) (err error) {
	h := sha1.New()
	h.Write([]byte(ip))
	ipHash := hex.EncodeToString(h.Sum(nil))

	var fileID uint
	if err = db.Model(&Files{}).
		Where(&Files{FileName: fileName}).
		Select("id").
		Scan(&fileID).Error; err != nil {
		return
	}

	return db.Model(&FileViews{}).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&FileViews{
			IpHash:  ipHash,
			FilesID: fileID,
		}).Error
}

func (db *Database) getFileViews(fileID uint) (count int64, err error) {
	err = db.Model(&FileViews{}).
		Where(&FileViews{FilesID: fileID}).
		Count(&count).Error

	return
}
