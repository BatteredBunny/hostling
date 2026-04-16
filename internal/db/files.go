package db

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Files struct {
	ID        uint `gorm:"primaryKey" json:"-"`
	CreatedAt time.Time
	UpdatedAt time.Time `                  json:"-"`

	FileName string `gorm:"uniqueIndex"` // Newly generated file name

	OriginalFileName string // Original file name from upload
	FileSize         uint
	MimeType         string

	Public bool // If false, only the uploader can see the file

	Views      []FileViews `gorm:"foreignKey:FilesID;constraint:OnDelete:CASCADE" json:"-"`
	ViewsCount uint        `gorm:"-"` // Used for export

	ExpiryDate time.Time `gorm:"default:null"` // Time when the file will be deleted

	UploaderID uint     `json:"-" gorm:"index"`
	Uploader   Accounts `json:"-" gorm:"foreignKey:UploaderID;constraint:OnDelete:CASCADE"`

	Tags []Tag `gorm:"many2many:file_tags;constraint:OnDelete:CASCADE"`
}

// Deletes file entry from database
func (db *Database) DeleteFileEntry(fileName string, accountID uint) (deleted bool, err error) {
	result := db.Where(&Files{FileName: fileName, UploaderID: accountID}).
		Delete(&Files{})
	return result.RowsAffected > 0, result.Error
}

type CreateFileEntryInput struct {
	Files Files

	UploadToken  uuid.NullUUID
	SessionToken uuid.NullUUID
}

var ErrNotAuthenticated = errors.New("not authenticated")

// Creates file entry in database
func (db *Database) CreateFileEntry(input CreateFileEntryInput) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var accountID uint
		now := time.Now()

		switch {
		case input.SessionToken.Valid:
			if err := tx.Model(&SessionTokens{}).
				Where(&SessionTokens{Token: input.SessionToken.UUID}).
				Where("expiry_date > ?", now).
				Select("account_id").
				First(&accountID).Error; err != nil {
				return err
			}
		case input.UploadToken.Valid:
			if err := tx.Model(&UploadTokens{}).
				Where(&UploadTokens{Token: input.UploadToken.UUID}).
				Select("account_id").
				First(&accountID).Error; err != nil {
				return err
			}
		default:
			return ErrNotAuthenticated
		}

		input.Files.UploaderID = accountID
		if err := tx.Model(&Files{}).Create(&input.Files).Error; err != nil {
			return err
		}

		if input.SessionToken.Valid {
			return tx.Model(&SessionTokens{}).
				Where(&SessionTokens{Token: input.SessionToken.UUID}).
				Update("last_used", now).Error
		}
		return tx.Model(&UploadTokens{}).
			Where(&UploadTokens{Token: input.UploadToken.UUID}).
			Update("last_used", now).Error
	})
}

// Only deletes database entry, actual file has to be deleted as well
func (db *Database) DeleteFilesFromAccount(accountID uint) (err error) {
	return db.Where(&Files{UploaderID: accountID}).
		Delete(&Files{}).Error
}

func (db *Database) GetAllFilesFromAccount(accountID uint) (files []Files, err error) {
	err = db.Model(&Files{}).
		Where(&Files{UploaderID: accountID}).
		Where("(expiry_date is not null AND expiry_date > ?) OR expiry_date is null", time.Now()).
		// Filters expired files
		Find(&files).Error

	return
}

func (db *Database) GetFileStats(accountID uint) (totalFiles uint, totalStorage uint, err error) {
	var result struct {
		TotalFiles   uint
		TotalStorage uint
	}

	err = db.Model(&Files{}).
		Select("COUNT(*) AS total_files, COALESCE(SUM(file_size), 0) AS total_storage").
		Where(&Files{UploaderID: accountID}).
		Where("(expiry_date is not null AND expiry_date > ?) OR expiry_date is null", time.Now()).
		// Filters expired files
		Scan(&result).Error
	if err != nil {
		return
	}

	totalFiles = result.TotalFiles
	totalStorage = result.TotalStorage

	return
}

func (db *Database) GetFileByName(fileName string) (file Files, err error) {
	err = db.Model(&Files{}).
		Where(&Files{FileName: fileName}).
		Where("(expiry_date is not null AND expiry_date > ?) OR expiry_date is null", time.Now()).
		First(&file).Error

	return
}

func (db *Database) FindExpiredFiles() (files []Files, err error) {
	err = db.Model(&Files{}).
		Where("expiry_date is not null AND expiry_date < ?", time.Now()).
		Find(&files).Error

	return
}

func (db *Database) DeleteExpiredFiles() (err error) {
	return db.Where("expiry_date is not null AND expiry_date < ?", time.Now()).
		Delete(&Files{}).Error
}

func (db *Database) GetFilesPaginatedFromAccount(
	accountID, skip, limit uint,
	sort string,
	desc bool,
	tag string, // Tag to filter by
	filter string,
) (files []Files, totalCount int64, err error) {
	baseQuery := func() *gorm.DB {
		q := db.Model(&Files{}).
			Where("files.uploader_id = ?", accountID).
			Where("files.expiry_date IS NULL OR files.expiry_date > ?", time.Now())

		switch {
		case filter == "untagged":
			q = q.Where("files.id NOT IN (SELECT files_id FROM file_tags)")
		case filter == "public":
			q = q.Where("files.public = ?", true)
		case filter == "private":
			q = q.Where("files.public = ?", false)
		case tag != "":
			q = q.Joins("JOIN file_tags ON file_tags.files_id = files.id").
				Where("file_tags.tag_name = ?", tag)
		}
		return q
	}

	if err = baseQuery().Distinct("files.id").Count(&totalCount).Error; err != nil {
		return
	}

	orderClauses := []clause.OrderByColumn{{Desc: desc}}
	if sort == "views" {
		orderClauses[0].Column = clause.Column{Name: "COUNT(file_views.id)", Raw: true}
	} else {
		orderClauses[0].Column = clause.Column{Table: "files", Name: sort}
	}

	orderClauses = append(orderClauses, clause.OrderByColumn{
		Column: clause.Column{Table: "files", Name: "id"},
		Desc:   desc,
	})

	tx := baseQuery().
		Offset(int(skip)).
		Limit(int(limit)).
		Preload("Views").
		Preload("Tags").
		Joins("LEFT JOIN file_views ON file_views.files_id = files.id").
		Select("files.*").
		Group("files.id")
	for _, o := range orderClauses {
		tx = tx.Order(o)
	}
	if err = tx.Find(&files).Error; err != nil {
		return
	}

	for i, file := range files {
		files[i].ViewsCount = uint(len(file.Views))
	}

	return
}

func (db *Database) ToggleFilePublic(fileName string, accountID uint) (newPublicStatus bool, err error) {
	err = db.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&Files{}).
			Where(&Files{FileName: fileName, UploaderID: accountID}).
			Where("(expiry_date is not null AND expiry_date > ?) OR expiry_date is null", time.Now()).
			Update("public", gorm.Expr("NOT public"))
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}

		var file Files
		if err := tx.Model(&Files{}).
			Where(&Files{FileName: fileName, UploaderID: accountID}).
			Select("public").
			First(&file).Error; err != nil {
			return err
		}
		newPublicStatus = file.Public
		return nil
	})
	return
}
