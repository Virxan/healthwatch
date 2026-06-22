// Package api_test holds the API contract feature tests. Unlike the
// checker feature tests, these hit a real, already-running Healthwatch
// instance over the network - so by default they are skipped (see
// TestFeatures) unless HEALTHWATCH_E2E=1 is set, which `just test-e2e`
// does after pointing HEALTHWATCH_BASE_URL at a deployed instance
// (locally via `just run`, or at the Service/Ingress exposed by the pod
// running in k3d).
package api_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/cucumber/godog"
)

type apiFeature struct {
	baseURL     string
	client      *http.Client
	resp        *http.Response
	body        []byte
	requestPath string
}

func (f *apiFeature) reset() {
	*f = apiFeature{client: &http.Client{Timeout: 5 * time.Second}}
}

func (f *apiFeature) theHealthwatchAPIBaseURLDefaultsTo(defaultURL string) error {
	if v := os.Getenv("HEALTHWATCH_BASE_URL"); v != "" {
		f.baseURL = v
	} else {
		f.baseURL = defaultURL
	}
	return nil
}

func (f *apiFeature) iGET(path string) error {
	f.requestPath = path
	resp, err := f.client.Get(f.baseURL + path)
	if err != nil {
		return fmt.Errorf("GET %s%s: %w", f.baseURL, path, err)
	}
	defer resp.Body.Close()

	f.resp = resp
	f.body, err = io.ReadAll(resp.Body)
	return err
}

func (f *apiFeature) theResponseStatusShouldBe(code int) error {
	if f.resp.StatusCode != code {
		return fmt.Errorf("GET %s: status = %d, want %d", f.requestPath, f.resp.StatusCode, code)
	}
	return nil
}

func (f *apiFeature) theResponseShouldBeAJSONArray() error {
	var v []json.RawMessage
	if err := json.Unmarshal(f.body, &v); err != nil {
		return fmt.Errorf("GET %s: response is not a JSON array: %w", f.requestPath, err)
	}
	return nil
}

func (f *apiFeature) theResponseContentTypeShouldBe(want string) error {
	got := f.resp.Header.Get("Content-Type")
	if got != want {
		return fmt.Errorf("GET %s: Content-Type = %q, want %q", f.requestPath, got, want)
	}
	return nil
}

func InitializeScenario(sc *godog.ScenarioContext) {
	f := &apiFeature{}
	f.reset()

	sc.Given(`^the Healthwatch API base URL defaults to "([^"]*)"$`, f.theHealthwatchAPIBaseURLDefaultsTo)
	sc.When(`^I GET "([^"]*)"$`, f.iGET)
	sc.Then(`^the response status should be (\d+)$`, f.theResponseStatusShouldBe)
	sc.Then(`^the response should be a JSON array$`, f.theResponseShouldBeAJSONArray)
	sc.Then(`^the response content type should be "([^"]*)"$`, f.theResponseContentTypeShouldBe)
}

// TestFeatures runs the API contract suite against a live Healthwatch
// instance. It is opt-in: run `just test-e2e` (or set HEALTHWATCH_E2E=1
// yourself) once an instance is reachable at HEALTHWATCH_BASE_URL
// (default http://localhost:8080), otherwise it skips - so a plain
// `go test ./...` or `just test` never depends on the network.
func TestFeatures(t *testing.T) {
	if os.Getenv("HEALTHWATCH_E2E") == "" {
		t.Skip("skipping API contract tests: set HEALTHWATCH_E2E=1 against a running instance (see `just test-e2e`)")
	}

	suite := godog.TestSuite{
		ScenarioInitializer: InitializeScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"api_contract.feature"},
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run API contract tests")
	}
}
