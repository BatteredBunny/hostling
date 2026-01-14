package cmd

import (
	"errors"
	"fmt"
	"net/http"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/BatteredBunny/hostling/cmd/db"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/google/uuid"
	"github.com/markbates/goth"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

func (app *Application) indexPage(c *gin.Context) {
	templateInput := gin.H{
		"Host":        c.Request.Host,
		"CurrentPage": "home",
		"Branding":    app.config.Branding,
		"Tagline":     app.config.Tagline,
	}

	_, account, loggedIn, err := app.validateAuthCookie(c)
	if errors.Is(err, ErrInvalidAuthCookie) {
		app.clearAuthCookie(c)
	} else if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	if loggedIn {
		// For top bar
		templateInput["LoggedIn"] = true
		templateInput["AccountID"] = account.ID
		templateInput["IsAdmin"] = account.AccountType == "ADMIN"
	}

	c.HTML(http.StatusOK, "index.gohtml", templateInput)
}

type AccountStats struct {
	db.Accounts

	SpaceUsed         uint
	InvitedBy         string
	FilesUploaded     int64
	You               bool
	SessionsCount     int64
	UploadTokensCount int64
	LastActivity      time.Time // Last session or upload token usage
}

func (app *Application) toAccountStats(account *db.Accounts, requesterAccountID uint) (stats AccountStats, err error) {
	files, err := app.db.GetAllFilesFromAccount(account.ID)
	if err != nil {
		return
	}

	stats = AccountStats{
		Accounts: *account,
		You:      account.ID == requesterAccountID,
	}

	stats.SessionsCount, err = app.db.GetSessionsCount(account.ID)
	if err != nil {
		log.Err(err).Msg("Failed to get session count")
	}

	stats.UploadTokensCount, err = app.db.GetUploadTokensCount(account.ID)
	if err != nil {
		log.Err(err).Msg("Failed to get upload token count")
	}

	stats.LastActivity, err = app.db.LastAccountActivity(account.ID)
	if err != nil {
		log.Err(err).Msg("Failed to get last activity")
	}

	if account.InvitedBy == 0 {
		stats.InvitedBy = "system"
	} else if account.InvitedBy > 0 {
		invitedBy, err := app.db.GetAccountByID(account.InvitedBy)
		if err == nil && invitedBy.GithubUsername != "" {
			stats.InvitedBy = fmt.Sprintf("%s (%d)", invitedBy.GithubUsername, invitedBy.ID)
		} else {
			stats.InvitedBy = strconv.Itoa(int(account.InvitedBy))
		}
	} else {
		stats.InvitedBy = strconv.Itoa(int(account.InvitedBy))
	}

	for _, file := range files {
		stats.SpaceUsed += file.FileSize
		stats.FilesUploaded++
	}

	if stats.GithubUsername == "" {
		stats.GithubUsername = "none"
	}

	return
}

func (app *Application) adminPage(c *gin.Context) {
	_, account, loggedIn, err := app.validateAuthCookie(c)
	if errors.Is(err, ErrInvalidAuthCookie) {
		app.clearAuthCookie(c)
	} else if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	if account.AccountType != "ADMIN" {
		c.Redirect(http.StatusTemporaryRedirect, "/login")
		return
	}

	templateInput := gin.H{
		"CurrentPage": "admin",
		"Branding":    app.config.Branding,
		"Tagline":     app.config.Tagline,
	}

	if loggedIn {
		// For top bar
		templateInput["LoggedIn"] = true
		templateInput["AccountID"] = account.ID
		templateInput["IsAdmin"] = account.AccountType == "ADMIN"

		accounts, err := app.db.GetAccounts()
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		var stats []AccountStats
		for _, account := range accounts {
			stat, err := app.toAccountStats(&account, account.ID)
			if err != nil {
				c.AbortWithStatus(http.StatusInternalServerError)
				return
			}

			stats = append(stats, stat)
		}

		templateInput["Accounts"] = stats
		templateInput["MaxUploadSize"] = uint(app.config.MaxUploadSize)
		templateInput["Version"] = Version
	}

	if loggedIn {
		c.HTML(http.StatusOK, "admin.gohtml", templateInput)
	} else {
		c.Redirect(http.StatusTemporaryRedirect, "/login")
	}
}

func (app *Application) userPage(c *gin.Context) {
	_, account, loggedIn, err := app.validateAuthCookie(c)
	if errors.Is(err, ErrInvalidAuthCookie) {
		app.clearAuthCookie(c)
	} else if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	templateInput := gin.H{
		"CurrentPage": "user",
		"Branding":    app.config.Branding,
		"Tagline":     app.config.Tagline,
	}

	if loggedIn && account.GithubID > 0 {
		templateInput["LinkedWithGithub"] = true
		templateInput["GithubUsername"] = account.GithubUsername
	}

	if loggedIn {
		// For top bar
		templateInput["LoggedIn"] = true
		templateInput["AccountID"] = account.ID
		templateInput["IsAdmin"] = account.AccountType == "ADMIN"

		templateInput["UnlinkedAccount"] = account.GithubID == 0

		templateInput["InviteCodes"], err = app.db.InviteCodesByAccount(account.ID)
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		uploadTokens, err := app.db.GetUploadTokens(account.ID)
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		templateInput["UploadTokens"] = uploadTokens
	}

	if loggedIn {
		c.HTML(http.StatusOK, "user.gohtml", templateInput)
	} else {
		c.Redirect(http.StatusTemporaryRedirect, "/login")
	}
}

func (app *Application) loginPage(c *gin.Context) {
	_, _, loggedIn, err := app.validateAuthCookie(c)
	if errors.Is(err, ErrInvalidAuthCookie) {
		app.clearAuthCookie(c)
	} else if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	var providers []string
	for _, provider := range goth.GetProviders() {
		providers = append(providers, provider.Name())
	}

	if loggedIn {
		c.Redirect(http.StatusTemporaryRedirect, "/user")
	} else {
		c.HTML(http.StatusOK, "login.gohtml", gin.H{
			"Providers":   providers,
			"CurrentPage": "login",
			"Branding":    app.config.Branding,
			"Tagline":     app.config.Tagline,
		})
	}
}

func (app *Application) registerPage(c *gin.Context) {
	_, _, loggedIn, err := app.validateAuthCookie(c)
	if errors.Is(err, ErrInvalidAuthCookie) {
		app.clearAuthCookie(c)
	} else if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	if loggedIn {
		c.Redirect(http.StatusTemporaryRedirect, "/user")
	} else {
		c.HTML(http.StatusOK, "register.gohtml", gin.H{
			"CurrentPage": "register",
			"Branding":    app.config.Branding,
			"Tagline":     app.config.Tagline,
		})
	}
}

func (app *Application) indexFiles(c *gin.Context) {
	c.Status(http.StatusOK)

	// Probably better ways to do this
	if strings.HasPrefix(c.Request.URL.Path, "/api") {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	// Looks if file exists in public folder then redirects there
	filePath := filepath.Join("public", path.Clean(c.Request.URL.Path))
	if file, err := publicFiles.Open(filePath); err == nil {
		file.Close()
		c.Redirect(http.StatusPermanentRedirect, path.Join("public", path.Clean(c.Request.URL.Path)))
		return
	}

	// Looks in database for uploaded file
	fileName := path.Base(path.Clean(c.Request.URL.Path))

	fileRecord, err := app.db.GetFileByName(fileName)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.Redirect(http.StatusTemporaryRedirect, "/")
		return
	} else if err != nil {
		log.Err(err).Msg("Failed to get file details")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	if !fileRecord.Public {
		// make sure its the uploader trying to access the file
		_, account, loggedIn, err := app.validateAuthCookie(c)
		if err != nil || !loggedIn || account.ID != fileRecord.UploaderID {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
	}

	if err := app.db.BumpFileViews(fileName, c.ClientIP()); err != nil {
		log.Err(err).Msg("Failed to bump file views")
	}

	switch app.config.FileStorageMethod {
	case fileStorageS3:
		c.Redirect(http.StatusTemporaryRedirect, "https://"+app.config.S3.CdnDomain+"/file/"+app.config.S3.Bucket+path.Clean(c.Request.URL.Path))
	case fileStorageLocal:
		c.File(filepath.Join(app.config.DataFolder, path.Clean(c.Request.URL.Path)))
	default:
		log.Err(ErrUnknownStorageMethod).Msg("No storage method chosen")
		c.AbortWithStatus(http.StatusInternalServerError)
	}
}

func (app *Application) newUploadTokenApi(c *gin.Context) {
	sessionToken, exists := c.Get("sessionToken")
	if !exists {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	account, err := app.db.GetAccountBySessionToken(sessionToken.(uuid.UUID))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	} else if err != nil {
		log.Err(err).Msg("Failed to fetch account by session token")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	var uploadToken uuid.UUID

	nickname := c.PostForm("nickname")

	if uploadToken, err = app.db.CreateUploadToken(account.ID, nickname); err != nil {
		log.Err(err).Msg("Failed to create upload token")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.String(http.StatusOK, uploadToken.String())
}

func (app *Application) deleteUploadTokenAPI(c *gin.Context) {
	sessionToken, exists := c.Get("sessionToken")
	if !exists {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	account, err := app.db.GetAccountBySessionToken(sessionToken.(uuid.UUID))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	} else if err != nil {
		log.Err(err).Msg("Failed to fetch account by session token")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	rawUploadToken := c.PostForm("upload_token")
	if rawUploadToken == "" {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	uploadToken, err := uuid.Parse(rawUploadToken)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	if err = app.db.DeleteUploadToken(account.ID, uploadToken); err != nil {
		log.Err(err).Msg("Failed to delete upload token")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.String(http.StatusOK, "Upload token deleted successfully")
}

func (app *Application) deleteInviteCodeAPI(c *gin.Context) {
	sessionToken, exists := c.Get("sessionToken")
	if !exists {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	account, err := app.db.GetAccountBySessionToken(sessionToken.(uuid.UUID))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	} else if err != nil {
		log.Err(err).Msg("Failed to fetch account by session token")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	inviteCode := c.PostForm("invite_code")

	if err = app.db.DeleteInviteCode(inviteCode, account.ID); err != nil {
		log.Err(err).Msg("Failed to delete invite code")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.String(http.StatusOK, "Invite code deleted successfully")
}

func (app *Application) deleteFilesAPI(c *gin.Context) {
	sessionToken, exists := c.Get("sessionToken")
	if !exists {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	account, err := app.db.GetAccountBySessionToken(sessionToken.(uuid.UUID))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	} else if err != nil {
		log.Err(err).Msg("Failed to fetch account by session token")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	if err = app.deleteFilesFromAccount(account.ID); err != nil {
		log.Err(err).Msg("Failed to delete files from account")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.String(http.StatusOK, "Files deleted")
}

func (app *Application) deleteFilesFromAccount(accountID uint) (err error) {
	files, err := app.db.GetAllFilesFromAccount(accountID)
	if err != nil {
		return
	}

	if err = app.db.DeleteFilesFromAccount(accountID); err != nil {
		return
	}

	for _, file := range files {
		if err = app.deleteFile(file.FileName); err != nil {
			log.Err(err).Msg("Failed to delete file")
		}
	}

	return
}

type FileStatsOutput struct {
	Count     uint     `json:"count"`
	SizeTotal uint     `json:"size_total"`
	Tags      []string `json:"tags"`
}

func (app *Application) fileStatsAPI(c *gin.Context) {
	sessionToken, exists := c.Get("sessionToken")
	if !exists {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	account, err := app.db.GetAccountBySessionToken(sessionToken.(uuid.UUID))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	} else if err != nil {
		log.Err(err).Msg("Failed to fetch account by session token")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	var output FileStatsOutput

	totalFiles, totalStorage, err := app.db.GetFileStats(account.ID)
	if err != nil {
		log.Err(err).Msg("Failed to get file stats")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	output.Count = totalFiles
	output.SizeTotal = totalStorage

	output.Tags, err = app.db.GetAccountTags(account.ID)
	if err != nil {
		log.Err(err).Msg("Failed to get account tags")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, output)
}

type FilesApiInput struct {
	Skip uint   `form:"skip,default=0"`          // Used for pagination
	Sort string `form:"sort,default=created_at"` // "created_at", "views", "file_size"
	Desc bool   `form:"desc,default=true"`       // true for descending, false for ascending
}

type FilesApiOutput struct {
	Files []db.Files `json:"files"`
	Count int64      `json:"count"`
}

func (app *Application) filesAPI(c *gin.Context) {
	sessionToken, exists := c.Get("sessionToken")
	if !exists {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	account, err := app.db.GetAccountBySessionToken(sessionToken.(uuid.UUID))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	} else if err != nil {
		log.Err(err).Msg("Failed to fetch account by session token")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	var input FilesApiInput
	if err = c.MustBindWith(&input, binding.Form); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	allowedSorts := []string{
		"created_at",
		"views",
		"file_size",
	}
	if !slices.Contains(allowedSorts, input.Sort) {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	// Api returns 8 files at a time
	var limit uint = 8

	var output FilesApiOutput
	output.Files, err = app.db.GetFilesPaginatedFromAccount(account.ID, input.Skip, limit, input.Sort, input.Desc)
	if err != nil {
		log.Err(err).Msg("Failed to get files from account")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	count, err := app.db.FilesAmountOnAccount(account.ID)
	if err != nil {
		log.Err(err).Msg("Failed to get files amount on account")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	output.Count = count

	c.JSON(http.StatusOK, output)
}
