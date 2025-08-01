//go:build !mock_test_off

// This will build if build tag mock_test_off is not parsed and will try to mock
// all tests in _test.go
package bitstamp

import (
	"log"
	"os"
	"testing"

	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

var mockTests = true

func TestMain(m *testing.M) {
	b = new(Bitstamp)
	if err := testexch.Setup(b); err != nil {
		log.Fatalf("Bitstamp Setup error: %s", err)
	}

	if err := testexch.MockHTTPInstance(b, "api"); err != nil {
		log.Fatalf("Bitstamp MockHTTPInstance error: %s", err)
	}

	os.Exit(m.Run())
}
