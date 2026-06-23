package checker_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cucumber/godog"

	"healthwatch/internal/checker"
	"healthwatch/internal/config"
)

// checkerFeature holds the state shared between the steps of a single
// scenario. A fresh logical state is established by reset(), called from
// an After hook, since godog re-runs the same registered step functions
// for every scenario in the feature file.
type checkerFeature struct {
	server     *httptest.Server
	httpClient *http.Client
	target     config.Target
	result     checker.Result
}

func (f *checkerFeature) reset() {
	if f.server != nil {
		f.server.Close()
	}
	*f = checkerFeature{}
}

func (f *checkerFeature) aTargetWebsiteThatRespondsWithStatus(code int) error {
	f.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(code)
	}))
	f.target = config.Target{Name: "feature-target", URL: f.server.URL, TimeoutSeconds: 2}
	return nil
}

func (f *checkerFeature) aTargetWebsiteThatIsUnreachable() error {
	f.server = httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
	f.target = config.Target{Name: "feature-target", URL: f.server.URL, TimeoutSeconds: 1}
	f.server.Close() // closed immediately: the URL is now unreachable
	return nil
}

func (f *checkerFeature) aTargetWebsiteServedOverHTTPSWithAValidCertificate() error {
	f.server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	f.httpClient = f.server.Client() // already trusts the test server's certificate
	f.target = config.Target{Name: "feature-target", URL: f.server.URL, TimeoutSeconds: 2}
	return nil
}

func (f *checkerFeature) healthwatchChecksTheTarget() error {
	var c *checker.Checker
	if f.httpClient != nil {
		c = checker.NewWithClient(f.httpClient)
	} else {
		c = checker.New(2 * time.Second)
	}
	f.result = c.Check(context.Background(), f.target)
	return nil
}

func (f *checkerFeature) theResultStatusShouldBe(status string) error {
	if string(f.result.Status) != status {
		return fmt.Errorf("got status %q, want %q (error: %s)", f.result.Status, status, f.result.Error)
	}
	return nil
}

func (f *checkerFeature) theResultShouldRecordANonNegativeLatency() error {
	if f.result.LatencyMS < 0 {
		return fmt.Errorf("got latency %dms, want >= 0", f.result.LatencyMS)
	}
	return nil
}

func (f *checkerFeature) theResultShouldHaveAnErrorMessage() error {
	if f.result.Error == "" {
		return fmt.Errorf("expected a non-empty error message, got none")
	}
	return nil
}

func (f *checkerFeature) theResultShouldReportAPositiveNumberOfTLSDaysRemaining() error {
	if f.result.TLSDaysRemaining == nil {
		return fmt.Errorf("expected TLSDaysRemaining to be set, got nil")
	}
	if *f.result.TLSDaysRemaining <= 0 {
		return fmt.Errorf("got %d TLS days remaining, want > 0", *f.result.TLSDaysRemaining)
	}
	return nil
}

func InitializeScenario(sc *godog.ScenarioContext) {
	f := &checkerFeature{}

	sc.Given(`^a target website that responds with status (\d+)$`, f.aTargetWebsiteThatRespondsWithStatus)
	sc.Given(`^a target website that is unreachable$`, f.aTargetWebsiteThatIsUnreachable)
	sc.Given(`^a target website served over HTTPS with a valid certificate$`, f.aTargetWebsiteServedOverHTTPSWithAValidCertificate)
	sc.When(`^Healthwatch checks the target$`, f.healthwatchChecksTheTarget)
	sc.Then(`^the result status should be "([^"]*)"$`, f.theResultStatusShouldBe)
	sc.Then(`^the result should record a non-negative latency$`, f.theResultShouldRecordANonNegativeLatency)
	sc.Then(`^the result should have an error message$`, f.theResultShouldHaveAnErrorMessage)
	sc.Then(`^the result should report a positive number of TLS days remaining$`, f.theResultShouldReportAPositiveNumberOfTLSDaysRemaining)

	sc.After(func(ctx context.Context, _ *godog.Scenario, err error) (context.Context, error) {
		f.reset()
		return ctx, err
	})
}

func TestFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: InitializeScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"checker.feature"},
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run checker feature tests")
	}
}
