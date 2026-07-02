package bitbucket

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"

	"devflow/internal/httpx"
)

var testServerSeq uint64

type testServer struct {
	URL  string
	host string
}

func newTestServer(t *testing.T, handler http.Handler) *testServer {
	t.Helper()

	id := atomic.AddUint64(&testServerSeq, 1)
	host := fmt.Sprintf("bitbucket.test-%d", id)
	httpx.RegisterTestServer(host, handler)
	t.Cleanup(func() {
		httpx.UnregisterTestServer(host)
	})

	return &testServer{
		URL:  "http://" + host,
		host: host,
	}
}

func (s *testServer) Close() {
	httpx.UnregisterTestServer(s.host)
}
