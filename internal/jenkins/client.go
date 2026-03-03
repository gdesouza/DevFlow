package jenkins

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"devflow/internal/config"
)

type Client struct {
	config     *config.JenkinsConfig
	httpClient *http.Client
	baseURL    string
}

type Build struct {
	Number    int    `json:"number"`
	Result    string `json:"result"`
	Timestamp int64  `json:"timestamp"`
	Duration  int64  `json:"duration"`
	URL       string `json:"url"`
	Building  bool   `json:"building"`
}

type Job struct {
	Name      string  `json:"name"`
	URL       string  `json:"url"`
	Builds    []Build `json:"builds"`
	LastBuild *Build  `json:"lastBuild"`
	Color     string  `json:"color"`
}

type BuildDetails struct {
	Number    int    `json:"number"`
	Result    string `json:"result"`
	Timestamp int64  `json:"timestamp"`
	Duration  int64  `json:"duration"`
	URL       string `json:"url"`
	Building  bool   `json:"building"`
	Actions   []struct {
		Causes []struct {
			ShortDescription string `json:"shortDescription"`
			UserName         string `json:"userName"`
		} `json:"causes"`
	} `json:"actions"`
}

type BuildStage struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Status          string `json:"status"`
	StartTimeMillis int64  `json:"startTimeMillis"`
	DurationMillis  int64  `json:"durationMillis"`
	Error           string `json:"error,omitempty"`
}

func NewClient(cfg *config.JenkinsConfig) *Client {
	return &Client{
		config:     cfg,
		httpClient: &http.Client{},
		baseURL:    strings.TrimSuffix(cfg.URL, "/"),
	}
}

func (c *Client) makeRequest(method, path string) (*http.Response, error) {
	url := fmt.Sprintf("%s/%s", c.baseURL, strings.TrimPrefix(path, "/"))

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Use API token authentication if configured
	if c.config.Username != "" && c.config.Token != "" {
		req.SetBasicAuth(c.config.Username, c.config.Token)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}

	return resp, nil
}

// GetJobBuilds retrieves recent builds for a job
func (c *Client) GetJobBuilds(jobName string, limit int) ([]Build, error) {
	path := fmt.Sprintf("/job/%s/api/json?tree=builds[number,result,timestamp,duration,url,building]{0,%d}", jobName, limit)

	resp, err := c.makeRequest("GET", path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status: %d, response: %s", resp.StatusCode, string(body))
	}

	var job Job
	if err := json.NewDecoder(resp.Body).Decode(&job); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return job.Builds, nil
}

// GetBuildLog retrieves the console log for a specific build
func (c *Client) GetBuildLog(jobName string, buildNumber int) (string, error) {
	path := fmt.Sprintf("/job/%s/%d/consoleText", jobName, buildNumber)

	resp, err := c.makeRequest("GET", path)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API request failed with status: %d, response: %s", resp.StatusCode, string(body))
	}

	log, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read log: %w", err)
	}

	return string(log), nil
}

// GetBuildStages retrieves the pipeline stages for a build
func (c *Client) GetBuildStages(jobName string, buildNumber int) ([]BuildStage, error) {
	path := fmt.Sprintf("/job/%s/%d/wfapi/describe", jobName, buildNumber)

	resp, err := c.makeRequest("GET", path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Pipeline API might not be available for non-pipeline jobs
		return nil, nil
	}

	var result struct {
		Stages []BuildStage `json:"stages"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Stages, nil
}

// GetFailedStepLog attempts to get the log for a failed stage
func (c *Client) GetFailedStepLog(jobName string, buildNumber int) (string, error) {
	stages, err := c.GetBuildStages(jobName, buildNumber)
	if err != nil {
		return "", err
	}

	if stages == nil {
		// Not a pipeline job, return full log
		return c.GetBuildLog(jobName, buildNumber)
	}

	// Find first failed stage
	var failedStage *BuildStage
	for i := range stages {
		if stages[i].Status == "FAILED" {
			failedStage = &stages[i]
			break
		}
	}

	if failedStage != nil {
		// Try to get stage-specific log
		path := fmt.Sprintf("/job/%s/%d/execution/node/%s/wfapi/log", jobName, buildNumber, failedStage.ID)
		resp, err := c.makeRequest("GET", path)
		if err == nil {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				log, err := io.ReadAll(resp.Body)
				if err == nil {
					return fmt.Sprintf("=== Failed Stage: %s ===\n%s", failedStage.Name, string(log)), nil
				}
			}
		}
	}

	// Fallback to full log
	return c.GetBuildLog(jobName, buildNumber)
}
