package internal

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
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

	account, loggedIn, ok := app.validateOrAbort(c)
	if !ok {
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

func (app *Application) adminPage(c *gin.Context) {
	account, ok := app.requireAuth(c)
	if !ok {
		return
	}

	if account.AccountType != "ADMIN" {
		c.Redirect(http.StatusTemporaryRedirect, "/")

		return
	}

	accounts, err := app.db.GetAccounts()
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	statsMap, err := app.db.AllAccountStats()
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	accountsByID := make(map[uint]db.Accounts, len(accounts))
	for _, a := range accounts {
		accountsByID[a.ID] = a
	}

	stats := make([]AccountStats, 0, len(accounts))
	for _, a := range accounts {
		agg := statsMap[a.ID]
		stat := AccountStats{
			Accounts:          a,
			You:               a.ID == account.ID,
			FilesUploaded:     agg.FilesUploaded,
			SpaceUsed:         agg.SpaceUsed,
			SessionsCount:     agg.SessionsCount,
			UploadTokensCount: agg.UploadTokensCount,
			LastActivity:      agg.LastActivity,
		}

		switch a.InvitedBy {
		case 0:
			stat.InvitedBy = "system"
		default:
			inviter, ok := accountsByID[a.InvitedBy]
			if ok && inviter.GithubUsername != "" {
				stat.InvitedBy = fmt.Sprintf("%s (%d)", inviter.GithubUsername, inviter.ID)
			} else {
				stat.InvitedBy = strconv.Itoa(int(a.InvitedBy))
			}
		}

		stats = append(stats, stat)
	}

	configured := app.getConfiguredProviders()
	c.HTML(http.StatusOK, "admin.gohtml", gin.H{
		"CurrentPage":           "admin",
		"Branding":              app.config.Branding,
		"Tagline":               app.config.Tagline,
		"LoggedIn":              true,
		"AccountID":             account.ID,
		"IsAdmin":               true,
		"HasAdminWarning":       app.hasAdminWarning(),
		"Accounts":              stats,
		"MaxUploadSize":         uint(app.config.MaxUploadSize),
		"Version":               Version,
		"NoProvidersConfigured": len(configured) == 0,
		"FailedProviders":       app.getFailedProviders(),
		"FileStorageMethod":     string(app.config.FileStorageMethod),
		"LoginProviders":        configured,
	})
}

func (app *Application) galleryPage(c *gin.Context) {
	account, ok := app.requireAuth(c)
	if !ok {
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
	account, ok := app.requireAuth(c)
	if !ok {
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
	account, ok := app.requireAuth(c)
	if !ok {
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
	_, loggedIn, ok := app.validateOrAbort(c)
	if !ok {
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
	_, loggedIn, ok := app.validateOrAbort(c)
	if !ok {
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
	// Probably better ways to do this
	if strings.HasPrefix(c.Request.URL.Path, "/api") {
		c.AbortWithStatus(http.StatusBadRequest)

		return
	}

	// Looks if file exists in public folder then redirects there
	cleaned := path.Clean(c.Request.URL.Path)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		c.AbortWithStatus(http.StatusBadRequest)

		return
	}
	if file, err := embed.PublicFiles().Open(cleaned); err == nil {
		info, statErr := file.Stat()
		file.Close()
		if statErr == nil && !info.IsDir() {
			c.Redirect(http.StatusFound, path.Join("/public", cleaned))

			return
		}
	}

	// Looks in database for uploaded file
	fileRecord, err := app.db.GetFileByName(path.Base(cleaned))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.Redirect(http.StatusTemporaryRedirect, "/")

		return
	} else if err != nil {
		log.Err(err).Msg("Failed to get file details")
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	if !fileRecord.Public {
		_, account, loggedIn, err := app.validateAuthCookie(c)
		if err != nil || !loggedIn || account.ID != fileRecord.UploaderID {
			c.Redirect(http.StatusTemporaryRedirect, "/")

			return
		}
	}

	fileName := fileRecord.FileName

	// Skip view bumps on Range/HEAD probes — media players issue many.
	if c.Request.Method == http.MethodGet && c.Request.Header.Get("Range") == "" {
		if err := app.db.BumpFileViews(fileRecord.ID, c.ClientIP(), deriveKey(app.appSecret, "view-hash")); err != nil {
			log.Err(err).Msg("Failed to bump file views")
		}
	}

	setUploadServeHeaders(c)
	c.Header(
		"Content-Disposition",
		formatContentDisposition(uploadDisposition(fileRecord.MimeType), fileRecord.OriginalFileName),
	)

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
		object, err := app.streamS3File(c.Request.Context(), fileName)
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

		c.Header("Content-Type", fileRecord.MimeType)
		http.ServeContent(c.Writer, c.Request, fileRecord.OriginalFileName, objectInfo.LastModified, object)
	} else {
		ctx, cancel := context.WithTimeout(c.Request.Context(), s3Timeout)
		defer cancel()
		reqParams := url.Values{
			"response-content-disposition": []string{
				formatContentDisposition(uploadDisposition(fileRecord.MimeType), fileRecord.OriginalFileName),
			},
			"response-content-type": []string{fileRecord.MimeType},
		}
		presignedURL, err := app.s3client.PresignedGetObject(ctx, app.config.S3.Bucket, fileName, time.Hour, reqParams)
		if err != nil {
			log.Err(err).Msg("Failed to generate presigned URL")
			c.AbortWithStatus(http.StatusInternalServerError)

			return
		}
		c.Redirect(http.StatusTemporaryRedirect, presignedURL.String())
	}
}

const (
	maxUploadTokensPerAccount = 100
	maxUploadTokenNickname    = 64
)

type InviteCodeAPIOutput struct {
	Code       string    `json:"code"`
	Uses       uint      `json:"uses"`
	ExpiryDate time.Time `json:"expiry_date"`
}

func (app *Application) uploadTokensAPI(c *gin.Context) {
	account, _ := getAccount(c)
	tokens, err := app.db.GetUploadTokens(account.ID)
	if err != nil {
		log.Err(err).Msg("Failed to list upload tokens")
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}
	if tokens == nil {
		tokens = []db.UiUploadToken{}
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, tokens)
}

func (app *Application) inviteCodesAPI(c *gin.Context) {
	account, _ := getAccount(c)
	invites, err := app.db.InviteCodesByAccount(account.ID)
	if err != nil {
		log.Err(err).Msg("Failed to list invite codes")
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}
	output := make([]InviteCodeAPIOutput, 0, len(invites))
	for _, invite := range invites {
		output = append(output, InviteCodeAPIOutput{
			Code: invite.Code, Uses: invite.Uses, ExpiryDate: invite.ExpiryDate,
		})
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, output)
}

func (app *Application) newUploadTokenAPI(c *gin.Context) {
	account, _ := getAccount(c)
	nickname := strings.TrimSpace(c.PostForm("nickname"))

	if nickname == "" {
		c.String(http.StatusBadRequest, "Token name is required")
		c.Abort()

		return
	}

	if len(nickname) > maxUploadTokenNickname {
		c.String(http.StatusBadRequest, "Token name too long")
		c.Abort()

		return
	}

	count, err := app.db.GetUploadTokensCount(account.ID)
	if err != nil {
		log.Err(err).Msg("Failed to count upload tokens")
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}
	if count >= maxUploadTokensPerAccount {
		c.String(http.StatusBadRequest, "Upload token limit reached")
		c.Abort()

		return
	}

	uploadToken, err := app.db.CreateUploadToken(account.ID, nickname)
	if err != nil {
		log.Err(err).Msg("Failed to create upload token")
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusCreated, uploadToken)
}

func (app *Application) deleteUploadTokenAPI(c *gin.Context) {
	account, _ := getAccount(c)

	id, err := strconv.ParseUint(c.PostForm("id"), 10, 32)
	if err != nil || id == 0 {
		c.AbortWithStatus(http.StatusBadRequest)

		return
	}

	if err = app.db.DeleteUploadToken(account.ID, uint(id)); err != nil {
		log.Err(err).Msg("Failed to delete upload token")
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	c.String(http.StatusOK, "Upload token deleted successfully")
}

func (app *Application) deleteInviteCodeAPI(c *gin.Context) {
	account, _ := getAccount(c)
	inviteCode := c.PostForm("invite_code")
	if inviteCode == "" {
		c.AbortWithStatus(http.StatusBadRequest)

		return
	}

	if err := app.db.DeleteInviteCode(inviteCode, account.ID); err != nil {
		log.Err(err).Msg("Failed to delete invite code")
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	c.String(http.StatusOK, "Invite code deleted successfully")
}

func (app *Application) deleteFilesAPI(c *gin.Context) {
	account, _ := getAccount(c)

	err := app.deleteFilesFromAccount(c.Request.Context(), account.ID)
	if errors.Is(err, ErrPartialDeleteFailed) {
		c.String(http.StatusInternalServerError, "Some files could not be deleted")

		return
	}
	if err != nil {
		log.Err(err).Msg("Failed to delete files from account")
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	c.String(http.StatusOK, "Files deleted")
}

func (app *Application) deleteFilesFromAccount(ctx context.Context, accountID uint) (err error) {
	files, err := app.db.GetAllFilesFromAccount(accountID)
	if err != nil {
		return
	}

	var failed int
	for _, file := range files {
		if err = app.deleteFile(ctx, file.FileName); err != nil {
			log.Err(err).Str("file", file.FileName).Msg("Failed to delete file from storage; keeping DB row for retry")
			failed++

			continue
		}
		if err = app.db.DeleteFileEntry(file.FileName, accountID); err != nil {
			log.Err(err).Str("file", file.FileName).Msg("Failed to delete file entry from database")
			failed++

			continue
		}
	}

	if failed > 0 {
		return ErrPartialDeleteFailed
	}

	return nil
}

type FileStatsOutput struct {
	Count     uint     `json:"count"`
	SizeTotal uint     `json:"size_total"`
	Tags      []string `json:"tags"`
}

func (app *Application) fileStatsAPI(c *gin.Context) {
	account, _ := getAccount(c)

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

// FileFilterInput is the tag/filter selection shared by the file listing and
// the filter tags API.
type FileFilterInput struct {
	// Repeatable; files must have every given tag.
	Tag []string `form:"tag"`
	// Repeatable: "untagged" for files without tags, "public" for public files, "private" for private files.
	// Multiple filters combine, e.g. filter=untagged&filter=private.
	Filter []string `form:"filter"`
}

// normalize validates the selection and lowercases tags, since they're stored lowercased.
func (input *FileFilterInput) normalize() bool {
	allowedFilters := []string{
		"",
		"untagged",
		"public",
		"private",
	}
	for _, f := range input.Filter {
		if !slices.Contains(allowedFilters, f) {
			return false
		}
	}
	// A file can't be both, so this can only be a client mistake.
	if slices.Contains(input.Filter, "public") && slices.Contains(input.Filter, "private") {
		return false
	}

	tags := make([]string, 0, len(input.Tag))
	for _, t := range input.Tag {
		if t = strings.ToLower(strings.TrimSpace(t)); t != "" {
			tags = append(tags, t)
		}
	}
	slices.Sort(tags)
	input.Tag = slices.Compact(tags)

	// Each tag is a subquery; no file can match more than this anyway.
	return len(input.Tag) <= db.MaxTagsPerFile
}

type FilesApiInput struct {
	FileFilterInput

	Skip uint   `form:"skip,default=0"`          // Used for pagination
	Sort string `form:"sort,default=created_at"` // "created_at", "views", "file_size"
	Desc bool   `form:"desc,default=true"`       // true for descending, false for ascending
}

type FilesApiOutput struct {
	Files []db.Files `json:"files"`
	Count int64      `json:"count"`
}

func (app *Application) filesAPI(c *gin.Context) {
	account, _ := getAccount(c)

	var input FilesApiInput
	if err := c.MustBindWith(&input, binding.Form); err != nil {
		_ = c.AbortWithError(http.StatusBadRequest, err)

		return
	}

	if input.Skip > maxPaginationSkip {
		c.AbortWithStatus(http.StatusBadRequest)

		return
	}

	if !input.normalize() {
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

	files, count, err := app.db.GetFilesPaginatedFromAccount(
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

	c.JSON(http.StatusOK, FilesApiOutput{Files: files, Count: count})
}

type FilterTagsApiOutput struct {
	Tags []string `json:"tags"`
}

// filterTagsAPI lists the tags on files matching the given tags and filters,
// so a client can offer only tags that still narrow the selection down.
func (app *Application) filterTagsAPI(c *gin.Context) {
	account, _ := getAccount(c)

	var input FileFilterInput
	if err := c.MustBindWith(&input, binding.Form); err != nil {
		_ = c.AbortWithError(http.StatusBadRequest, err)

		return
	}

	if !input.normalize() {
		c.AbortWithStatus(http.StatusBadRequest)

		return
	}

	tags, err := app.db.GetFilteredTags(account.ID, input.Tag, input.Filter)
	if err != nil {
		log.Err(err).Msg("Failed to get filter tags")
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}
	if tags == nil {
		tags = []string{}
	}

	c.JSON(http.StatusOK, FilterTagsApiOutput{Tags: tags})
}
