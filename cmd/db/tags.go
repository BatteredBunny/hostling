package db

import "time"

type Tag struct {
	Name string `gorm:"primaryKey"`
}

func (db *Database) AddTagToFile(fileName string, tagName string, accountID uint) (err error) {
	var file Files
	if err = db.Where("file_name = ? AND uploader_id = ?", fileName, accountID).First(&file).Error; err != nil {
		return
	}

	tag := Tag{Name: tagName}

	err = db.Model(&file).Association("Tags").Append(&tag)

	return
}

func (db *Database) FileHasTag(fileName string, tagName string, accountID uint) (hasTag bool, err error) {
	var file Files
	if err = db.Where("file_name = ? AND uploader_id = ?", fileName, accountID).First(&file).Error; err != nil {
		return
	}

	result := db.Model(&file).Where("name = ?", tagName).Association("Tags")
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

	err = db.Model(&file).Association("Tags").Delete(&Tag{Name: tagName})

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

func (db *Database) GetUserTags(accountID uint) (tags []string, err error) {
	err = db.Table("tags").
		Joins("JOIN file_tags ON file_tags.tag_name = tags.name").
		Joins("JOIN files ON files.id = file_tags.files_id").
		Where("files.uploader_id = ?", accountID).
		Where("(expiry_date is not null AND expiry_date > ?) OR expiry_date is null", time.Now()).
		Distinct("tags.name").
		Pluck("name", &tags).Error

	return tags, err
}
