package db

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Tag struct {
	Name string `gorm:"primaryKey"`
}

const (
	TagMaxLength   = 25
	MaxTagsPerFile = 50
)

var (
	ErrTagTooLong       = errors.New("tag too long")
	ErrTooManyTags      = errors.New("too many tags")
	ErrTagAlreadyOnFile = errors.New("file already has this tag")
)

func (db *Database) AddTagToFile(fileName string, tagName string, accountID uint) (err error) {
	if len(tagName) > TagMaxLength {
		err = ErrTagTooLong

		return
	}
	tagName = strings.ToLower(tagName)

	return db.Transaction(func(tx *gorm.DB) error {
		var file Files
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("file_name = ? AND uploader_id = ?", fileName, accountID).
			First(&file).Error; err != nil {
			return err
		}

		assoc := tx.Model(&file).Association("Tags")
		if assoc.Error != nil {
			return assoc.Error
		}

		existing := tx.Model(&file).Where("tags.name = ?", tagName).Association("Tags")
		if existing.Error != nil {
			return existing.Error
		}
		if existing.Count() > 0 {
			return ErrTagAlreadyOnFile
		}

		if assoc.Count() >= MaxTagsPerFile {
			return ErrTooManyTags
		}

		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&Tag{Name: tagName}).Error; err != nil {
			return err
		}

		return tx.Model(&file).Association("Tags").Append(&Tag{Name: tagName})
	})
}

func (db *Database) FileHasTag(fileName string, tagName string, accountID uint) (hasTag bool, err error) {
	var file Files
	if err = db.Where("file_name = ? AND uploader_id = ?", fileName, accountID).First(&file).Error; err != nil {
		return
	}

	result := db.Model(&file).Where("tags.name = ?", strings.ToLower(tagName)).Association("Tags")
	if err = result.Error; err != nil {
		return
	}

	hasTag = result.Count() > 0

	return
}

func (db *Database) RemoveTagFromFile(fileName string, tagName string, accountID uint) (err error) {
	var file Files
	if err = db.Where("file_name = ? AND uploader_id = ?", fileName, accountID).First(&file).Error; err != nil {
		return
	}

	err = db.Model(&file).Association("Tags").Delete(&Tag{Name: strings.ToLower(tagName)})

	return
}

func (db *Database) CleanupOrphanedTags() (deleted int64, err error) {
	// Use a subquery to find all tag names currently in use
	// Then we delete any tag whose name is NOT in that list
	result := db.Where("name NOT IN (SELECT tag_name FROM file_tags)").Delete(&Tag{})
	if result.Error != nil {
		return 0, result.Error
	}

	return result.RowsAffected, nil
}

func (db *Database) GetAccountTags(accountID uint) (tags []string, err error) {
	err = db.Table("tags").
		Joins("JOIN file_tags ON file_tags.tag_name = tags.name").
		Joins("JOIN files ON files.id = file_tags.files_id").
		Where("files.uploader_id = ?", accountID).
		Where("(expiry_date is not null AND expiry_date > ?) OR expiry_date is null", time.Now()).
		Distinct("tags.name").
		Pluck("name", &tags).Error

	return tags, err
}
