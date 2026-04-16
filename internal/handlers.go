package internal

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	embed "github.com/BatteredBunny/hostling"
	"github.com/BatteredBunny/hostling/internal/db"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

func (app *Application) hasAdminWarning() bool {
	app.providersMutex.RLock()
	defer app.providersMutex.RUnlock()
	return len(app.configuredProviders) == 0 || len(app.failedProviders) > 0
}

func (app *Application) getFailedProviders() []string {
	app.providersMutex.RLock()
	defer app.providersMutex.RUnlock()
	return append([]string(nil), app.failedProviders...)
}

func (app *Application) getConfiguredProviders() []string {
	app.providersMutex.RLock()
	defer app.providersMutex.RUnlock()
	return append([]string(nil), app.configuredProviders...)
}

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

		isAdmin := account.AccountType == "ADMIN"
		templateInput["IsAdmin"] = isAdmin

		if isAdmin {
			templateInput["HasAdminWarning"] = app.hasAdminWarning()
		}
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
		templateInput["HasAdminWarning"] = app.hasAdminWarning()

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

		configured := app.getConfiguredProviders()
		templateInput["Accounts"] = stats
		templateInput["MaxUploadSize"] = uint(app.config.MaxUploadSize)
		templateInput["Version"] = Version
		templateInput["NoProvidersConfigured"] = len(configured) == 0
		templateInput["FailedProviders"] = app.getFailedProviders()
		templateInput["FileStorageMethod"] = string(app.config.FileStorageMethod)
		templateInput["LoginProviders"] = configured
	}

	if loggedIn {
		c.HTML(http.StatusOK, "admin.gohtml", templateInput)
	} else {
		c.Redirect(http.StatusTemporaryRedirect, "/login")
	}
}

func (app *Application) galleryPage(c *gin.Context) {
	_, account, loggedIn, err := app.validateAuthCookie(c)
	if errors.Is(err, ErrInvalidAuthCookie) {
		app.clearAuthCookie(c)
	} else if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	if !loggedIn {
		c.Redirect(http.StatusTemporaryRedirect, "/login")
		return
	}

	isAdmin := account.AccountType == "ADMIN"
	templateInput := gin.H{
		"CurrentPage": "gallery",
		"Branding":    app.config.Branding,
		"Tagline":     app.config.Tagline,
		"LoggedIn":    true,
		"AccountID":   account.ID,
		"IsAdmin":     isAdmin,
	}

	if isAdmin {
		templateInput["HasAdminWarning"] = app.hasAdminWarning()
	}

	c.HTML(http.StatusOK, "gallery.gohtml", templateInput)
}

func (app *Application) settingsPage(c *gin.Context) {
	_, account, loggedIn, err := app.validateAuthCookie(c)
	if errors.Is(err, ErrInvalidAuthCookie) {
		app.clearAuthCookie(c)
	} else if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	if !loggedIn {
		c.Redirect(http.StatusTemporaryRedirect, "/login")
		return
	}

	isAdmin := account.AccountType == "ADMIN"
	templateInput := gin.H{
		"CurrentPage": "settings",
		"Branding":    app.config.Branding,
		"Tagline":     app.config.Tagline,
		"LoggedIn":    true,
		"AccountID":   account.ID,
		"IsAdmin":     isAdmin,
	}

	if isAdmin {
		templateInput["HasAdminWarning"] = app.hasAdminWarning()
	}

	var providers []ProviderInfo
	unlinkedAccount := true

	configured := app.getConfiguredProviders()
	for _, providerName := range configured {
		info := ProviderInfo{
			Name:        providerName,
			Icon:        ProviderToIcon(providerName),
			LinkingText: ProviderToLinkingText(providerName),
		}

		switch providerName {
		case "github":
			if account.GithubID > 0 {
				info.IsLinked = true
				info.Username = account.GithubUsername
				info.ProfileURL = "https://github.com/" + account.GithubUsername
				unlinkedAccount = false
			}
		case "openid-connect":
			if account.OIDCID != "" {
				info.IsLinked = true
				info.Username = account.OIDCUsername
				unlinkedAccount = false
			}
		}

		providers = append(providers, info)
	}

	templateInput["Providers"] = providers
	templateInput["NoProvidersConfigured"] = len(configured) == 0
	templateInput["UnlinkedAccount"] = unlinkedAccount

	c.HTML(http.StatusOK, "settings.gohtml", templateInput)
}

func (app *Application) tokensPage(c *gin.Context) {
	_, account, loggedIn, err := app.validateAuthCookie(c)
	if errors.Is(err, ErrInvalidAuthCookie) {
		app.clearAuthCookie(c)
	} else if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	if !loggedIn {
		c.Redirect(http.StatusTemporaryRedirect, "/login")
		return
	}

	isAdmin := account.AccountType == "ADMIN"
	templateInput := gin.H{
		"CurrentPage": "tokens",
		"Branding":    app.config.Branding,
		"Tagline":     app.config.Tagline,
		"LoggedIn":    true,
		"AccountID":   account.ID,
		"IsAdmin":     isAdmin,
	}

	if isAdmin {
		templateInput["HasAdminWarning"] = app.hasAdminWarning()
	}

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

	c.HTML(http.StatusOK, "tokens.gohtml", templateInput)
}

type LoginProvider struct {
	Name string
	Icon string // lucide-icon name, should probably be replaced with simpleicons.org when supporting more login platforms
}

type ProviderInfo struct {
	Name        string
	Icon        string // lucide-icon name, should probably be replaced with simpleicons.org when supporting more login platforms
	LinkingText string // e.g., "Link with GitHub"
	IsLinked    bool   // whether the account is linked to this provider
	Username    string // Used for displaying linked username
	ProfileURL  string // Profile URL if the provider supports it, e.g for github you can open your profile
}

func ProviderToIcon(provider string) string {
	switch provider {
	case "github":
		return "github"
	case "openid-connect":
		return "key-round"
	default:
		return "key-square"
	}
}

func ProviderToLinkingText(provider string) string {
	switch provider {
	case "github":
		return "Link with GitHub"
	case "openid-connect":
		return "Link with OpenID Connect"
	default:
		return "Link with " + provider
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

	configured := app.getConfiguredProviders()
	var providers []LoginProvider
	for _, name := range configured {
		providers = append(providers, LoginProvider{
			Name: name,
			Icon: ProviderToIcon(name),
		})
	}

	if loggedIn {
		c.Redirect(http.StatusTemporaryRedirect, "/gallery")
	} else {
		c.HTML(http.StatusOK, "login.gohtml", gin.H{
			"Providers":             providers,
			"NoProvidersConfigured": len(configured) == 0,
			"CurrentPage":           "login",
			"Branding":              app.config.Branding,
			"Tagline":               app.config.Tagline,
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
		c.Redirect(http.StatusTemporaryRedirect, "/gallery")
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
	if file, err := embed.PublicFiles().Open(filePath); err == nil {
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
		app.handles3File(fileName, fileRecord, c)
	case fileStorageLocal:
		c.File(filepath.Join(app.config.DataFolder, fileName))
	default:
		log.Err(ErrUnknownStorageMethod).Msg("No storage method chosen")
		c.AbortWithStatus(http.StatusInternalServerError)
	}
}

func (app *Application) handles3File(fileName string, fileRecord db.Files, c *gin.Context) {
	if app.config.S3.ProxyFiles {
		object, err := app.streamS3File(fileName)
		if err != nil {
			log.Err(err).Str("file", fileName).Msg("Failed to retrieve file from S3")
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		defer object.Close()

		objectInfo, err := object.Stat()
		if err != nil {
			log.Err(err).Str("file", fileName).Msg("Failed to get S3 object info")
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		c.Header("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", fileRecord.OriginalFileName))
		c.Header("Content-Type", fileRecord.MimeType)
		http.ServeContent(c.Writer, c.Request, fileRecord.OriginalFileName, objectInfo.LastModified, object)
	} else {
		presignedURL, err := app.s3client.PresignedGetObject(
			context.Background(),
			app.config.S3.Bucket,
			fileName,
			time.Hour,
			nil,
		)
		if err != nil {
			log.Err(err).Msg("Failed to generate presigned URL")
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		c.Redirect(http.StatusTemporaryRedirect, presignedURL.String())
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
		_ = c.AbortWithError(http.StatusBadRequest, err)

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
	Skip   uint   `form:"skip,default=0"`          // Used for pagination
	Sort   string `form:"sort,default=created_at"` // "created_at", "views", "file_size"
	Desc   bool   `form:"desc,default=true"`       // true for descending, false for ascending
	Tag    string `form:"tag"`                     // optional tag filter
	Filter string `form:"filter"`                  // "untagged" for files without tags, "public" for public files, "private" for private files
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
		_ = c.AbortWithError(http.StatusBadRequest, err)

		return
	}

	allowedFilters := []string{
		"",
		"untagged",
		"public",
		"private",
	}
	if !slices.Contains(allowedFilters, input.Filter) {
		c.AbortWithStatus(http.StatusBadRequest)
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
	output.Files, output.Count, err = app.db.GetFilesPaginatedFromAccount(
		account.ID,
		input.Skip,
		limit,
		input.Sort,
		input.Desc,
		input.Tag,
		input.Filter,
	)
	if err != nil {
		log.Err(err).Msg("Failed to get files from account")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, output)
}
