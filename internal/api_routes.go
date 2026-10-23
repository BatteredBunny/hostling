package internal

import (
	"encoding/json"
	"errors"
	"github.com/BatteredBunny/hostling/internal/db"
	"io"
	"mime/multipart"
	"net/http"
	"sync"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humagin"
	"github.com/gin-gonic/gin"
)

// RequestContext exposes the existing Gin authentication and cookie helpers to
// typed handlers. It is resolved per request and never appears in the schema.
type RequestContext struct {
	Gin *gin.Context `json:"-"`
}

func (r *RequestContext) Resolve(ctx huma.Context) []error {
	r.Gin = humagin.Unwrap(ctx)

	return nil
}

type (
	EmptyInput       struct{ RequestContext }
	FormInput[T any] struct {
		RequestContext
		RawBody huma.MultipartFormFiles[T]
	}
)
type FilesInput struct {
	RequestContext
	FilesApiInput
}
type FilterTagsInput struct {
	RequestContext
	FileFilterInput
}

type JSONOutput[T any] struct {
	CacheControl string `header:"Cache-Control"`
	Body         T
}

func jsonOutput[T any](body T) *JSONOutput[T] {
	return &JSONOutput[T]{Body: body, CacheControl: "no-store"}
}

type TextOutput struct {
	Status   int
	Location string `header:"Location"`
	Body     []byte `                  contentType:"text/plain"`
}

func textOutput(body string) *TextOutput {
	return &TextOutput{Status: http.StatusOK, Body: []byte(body)}
}

type RegistrationOutput struct {
	Status   int
	Location string `header:"Location"`
	Body     RedirectOutput
}

type NewUploadTokenForm struct {
	Nickname UploadTokenName `form:"nickname" required:"true"`
}
type DeleteUploadTokenForm struct {
	ID uint `form:"id" required:"true" minimum:"1"`
}
type DeleteInviteCodeForm struct {
	Code string `form:"invite_code" required:"true"`
}
type UploadForm struct {
	File            UploadedFile `form:"file"             required:"true"  contentType:"application/octet-stream"`
	UploadToken     string       `form:"upload_token"     required:"false"`
	ExpiryDate      string       `form:"expiry_date"      required:"false"`
	ExpiryTimestamp string       `form:"expiry_timestamp" required:"false"`
	Plain           bool         `form:"plain"            required:"false"                                        default:"false"`
	Tag             UploadTags   `form:"tag"              required:"false"`
}

// Schema providers keep API constraints tied to the domain constants.
type UploadTokenName string

func (UploadTokenName) Schema(registry huma.Registry) *huma.Schema {
	minimum, maximum := 1, maxUploadTokenNickname
	registry.Map()["UploadTokenName"] = &huma.Schema{Type: "string", MinLength: &minimum, MaxLength: &maximum}

	return &huma.Schema{Ref: "#/components/schemas/UploadTokenName"}
}

type TagName string

func (TagName) Schema(registry huma.Registry) *huma.Schema {
	maximum := db.TagMaxLength
	registry.Map()["TagName"] = &huma.Schema{Type: "string", MaxLength: &maximum}

	return &huma.Schema{Ref: "#/components/schemas/TagName"}
}

type UploadTags []string

func (UploadTags) Schema(registry huma.Registry) *huma.Schema {
	maximum := db.MaxTagsPerFile
	registry.Map()["UploadTags"] = &huma.Schema{Type: "array", MaxItems: &maximum, Items: TagName("").Schema(registry)}

	return &huma.Schema{Ref: "#/components/schemas/UploadTags"}
}

// UploadedFile describes the binary input without opening it during validation.
// Huma 2.39's FormFile MIME validator opens an extra descriptor without closing
// it. Resolve the header here; the upload handler owns the single open reader.
type UploadedFile string

func (UploadedFile) Schema(huma.Registry) *huma.Schema {
	return &huma.Schema{Type: "string", Format: "binary"}
}

type UploadInput struct {
	RequestContext
	RawBody    huma.MultipartFormFiles[UploadForm]
	FileHeader *multipart.FileHeader `json:"-"`
}

func (r *UploadInput) Resolve(ctx huma.Context) []error {
	r.RequestContext.Resolve(ctx)
	headers := r.RawBody.Form.File["file"]
	if len(headers) != 1 {
		return []error{huma.Error400BadRequest("Exactly one file is required")}
	}
	r.FileHeader = headers[0]

	return nil
}

var apiErrorSetup sync.Once

// Huma's error factory is process-wide. Configure it once, explicitly when
// creating our API, to preserve the existing 400 validation status.
func configureAPIErrors() {
	apiErrorSetup.Do(func() {
		huma.NewErrorWithContext = func(_ huma.Context, status int, message string, errs ...error) huma.StatusError {
			if status == http.StatusUnprocessableEntity {
				status = http.StatusBadRequest
			}

			return huma.NewError(status, message, errs...)
		}
	})
}

// Support existing URL-encoded forms as well as multipart forms. The Gin
// middleware has already applied the request limit and parsed the form.
type (
	humaContext = huma.Context
	formContext struct{ humaContext }
)

func (c formContext) Unwrap() huma.Context { return c.humaContext }

func (c formContext) GetMultipartForm() (*multipart.Form, error) {
	form, err := c.humaContext.GetMultipartForm()
	if errors.Is(err, http.ErrNotMultipart) {
		request := humagin.Unwrap(c.humaContext).Request

		return &multipart.Form{Value: request.PostForm}, nil
	}

	return form, err
}

func apiConfig() huma.Config {
	config := huma.DefaultConfig("Hostling API", "1")
	config.OpenAPIPath, config.DocsPath, config.SchemasPath = "", "", ""
	format := huma.DefaultJSONFormat
	format.Marshal = func(writer io.Writer, value any) error {
		body, err := json.Marshal(value)
		if err != nil {
			return err
		}
		_, err = writer.Write(body)

		return err
	}
	config.Formats = map[string]huma.Format{"application/json": format, "json": format}
	config.CreateHooks = nil // Keep response bodies free of documentation fields.
	config.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		"session": {Type: "apiKey", In: "cookie", Name: AUTH_COOKIE},
	}
	config.OnAddOperation = append(config.OnAddOperation, func(_ *huma.OpenAPI, op *huma.Operation) {
		if op.RequestBody != nil {
			op.RequestBody.Required = true
		}
		if response, exists := op.Responses["422"]; exists {
			if op.Responses["400"] == nil {
				op.Responses["400"] = response
			}
			delete(op.Responses, "422")
		}
	})

	return config
}

// Shared metadata does not describe request/response types; Huma derives those
// directly from each typed handler below.
func apiOperation(id, method, path string, status int) huma.Operation {
	return huma.Operation{
		OperationID:   id,
		Method:        method,
		Path:          path,
		DefaultStatus: status,
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusNotFound,
			http.StatusRequestEntityTooLarge,
			http.StatusTooManyRequests,
			http.StatusInternalServerError,
		},
	}
}

// The same typed registrations are used by the running server and the schema
// command. Creating the schema does not initialize storage or call handlers.
func (app *Application) registerAPIs(router *gin.Engine) huma.API {
	configureAPIErrors()
	config := apiConfig()
	makeAPI := func(middleware ...gin.HandlerFunc) *huma.Group {
		// Empty-prefix Gin groups keep middleware scoped while operations retain
		// their complete public paths in the shared OpenAPI document.
		api := humagin.NewWithGroup(router, router.Group("", middleware...), config)
		api.UseMiddleware(func(ctx huma.Context, next func(huma.Context)) { next(formContext{ctx}) })

		return huma.NewGroup(api)
	}
	public := makeAPI()
	auth := makeAPI(app.ratelimitMiddleware(), app.apiMiddleware())
	upload := makeAPI(
		app.ratelimitMiddleware(),
		app.multipartFormMiddleware(0),
		app.hasUploadOrSessionTokenMiddleware(),
	)
	account := makeAPI(app.ratelimitMiddleware(), app.apiMiddleware(), app.verifySessionAuthentication())
	admin := makeAPI(app.ratelimitMiddleware(), app.apiMiddleware(), app.verifySessionAuthentication(), app.isAdmin())

	for _, group := range []*huma.Group{account, admin} {
		group.UseSimpleModifier(func(op *huma.Operation) {
			op.Security = []map[string][]string{{"session": {}}}
		})
	}
	huma.Register(
		public,
		apiOperation("getContext", http.MethodGet, "/api/context", http.StatusOK),
		app.frontendContextAPI,
	)
	registration := apiOperation("register", http.MethodPost, "/api/auth/register", http.StatusCreated)
	registration.Responses = map[string]*huma.Response{
		"303": {Description: "Redirect to gallery for browser form submissions"},
	}
	huma.Register(auth, registration, app.registerApi)
	uploadOperation := apiOperation("uploadFile", http.MethodPost, "/api/file/upload", http.StatusOK)
	uploadOperation.Responses = map[string]*huma.Response{
		"307": {Description: "Redirect to uploaded file when plain is false"},
	}
	uploadOperation.MaxBodyBytes = -1 // The Gin body-size middleware owns the upload limit.
	uploadOperation.BodyReadTimeout = -1
	uploadOperation.Description = "Accepts a session cookie or upload_token form field. Set plain=true for a text URL; otherwise redirects to the uploaded file."
	huma.Register(upload, uploadOperation, app.uploadFileAPI)
	huma.Register(
		account,
		apiOperation("getSettings", http.MethodGet, "/api/account/settings", http.StatusOK),
		app.settingsAPI,
	)
	huma.Register(
		account,
		apiOperation("deleteAccount", http.MethodDelete, "/api/account/", http.StatusOK),
		app.accountDeleteAPI,
	)
	huma.Register(
		account,
		apiOperation("listUploadTokens", http.MethodGet, "/api/account/upload_tokens", http.StatusOK),
		app.uploadTokensAPI,
	)
	huma.Register(
		account,
		apiOperation("createUploadToken", http.MethodPost, "/api/account/upload_tokens", http.StatusCreated),
		app.newUploadTokenAPI,
	)
	huma.Register(
		account,
		apiOperation("deleteUploadToken", http.MethodDelete, "/api/account/upload_tokens", http.StatusOK),
		app.deleteUploadTokenAPI,
	)
	huma.Register(
		account,
		apiOperation("listInviteCodes", http.MethodGet, "/api/account/invite_code", http.StatusOK),
		app.inviteCodesAPI,
	)
	huma.Register(
		account,
		apiOperation("deleteInviteCode", http.MethodDelete, "/api/account/invite_code", http.StatusOK),
		app.deleteInviteCodeAPI,
	)
	huma.Register(account, apiOperation("listFiles", http.MethodGet, "/api/account/files", http.StatusOK), app.filesAPI)
	huma.Register(
		account,
		apiOperation("deleteAllFiles", http.MethodDelete, "/api/account/files", http.StatusOK),
		app.deleteFilesAPI,
	)
	huma.Register(
		account,
		apiOperation("getFileStats", http.MethodGet, "/api/account/files/stats", http.StatusOK),
		app.fileStatsAPI,
	)
	huma.Register(
		account,
		apiOperation("getFilterTags", http.MethodGet, "/api/account/files/tags", http.StatusOK),
		app.filterTagsAPI,
	)
	huma.Register(
		account,
		apiOperation("deleteFile", http.MethodDelete, "/api/account/file", http.StatusOK),
		app.deleteFileAPI,
	)
	huma.Register(
		account,
		apiOperation("toggleFileVisibility", http.MethodPost, "/api/account/file/public", http.StatusOK),
		app.toggleFilePublicAPI,
	)
	huma.Register(
		account,
		apiOperation("addFileTag", http.MethodPost, "/api/account/file/tag", http.StatusOK),
		app.addTagAPI,
	)
	huma.Register(
		account,
		apiOperation("removeFileTag", http.MethodDelete, "/api/account/file/tag", http.StatusOK),
		app.deleteTagAPI,
	)
	huma.Register(
		admin,
		apiOperation("getAdminOverview", http.MethodGet, "/api/admin/overview", http.StatusOK),
		app.adminOverviewAPI,
	)
	huma.Register(
		admin,
		apiOperation("deleteUser", http.MethodDelete, "/api/admin/user", http.StatusOK),
		app.adminDeleteAccount,
	)
	huma.Register(
		admin,
		apiOperation("deleteUserFiles", http.MethodDelete, "/api/admin/files", http.StatusOK),
		app.adminDeleteFiles,
	)
	huma.Register(
		admin,
		apiOperation("deleteUserSessions", http.MethodDelete, "/api/admin/sessions", http.StatusOK),
		app.adminDeleteSessions,
	)
	huma.Register(
		admin,
		apiOperation("deleteUserUploadTokens", http.MethodDelete, "/api/admin/upload_tokens", http.StatusOK),
		app.adminDeleteUploadTokens,
	)
	huma.Register(
		admin,
		apiOperation("giveInviteCode", http.MethodPost, "/api/admin/give_invite_code", http.StatusOK),
		app.adminGiveInviteCode,
	)

	return public
}

func OpenAPISchema() ([]byte, error) {
	app := &Application{}
	api := app.registerAPIs(gin.New())

	return json.MarshalIndent(api.OpenAPI(), "", "  ")
}
