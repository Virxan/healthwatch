package checker

import "testing"

// tlsDaysRemaining is unexported, so this one stays as a white-box test;
// the behavioural TLS test (a full Check() against a real TLS server)
// lives in checker_test.go using the exported NewWithClient constructor.
func TestTLSDaysRemainingNilForPlainHTTP(t *testing.T) {
	if got := tlsDaysRemaining(nil); got != nil {
		t.Errorf("tlsDaysRemaining(nil) = %v, want nil", *got)
	}
}
