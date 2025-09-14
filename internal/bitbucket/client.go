package bitbucket

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"devflow/internal/config"
)

type Client struct {
	config      *config.BitbucketConfig
	httpClient  *http.Client
	rateLimiter chan struct{}
}

type PullRequest struct {
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
	} `json:"source"`
	Destination struct {
		Branch struct {
			Name string `json:"name"`
		} `json:"branch"`
	} `json:"destination"`
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
	Name        string `json:"name"`
	FullName    string `json:"full_name"`
	Description string `json:"description"`
	IsPrivate   bool   `json:"is_private"`
	Language    string `json:"language"`
	CreatedOn   string `json:"created_on"`
	UpdatedOn   string `json:"updated_on"`
	Size        int64  `json:"size"`
}

type RepositoriesResponse struct {
	Values []Repository `json:"values"`
	Next   string       `json:"next"`
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
		httpClient:  &http.Client{},
		rateLimiter: rateLimiter,
	}
}

func (c *Client) makeRequest(method, endpoint string, body interface{}) (*http.Response, error) {
	return c.makeRequestWithRetry(method, endpoint, body, 3)
}

func (c *Client) makeRequestWithRetry(method, endpoint string, body interface{}, maxRetries int) (*http.Response, error) {
	url := fmt.Sprintf("https://api.bitbucket.org/2.0/%s", endpoint)

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

		// Use Basic auth since Bearer auth is giving 401
		req.SetBasicAuth(c.config.Username, c.config.Token)
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to make request: %w", err)
		}

		// Handle rate limiting
		if resp.StatusCode == 429 {
			resp.Body.Close()
			if attempt < maxRetries-1 {
				// Exponential backoff: wait longer between retries
				waitTime := time.Duration(attempt+1) * 2 * time.Second
				fmt.Printf("Rate limited, waiting %v before retry %d/%d...\n", waitTime, attempt+1, maxRetries-1)
				time.Sleep(waitTime)
				continue
			}
			return nil, fmt.Errorf("rate limit exceeded after %d retries", maxRetries)
		}

		// For other errors, don't retry
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
	endpoints := []string{"workspaces", fmt.Sprintf("repositories/%s", c.config.Workspace)}

	for _, endpoint := range endpoints {
		resp, err := c.makeRequest("GET", endpoint, nil)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			return nil
		}
	}

	return fmt.Errorf("authentication test failed")
}

// TestBasicAuth tests authentication using Basic auth instead of Bearer
func (c *Client) TestBasicAuth() error {
	// Create a separate request with Basic auth
	url := "https://api.bitbucket.org/2.0/workspaces"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}

	// Use Basic auth instead of Bearer
	req.SetBasicAuth(c.config.Username, c.config.Token)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil
	}

	return fmt.Errorf("basic auth test failed with status: %d", resp.StatusCode)
}

// GetPullRequests retrieves pull requests for a repository
func (c *Client) GetPullRequests(repoSlug string) ([]PullRequest, error) {
	endpoint := fmt.Sprintf("repositories/%s/%s/pullrequests", c.config.Workspace, repoSlug)

	resp, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status: %d, response: %s", resp.StatusCode, string(body))
	}

	var prResp PullRequestsResponse
	if err := json.NewDecoder(resp.Body).Decode(&prResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return prResp.Values, nil
}

// GetPullRequestsWithReviewers retrieves pull requests with reviewer information for a specific repository
func (c *Client) GetPullRequestsWithReviewers(repoSlug string) ([]PullRequestWithReviewers, error) {
	// First, get the basic list of PRs
	basicPRs, err := c.GetPullRequests(repoSlug)
	if err != nil {
		return nil, fmt.Errorf("failed to get basic PR list: %w", err)
	}

	if len(basicPRs) == 0 {
		return []PullRequestWithReviewers{}, nil
	}

	// For each PR, fetch detailed information to get reviewers
	var prsWithReviewers []PullRequestWithReviewers
	fmt.Printf("Fetching reviewer details for %d PRs in %s...\n", len(basicPRs), repoSlug)

	for i, basicPR := range basicPRs {
		if i > 0 && i%5 == 0 {
			fmt.Printf("Processed %d/%d PRs...\n", i, len(basicPRs))
		}

		details, err := c.GetPullRequestDetails(repoSlug, basicPR.ID)
		if err != nil {
			// Log error but continue with other PRs
			fmt.Printf("Warning: Failed to get details for PR #%d: %v\n", basicPR.ID, err)
			continue
		}

		// Convert detailed PR to PR with reviewers format
		prWithReviewers := PullRequestWithReviewers{
			ID:    details.ID,
			Title: details.Title,
			State: details.State,
			Author: struct {
				DisplayName string `json:"display_name"`
			}{
				DisplayName: details.Author.DisplayName,
			},
			Source: struct {
				Branch struct {
					Name string `json:"name"`
				} `json:"branch"`
				Repository struct {
					Name string `json:"name"`
				} `json:"repository"`
			}{
				Branch: struct {
					Name string `json:"name"`
				}{
					Name: details.Source.Branch.Name,
				},
				Repository: struct {
					Name string `json:"name"`
				}{
					Name: details.Source.Repository.Name,
				},
			},
			Destination: struct {
				Branch struct {
					Name string `json:"name"`
				} `json:"branch"`
				Repository struct {
					Name string `json:"name"`
				} `json:"repository"`
			}{
				Branch: struct {
					Name string `json:"name"`
				}{
					Name: details.Destination.Branch.Name,
				},
				Repository: struct {
					Name string `json:"name"`
				}{
					Name: details.Destination.Repository.Name,
				},
			},
			Reviewers: func() []struct {
				DisplayName string `json:"display_name"`
				UUID        string `json:"uuid"`
			} {
				var reviewers []struct {
					DisplayName string `json:"display_name"`
					UUID        string `json:"uuid"`
				}
				for _, r := range details.Reviewers {
					reviewers = append(reviewers, struct {
						DisplayName string `json:"display_name"`
						UUID        string `json:"uuid"`
					}{
						DisplayName: r.DisplayName,
						UUID:        "", // UUID might not be available in details
					})
				}
				return reviewers
			}(),
		}

		prsWithReviewers = append(prsWithReviewers, prWithReviewers)
	}

	return prsWithReviewers, nil
}

// GetWorkspacePullRequestsForUser retrieves all PRs where the user is a reviewer across the entire workspace
func (c *Client) GetWorkspacePullRequestsForUser(username string) ([]PullRequestWithReviewers, error) {
	endpoint := fmt.Sprintf("workspaces/%s/pullrequests/%s", c.config.Workspace, username)

	resp, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status: %d, response: %s", resp.StatusCode, string(body))
	}

	var prResp PullRequestsWithReviewersResponse
	if err := json.NewDecoder(resp.Body).Decode(&prResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return prResp.Values, nil
}

// GetPullRequestDetails retrieves detailed information about a specific pull request
func (c *Client) GetPullRequestDetails(repoSlug string, prID int) (*PullRequestDetails, error) {
	endpoint := fmt.Sprintf("repositories/%s/%s/pullrequests/%d", c.config.Workspace, repoSlug, prID)

	resp, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status: %d, response: %s", resp.StatusCode, string(body))
	}

	var pr PullRequestDetails
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &pr, nil
}

// GetRepositories retrieves all repositories in the workspace with pagination support
func (c *Client) GetRepositories() ([]Repository, error) {
	var allRepos []Repository
	endpoint := fmt.Sprintf("repositories/%s?pagelen=100", c.config.Workspace)

	for endpoint != "" {
		resp, err := c.makeRequest("GET", endpoint, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to make request: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("API request failed with status: %d, response: %s", resp.StatusCode, string(body))
		}

		var repoResp RepositoriesResponse
		if err := json.NewDecoder(resp.Body).Decode(&repoResp); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		resp.Body.Close()

		// Add repositories from this page
		allRepos = append(allRepos, repoResp.Values...)

		// Check if there's a next page
		if repoResp.Next != "" {
			// Extract the relative path from the next URL
			// The next URL will be something like: https://api.bitbucket.org/2.0/repositories/workspace?page=2
			// We need to extract just the path part after the base URL
			if idx := strings.Index(repoResp.Next, "/2.0/"); idx != -1 {
				endpoint = repoResp.Next[idx+4:] // Skip "/2.0"
			} else {
				// Fallback: if we can't parse the next URL, stop pagination
				break
			}
		} else {
			// No more pages
			endpoint = ""
		}
	}

	return allRepos, nil
}

// GetRepositoriesPaged retrieves repositories for a specific page with total count
func (c *Client) GetRepositoriesPaged(page, size int) ([]Repository, int, error) {
	// Adjust for 0-based page indexing in API vs 1-based user input
	apiPage := page
	if apiPage < 0 {
		apiPage = 0
	}

	// First, get the first page to determine total count if we haven't cached it
	if apiPage == 0 {
		return c.getFirstPageWithTotal(size)
	}

	// For subsequent pages, we need to get the total count first
	totalCount, err := c.getTotalRepositoryCount(size)
	if err != nil {
		// Fallback to estimating
		totalCount = 1000
	}

	// Now get the specific page
	endpoint := fmt.Sprintf("repositories/%s?page=%d&pagelen=%d", c.config.Workspace, apiPage+1, size)

	resp, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, 0, fmt.Errorf("API request failed with status: %d, response: %s", resp.StatusCode, string(body))
	}

	var repoResp RepositoriesResponse
	if err := json.NewDecoder(resp.Body).Decode(&repoResp); err != nil {
		return nil, 0, fmt.Errorf("failed to decode response: %w", err)
	}

	return repoResp.Values, totalCount, nil
}

// getFirstPageWithTotal gets the first page and calculates total count
func (c *Client) getFirstPageWithTotal(size int) ([]Repository, int, error) {
	endpoint := fmt.Sprintf("repositories/%s?page=1&pagelen=%d", c.config.Workspace, size)

	resp, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, 0, fmt.Errorf("API request failed with status: %d, response: %s", resp.StatusCode, string(body))
	}

	var repoResp RepositoriesResponse
	if err := json.NewDecoder(resp.Body).Decode(&repoResp); err != nil {
		return nil, 0, fmt.Errorf("failed to decode response: %w", err)
	}

	// Calculate total count based on whether there's a next page
	totalCount := len(repoResp.Values)
	if repoResp.Next != "" {
		// Estimate total by making another request to get a better count
		// For now, we'll use a conservative estimate
		totalCount = len(repoResp.Values) * 5 // Rough estimate
	}

	return repoResp.Values, totalCount, nil
}

// getTotalRepositoryCount attempts to get the total repository count
func (c *Client) getTotalRepositoryCount(size int) (int, error) {
	// Make a request with a small page size to count pages
	endpoint := fmt.Sprintf("repositories/%s?page=1&pagelen=1", c.config.Workspace)

	resp, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("failed to get repository count")
	}

	var repoResp RepositoriesResponse
	if err := json.NewDecoder(resp.Body).Decode(&repoResp); err != nil {
		return 0, err
	}

	// If there's no next page, we have the total count
	if repoResp.Next == "" {
		return len(repoResp.Values), nil
	}

	// Estimate based on having a next page
	return 50, nil // Conservative estimate
}

// CreatePullRequest creates a new pull request
func (c *Client) CreatePullRequest(repoSlug, title, sourceBranch, destinationBranch string) (*PullRequest, error) {
	endpoint := fmt.Sprintf("repositories/%s/%s/pullrequests", c.config.Workspace, repoSlug)

	body := map[string]interface{}{
		"title": title,
		"source": map[string]interface{}{
			"branch": map[string]string{
				"name": sourceBranch,
			},
		},
		"destination": map[string]interface{}{
			"branch": map[string]string{
				"name": destinationBranch,
			},
		},
	}

	resp, err := c.makeRequest("POST", endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status: %d, response: %s", resp.StatusCode, string(body))
	}

	var pr PullRequest
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &pr, nil
}
