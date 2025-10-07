package jira

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

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
		Attachment []Attachment `json:"attachment"`
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
	endpoint := fmt.Sprintf("issue/%s?fields=summary,description,status,priority,assignee,reporter,created,updated,comment,attachment", issueKey)

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
}

// CreateIssue creates a new Jira issue with extended options
func (c *Client) CreateIssue(opts CreateIssueOptions) (*Issue, error) {
	endpoint := "issue"

	if opts.IssueType == "" {
		opts.IssueType = "Task"
	}

	fields := map[string]interface{}{
		"project":     map[string]string{"key": opts.ProjectKey},
		"summary":     opts.Summary,
		"description": opts.Description,
		"issuetype":   map[string]string{"name": opts.IssueType},
	}

	if opts.Priority != "" {
		fields["priority"] = map[string]string{"name": opts.Priority}
	}
	if len(opts.Labels) > 0 {
		fields["labels"] = opts.Labels
	}
	// Assignee handling (Jira Cloud now prefers accountId; using name may fail depending on config)
	if opts.Assignee != "" {
		fields["assignee"] = map[string]string{"name": opts.Assignee}
	}
	// Story points custom field is instance specific; common classic customfield_10016
	if opts.StoryPoints > 0 {
		fields["customfield_10016"] = opts.StoryPoints
	}
	// Epic linking varies; placeholder using common classic field customfield_10014
	if opts.Epic != "" {
		fields["customfield_10014"] = opts.Epic
	}
	// Sprint is also custom; commonly customfield_10020
	if opts.Sprint != "" {
		fields["customfield_10020"] = opts.Sprint
	}

	body := map[string]interface{}{"fields": fields}

	resp, err := c.makeRequest("POST", endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		responseBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status: %d, body: %s", resp.StatusCode, string(responseBody))
	}

	var issue Issue
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
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
