package judas

import (
	"log"
	"net/http"
	"net/url"
	"testing"
)

func testLogger(t *testing.T) *log.Logger {
	return log.New(testWriter{t}, "test", log.LstdFlags)
}

type testWriter struct {
	t *testing.T
}

func (tw testWriter) Write(p []byte) (n int, err error) {
	tw.t.Log(string(p))
	return len(p), nil
}

func TestPhishingProxyModifiesRequestCorrectly(t *testing.T) {
	targetURL, _ := url.Parse("https://target.com")
	proxy := &phishingProxy{
		TargetURL: targetURL,
		Logger:    testLogger(t),
	}

	// Simulate a phishing attack where the user clicked through entirely through judas.
	request, _ := http.NewRequest("POST", "https://phishingsite.com/login", nil)
	request.Header.Set("Referer", "https://phishingsite.com")
	request.Header.Set("Origin", "https://phishingsite.com")
	proxy.Director(request)

	expectedHost := targetURL.Host
	actualHost := request.URL.Host
	if expectedHost != actualHost {
		t.Fatalf("Unexpected Host header, expected %s, got %s", expectedHost, actualHost)
	}

	actualReferer := request.Referer()
	expectedReferer := targetURL.String()
	if actualReferer != expectedReferer {
		t.Fatalf("Unexpected Referer header, expected %s, got %s", expectedReferer, actualReferer)
	}

	actualOrigin := request.Header.Get("Origin")
	expectedOrigin := targetURL.String()
	if actualOrigin != expectedOrigin {
		t.Fatalf("Unexpected Origin header, expected %s, got %s", expectedOrigin, actualOrigin)
	}
}

func TestPhishingProxyModifiesResponseCorrectly(t *testing.T) {
	targetURL, _ := url.Parse("https://target.com")

	_ = &phishingProxy{
		TargetURL: targetURL,
		Logger:    testLogger(t),
	}
	// Create HTTP response from text
	// Modify it with the proxy
	// Ensure it's correctly modified to be understood by the victim's browser

}
