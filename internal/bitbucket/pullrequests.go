package bitbucket

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// GetPullRequests retrieves pull requests for a repository.
func (c *Client) GetPullRequests(repoSlug string) ([]PullRequest, error) {
	endpoint := fmt.Sprintf("repositories/%s/%s/pullrequests", c.config.Workspace, repoSlug)

	var allPRs []PullRequest
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

		var prResp PullRequestsResponse
		if err := json.NewDecoder(resp.Body).Decode(&prResp); err != nil {
			if err := resp.Body.Close(); err != nil {
				fmt.Printf("warning: failed to close response body: %v\n", err)
			}
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		allPRs = append(allPRs, prResp.Values...)

		defer func() {
			if err := resp.Body.Close(); err != nil {
				fmt.Printf("warning: failed to close response body: %v\n", err)
			}
		}()

		if prResp.Next != "" {
			endpoint = strings.TrimPrefix(prResp.Next, c.baseURL+"/")
		} else {
			endpoint = ""
		}
	}

	return allPRs, nil
}

// GetParticipatingPullRequests retrieves pull requests where the user participates (author, reviewer, etc.).
func (c *Client) GetParticipatingPullRequests(repoSlug, username string) ([]PullRequest, error) {
	query := url.QueryEscape(fmt.Sprintf("participants.username=\"%s\"", username))
	endpoint := fmt.Sprintf("repositories/%s/%s/pullrequests?q=%s", c.config.Workspace, repoSlug, query)

	var allPRs []PullRequest
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

		var prResp PullRequestsResponse
		if err := json.NewDecoder(resp.Body).Decode(&prResp); err != nil {
			if err := resp.Body.Close(); err != nil {
				fmt.Printf("warning: failed to close response body: %v\n", err)
			}
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		allPRs = append(allPRs, prResp.Values...)

		defer func() {
			if err := resp.Body.Close(); err != nil {
				fmt.Printf("warning: failed to close response body: %v\n", err)
			}
		}()

		if prResp.Next != "" {
			endpoint = strings.TrimPrefix(prResp.Next, c.baseURL+"/")
		} else {
			endpoint = ""
		}
	}

	return allPRs, nil
}

// GetPullRequestsWithReviewers retrieves pull requests with reviewer information for a specific repository.
func (c *Client) GetPullRequestsWithReviewers(repoSlug string) ([]PullRequestWithReviewers, error) {
	basicPRs, err := c.GetPullRequests(repoSlug)
	if err != nil {
		return nil, fmt.Errorf("failed to get basic PR list: %w", err)
	}

	if len(basicPRs) == 0 {
		return []PullRequestWithReviewers{}, nil
	}

	var prsWithReviewers []PullRequestWithReviewers
	fmt.Printf("Fetching reviewer details for %d PRs in %s...\n", len(basicPRs), repoSlug)

	for i, basicPR := range basicPRs {
		if i > 0 && i%5 == 0 {
			fmt.Printf("Processed %d/%d PRs...\n", i, len(basicPRs))
		}

		details, err := c.GetPullRequestDetails(repoSlug, basicPR.ID)
		if err != nil {
			fmt.Printf("Warning: Failed to get details for PR #%d: %v\n", basicPR.ID, err)
			continue
		}

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
						UUID:        "",
					})
				}
				return reviewers
			}(),
		}

		prsWithReviewers = append(prsWithReviewers, prWithReviewers)
	}

	return prsWithReviewers, nil
}

// GetWorkspacePullRequestsForUser retrieves all PRs where the user is a reviewer across the entire workspace.
func (c *Client) GetWorkspacePullRequestsForUser(username string) ([]PullRequestWithReviewers, error) {
	endpoint := fmt.Sprintf("workspaces/%s/pullrequests/%s", c.config.Workspace, username)

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
		return nil, fmt.Errorf("API request failed with status: %d, response: %s", resp.StatusCode, string(body))
	}

	var prResp PullRequestsWithReviewersResponse
	if err := json.NewDecoder(resp.Body).Decode(&prResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return prResp.Values, nil
}

// GetPullRequestDetails retrieves detailed information about a specific pull request.
func (c *Client) GetPullRequestDetails(repoSlug string, prID int) (*PullRequestDetails, error) {
	endpoint := fmt.Sprintf("repositories/%s/%s/pullrequests/%d", c.config.Workspace, repoSlug, prID)

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
		return nil, fmt.Errorf("API request failed with status: %d, response: %s", resp.StatusCode, string(body))
	}

	var pr PullRequestDetails
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &pr, nil
}

// GetPullRequestDiff retrieves the diff for a specific pull request.
func (c *Client) GetPullRequestDiff(repoSlug string, prID int) (string, error) {
	endpoint := fmt.Sprintf("repositories/%s/%s/pullrequests/%d/diff", c.config.Workspace, repoSlug, prID)

	resp, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("warning: failed to close response body: %v\n", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API request failed with status: %d, response: %s", resp.StatusCode, string(body))
	}

	diff, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read diff: %w", err)
	}

	return string(diff), nil
}

// GetPullRequestCommits retrieves commits for a given pull request.
func (c *Client) GetPullRequestCommits(repoSlug string, prID int) ([]Commit, error) {
	endpoint := fmt.Sprintf("repositories/%s/%s/pullrequests/%d/commits", c.config.Workspace, repoSlug, prID)

	var allCommits []Commit
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

		var commitsResp CommitsResponse
		if err := json.NewDecoder(resp.Body).Decode(&commitsResp); err != nil {
			if err := resp.Body.Close(); err != nil {
				fmt.Printf("warning: failed to close response body: %v\n", err)
			}
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		allCommits = append(allCommits, commitsResp.Values...)

		defer func() {
			if err := resp.Body.Close(); err != nil {
				fmt.Printf("warning: failed to close response body: %v\n", err)
			}
		}()

		if commitsResp.Next != "" {
			endpoint = strings.TrimPrefix(commitsResp.Next, c.baseURL+"/")
		} else {
			endpoint = ""
		}
	}

	return allCommits, nil
}

