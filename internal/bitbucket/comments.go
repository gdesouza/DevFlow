package bitbucket

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// GetPullRequestComments retrieves all comments for a given pull request.
func (c *Client) GetPullRequestComments(repoSlug string, prID int) ([]Comment, error) {
	endpoint := fmt.Sprintf("repositories/%s/%s/pullrequests/%d/comments", c.config.Workspace, repoSlug, prID)

	var allComments []Comment
	for endpoint != "" {
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

		var commentsResp CommentsResponse
		if err := json.NewDecoder(resp.Body).Decode(&commentsResp); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		allComments = append(allComments, commentsResp.Values...)

		if commentsResp.Next != "" {
			endpoint = strings.TrimPrefix(commentsResp.Next, c.baseURL+"/")
		} else {
			endpoint = ""
		}
	}

	return allComments, nil
}

// CreatePullRequestComment adds a new comment to a pull request.
func (c *Client) CreatePullRequestComment(repoSlug string, prID int, content string) (*Comment, error) {
	endpoint := fmt.Sprintf("repositories/%s/%s/pullrequests/%d/comments", c.config.Workspace, repoSlug, prID)

	payload := map[string]interface{}{
		"content": map[string]string{
			"raw": content,
		},
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

	var comment Comment
	if err := json.NewDecoder(resp.Body).Decode(&comment); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &comment, nil
}

// CreatePullRequestInlineComment adds an inline comment anchored to a specific file and line in a pull request diff.
func (c *Client) CreatePullRequestInlineComment(repoSlug string, prID int, content string, filePath string, line int) (*Comment, error) {
	endpoint := fmt.Sprintf("repositories/%s/%s/pullrequests/%d/comments", c.config.Workspace, repoSlug, prID)

	payload := map[string]interface{}{
		"content": map[string]string{
			"raw": content,
		},
		"inline": map[string]interface{}{
			"path": filePath,
			"to":   line,
		},
	}

	resp, err := c.makeRequest("POST", endpoint, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("warning: failed to close response body: %v\n", err)
		}
	}()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status: %d, response: %s", resp.StatusCode, string(body))
	}

	var comment Comment
	if err := json.NewDecoder(resp.Body).Decode(&comment); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &comment, nil
}

// ReplyToPullRequestComment replies to a specific comment thread.
func (c *Client) ReplyToPullRequestComment(repoSlug string, prID int, parentCommentID int, content string) (*Comment, error) {
	endpoint := fmt.Sprintf("repositories/%s/%s/pullrequests/%d/comments", c.config.Workspace, repoSlug, prID)

	payload := map[string]interface{}{
		"content": map[string]string{
			"raw": content,
		},
		"parent": map[string]int{
			"id": parentCommentID,
		},
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

	var comment Comment
	if err := json.NewDecoder(resp.Body).Decode(&comment); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &comment, nil
}

