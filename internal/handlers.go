package internal

import (
	"context"
	"errors"
	"fmt"
	"github.com/danielgtaylor/huma/v2"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	embed "github.com/BatteredBunny/hostling"
	"github.com/BatteredBunny/hostling/internal/db"
	"github.com/gin-gonic/gin"
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

func (app *Application) indexPage(c *gin.Context) { app.staticPage(c, "index") }

type AccountStats struct {
	ID             uint      `json:"ID"`
	CreatedAt      time.Time `json:"CreatedAt"`
	GithubUsername string    `json:"GithubUsername"`
	OIDCUsername   string    `json:"OIDCUsername"`
	AccountType    string    `json:"AccountType"`

	SpaceUsed         uint      `json:"SpaceUsed"`
	InvitedBy         string    `json:"InvitedBy"`
	FilesUploaded     int64     `json:"FilesUploaded"`
	You               bool      `json:"You"`
	SessionsCount     int64     `json:"SessionsCount"`
	UploadTokensCount int64     `json:"UploadTokensCount"`
	LastActivity      time.Time `json:"LastActivity"` // Last session or upload token usage
}

func (app *Application) adminOverviewAPI(
	_ context.Context,
	request *EmptyInput,
) (*JSONOutput[AdminOverviewOutput], error) {
	c := request.Gin

	account, ok := getAccount(c)
	if !ok {
		return nil, huma.NewError(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
	}

	if account.AccountType != "ADMIN" {
		return nil, huma.NewError(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
	}

	database := app.db.WithContext(c.Request.Context())
	accounts, err := database.GetAccounts()
	if err != nil {
		return nil, huma.NewError(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}

	statsMap, err := database.AllAccountStats()
	if err != nil {
		return nil, huma.NewError(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}

	accountsByID := make(map[uint]db.Accounts, len(accounts))
	for _, a := range accounts {
		accountsByID[a.ID] = a
	}

	stats := make([]AccountStats, 0, len(accounts))
	for _, a := range accounts {
		agg := statsMap[a.ID]
		stat := AccountStats{
			ID:                a.ID,
			CreatedAt:         a.CreatedAt,
			GithubUsername:    a.GithubUsername,
			OIDCUsername:      a.OIDCUsername,
			AccountType:       a.AccountType,
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

	return jsonOutput(AdminOverviewOutput{
		Accounts:              stats,
		MaxUploadSize:         app.config.MaxUploadSize,
		Version:               Version,
		NoProvidersConfigured: len(configured) == 0,
		FailedProviders:       app.getFailedProviders(),
		FileStorageMethod:     string(app.config.FileStorageMethod),
		LoginProviders:        configured,
	}), nil
}

func (app *Application) galleryPage(c *gin.Context) {
	_, ok := app.requireAuth(c)
	if !ok {
		return
	}
	app.staticPage(c, "gallery")
}

func (app *Application) settingsAPI(_ context.Context, request *EmptyInput) (*JSONOutput[SettingsOutput], error) {
	c := request.Gin

	account, ok := getAccount(c)
	if !ok {
		return nil, huma.NewError(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
	}

	data := app.settingsData(account)

	return jsonOutput(data), nil
}

func (app *Application) settingsData(account db.Accounts) SettingsOutput {
	providers := make([]ProviderInfo, 0)
	unlinkedAccount := true

	configured := app.getConfiguredProviders()
	for _, providerName := range configured {
		info := ProviderInfo{Name: providerName}

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

	return SettingsOutput{
		Providers:             providers,
		NoProvidersConfigured: len(configured) == 0,
		UnlinkedAccount:       unlinkedAccount,
	}
}

func (app *Application) tokensPage(c *gin.Context) {
	_, ok := app.requireAuth(c)
	if !ok {
		return
	}
	app.staticPage(c, "tokens")
}

func (app *Application) loginPage(c *gin.Context) {
	_, loggedIn, ok := app.validateOrAbort(c)
	if !ok {
		return
	}
	if loggedIn {
		c.Redirect(http.StatusTemporaryRedirect, "/gallery")

		return
	}
	app.staticPage(c, "login")
}

func (app *Application) registerPage(c *gin.Context) {
	_, loggedIn, ok := app.validateOrAbort(c)
	if !ok {
		return
	}
	if loggedIn {
		c.Redirect(http.StatusTemporaryRedirect, "/gallery")

		return
	}
	app.staticPage(c, "register")
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
	database := app.db.WithContext(c.Request.Context())
	fileRecord, err := database.GetFileByName(path.Base(cleaned))
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
		if err := database.BumpFileViews(
			fileRecord.ID,
			c.ClientIP(),
			deriveKey(app.appSecret, "view-hash"),
		); err != nil {
			log.Err(err).Msg("Failed to bump file views")
		}
	}

	setUploadServeHeaders(c)
	c.Header(
		"Content-Disposition",
		formatContentDisposition(
			uploadDisposition(fileRecord.MimeType),
			fileRecord.OriginalFileName,
		),
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
		http.ServeContent(
			c.Writer,
			c.Request,
			fileRecord.OriginalFileName,
			objectInfo.LastModified,
			object,
		)
	} else {
		ctx, cancel := context.WithTimeout(c.Request.Context(), s3Timeout)
		defer cancel()
		reqParams := url.Values{
			"response-content-disposition": []string{
				formatContentDisposition(
					uploadDisposition(fileRecord.MimeType),
					fileRecord.OriginalFileName,
				),
			},
			"response-content-type": []string{fileRecord.MimeType},
		}
		presignedURL, err := app.s3client.PresignedGetObject(
			ctx,
			app.config.S3.Bucket,
			fileName,
			time.Hour,
			reqParams,
		)
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

func (app *Application) uploadTokensAPI(
	_ context.Context,
	request *EmptyInput,
) (*JSONOutput[[]db.UiUploadToken], error) {
	c := request.Gin

	account, _ := getAccount(c)
	tokens, err := app.db.WithContext(c.Request.Context()).GetUploadTokens(account.ID)
	if err != nil {
		log.Err(err).Msg("Failed to list upload tokens")

		return nil, huma.NewError(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}
	if tokens == nil {
		tokens = []db.UiUploadToken{}
	}

	return jsonOutput(tokens), nil
}

func (app *Application) inviteCodesAPI(
	_ context.Context,
	request *EmptyInput,
) (*JSONOutput[[]InviteCodeAPIOutput], error) {
	c := request.Gin

	account, _ := getAccount(c)
	invites, err := app.db.WithContext(c.Request.Context()).InviteCodesByAccount(account.ID)
	if err != nil {
		log.Err(err).Msg("Failed to list invite codes")

		return nil, huma.NewError(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}
	output := make([]InviteCodeAPIOutput, 0, len(invites))
	for _, invite := range invites {
		output = append(output, InviteCodeAPIOutput{
			Code: invite.Code, Uses: invite.Uses, ExpiryDate: invite.ExpiryDate,
		})
	}

	return jsonOutput(output), nil
}

func (app *Application) newUploadTokenAPI(
	_ context.Context,
	request *FormInput[NewUploadTokenForm],
) (*JSONOutput[db.CreatedUploadToken], error) {
	c := request.Gin
	input := request.RawBody.Data()

	account, _ := getAccount(c)
	nickname := strings.TrimSpace(string(input.Nickname))

	if nickname == "" {
		return nil, huma.NewError(http.StatusBadRequest, "Token name is required")
	}

	if utf8.RuneCountInString(nickname) > maxUploadTokenNickname {
		return nil, huma.NewError(http.StatusBadRequest, "Token name too long")
	}

	uploadToken, err := app.db.WithContext(c.Request.Context()).
		CreateUploadTokenWithinLimit(account.ID, nickname, maxUploadTokensPerAccount)
	if errors.Is(err, db.ErrUploadTokenLimit) {
		return nil, huma.NewError(http.StatusBadRequest, "Upload token limit reached")
	}
	if err != nil {
		log.Err(err).Msg("Failed to create upload token")

		return nil, huma.NewError(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}

	return &JSONOutput[db.CreatedUploadToken]{Body: uploadToken, CacheControl: "no-store"}, nil
}

func (app *Application) deleteUploadTokenAPI(
	_ context.Context,
	request *FormInput[DeleteUploadTokenForm],
) (*TextOutput, error) {
	c := request.Gin
	input := request.RawBody.Data()

	account, _ := getAccount(c)

	id := input.ID
	if id == 0 {
		return nil, huma.NewError(http.StatusBadRequest, http.StatusText(http.StatusBadRequest))
	}

	if err := app.db.WithContext(c.Request.Context()).DeleteUploadToken(account.ID, id); err != nil {
		log.Err(err).Msg("Failed to delete upload token")

		return nil, huma.NewError(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}

	return textOutput("Upload token deleted successfully"), nil
}

func (app *Application) deleteInviteCodeAPI(
	_ context.Context,
	request *FormInput[DeleteInviteCodeForm],
) (*TextOutput, error) {
	c := request.Gin
	input := request.RawBody.Data()

	account, _ := getAccount(c)
	inviteCode := input.Code
	if inviteCode == "" {
		return nil, huma.NewError(http.StatusBadRequest, http.StatusText(http.StatusBadRequest))
	}

	if err := app.db.WithContext(c.Request.Context()).DeleteInviteCode(inviteCode, account.ID); err != nil {
		log.Err(err).Msg("Failed to delete invite code")

		return nil, huma.NewError(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}

	return textOutput("Invite code deleted successfully"), nil
}

func (app *Application) deleteFilesAPI(_ context.Context, request *EmptyInput) (*TextOutput, error) {
	c := request.Gin

	account, _ := getAccount(c)

	err := app.deleteFilesFromAccount(c.Request.Context(), account.ID)
	if errors.Is(err, ErrPartialDeleteFailed) {
		return nil, huma.NewError(http.StatusInternalServerError, "Some files could not be deleted")
	}
	if err != nil {
		log.Err(err).Msg("Failed to delete files from account")

		return nil, huma.NewError(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}

	return textOutput("Files deleted"), nil
}

func (app *Application) deleteFilesFromAccount(ctx context.Context, accountID uint) (err error) {
	database := app.db.WithContext(ctx)
	files, err := database.GetAllFilesFromAccount(accountID)
	if err != nil {
		return
	}

	var failed int
	for _, file := range files {
		if err = app.deleteFile(ctx, file.FileName); err != nil {
			log.Err(err).
				Str("file", file.FileName).
				Msg("Failed to delete file from storage; keeping DB row for retry")
			failed++

			continue
		}
		if err = database.DeleteFileEntry(file.FileName, accountID); err != nil {
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

func (app *Application) fileStatsAPI(_ context.Context, request *EmptyInput) (*JSONOutput[FileStatsOutput], error) {
	c := request.Gin

	account, _ := getAccount(c)

	var output FileStatsOutput

	database := app.db.WithContext(c.Request.Context())
	totalFiles, totalStorage, err := database.GetFileStats(account.ID)
	if err != nil {
		log.Err(err).Msg("Failed to get file stats")

		return nil, huma.NewError(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}

	output.Count = totalFiles
	output.SizeTotal = totalStorage

	output.Tags, err = database.GetAccountTags(account.ID)
	if err != nil {
		log.Err(err).Msg("Failed to get account tags")

		return nil, huma.NewError(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}

	return jsonOutput(output), nil
}

// FileFilterInput is the tag/filter selection shared by the file listing and
// the filter tags API.
type FileFilterInput struct {
	// Repeatable; files must have every given tag.
	Tag []string `form:"tag" query:"tag,explode"`
	// Repeatable: "untagged" for files without tags, "public" for public files, "private" for private files.
	// Multiple filters combine, e.g. filter=untagged&filter=private.
	Filter []string `form:"filter" query:"filter,explode"`
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

	Skip uint   `form:"skip,default=0"          query:"skip" default:"0"`                                            // Used for pagination
	Sort string `form:"sort,default=created_at" query:"sort" default:"created_at" enum:"created_at,views,file_size"` // "created_at", "views", "file_size"
	Desc bool   `form:"desc,default=true"       query:"desc" default:"true"`                                         // true for descending, false for ascending
}

type FilesApiOutput struct {
	Files []db.Files `json:"files"`
	Count int64      `json:"count"`
}

func (app *Application) filesAPI(_ context.Context, request *FilesInput) (*JSONOutput[FilesApiOutput], error) {
	c := request.Gin
	input := &request.FilesApiInput

	account, _ := getAccount(c)

	if input.Skip > maxPaginationSkip {
		return nil, huma.NewError(http.StatusBadRequest, http.StatusText(http.StatusBadRequest))
	}

	if !input.normalize() {
		return nil, huma.NewError(http.StatusBadRequest, http.StatusText(http.StatusBadRequest))
	}

	allowedSorts := []string{
		"created_at",
		"views",
		"file_size",
	}
	if !slices.Contains(allowedSorts, input.Sort) {
		return nil, huma.NewError(http.StatusBadRequest, http.StatusText(http.StatusBadRequest))
	}

	// Api returns 8 files at a time
	var limit uint = 8

	files, count, err := app.db.WithContext(c.Request.Context()).GetFilesPaginatedFromAccount(
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

		return nil, huma.NewError(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}

	return jsonOutput(FilesApiOutput{Files: files, Count: count}), nil
}

type FilterTagsApiOutput struct {
	Tags []string `json:"tags"`
}

// filterTagsAPI lists the tags on files matching the given tags and filters,
// so a client can offer only tags that still narrow the selection down.
func (app *Application) filterTagsAPI(
	_ context.Context,
	request *FilterTagsInput,
) (*JSONOutput[FilterTagsApiOutput], error) {
	c := request.Gin
	input := &request.FileFilterInput

	account, _ := getAccount(c)

	if !input.normalize() {
		return nil, huma.NewError(http.StatusBadRequest, http.StatusText(http.StatusBadRequest))
	}

	tags, err := app.db.WithContext(c.Request.Context()).GetFilteredTags(account.ID, input.Tag, input.Filter)
	if err != nil {
		log.Err(err).Msg("Failed to get filter tags")

		return nil, huma.NewError(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}
	if tags == nil {
		tags = []string{}
	}

	return jsonOutput(FilterTagsApiOutput{Tags: tags}), nil
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
	app.staticPage(c, "admin")
}
