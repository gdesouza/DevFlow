package bitbucket

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"devflow/internal/httpx"
)

// GetPipelines returns a list of pipelines for the given repository.
// Results are sorted by build number descending (newest first) and limited by the pagelen param.
func (c *Client) GetPipelines(repoSlug string, limit int) ([]Pipeline, error) {
	endpoint := fmt.Sprintf("repositories/%s/%s/pipelines/?sort=-created_on&pagelen=%d", c.config.Workspace, repoSlug, limit)

	var allPipelines []Pipeline
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

		var pipelinesResp PipelinesResponse
		if err := json.NewDecoder(resp.Body).Decode(&pipelinesResp); err != nil {
			if err := resp.Body.Close(); err != nil {
				fmt.Printf("warning: failed to close response body: %v\n", err)
			}
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		if err := resp.Body.Close(); err != nil {
			fmt.Printf("warning: failed to close response body: %v\n", err)
		}

		allPipelines = append(allPipelines, pipelinesResp.Values...)

		if len(allPipelines) >= limit || pipelinesResp.Next == "" {
			break
		}
		endpoint = strings.TrimPrefix(pipelinesResp.Next, c.baseURL+"/")
	}

	if len(allPipelines) > limit {
		allPipelines = allPipelines[:limit]
	}
	return allPipelines, nil
}

// GetPipeline returns details for a single pipeline by its UUID or build number string.
func (c *Client) GetPipeline(repoSlug, pipelineUUID string) (*Pipeline, error) {
	endpoint := fmt.Sprintf("repositories/%s/%s/pipelines/%s", c.config.Workspace, repoSlug, pipelineUUID)

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

	var pipeline Pipeline
	if err := json.NewDecoder(resp.Body).Decode(&pipeline); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &pipeline, nil
}

// GetPipelineSteps returns the steps for a pipeline.
func (c *Client) GetPipelineSteps(repoSlug, pipelineUUID string) ([]PipelineStep, error) {
	endpoint := fmt.Sprintf("repositories/%s/%s/pipelines/%s/steps/", c.config.Workspace, repoSlug, pipelineUUID)

	var allSteps []PipelineStep
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

		var stepsResp PipelineStepsResponse
		if err := json.NewDecoder(resp.Body).Decode(&stepsResp); err != nil {
			if err := resp.Body.Close(); err != nil {
				fmt.Printf("warning: failed to close response body: %v\n", err)
			}
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		if err := resp.Body.Close(); err != nil {
			fmt.Printf("warning: failed to close response body: %v\n", err)
		}

		allSteps = append(allSteps, stepsResp.Values...)

		if stepsResp.Next == "" {
			break
		}
		endpoint = strings.TrimPrefix(stepsResp.Next, c.baseURL+"/")
	}

	return allSteps, nil
}

// GetPipelineStepLog returns the raw log output for a pipeline step.
// The Bitbucket log endpoint returns text/plain, so this method sets
// Accept: */* explicitly to avoid a 406 Not Acceptable response.
func (c *Client) GetPipelineStepLog(repoSlug, pipelineUUID, stepUUID string) (string, error) {
	endpoint := fmt.Sprintf("repositories/%s/%s/pipelines/%s/steps/%s/log", c.config.Workspace, repoSlug, pipelineUUID, stepUUID)
	rawURL := fmt.Sprintf("%s/%s", c.baseURL, endpoint)

	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpx.ApplyAuth(req, c.config.Username, c.config.Token)
	req.Header.Set("Accept", "*/*")

	resp, err := c.httpClient.Do(req)
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

	logBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read log response: %w", err)
	}

	return string(logBytes), nil
}
