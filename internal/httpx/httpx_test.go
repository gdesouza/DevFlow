package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestApplyBasicAuthAndApplyAuth(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "https://example.com", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	ApplyBasicAuth(req, "user", "token")
	if got, want := req.Header.Get("Authorization"), "Basic dXNlcjp0b2tlbg=="; got != want {
		t.Fatalf("ApplyBasicAuth authorization = %q, want %q", got, want)
	}

	req, err = http.NewRequest(http.MethodGet, "https://example.com", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	ApplyBasicAuth(req, "", "token")
	if got := req.Header.Get("Authorization"); got != "" {
		t.Fatalf("ApplyBasicAuth with missing username should not set auth, got %q", got)
	}

	req, err = http.NewRequest(http.MethodGet, "https://example.com", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	ApplyAuth(req, "user", "token")
	if got, want := req.Header.Get("Authorization"), "Basic dXNlcjp0b2tlbg=="; got != want {
		t.Fatalf("ApplyAuth basic auth = %q, want %q", got, want)
	}

	req, err = http.NewRequest(http.MethodGet, "https://example.com", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	ApplyAuth(req, "", "token")
	if got, want := req.Header.Get("Authorization"), "Bearer token"; got != want {
		t.Fatalf("ApplyAuth bearer = %q, want %q", got, want)
	}
}

func TestRegisterTestServerAndNewClient(t *testing.T) {
	host := "fake.example"
	RegisterTestServer(host, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer token" {
			t.Fatalf("expected bearer auth header, got %q", r.Header.Get("Authorization"))
		}
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte("ok"))
	}))
	defer UnregisterTestServer(host)

	client := NewClient(time.Second)
	req, err := http.NewRequest(http.MethodGet, "https://"+host+"/ping", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	ApplyAuth(req, "", "token")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("client.Do: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusTeapot {
		t.Fatalf("unexpected status code: %d", resp.StatusCode)
	}
}

func TestRoundTripFallsBackToBaseTransport(t *testing.T) {
	wantErr := http.ErrUseLastResponse
	transport := &testAwareTransport{base: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return nil, wantErr
	})}
	req := httptest.NewRequest(http.MethodGet, "https://example.com", nil)

	_, err := transport.RoundTrip(req)
	if err != wantErr {
		t.Fatalf("expected fallback error %v, got %v", wantErr, err)
	}
}

type roundTripperFunc func(req *http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
