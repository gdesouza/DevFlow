package httpx

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"time"
)

var testServerRegistry sync.Map

// NewClient returns an HTTP client with a fixed timeout.
func NewClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout:   timeout,
		Transport: &testAwareTransport{base: http.DefaultTransport},
	}
}

type testAwareTransport struct {
	base http.RoundTripper
}

func (t *testAwareTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if handler, ok := testServerRegistry.Load(req.URL.Host); ok {
		recorder := httptest.NewRecorder()
		handler.(http.Handler).ServeHTTP(recorder, req)
		return recorder.Result(), nil
	}

	if t.base != nil {
		return t.base.RoundTrip(req)
	}

	return http.DefaultTransport.RoundTrip(req)
}

// RegisterTestServer registers an in-memory handler for a fake host used by tests.
func RegisterTestServer(host string, handler http.Handler) {
	testServerRegistry.Store(host, handler)
}

// UnregisterTestServer removes a registered in-memory handler.
func UnregisterTestServer(host string) {
	testServerRegistry.Delete(host)
}

// ApplyBasicAuth sets HTTP Basic auth when credentials are present.
func ApplyBasicAuth(req *http.Request, username, token string) {
	if username == "" || token == "" {
		return
	}

	req.SetBasicAuth(username, token)
}

// ApplyAuth sets Basic auth when a username is present, otherwise falls back to Bearer auth.
func ApplyAuth(req *http.Request, username, token string) {
	if username != "" {
		req.SetBasicAuth(username, token)
		return
	}

	req.Header.Set("Authorization", "Bearer "+token)
}
