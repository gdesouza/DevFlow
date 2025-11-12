package bitbucket

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"devflow/internal/config"
)

// fakeTransport allows test code to return custom responses per request.
type fakeTransport struct {
	mu      sync.Mutex
	calls   int
	handler func(req *http.Request, call int) *http.Response
}

func (f *fakeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	f.mu.Lock()
	call := f.calls
	f.calls++
	h := f.handler
	f.mu.Unlock()
	if h != nil {
		return h(req, call), nil
	}
	return &http.Response{StatusCode: 500, Body: io.NopCloser(strings.NewReader("")), Header: http.Header{}}, nil
}

func TestMakeRequestWithRetry_RateLimitExceeded(t *testing.T) {
	cfg := &config.BitbucketConfig{Workspace: "w"}
	c := NewClient(cfg)
	// disable rate limiter for tests
	c.rateLimiter = nil

	ft := &fakeTransport{handler: func(req *http.Request, call int) *http.Response {
		// always return 429
		return &http.Response{StatusCode: 429, Body: io.NopCloser(strings.NewReader("")), Header: http.Header{}}
	}}
	c.httpClient = &http.Client{Transport: ft}

	_, err := c.makeRequestWithRetry("GET", "some/endpoint", nil, 2)
	if err == nil || !strings.Contains(err.Error(), "rate limit exceeded") {
		t.Fatalf("expected rate limit exceeded error, got: %v", err)
	}
}

func TestMakeRequestWithRetry_SucceedsAfterRetry(t *testing.T) {
	cfg := &config.BitbucketConfig{Workspace: "w"}
	c := NewClient(cfg)
	c.rateLimiter = nil

	ft := &fakeTransport{handler: func(req *http.Request, call int) *http.Response {
		if call == 0 {
			return &http.Response{StatusCode: 429, Body: io.NopCloser(strings.NewReader("")), Header: http.Header{}}
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("ok")), Header: http.Header{"Content-Type": {"application/json"}}}
	}}
	c.httpClient = &http.Client{Transport: ft}

	resp, err := c.makeRequestWithRetry("GET", "some/endpoint", nil, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if string(b) != "ok" {
		t.Fatalf("expected body 'ok', got %q", string(b))
	}
}

func TestSetCommitStatus_Success(t *testing.T) {
	cfg := &config.BitbucketConfig{Workspace: "w"}
	c := NewClient(cfg)
	c.rateLimiter = nil

	ft := &fakeTransport{handler: func(req *http.Request, call int) *http.Response {
		// Verify method/path
		if req.Method != "POST" {
			return &http.Response{StatusCode: 400, Body: io.NopCloser(strings.NewReader("wrong method")), Header: http.Header{}}
		}
		if !strings.Contains(req.URL.Path, "/commit/") || !strings.Contains(req.URL.Path, "/statuses/build") {
			return &http.Response{StatusCode: 404, Body: io.NopCloser(strings.NewReader("not found")), Header: http.Header{}}
		}
		// Read and verify body
		body, _ := io.ReadAll(req.Body)
		var payload map[string]string
		_ = json.Unmarshal(body, &payload)
		if payload["state"] != "SUCCESSFUL" || payload["key"] != "CI" {
			return &http.Response{StatusCode: 400, Body: io.NopCloser(strings.NewReader("invalid payload")), Header: http.Header{}}
		}
		respBody := `{"state":"SUCCESSFUL","key":"CI","name":"CI","url":"http://ci","description":"desc"}`
		return &http.Response{StatusCode: 201, Body: io.NopCloser(strings.NewReader(respBody)), Header: http.Header{"Content-Type": {"application/json"}}}
	}}
	c.httpClient = &http.Client{Transport: ft}

	status, err := c.SetCommitStatus("repo", "abcdef", "SUCCESSFUL", "CI", "CI", "http://ci", "desc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.Key != "CI" || status.State != "SUCCESSFUL" || status.URL != "http://ci" {
		t.Fatalf("unexpected status returned: %+v", status)
	}
}

func TestGetRepositoryReadme_Found(t *testing.T) {
	cfg := &config.BitbucketConfig{Workspace: "w"}
	c := NewClient(cfg)
	c.rateLimiter = nil

	ft := &fakeTransport{handler: func(req *http.Request, call int) *http.Response {
		if strings.HasSuffix(req.URL.Path, "/src/HEAD/README.md") {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("# Hello")), Header: http.Header{"Content-Type": {"text/plain"}}}
		}
		return &http.Response{StatusCode: 404, Body: io.NopCloser(strings.NewReader("not found")), Header: http.Header{}}
	}}
	c.httpClient = &http.Client{Transport: ft}

	name, contents, err := c.GetRepositoryReadme("repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "README.md" {
		t.Fatalf("expected README.md, got %s", name)
	}
	if !strings.Contains(contents, "Hello") {
		t.Fatalf("unexpected contents: %s", contents)
	}
}

func TestGetRepositories_Pagination(t *testing.T) {
	cfg := &config.BitbucketConfig{Workspace: "w"}
	c := NewClient(cfg)
	c.rateLimiter = nil

	ft := &fakeTransport{handler: func(req *http.Request, call int) *http.Response {
		switch call {
		case 0:
			// first page with next
			body := `{"values":[{"name":"r1","full_name":"w/r1"}], "next":"https://api.bitbucket.org/2.0/repositories/w?page=2"}`
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
		case 1:
			body := `{"values":[{"name":"r2","full_name":"w/r2"}], "next":""}`
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
		default:
			return &http.Response{StatusCode: 404, Body: io.NopCloser(strings.NewReader("not found")), Header: http.Header{}}
		}
	}}
	c.httpClient = &http.Client{Transport: ft}

	repos, err := c.GetRepositories()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repos) != 2 {
		t.Fatalf("expected 2 repos, got %d", len(repos))
	}
}

func TestGetTotalRepositoryCount_NoNext(t *testing.T) {
	cfg := &config.BitbucketConfig{Workspace: "w"}
	c := NewClient(cfg)
	c.rateLimiter = nil

	ft := &fakeTransport{handler: func(req *http.Request, call int) *http.Response {
		body := `{"values":[{"name":"r1","full_name":"w/r1"}], "next":""}`
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
	}}
	c.httpClient = &http.Client{Transport: ft}

	count, err := c.getTotalRepositoryCount(10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected count 1, got %d", count)
	}
}

func TestGetTotalRepositoryCount_WithNext(t *testing.T) {
	cfg := &config.BitbucketConfig{Workspace: "w"}
	c := NewClient(cfg)
	c.rateLimiter = nil

	ft := &fakeTransport{handler: func(req *http.Request, call int) *http.Response {
		body := `{"values":[], "next":"https://api.bitbucket.org/2.0/repositories/w?page=2"}`
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
	}}
	c.httpClient = &http.Client{Transport: ft}

	count, err := c.getTotalRepositoryCount(10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 50 {
		t.Fatalf("expected estimate 50, got %d", count)
	}
}

func TestGetRepositoryMainBranch_WithAndWithoutMainBranch(t *testing.T) {
	cfg := &config.BitbucketConfig{Workspace: "w"}
	c := NewClient(cfg)
	c.rateLimiter = nil

	ft := &fakeTransport{handler: func(req *http.Request, call int) *http.Response {
		switch call {
		case 0:
			body := `{"mainbranch":{"name":"develop"}}`
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
		case 1:
			body := `{"name":"repo-no-main"}`
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
		default:
			return &http.Response{StatusCode: 404, Body: io.NopCloser(strings.NewReader("not found")), Header: http.Header{}}
		}
	}}
	c.httpClient = &http.Client{Transport: ft}

	mb, err := c.GetRepositoryMainBranch("repo1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mb != "develop" {
		t.Fatalf("expected develop, got %s", mb)
	}

	mb2, err := c.GetRepositoryMainBranch("repo2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mb2 != "main" {
		t.Fatalf("expected fallback main, got %s", mb2)
	}
}

func TestGetPullRequestsAndDetails(t *testing.T) {
	cfg := &config.BitbucketConfig{Workspace: "w"}
	c := NewClient(cfg)
	c.rateLimiter = nil

	// This fakeTransport will respond to the PR list first, then the PR details
	ft := &fakeTransport{handler: func(req *http.Request, call int) *http.Response {
		// call 0 -> list PRs
		if strings.HasSuffix(req.URL.Path, "/pullrequests") {
			body := `{"values":[{"id":1,"title":"PR 1","state":"OPEN","description":"d1","author":{"display_name":"Alice"}}]}`
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
		}
		// call for details
		if strings.Contains(req.URL.Path, "/pullrequests/1") {
			body := `{"id":1,"title":"PR 1","state":"OPEN","description":"d1","author":{"display_name":"Alice"},"source":{"branch":{"name":"feature"},"repository":{"name":"repo"}},"destination":{"branch":{"name":"main"},"repository":{"name":"repo"}},"reviewers":[{"display_name":"Bob"}]}`
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
		}
		return &http.Response{StatusCode: 404, Body: io.NopCloser(strings.NewReader("not found")), Header: http.Header{}}
	}}
	c.httpClient = &http.Client{Transport: ft}

	prs, err := c.GetPullRequests("repo")
	if err != nil {
		t.Fatalf("GetPullRequests error: %v", err)
	}
	if len(prs) != 1 {
		t.Fatalf("expected 1 PR, got %d", len(prs))
	}
	if prs[0].Title != "PR 1" {
		t.Fatalf("unexpected PR title: %s", prs[0].Title)
	}

	details, err := c.GetPullRequestDetails("repo", 1)
	if err != nil {
		t.Fatalf("GetPullRequestDetails error: %v", err)
	}
	if details.Source.Branch.Name != "feature" || details.Destination.Branch.Name != "main" {
		t.Fatalf("unexpected branches in details: %v -> %v", details.Source.Branch.Name, details.Destination.Branch.Name)
	}
	if len(details.Reviewers) != 1 || details.Reviewers[0].DisplayName != "Bob" {
		t.Fatalf("unexpected reviewers: %+v", details.Reviewers)
	}
}
