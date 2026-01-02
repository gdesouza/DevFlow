package jira

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"devflow/internal/config"
)

// fakeTransport allows tests to inspect requests and return custom responses.
type fakeTransport struct {
	fn func(req *http.Request) *http.Response
}

func (f fakeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return f.fn(req), nil
}

func makeResp(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     map[string][]string{"Content-Type": []string{"application/json"}},
	}
}

func TestNewClient(t *testing.T) {
	cfg := &config.JiraConfig{URL: "http://jira.example", Username: "me", Token: "tok"}
	c := NewClient(cfg)
	if c.config != cfg {
		t.Fatalf("expected config to be set")
	}
	if c.httpClient == nil {
		t.Fatalf("expected httpClient to be non-nil")
	}
}

func TestMakeRequest_SetsAuthAndHeaders(t *testing.T) {
	cfg := &config.JiraConfig{URL: "http://jira.example", Username: "me", Token: "tok"}
	c := NewClient(cfg)
	c.httpClient = &http.Client{Transport: fakeTransport{fn: func(req *http.Request) *http.Response {
		// verify basic auth
		user, pass, ok := req.BasicAuth()
		if !ok || user != "me" || pass != "tok" {
			t.Fatalf("basic auth not set correctly")
		}
		if req.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("content-type header not set")
		}
		return makeResp(200, "{}")
	}}}

	resp, err := c.makeRequest("GET", "status", nil)
	if err != nil {
		t.Fatalf("makeRequest failed: %v", err)
	}
	defer resp.Body.Close()
}

func TestGetMyIssues_Success(t *testing.T) {
	cfg := &config.JiraConfig{URL: "http://jira.example", Username: "me", Token: "tok"}
	c := NewClient(cfg)
	payload := SearchResponse{Issues: []Issue{{Key: "ABC-1"}}}
	b, _ := json.Marshal(payload)
	c.httpClient = &http.Client{Transport: fakeTransport{fn: func(req *http.Request) *http.Response {
		return makeResp(200, string(b))
	}}}

	issues, err := c.GetMyIssues()
	if err != nil {
		t.Fatalf("GetMyIssues failed: %v", err)
	}
	if len(issues) != 1 || issues[0].Key != "ABC-1" {
		t.Fatalf("unexpected issues returned: %+v", issues)
	}
}

func TestGetIssueDetails_Success(t *testing.T) {
	cfg := &config.JiraConfig{URL: "http://jira.example", Username: "me", Token: "tok"}
	c := NewClient(cfg)
	var det IssueDetails
	det.Key = "PRJ-1"
	det.Fields.Summary = "s"
	b, _ := json.Marshal(det)
	c.httpClient = &http.Client{Transport: fakeTransport{fn: func(req *http.Request) *http.Response {
		return makeResp(200, string(b))
	}}}

	got, err := c.GetIssueDetails("PRJ-1")
	if err != nil {
		t.Fatalf("GetIssueDetails failed: %v", err)
	}
	if got.Key != "PRJ-1" {
		t.Fatalf("unexpected key: %s", got.Key)
	}
}

func TestCreateIssue_SuccessAndRetry(t *testing.T) {
	cfg := &config.JiraConfig{URL: "http://jira.example", Username: "me", Token: "tok"}
	c := NewClient(cfg)

	// Counter to simulate first-call bad request then success on retry when custom fields removed.
	call := 0
	c.httpClient = &http.Client{Transport: fakeTransport{fn: func(req *http.Request) *http.Response {
		call++
		body, _ := io.ReadAll(req.Body)
		bs := string(body)
		// First call: contains custom fields -> return 400 with errors
		if call == 1 && strings.Contains(bs, "customfield_10014") {
			errPayload := map[string]map[string]string{"errors": map[string]string{"customfield_10014": "unsupported"}}
			b, _ := json.Marshal(errPayload)
			return makeResp(400, string(b))
		}
		// Otherwise, return created
		created := Issue{Key: "CRE-1"}
		b, _ := json.Marshal(created)
		return &http.Response{StatusCode: 201, Body: io.NopCloser(bytes.NewBuffer(b)), Header: map[string][]string{"Content-Type": []string{"application/json"}}}
	}}}

	opts := CreateIssueOptions{
		ProjectKey:  "PRJ",
		Summary:     "sum",
		Description: "line1\n\nline3",
		IssueType:   "Task",
		Priority:    "High",
		Assignee:    "user",
		Labels:      []string{"a", "b"},
		StoryPoints: 3.5,
		Epic:        "EPIC-1",
		Sprint:      "SPR-1",
		Team:        "42",
	}

	issue, err := c.CreateIssue(opts)
	if err != nil {
		t.Fatalf("CreateIssue failed: %v", err)
	}
	if issue.Key != "CRE-1" {
		t.Fatalf("unexpected issue key: %s", issue.Key)
	}
	if call < 1 {
		t.Fatalf("expected at least one request")
	}
}

func TestAddCommentAndRemoteLink(t *testing.T) {
	cfg := &config.JiraConfig{URL: "http://jira.example", Username: "me", Token: "tok"}
	c := NewClient(cfg)
	c.httpClient = &http.Client{Transport: fakeTransport{fn: func(req *http.Request) *http.Response {
		if strings.Contains(req.URL.Path, "comment") {
			return makeResp(201, "{}")
		}
		if strings.Contains(req.URL.Path, "remotelink") {
			return makeResp(200, "{}")
		}
		return makeResp(400, "{\"error\":\"bad\"}")
	}}}

	if err := c.AddComment("PRJ-1", "hi"); err != nil {
		t.Fatalf("AddComment failed: %v", err)
	}
	if err := c.AddRemoteLink("PRJ-1", "http://x", "t", "s"); err != nil {
		t.Fatalf("AddRemoteLink failed: %v", err)
	}
}

func TestFindMentions_Success(t *testing.T) {
	cfg := &config.JiraConfig{URL: "http://jira.example", Username: "me", Token: "tok"}
	c := NewClient(cfg)
	payload := SearchResponse{Issues: []Issue{{Key: "ABC-2"}}}
	b, _ := json.Marshal(payload)
	c.httpClient = &http.Client{Transport: fakeTransport{fn: func(req *http.Request) *http.Response {
		return makeResp(200, string(b))
	}}}

	issues, err := c.FindMentions()
	if err != nil {
		t.Fatalf("FindMentions failed: %v", err)
	}
	if len(issues) != 1 || issues[0].Key != "ABC-2" {
		t.Fatalf("unexpected issues: %+v", issues)
	}
}

func TestListProjects_Success(t *testing.T) {
	cfg := &config.JiraConfig{URL: "http://jira.example", Username: "me", Token: "tok"}
	c := NewClient(cfg)
	projects := []Project{
		{ID: "10000", Key: "PROJ", Name: "My Project"},
		{ID: "10001", Key: "TEST", Name: "Test Project"},
	}
	projects[0].Lead.DisplayName = "John Doe"
	b, _ := json.Marshal(projects)
	c.httpClient = &http.Client{Transport: fakeTransport{fn: func(req *http.Request) *http.Response {
		// Verify the endpoint
		if !strings.Contains(req.URL.Path, "project") {
			t.Fatalf("unexpected endpoint: %s", req.URL.Path)
		}
		return makeResp(200, string(b))
	}}}

	result, err := c.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects failed: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(result))
	}
	if result[0].Key != "PROJ" || result[0].Name != "My Project" {
		t.Fatalf("unexpected first project: %+v", result[0])
	}
	if result[0].Lead.DisplayName != "John Doe" {
		t.Fatalf("unexpected lead: %s", result[0].Lead.DisplayName)
	}
}

func TestListProjects_APIError(t *testing.T) {
	cfg := &config.JiraConfig{URL: "http://jira.example", Username: "me", Token: "tok"}
	c := NewClient(cfg)
	c.httpClient = &http.Client{Transport: fakeTransport{fn: func(req *http.Request) *http.Response {
		return makeResp(403, `{"errorMessages":["Forbidden"]}`)
	}}}

	_, err := c.ListProjects()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "403") {
		t.Fatalf("expected 403 in error, got: %v", err)
	}
}
