package db

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Files struct {
	ID        uint `gorm:"primaryKey" json:"-"`
	CreatedAt time.Time
	UpdatedAt time.Time      `json:"-"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	FileName string // Newly generated file name

	OriginalFileName string // Original file name from upload
	FileSize         uint
	MimeType         string

	Public bool // If false, only the uploader can see the file

	Views      []FileViews `gorm:"foreignKey:FilesID" json:"-"`
	ViewsCount uint        `gorm:"-"` // Used for export

	ExpiryDate time.Time `gorm:"default:null"` // Time when the file will be deleted

	UploaderID uint     `json:"-"`
	Uploader   Accounts `gorm:"foreignKey:UploaderID" json:"-"`

	Tags []Tag `gorm:"many2many:file_tags;"`
}

type FileViews struct {
	gorm.Model

	// Each IP counts once as a view
	IpHash string `gorm:"index:,unique,composite:hash_collision"`

	FilesID uint `gorm:"index:,unique,composite:hash_collision"`
}

// TODO: Doesn't work
func (f *Files) AfterDelete(db *gorm.DB) (err error) {
	err = db.Model(&FileViews{}).
		Where("files_id = ?", f.ID).
		Delete(&FileViews{}).Error

	return
}

func (db *Database) getFileViews(fileID uint) (count int64, err error) {
	err = db.Model(&FileViews{}).
		Where(&FileViews{FilesID: fileID}).
		Count(&count).Error

	return
}

// Deletes file entry from database
func (db *Database) DeleteFileEntry(fileName string, accountID uint) (err error) {
	var file Files
	if err = db.Where("file_name = ? AND uploader_id = ?", fileName, accountID).First(&file).Error; err != nil {
		return
	}

	return db.Select("Tags").Delete(&file).Error
}

type CreateFileEntryInput struct {
	Files Files

	UploadToken  uuid.NullUUID
	SessionToken uuid.NullUUID
}

var ErrNotAuthenticated = errors.New("not authenticated")

// Creates file entry in database
func (db *Database) CreateFileEntry(input CreateFileEntryInput) (err error) {
	var account Accounts
	if input.SessionToken.Valid {
		account, err = db.GetAccountBySessionToken(input.SessionToken.UUID)
		if err != nil {
			return
		}
	} else if input.UploadToken.Valid {
		account, err = db.GetAccountByUploadToken(input.UploadToken.UUID)
		if err != nil {
			return
		}
	} else {
		// This shouldnt happen but just in case
		return ErrNotAuthenticated
	}

	input.Files.UploaderID = account.ID

	return db.Model(&Files{}).Create(&input.Files).Error
}

// Only deletes database entry, actual file has to be deleted as well
func (db *Database) DeleteFilesFromAccount(accountID uint) (err error) {
	return db.Model(&Files{}).
		Where(&Files{UploaderID: accountID}).
		Delete(&Files{}).Error
}

func (db *Database) FilesAmountOnAccount(accountID uint) (count int64, err error) {
	err = db.Model(&Files{}).
		Where(&Files{UploaderID: accountID}).
		Where("(expiry_date is not null AND expiry_date > ?) OR expiry_date is null", time.Now()). // Filters expired files
		Count(&count).Error

	return
}

func (db *Database) GetAllFilesFromAccount(accountID uint) (files []Files, err error) {
	err = db.Model(&Files{}).
		Where(&Files{UploaderID: accountID}).
		Where("(expiry_date is not null AND expiry_date > ?) OR expiry_date is null", time.Now()). // Filters expired files
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
		Where("(expiry_date is not null AND expiry_date > ?) OR expiry_date is null", time.Now()). // Filters expired files
		Scan(&result).Error
	if err != nil {
		return
	}

	totalFiles = result.TotalFiles
	totalStorage = result.TotalStorage

	return
}

// Looks if file exists in database
func (db *Database) FileExists(fileName string) (bool, error) {
	var count int64
	if err := db.Model(&Files{}).
		Where(&Files{FileName: fileName}).
		Where("(expiry_date is not null AND expiry_date > ?) OR expiry_date is null", time.Now()).
		Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (db *Database) GetFileByName(fileName string) (file Files, err error) {
	err = db.Model(&Files{}).
		Where(&Files{FileName: fileName}).
		Where("(expiry_date is not null AND expiry_date > ?) OR expiry_date is null", time.Now()).
		First(&file).Error

	return
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

func (db *Database) FindExpiredFiles() (files []Files, err error) {
	err = db.Model(&Files{}).
		Where("expiry_date is not null AND expiry_date < ?", time.Now()).
		Find(&files).Error

	return
}

func (db *Database) DeleteExpiredFiles() (err error) {
	return db.Model(&Files{}).
		Where("expiry_date is not null AND expiry_date < ?", time.Now()).
		Delete(&Files{}).Error
}

func (db *Database) GetFilesPaginatedFromAccount(
	accountID, skip, limit uint,
	sort string,
	desc bool,
	tag string, // Tag to filter by
) (files []Files, err error) {
	query := db.Model(&Files{}).
		Where("files.uploader_id = ?", accountID).
		Where("files.expiry_date IS NULL OR files.expiry_date > ?", time.Now())

	// Add tag filter if provided
	if tag != "" {
		query = query.Joins("JOIN file_tags ON file_tags.files_id = files.id").
			Where("file_tags.tag_name = ?", tag)
	}

	if err = query.
		Offset(int(skip)).
		Limit(int(limit)).
		Preload("Views").
		Preload("Tags").
		Joins("LEFT JOIN file_views ON file_views.files_id = files.id").
		Select("files.*").
		Group("files.id").
		Order(clause.OrderByColumn{
			Column: clause.Column{Table: "files", Name: sort},
			Desc:   desc,
		}).Find(&files).Error; err != nil {
		return
	}

	for i, file := range files {
		files[i].ViewsCount = uint(len(file.Views))
	}

	return
}

func (db *Database) ToggleFilePublic(fileName string, accountID uint) (newPublicStatus bool, err error) {
	var file Files

	if err = db.Model(&Files{}).
		Where(&Files{FileName: fileName, UploaderID: accountID}).
		Where("(expiry_date is not null AND expiry_date > ?) OR expiry_date is null", time.Now()).
		First(&file).Error; err != nil {
		return
	}

	newPublicStatus = !file.Public

	err = db.Model(&Files{}).
		Where(&Files{FileName: fileName, UploaderID: accountID}).
		Update("public", newPublicStatus).Error

	return
}
