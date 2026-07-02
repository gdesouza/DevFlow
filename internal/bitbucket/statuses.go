package bitbucket

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// GetCommitStatuses retrieves build/status information for a given commit hash.
func (c *Client) GetCommitStatuses(repoSlug, commitHash string) ([]CommitStatus, error) {
	endpoint := fmt.Sprintf("repositories/%s/%s/commit/%s/statuses", c.config.Workspace, repoSlug, commitHash)

	var allStatuses []CommitStatus
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

		var statusesResp CommitStatusesResponse
		if err := json.NewDecoder(resp.Body).Decode(&statusesResp); err != nil {
			if err := resp.Body.Close(); err != nil {
				fmt.Printf("warning: failed to close response body: %v\n", err)
			}
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		allStatuses = append(allStatuses, statusesResp.Values...)

		defer func() {
			if err := resp.Body.Close(); err != nil {
				fmt.Printf("warning: failed to close response body: %v\n", err)
			}
		}()

		if statusesResp.Next != "" {
			endpoint = strings.TrimPrefix(statusesResp.Next, c.baseURL+"/")
		} else {
			endpoint = ""
		}
	}

	return allStatuses, nil
}

// SetCommitStatus creates or updates a build/status for a commit.
// Bitbucket upserts a status when the same key is reused.
func (c *Client) SetCommitStatus(repoSlug, commitHash, state, key, name, urlStr, description string) (*CommitStatus, error) {
	endpoint := fmt.Sprintf("repositories/%s/%s/commit/%s/statuses/build", c.config.Workspace, repoSlug, commitHash)
	payload := map[string]string{
		"state":       state,
		"key":         key,
		"name":        name,
		"url":         urlStr,
		"description": description,
	}
	resp, err := c.makeRequest("POST", endpoint, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer func() {
		defer func() {
			if err := resp.Body.Close(); err != nil {
				fmt.Printf("warning: failed to close response body: %v\n", err)
			}
		}()
	}()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status: %d, response: %s", resp.StatusCode, string(body))
	}
	var status CommitStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &status, nil
}

