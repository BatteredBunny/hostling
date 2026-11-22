package internal

import (
	"context"
	"github.com/BatteredBunny/hostling/internal/db"
	"github.com/danielgtaylor/huma/v2"
	"net/http"
	"os"

	embed "github.com/BatteredBunny/hostling"
	"github.com/BatteredBunny/hostling/internal/tags"
	"github.com/gin-gonic/gin"
)

const defaultDevFrontendURL = "http://localhost:4321"

func devFrontendURL() string {
	if url := os.Getenv("HOSTLING_FRONTEND_URL"); url != "" {
		return url
	}

	return defaultDevFrontendURL
}

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
	admin := loggedIn && account.AccountType == db.AccountTypeAdmin

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

// Every page of the single-page app gets the same shell; the app itself
// decides what to show, and the APIs decide what each account may see.
var appPages = []string{"/", "/login", "/register", "/gallery", "/settings", "/tokens", "/admin"}

func (app *Application) appShell(c *gin.Context) {
	if tags.DevMode {
		c.Redirect(http.StatusTemporaryRedirect, devFrontendURL()+c.Request.URL.RequestURI())

		return
	}
	file, err := embed.BuildFiles().Open("index.html")
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}
	defer file.Close()
	c.Header("Cache-Control", "no-store")
	c.DataFromReader(http.StatusOK, -1, "text/html; charset=utf-8", file, nil)
}
