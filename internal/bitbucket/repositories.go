package bitbucket

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// GetRepositories retrieves all repositories in the workspace with pagination support.
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
			if err := resp.Body.Close(); err != nil {
				fmt.Printf("warning: failed to close response body: %v\n", err)
			}
			return nil, fmt.Errorf("API request failed with status: %d, response: %s", resp.StatusCode, string(body))
		}

		var repoResp RepositoriesResponse
		if err := json.NewDecoder(resp.Body).Decode(&repoResp); err != nil {
			if err := resp.Body.Close(); err != nil {
				fmt.Printf("warning: failed to close response body: %v\n", err)
			}
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("warning: failed to close response body: %v\n", err)
		}

		allRepos = append(allRepos, repoResp.Values...)

		if repoResp.Next != "" {
			endpoint = strings.TrimPrefix(repoResp.Next, c.baseURL+"/")
			if endpoint == repoResp.Next {
				break
			}
		} else {
			endpoint = ""
		}
	}

	return allRepos, nil
}

// GetRepositoriesPaged retrieves repositories for a specific page with total count.
func (c *Client) GetRepositoriesPaged(page, size int) ([]Repository, int, error) {
	apiPage := page
	if apiPage < 0 {
		apiPage = 0
	}

	if apiPage == 0 {
		return c.getFirstPageWithTotal(size)
	}

	totalCount, err := c.getTotalRepositoryCount(size)
	if err != nil {
		totalCount = 1000
	}

	endpoint := fmt.Sprintf("repositories/%s?page=%d&pagelen=%d", c.config.Workspace, apiPage+1, size)

	resp, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to make request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("warning: failed to close response body: %v\n", err)
		}
	}()

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

// getFirstPageWithTotal gets the first page and calculates total count.
func (c *Client) getFirstPageWithTotal(size int) ([]Repository, int, error) {
	endpoint := fmt.Sprintf("repositories/%s?page=1&pagelen=%d", c.config.Workspace, size)

	resp, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to make request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("warning: failed to close response body: %v\n", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, 0, fmt.Errorf("API request failed with status: %d, response: %s", resp.StatusCode, string(body))
	}

	var repoResp RepositoriesResponse
	if err := json.NewDecoder(resp.Body).Decode(&repoResp); err != nil {
		return nil, 0, fmt.Errorf("failed to decode response: %w", err)
	}

	totalCount := repoResp.Size
	if totalCount == 0 {
		totalCount = len(repoResp.Values)
	}

	return repoResp.Values, totalCount, nil
}

// getTotalRepositoryCount attempts to get the total repository count.
func (c *Client) getTotalRepositoryCount(size int) (int, error) {
	endpoint := fmt.Sprintf("repositories/%s?page=1&pagelen=1", c.config.Workspace)

	resp, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return 0, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("warning: failed to close response body: %v\n", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("failed to get repository count")
	}

	var repoResp RepositoriesResponse
	if err := json.NewDecoder(resp.Body).Decode(&repoResp); err != nil {
		return 0, err
	}

	if repoResp.Size > 0 {
		return repoResp.Size, nil
	}
	return len(repoResp.Values), nil
}

// GetRepository retrieves detailed information about a repository by slug or UUID.
func (c *Client) GetRepository(repoIdentifier string) (*Repository, error) {
	repositoryID := strings.Trim(repoIdentifier, "{}")
	if isRepositoryUUID(repositoryID) {
		if repositories, err := c.GetRepositories(); err == nil {
			for _, repository := range repositories {
				if strings.EqualFold(strings.Trim(repository.UUID, "{}"), repositoryID) {
					return &repository, nil
				}
			}
		}
	}
	endpoint := fmt.Sprintf("repositories/%s/%s", c.config.Workspace, url.PathEscape(repositoryID))
	resp, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("warning: failed to close response body: %v\n", err)
		}
	}()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed for repository %s with status: %d, response: %s", repoIdentifier, resp.StatusCode, string(body))
	}
	var repo Repository
	if err := json.NewDecoder(resp.Body).Decode(&repo); err != nil {
		return nil, fmt.Errorf("failed to decode repository response: %w", err)
	}
	return &repo, nil
}

func isRepositoryUUID(value string) bool {
	if len(value) != 36 {
		return false
	}
	for index, character := range value {
		if index == 8 || index == 13 || index == 18 || index == 23 {
			if character != '-' {
				return false
			}
			continue
		}
		if !strings.ContainsRune("0123456789abcdefABCDEF", character) {
			return false
		}
	}
	return true
}

// GetRepositoryMainBranch returns the repository's main branch name or a fallback.
func (c *Client) GetRepositoryMainBranch(repoSlug string) (string, error) {
	repo, err := c.GetRepository(repoSlug)
	if err != nil {
		return "", err
	}
	if repo.MainBranch.Name != "" {
		return repo.MainBranch.Name, nil
	}
	return "main", nil
}

// GetRepositoryReadme fetches README file content if present.
// Returns the matched filename and its contents.
func (c *Client) GetRepositoryReadme(repoSlug string) (string, string, error) {
	candidates := []string{
		"README.md", "README.MD", "Readme.md", "readme.md",
		"README.markdown", "README.txt", "README", "readme", "Readme",
	}
	for _, name := range candidates {
		endpoint := fmt.Sprintf("repositories/%s/%s/src/HEAD/%s", c.config.Workspace, repoSlug, name)
		resp, err := c.makeRequest("GET", endpoint, nil)
		if err != nil {
			continue
		}
		if resp.StatusCode == http.StatusOK {
			data, _ := io.ReadAll(resp.Body)
			if err := resp.Body.Close(); err != nil {
				fmt.Printf("warning: failed to close response body: %v\n", err)
			}
			return name, string(data), nil
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				fmt.Printf("warning: failed to close response body: %v\n", err)
			}
		}()
	}
	return "", "", fmt.Errorf("no README found in repository %s", repoSlug)
}

// CreatePullRequest creates a new pull request with description and reviewers.
func (c *Client) CreatePullRequest(repoSlug, title, description, sourceBranch, destinationBranch string, reviewers []string) (*PullRequest, error) {
	endpoint := fmt.Sprintf("repositories/%s/%s/pullrequests", c.config.Workspace, repoSlug)

	var reviewerObjs []map[string]string
	for _, r := range reviewers {
		if strings.TrimSpace(r) == "" {
			continue
		}
		reviewerObjs = append(reviewerObjs, map[string]string{"username": r})
	}

	body := map[string]interface{}{
		"title":       title,
		"description": description,
		"source": map[string]interface{}{
			"branch": map[string]string{"name": sourceBranch},
		},
		"destination": map[string]interface{}{
			"branch": map[string]string{"name": destinationBranch},
		},
	}
	if len(reviewerObjs) > 0 {
		body["reviewers"] = reviewerObjs
	}

	resp, err := c.makeRequest("POST", endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("warning: failed to close response body: %v\n", err)
		}
	}()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status: %d, response: %s", resp.StatusCode, string(respBody))
	}

	var pr PullRequest
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &pr, nil
}
