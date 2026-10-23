package internal

import (
	"context"
	"github.com/danielgtaylor/huma/v2"
	"net/http"

	embed "github.com/BatteredBunny/hostling"
	"github.com/gin-gonic/gin"
)

type FrontendContext struct {
	Branding        string   `json:"branding"`
	Tagline         string   `json:"tagline"`
	LoggedIn        bool     `json:"logged_in"`
	AccountID       uint     `json:"account_id"`
	IsAdmin         bool     `json:"is_admin"`
	HasAdminWarning bool     `json:"has_admin_warning"`
	Providers       []string `json:"providers"`
}

type ProviderInfo struct {
	Name       string `json:"Name"`
	IsLinked   bool   `json:"IsLinked"`
	Username   string `json:"Username"`
	ProfileURL string `json:"ProfileURL"`
}

type SettingsOutput struct {
	Providers             []ProviderInfo `json:"providers"`
	NoProvidersConfigured bool           `json:"no_providers_configured"`
	UnlinkedAccount       bool           `json:"unlinked_account"`
}

type AdminOverviewOutput struct {
	Accounts              []AccountStats `json:"accounts"`
	MaxUploadSize         int64          `json:"max_upload_size"`
	Version               string         `json:"version"`
	NoProvidersConfigured bool           `json:"no_providers_configured"`
	FailedProviders       []string       `json:"failed_providers"`
	FileStorageMethod     string         `json:"file_storage_method"`
	LoginProviders        []string       `json:"login_providers"`
}

type RedirectOutput struct {
	Redirect string `json:"redirect"`
}

func (app *Application) frontendContextAPI(
	_ context.Context,
	request *EmptyInput,
) (*JSONOutput[FrontendContext], error) {
	c := request.Gin

	account, loggedIn, ok := app.validateOrAbort(c)
	if !ok {
		return nil, huma.Error400BadRequest("Could not validate session")
	}
	providers := append([]string{}, app.getConfiguredProviders()...)
	admin := loggedIn && account.AccountType == "ADMIN"

	return jsonOutput(FrontendContext{
		Branding:        app.config.Branding,
		Tagline:         app.config.Tagline,
		LoggedIn:        loggedIn,
		AccountID:       account.ID,
		IsAdmin:         admin,
		HasAdminWarning: admin && app.hasAdminWarning(),
		Providers:       providers,
	}), nil
}

// The UI is built by Astro. Request-specific data is fetched through APIs.
func (app *Application) staticPage(c *gin.Context, page string) {
	file, err := embed.BuildFiles().Open(page + "/index.html")
	if page == "index" && err != nil {
		file, err = embed.BuildFiles().Open("index.html")
	}
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}
	defer file.Close()
	c.Header("Cache-Control", "no-store")
	c.DataFromReader(http.StatusOK, -1, "text/html; charset=utf-8", file, nil)
}

func (app *Application) settingsPage(c *gin.Context) {
	if _, ok := app.requireAuth(c); !ok {
		return
	}
	app.staticPage(c, "settings")
}
