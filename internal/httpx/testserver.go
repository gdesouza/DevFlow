package httpx

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"
)

var testServerSeq uint64

type TestServer struct {
	URL  string
	host string
}

// NewTestServer registers an in-memory HTTP handler and returns a fake server URL.
func NewTestServer(t *testing.T, handler http.Handler) *TestServer {
	t.Helper()

	id := atomic.AddUint64(&testServerSeq, 1)
	host := fmt.Sprintf("httptest-%d.local", id)
	RegisterTestServer(host, handler)
	t.Cleanup(func() {
		UnregisterTestServer(host)
	})

	return &TestServer{
		URL:  "http://" + host,
		host: host,
	}
}

// Close removes the registered handler.
func (s *TestServer) Close() {
	UnregisterTestServer(s.host)
}
