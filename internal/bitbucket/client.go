package bitbucket

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"devflow/internal/config"
	"devflow/internal/httpx"
)

type Client struct {
	config      *config.BitbucketConfig
	httpClient  *http.Client
	rateLimiter chan struct{}
	baseURL     string
}

type PullRequest struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	State       string `json:"state"`
	Description string `json:"description"`
	Author      struct {
		DisplayName string `json:"display_name"`
	} `json:"author"`
	Source struct {
		Branch struct {
			Name string `json:"name"`
		} `json:"branch"`
	} `json:"source"`
	Destination struct {
		Branch struct {
			Name string `json:"name"`
		} `json:"branch"`
	} `json:"destination"`
	Links struct {
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
	} `json:"links"`
}

type PullRequestWithReviewers struct {
	ID     int    `json:"id"`
	Title  string `json:"title"`
	State  string `json:"state"`
	Author struct {
		DisplayName string `json:"display_name"`
	} `json:"author"`
	Source struct {
		Branch struct {
			Name string `json:"name"`
		} `json:"branch"`
		Repository struct {
			Name string `json:"name"`
		} `json:"repository"`
	} `json:"source"`
	Destination struct {
		Branch struct {
			Name string `json:"name"`
		} `json:"branch"`
		Repository struct {
			Name string `json:"name"`
		} `json:"repository"`
	} `json:"destination"`
	Reviewers []struct {
		DisplayName string `json:"display_name"`
		UUID        string `json:"uuid"`
	} `json:"reviewers"`
}

type PullRequestsWithReviewersResponse struct {
	Values []PullRequestWithReviewers `json:"values"`
}

type Repository struct {
	UUID        string `json:"uuid"`
	Name        string `json:"name"`
	FullName    string `json:"full_name"`
	Description string `json:"description"`
	IsPrivate   bool   `json:"is_private"`
	Language    string `json:"language"`
	CreatedOn   string `json:"created_on"`
	UpdatedOn   string `json:"updated_on"`
	Size        int64  `json:"size"`
	MainBranch  struct {
		Name string `json:"name"`
	} `json:"mainbranch"`
}

type RepositoriesResponse struct {
	Values []Repository `json:"values"`
	Next   string       `json:"next"`
	Size   int          `json:"size"`
}

type PullRequestDetails struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	State       string `json:"state"`
	Description string `json:"description"`
	CreatedOn   string `json:"created_on"`
	UpdatedOn   string `json:"updated_on"`
	Author      struct {
		DisplayName string `json:"display_name"`
	} `json:"author"`
	Source struct {
		Branch struct {
			Name string `json:"name"`
		} `json:"branch"`
		Repository struct {
			Name string `json:"name"`
		} `json:"repository"`
	} `json:"source"`
	Destination struct {
		Branch struct {
			Name string `json:"name"`
		} `json:"branch"`
		Repository struct {
			Name string `json:"name"`
		} `json:"repository"`
	} `json:"destination"`
	Reviewers []struct {
		DisplayName string `json:"display_name"`
	} `json:"reviewers"`
}

type PullRequestsResponse struct {
	Values []PullRequest `json:"values"`
	Next   string        `json:"next"`
}

func NewClient(cfg *config.BitbucketConfig) *Client {
	// Create a rate limiter that allows 2 requests per second (more conservative)
	rateLimiter := make(chan struct{}, 2)
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond) // 2 requests per second = 1 request every 500ms
		defer ticker.Stop()
		for range ticker.C {
			select {
			case rateLimiter <- struct{}{}:
			default:
			}
		}
	}()

	return &Client{
		config:      cfg,
		httpClient:  httpx.NewClient(30 * time.Second),
		rateLimiter: rateLimiter,
		baseURL:     "https://api.bitbucket.org/2.0",
	}
}

func (c *Client) makeRequest(method, endpoint string, body interface{}) (*http.Response, error) {
	return c.makeRequestWithRetry(method, endpoint, body, 3)
}

func (c *Client) makeRequestWithRetry(method, endpoint string, body interface{}, maxRetries int) (*http.Response, error) {
	url := fmt.Sprintf("%s/%s", c.baseURL, endpoint)

	for attempt := 0; attempt < maxRetries; attempt++ {
		// Rate limiting
		if c.rateLimiter != nil {
			<-c.rateLimiter
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

		// Authentication selection:
		// - If a username (email) is configured, prefer Basic (personal API token)
		// - If no username, assume a resource access token and use Bearer
		// No automatic fallback to avoid masking misconfiguration.
		httpx.ApplyAuth(req, c.config.Username, c.config.Token)
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to make request: %w", err)
		}

		// Handle rate limiting
		if resp.StatusCode == 429 {
			if err := resp.Body.Close(); err != nil {
				fmt.Printf("warning: failed to close response body: %v\n", err)
			}
			if attempt < maxRetries-1 {
				// Exponential backoff: wait longer between retries
				waitTime := time.Duration(attempt+1) * 2 * time.Second
				fmt.Printf("Rate limited, waiting %v before retry %d/%d...\n", waitTime, attempt+1, maxRetries-1)
				time.Sleep(waitTime)
				continue
			}
			return nil, fmt.Errorf("rate limit exceeded after %d retries", maxRetries)
		}

		// Return early on HTTP errors (leave body for caller to inspect)
		if resp.StatusCode >= 400 {
			return resp, nil
		}

		return resp, nil
	}

	return nil, fmt.Errorf("max retries exceeded")
}

// TestAuth tests basic authentication with a simple API call
func (c *Client) TestAuth() error {
	// Try endpoints that match the user's scopes
	endpoints := []string{"workspaces"}
	if c.config.Workspace != "" {
		endpoints = append(endpoints, fmt.Sprintf("repositories/%s", c.config.Workspace))
	}

	for _, endpoint := range endpoints {
		resp, err := c.makeRequest("GET", endpoint, nil)
		if err != nil {
			continue
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				fmt.Printf("warning: failed to close response body: %v\n", err)
			}
		}()

		if resp.StatusCode == http.StatusOK {
			return nil
		}
	}

	return fmt.Errorf("authentication test failed")
}

// TestBasicAuth tests authentication using Basic auth instead of Bearer
func (c *Client) TestBasicAuth() error {
	// Create a separate request with Basic auth
	url := fmt.Sprintf("%s/workspaces", c.baseURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}

	// Use Basic auth instead of Bearer
	httpx.ApplyBasicAuth(req, c.config.Username, c.config.Token)
	req.Header.Set("Accept", "application/json")

	client := c.httpClient
	if client == nil {
		client = httpx.NewClient(30 * time.Second)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer func() {
		defer func() {
			if err := resp.Body.Close(); err != nil {
				fmt.Printf("warning: failed to close response body: %v\n", err)
			}
		}()
	}()

	if resp.StatusCode == http.StatusOK {
		return nil
	}

	return fmt.Errorf("basic auth test failed with status: %d", resp.StatusCode)
}
