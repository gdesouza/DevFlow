package jira

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
	Issues []Issue `json:"issues"`
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

// GetMyIssues retrieves issues assigned to the current user
func (c *Client) GetMyIssues() ([]Issue, error) {
	jql := "assignee = currentUser() ORDER BY updated DESC"
	encodedJQL := url.QueryEscape(jql)
	endpoint := fmt.Sprintf("search/jql?jql=%s&fields=key,summary,description,status,assignee,priority,sprint", encodedJQL)

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

	return searchResp.Issues, nil
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

// FindMentions searches for issues where the current user is mentioned
func (c *Client) FindMentions() ([]Issue, error) {
	jql := fmt.Sprintf("text ~ \"%s\" ORDER BY updated DESC", c.config.Username)
	encodedJQL := url.QueryEscape(jql)
	endpoint := fmt.Sprintf("search/jql?jql=%s&fields=key,summary,description,status,assignee,priority,updated&maxResults=50", encodedJQL)

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

	return searchResp.Issues, nil
}
