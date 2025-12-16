package jira

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"devflow/internal/config"
)

type Client struct {
	config     *config.JiraConfig
	httpClient *http.Client
}

type Issue struct {
	Key    string `json:"key"`
	Fields struct {
		Summary     string      `json:"summary"`
		Description interface{} `json:"description"`
		Status      struct {
			Name string `json:"name"`
		} `json:"status"`
		Assignee struct {
			DisplayName string `json:"displayName"`
		} `json:"assignee"`
		Priority struct {
			Name string `json:"name"`
		} `json:"priority"`
		Sprint  interface{} `json:"sprint"`
		Updated string      `json:"updated"`
		Created string      `json:"created"`
	} `json:"fields"`
}

type SearchResponse struct {
	Issues        []Issue `json:"issues"`
	StartAt       int     `json:"startAt"`
	MaxResults    int     `json:"maxResults"`
	Total         int     `json:"total"`
	NextPageToken string  `json:"nextPageToken"`
	IsLast        bool    `json:"isLast"`
}

// IssueDetails represents detailed information about a Jira issue
type IssueDetails struct {
	Key    string `json:"key"`
	Fields struct {
		Summary     string      `json:"summary"`
		Description interface{} `json:"description"`
		Status      struct {
			Name string `json:"name"`
		} `json:"status"`
		Priority struct {
			Name string `json:"name"`
		} `json:"priority"`
		Assignee struct {
			DisplayName string `json:"displayName"`
		} `json:"assignee"`
		Reporter struct {
			DisplayName string `json:"displayName"`
		} `json:"reporter"`
		Created string `json:"created"`
		Updated string `json:"updated"`
		Comment struct {
			Comments []Comment `json:"comments"`
		} `json:"comment"`
		Attachment   []Attachment `json:"attachment"`
		TeamAssigned struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"customfield_11887"`
	} `json:"fields"`
}

// Comment represents a Jira comment
type Comment struct {
	Author struct {
		DisplayName string `json:"displayName"`
	} `json:"author"`
	Body    interface{} `json:"body"`
	Created string      `json:"created"`
	Updated string      `json:"updated"`
}

// Attachment represents a Jira attachment
type Attachment struct {
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
	Created  string `json:"created"`
}

func NewClient(cfg *config.JiraConfig) *Client {
	return &Client{
		config:     cfg,
		httpClient: &http.Client{},
	}
}

func (c *Client) makeRequest(method, endpoint string, body interface{}) (*http.Response, error) {
	url := fmt.Sprintf("%s/rest/api/3/%s", c.config.URL, endpoint)

	// Debug: print full URL if enabled
	if os.Getenv("DEVFLOW_DEBUG") == "1" || strings.ToLower(os.Getenv("DEVFLOW_DEBUG")) == "true" {
		log.Printf("Jira request: %s %s", method, url)
	}

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.config.Username, c.config.Token)
	req.Header.Set("Content-Type", "application/json")

	return c.httpClient.Do(req)
}

// Search performs a Jira search using either raw JQL or a free-text query.
// If isJQL is true, the provided query is used as JQL directly. Otherwise
// the query is treated as free text and converted to a `text ~ "..."` JQL.
func (c *Client) Search(query string, isJQL bool, maxResults int, startAtArg int) ([]Issue, error) {
	// Backwards compatible: if fetchAll behavior is desired, callers should use SearchAll.
	var jql string
	if isJQL {
		jql = query
		// If the provided raw JQL does not include an ORDER BY clause, append a default
		// to ensure deterministic paging. This helps avoid repeated/overlapping pages.
		if !strings.Contains(strings.ToLower(jql), "order by") {
			jql = strings.TrimSpace(jql) + " ORDER BY updated DESC"
		}
	} else {
		// Protect empty query
		trimmed := strings.TrimSpace(query)
		if trimmed == "" {
			// default to issues assigned to current user
			jql = "assignee = currentUser() ORDER BY updated DESC"
		} else {
			jql = fmt.Sprintf("text ~ \"%s\" ORDER BY updated DESC", trimmed)
		}
	}

	encodedJQL := url.QueryEscape(jql)
	baseEndpoint := fmt.Sprintf("search/jql?jql=%s&fields=key,summary,description,status,assignee,priority,sprint", encodedJQL)

	// If maxResults <= 0, behave as before: single request leaving server to use its default
	if maxResults <= 0 {
		endpoint := baseEndpoint
		resp, err := c.makeRequest("GET", endpoint, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to make request: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("API request failed with status: %d, response: %s", resp.StatusCode, string(body))
		}

		var searchResp SearchResponse
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}
		if os.Getenv("DEVFLOW_DEBUG") == "1" || strings.ToLower(os.Getenv("DEVFLOW_DEBUG")) == "true" {
			// Print raw response body (truncated)
			bodyStr := string(data)
			if len(bodyStr) > 2000 {
				bodyStr = bodyStr[:2000] + "..."
			}
			log.Printf("Jira raw response body: %s", bodyStr)
		}
		if err := json.Unmarshal(data, &searchResp); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		// Debug: print paging info
		if os.Getenv("DEVFLOW_DEBUG") == "1" || strings.ToLower(os.Getenv("DEVFLOW_DEBUG")) == "true" {
			log.Printf("Jira response paging: startAt=%d maxResults=%d total=%d issues=%d", searchResp.StartAt, searchResp.MaxResults, searchResp.Total, len(searchResp.Issues))
		}

		return searchResp.Issues, nil

	}

	// When maxResults > 0, implement paging to retrieve up to maxResults items
	collected := make([]Issue, 0)
	startAt := startAtArg
	perPage := maxResults

	// Cap for fallback size to avoid huge single requests when server omits metadata.
	const fallbackCap = 500

	// Some Jira servers may have a hard upper limit; request in chunks of at most perPage
	for {
		endpoint := fmt.Sprintf("%s&maxResults=%d&startAt=%d", baseEndpoint, perPage, startAt)
		resp, err := c.makeRequest("GET", endpoint, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to make request: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("API request failed with status: %d, response: %s", resp.StatusCode, string(body))
		}

		var searchResp SearchResponse
		if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		// Debug: print paging info
		if os.Getenv("DEVFLOW_DEBUG") == "1" || strings.ToLower(os.Getenv("DEVFLOW_DEBUG")) == "true" {
			log.Printf("Jira response paging: startAt=%d maxResults=%d total=%d issues=%d nextPageToken=%q isLast=%v", searchResp.StartAt, searchResp.MaxResults, searchResp.Total, len(searchResp.Issues), searchResp.NextPageToken, searchResp.IsLast)
		}

		returned := len(searchResp.Issues)
		if returned > 0 {
			needed := maxResults - len(collected)
			if needed <= 0 {
				break
			}
			if returned <= needed {
				collected = append(collected, searchResp.Issues...)
			} else {
				collected = append(collected, searchResp.Issues[:needed]...)
			}
		}

		// If the server provides a nextPageToken, use token-based pagination (prefer token over startAt)
		if searchResp.NextPageToken != "" {
			// Loop using the token until we either have enough results, hit isLast, or token is empty
			token := searchResp.NextPageToken
			for token != "" && !searchResp.IsLast && len(collected) < maxResults {
				if os.Getenv("DEVFLOW_DEBUG") == "1" || strings.ToLower(os.Getenv("DEVFLOW_DEBUG")) == "true" {
					log.Printf("Jira token-based paging: requesting next page with token=%s", token)
				}
				// Build endpoint with token (GET style)
				endpoint := fmt.Sprintf("%s&maxResults=%d&pageToken=%s", baseEndpoint, perPage, url.QueryEscape(token))
				resp, err := c.makeRequest("GET", endpoint, nil)
				if err != nil {
					return nil, fmt.Errorf("failed to make token-based request: %w", err)
				}
				defer resp.Body.Close()
				if resp.StatusCode != http.StatusOK {
					body, _ := io.ReadAll(resp.Body)
					return nil, fmt.Errorf("API request failed (token) with status: %d, response: %s", resp.StatusCode, string(body))
				}
				var tr SearchResponse
				if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
					return nil, fmt.Errorf("failed to decode token response: %w", err)
				}
				// Debug
				if os.Getenv("DEVFLOW_DEBUG") == "1" || strings.ToLower(os.Getenv("DEVFLOW_DEBUG")) == "true" {
					log.Printf("Jira token response: nextPageToken=%q isLast=%v issues=%d", tr.NextPageToken, tr.IsLast, len(tr.Issues))
				}
				// Append items (respecting maxResults)
				if len(tr.Issues) > 0 {
					needed := maxResults - len(collected)
					if needed <= 0 {
						break
					}
					if len(tr.Issues) <= needed {
						collected = append(collected, tr.Issues...)
					} else {
						collected = append(collected, tr.Issues[:needed]...)
					}
				}
				// Prepare next iteration
				token = tr.NextPageToken
				searchResp.IsLast = tr.IsLast
			}
			// After token loop return collected (we paged to satisfy the request)
			if len(collected) > maxResults {
				collected = collected[:maxResults]
			}
			return collected, nil
		}

		// If server didn't provide paging metadata (total/maxResults==0) but returned issues,
		// attempt a fallback for requested pages > 0: request a larger page at startAt=0 and slice client-side.
		if (searchResp.Total == 0 || searchResp.MaxResults == 0) && startAtArg > 0 {
			if os.Getenv("DEVFLOW_DEBUG") == "1" || strings.ToLower(os.Getenv("DEVFLOW_DEBUG")) == "true" {
				log.Printf("Jira server omitted paging metadata; attempting client-side paging fallback")
			}
			// Try to fetch up to startAtArg+perPage items from the server in a single request
			fallbackSize := startAtArg + perPage
			if fallbackSize > fallbackCap {
				return nil, fmt.Errorf("requested page would require a large fallback of %d items which exceeds the cap of %d; please choose a smaller page or increase --max-results accordingly", fallbackSize, fallbackCap)
			}
			fallbackEndpoint := fmt.Sprintf("%s&maxResults=%d&startAt=%d", baseEndpoint, fallbackSize, 0)
			fbResp, err := c.makeRequest("GET", fallbackEndpoint, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to make fallback request: %w", err)
			}
			defer fbResp.Body.Close()
			if fbResp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(fbResp.Body)
				return nil, fmt.Errorf("API request failed (fallback) with status: %d, response: %s", fbResp.StatusCode, string(body))
			}
			var fbSearchResp SearchResponse
			fbData, err := io.ReadAll(fbResp.Body)
			if err != nil {
				return nil, fmt.Errorf("failed to read fallback response body: %w", err)
			}
			if os.Getenv("DEVFLOW_DEBUG") == "1" || strings.ToLower(os.Getenv("DEVFLOW_DEBUG")) == "true" {
				bodyStr := string(fbData)
				if len(bodyStr) > 2000 {
					bodyStr = bodyStr[:2000] + "..."
				}
				log.Printf("Jira raw fallback response body: %s", bodyStr)
			}
			if err := json.Unmarshal(fbData, &fbSearchResp); err != nil {
				return nil, fmt.Errorf("failed to decode fallback response: %w", err)
			}
			// Compute slice range
			start := startAtArg
			end := startAtArg + perPage
			if start >= len(fbSearchResp.Issues) {
				// fallback response doesn't contain enough items to reach requested page
				return nil, fmt.Errorf("fallback response contained %d issues which is insufficient for requested startAt %d", len(fbSearchResp.Issues), startAtArg)
			}
			if end > len(fbSearchResp.Issues) {
				end = len(fbSearchResp.Issues)
			}
			pageSlice := fbSearchResp.Issues[start:end]
			// Replace collected with the slice (since fallback was intended to return this page)
			collected = append([]Issue{}, pageSlice...)
			return collected, nil
		}

		// Determine if we should continue using the actual number of issues returned
		if searchResp.StartAt+returned >= searchResp.Total {
			break
		}
		// Safety: if server returns 0 issues, avoid infinite loop
		if returned == 0 {
			break
		}
		startAt = searchResp.StartAt + returned
		// Another safety: if collected reached requested maxResults, stop
		if len(collected) >= maxResults {
			break
		}
	}

	return collected, nil
}

// GetMyIssues retrieves issues assigned to the current user
func (c *Client) GetMyIssues() ([]Issue, error) {
	// Default to single-page search with server default paging
	return c.Search("", false, 0, 0)
}

// SearchAll retrieves issues following token-based or startAt pagination until completion
// It respects maxResultsPerPage if >0; if maxTotal <= 0 it will fetch all available issues.
func (c *Client) SearchAll(query string, isJQL bool, maxResultsPerPage int, maxTotal int) ([]Issue, error) {
	collected := make([]Issue, 0)
	seen := make(map[string]struct{})

	// Helper to append unique issues
	appendUnique := func(items []Issue) {
		for _, it := range items {
			if _, ok := seen[it.Key]; ok {
				continue
			}
			seen[it.Key] = struct{}{}
			collected = append(collected, it)
		}
	}

	// First request uses startAt=0 (Search already handles building JQL)
	issues, err := c.Search(query, isJQL, maxResultsPerPage, 0)
	if err != nil {
		return nil, err
	}
	appendUnique(issues)
	if maxTotal > 0 && len(collected) >= maxTotal {
		return collected[:maxTotal], nil
	}

	// Attempt token-based continuation if available
	// Make an initial lightweight request to get the first response with tokens if needed
	// We'll reuse Search flow by making direct calls similar to Search's token loop.
	// Build base JQL and endpoint pieces similar to Search
	var jql string
	if isJQL {
		jql = query
		if !strings.Contains(strings.ToLower(jql), "order by") {
			jql = strings.TrimSpace(jql) + " ORDER BY updated DESC"
		}
	} else {
		trimmed := strings.TrimSpace(query)
		if trimmed == "" {
			jql = "assignee = currentUser() ORDER BY updated DESC"
		} else {
			jql = fmt.Sprintf("text ~ \"%s\" ORDER BY updated DESC", trimmed)
		}
	}
	encodedJQL := url.QueryEscape(jql)
	baseEndpoint := fmt.Sprintf("search/jql?jql=%s&fields=key,summary,description,status,assignee,priority,sprint", encodedJQL)

	// Try token-based paging first by requesting with maxResultsPerPage and seeing if NextPageToken appears
	token := ""
	for {
		endpoint := baseEndpoint
		if maxResultsPerPage > 0 {
			endpoint = fmt.Sprintf("%s&maxResults=%d", baseEndpoint, maxResultsPerPage)
		}
		if token != "" {
			endpoint = fmt.Sprintf("%s&pageToken=%s", endpoint, url.QueryEscape(token))
		}
		resp, err := c.makeRequest("GET", endpoint, nil)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("API request failed with status: %d, response: %s", resp.StatusCode, string(body))
		}
		var sr SearchResponse
		if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
			return nil, err
		}
		appendUnique(sr.Issues)
		if maxTotal > 0 && len(collected) >= maxTotal {
			return collected[:maxTotal], nil
		}
		if sr.IsLast || sr.NextPageToken == "" {
			break
		}
		// Continue with token
		token = sr.NextPageToken
	}

	return collected, nil
}

// GetIssueDetails retrieves detailed information about a specific issue
func (c *Client) GetIssueDetails(issueKey string) (*IssueDetails, error) {
	endpoint := fmt.Sprintf("issue/%s?fields=summary,description,status,priority,assignee,reporter,created,updated,comment,attachment,customfield_11887", issueKey)

	resp, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status: %d, response: %s", resp.StatusCode, string(body))
	}

	var issue IssueDetails
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &issue, nil
}

// CreateIssueOptions holds optional fields for issue creation
type CreateIssueOptions struct {
	ProjectKey  string
	Summary     string
	Description string
	IssueType   string
	Priority    string
	Assignee    string
	Labels      []string
	Epic        string
	StoryPoints float64
	Sprint      string
	Team        string
}

// CreateIssue creates a new Jira issue with extended options
func (c *Client) CreateIssue(opts CreateIssueOptions) (*Issue, error) {
	endpoint := "issue"

	if opts.IssueType == "" {
		opts.IssueType = "Task"
	}

	toADF := func(text string) interface{} {
		if strings.TrimSpace(text) == "" {
			return map[string]interface{}{
				"type":    "doc",
				"version": 1,
				"content": []interface{}{map[string]interface{}{"type": "paragraph", "content": []interface{}{}}},
			}
		}
		lines := strings.Split(text, "\n")
		paragraphs := make([]interface{}, 0, len(lines))
		for _, line := range lines {
			line = strings.TrimRight(line, "\r")
			if line == "" {
				paragraphs = append(paragraphs, map[string]interface{}{"type": "paragraph", "content": []interface{}{}})
				continue
			}
			paragraphs = append(paragraphs, map[string]interface{}{
				"type":    "paragraph",
				"content": []interface{}{map[string]interface{}{"type": "text", "text": line}},
			})
		}
		return map[string]interface{}{"type": "doc", "version": 1, "content": paragraphs}
	}

	fields := map[string]interface{}{
		"project":     map[string]string{"key": opts.ProjectKey},
		"summary":     opts.Summary,
		"description": toADF(opts.Description),
		"issuetype":   map[string]string{"name": opts.IssueType},
	}

	if opts.Priority != "" {
		fields["priority"] = map[string]string{"name": opts.Priority}
	}
	if len(opts.Labels) > 0 {
		fields["labels"] = opts.Labels
	}
	if opts.Assignee != "" {
		fields["assignee"] = map[string]string{"name": opts.Assignee}
	}
	if opts.StoryPoints > 0 {
		fields["customfield_10016"] = opts.StoryPoints
	}
	if opts.Epic != "" {
		fields["customfield_10014"] = opts.Epic
	}
	if opts.Sprint != "" {
		fields["customfield_10020"] = opts.Sprint
	}
	if opts.Team != "" {
		// Team Assigned usually expects an object with id or name. Accept numeric or string.
		if _, err := strconv.Atoi(opts.Team); err == nil {
			fields["customfield_11887"] = map[string]string{"id": opts.Team}
		} else {
			fields["customfield_11887"] = map[string]string{"name": opts.Team}
		}
	}

	attempt := func(f map[string]interface{}) (*http.Response, []byte, error) {
		body := map[string]interface{}{"fields": f}
		resp, err := c.makeRequest("POST", endpoint, body)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to make request: %w", err)
		}
		data, _ := io.ReadAll(resp.Body)
		return resp, data, nil
	}

	resp, data, err := attempt(fields)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusBadRequest {
		var errPayload struct {
			Errors map[string]string `json:"errors"`
		}
		_ = json.Unmarshal(data, &errPayload)

		removed := []string{}
		for _, cf := range []string{"customfield_10014", "customfield_10016", "customfield_10020", "customfield_11887"} {
			if msg, bad := errPayload.Errors[cf]; bad && msg != "" {
				if _, present := fields[cf]; present {
					delete(fields, cf)
					removed = append(removed, cf)
				}
			}
		}
		if len(removed) > 0 {
			resp.Body.Close()
			resp2, data2, err2 := attempt(fields)
			if err2 != nil {
				return nil, err2
			}
			defer resp2.Body.Close()
			if resp2.StatusCode != http.StatusCreated {
				return nil, fmt.Errorf("API request failed (after retry) with status: %d, body: %s", resp2.StatusCode, string(data2))
			}
			var issue Issue
			if err := json.Unmarshal(data2, &issue); err != nil {
				return nil, fmt.Errorf("failed to decode response: %w", err)
			}
			fmt.Printf("Warning: omitted unsupported custom fields: %v\n", removed)
			return &issue, nil
		}
	}

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("API request failed with status: %d, body: %s", resp.StatusCode, string(data))
	}

	var issue Issue
	if err := json.Unmarshal(data, &issue); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &issue, nil
}

// AddComment adds a comment to an issue
func (c *Client) AddComment(issueKey, body string) error {
	endpoint := fmt.Sprintf("issue/%s/comment", issueKey)
	payload := map[string]interface{}{
		"body": map[string]interface{}{
			"type":    "doc",
			"version": 1,
			"content": []interface{}{map[string]interface{}{
				"type":    "paragraph",
				"content": []interface{}{map[string]interface{}{"type": "text", "text": body}},
			}},
		},
	}
	resp, err := c.makeRequest("POST", endpoint, payload)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status: %d, body: %s", resp.StatusCode, string(data))
	}
	return nil
}

// AddRemoteLink adds a remote link to an issue
func (c *Client) AddRemoteLink(issueKey, linkURL, title, summary string) error {
	endpoint := fmt.Sprintf("issue/%s/remotelink", issueKey)
	obj := map[string]interface{}{
		"url": linkURL,
	}
	if title != "" {
		obj["title"] = title
	}
	if summary != "" {
		obj["summary"] = summary
	}
	payload := map[string]interface{}{
		"object": obj,
	}
	resp, err := c.makeRequest("POST", endpoint, payload)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status: %d, body: %s", resp.StatusCode, string(data))
	}
	return nil
}

// FindMentions searches for issues where the current user is mentioned
func (c *Client) FindMentions() ([]Issue, error) {
	query := fmt.Sprintf("text ~ \"%s\" ORDER BY updated DESC", c.config.Username)
	// Use Search with isJQL=true because query already contains JQL syntax
	issues, err := c.Search(query, true, 50, 0)
	if err != nil {
		return nil, err
	}
	return issues, nil
}

// UpdateIssue updates the specified fields on an existing issue.
// The caller should pass a map where keys are Jira field keys (e.g., "summary",
// "description", "priority", "assignee", "labels", "customfield_10016", etc.).
func (c *Client) UpdateIssue(issueKey string, fields map[string]interface{}) error {
	if len(fields) == 0 {
		return nil
	}

	// Convert description from plain string to ADF if present and is a string
	if d, ok := fields["description"].(string); ok {
		if strings.TrimSpace(d) == "" {
			fields["description"] = map[string]interface{}{
				"type":    "doc",
				"version": 1,
				"content": []interface{}{map[string]interface{}{"type": "paragraph", "content": []interface{}{}}},
			}
		} else {
			lines := strings.Split(d, "\n")
			paragraphs := make([]interface{}, 0, len(lines))
			for _, line := range lines {
				line = strings.TrimRight(line, "\r")
				if line == "" {
					paragraphs = append(paragraphs, map[string]interface{}{"type": "paragraph", "content": []interface{}{}})
					continue
				}
				paragraphs = append(paragraphs, map[string]interface{}{
					"type":    "paragraph",
					"content": []interface{}{map[string]interface{}{"type": "text", "text": line}},
				})
			}
			fields["description"] = map[string]interface{}{"type": "doc", "version": 1, "content": paragraphs}
		}
	}

	endpoint := fmt.Sprintf("issue/%s", issueKey)
	body := map[string]interface{}{"fields": fields}

	resp, err := c.makeRequest("PUT", endpoint, body)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Jira returns 204 No Content on success for update
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status: %d, body: %s", resp.StatusCode, string(data))
	}
	return nil
}
