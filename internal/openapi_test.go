package internal

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/BatteredBunny/hostling/internal/db"
)

func TestOpenAPISchemaIsCurrent(t *testing.T) {
	generated, err := OpenAPISchema()
	if err != nil {
		t.Fatal(err)
	}
	committed, err := os.ReadFile("../frontend/openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bytes.TrimSpace(generated), bytes.TrimSpace(committed)) {
		t.Fatal("OpenAPI schema is stale; run just generate-api")
	}
	if bytes.Contains(generated, []byte("TokenHash")) ||
		bytes.Contains(generated, []byte("UploaderID")) {
		t.Fatal("schema exposes internal model fields")
	}
}

func TestOpenAPIRoutesAndBodies(t *testing.T) {
	ts := newTestServer(t)
	schema, err := OpenAPISchema()
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		Paths map[string]map[string]struct {
			ID          string `json:"operationId"`
			Parameters  []struct{ Name, In string }
			RequestBody struct {
				Required bool
				Content  map[string]json.RawMessage
			}
		}
		Components struct {
			Schemas map[string]struct {
				Properties map[string]json.RawMessage
				Required   []string
				MaxLength  int
				MaxItems   int
			}
		}
	}
	if err := json.Unmarshal(schema, &document); err != nil {
		t.Fatal(err)
	}
	actual := map[string]bool{}
	for _, route := range ts.app.Router.Routes() {
		actual[route.Method+" "+route.Path] = true
	}
	ids := map[string]bool{}
	for path, methods := range document.Paths {
		for method, operation := range methods {
			if !actual[strings.ToUpper(method)+" "+path] {
				t.Errorf("unregistered operation %s", operation.ID)
			}
			if operation.ID == "" || ids[operation.ID] {
				t.Errorf("invalid operation ID %s", operation.ID)
			}
			ids[operation.ID] = true
			if len(operation.RequestBody.Content) > 0 && method != "get" {
				if _, ok := operation.RequestBody.Content["multipart/form-data"]; !ok {
					t.Errorf("%s must use multipart form bodies", operation.ID)
				}
				for _, parameter := range operation.Parameters {
					if parameter.In == "query" {
						t.Errorf("%s has form fields in query", operation.ID)
					}
				}
			}
		}
	}
	if len(ids) != 24 {
		t.Errorf("expected 24 operations, got %d", len(ids))
	}

	for _, field := range []string{"FileName", "Tags", "Public"} {
		if _, ok := document.Components.Schemas["Files"].Properties[field]; !ok {
			t.Errorf("missing file field %s", field)
		}
	}
	if document.Components.Schemas["TagName"].MaxLength != db.TagMaxLength {
		t.Fatal("tag name constraint must be available to generated frontend schemas")
	}
	if document.Components.Schemas["UploadTags"].MaxItems != db.MaxTagsPerFile {
		t.Fatal("tag count constraint must be available to generated frontend schemas")
	}
	anon := ts.newClient()
	if status, _ := anon.get("/api/openapi.json"); status != http.StatusOK {
		t.Fatal("schema endpoint unavailable")
	}
}

func TestRegistrationJSONResponse(t *testing.T) {
	ts := newTestServer(t)
	client := ts.newClient()
	request, err := http.NewRequest(
		http.MethodPost,
		client.base+"/api/auth/register",
		strings.NewReader("code="+testInitialToken),
	)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Accept", "application/json")
	response, err := client.http.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("status %d: %s", response.StatusCode, body)
	}
	var redirect RedirectOutput
	if err := json.Unmarshal(body, &redirect); err != nil || redirect.Redirect != "/gallery" {
		t.Fatalf("invalid registration response: %s", body)
	}
	if status, _ := client.get("/api/account/settings"); status != http.StatusOK {
		t.Fatal("JSON registration did not establish a session")
	}
}
