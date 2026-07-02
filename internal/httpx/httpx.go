package httpx

import (
	"net/http"
	"time"
)

// NewClient returns an HTTP client with a fixed timeout.
func NewClient(timeout time.Duration) *http.Client {
	return &http.Client{Timeout: timeout}
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
