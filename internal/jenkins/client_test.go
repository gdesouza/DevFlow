package jenkins

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"devflow/internal/config"
)

func TestNewClient(t *testing.T) {
	cfg := &config.JenkinsConfig{
		URL:      "https://jenkins.example.com",
		Username: "testuser",
		Token:    "testtoken",
	}

	client := NewClient(cfg)

	if client == nil {
		t.Fatal("NewClient returned nil")
	}

	if client.baseURL != "https://jenkins.example.com" {
		t.Errorf("Expected baseURL to be 'https://jenkins.example.com', got '%s'", client.baseURL)
	}

	if client.config != cfg {
		t.Error("Config not set correctly")
	}
}

func TestGetJobBuilds(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/job/test-job/api/json" {
			t.Errorf("Expected path '/job/test-job/api/json', got '%s'", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"builds": [
				{
					"number": 123,
					"result": "SUCCESS",
					"timestamp": 1640000000000,
					"duration": 60000,
					"url": "https://jenkins.example.com/job/test-job/123/",
					"building": false
				},
				{
					"number": 122,
					"result": "FAILURE",
					"timestamp": 1639900000000,
					"duration": 45000,
					"url": "https://jenkins.example.com/job/test-job/122/",
					"building": false
				}
			]
		}`))
	}))
	defer server.Close()

	cfg := &config.JenkinsConfig{
		URL:      server.URL,
		Username: "testuser",
		Token:    "testtoken",
	}

	client := NewClient(cfg)
	builds, err := client.GetJobBuilds("test-job", 10)

	if err != nil {
		t.Fatalf("GetJobBuilds failed: %v", err)
	}

	if len(builds) != 2 {
		t.Errorf("Expected 2 builds, got %d", len(builds))
	}

	if builds[0].Number != 123 {
		t.Errorf("Expected build number 123, got %d", builds[0].Number)
	}

	if builds[0].Result != "SUCCESS" {
		t.Errorf("Expected result 'SUCCESS', got '%s'", builds[0].Result)
	}
}

func TestGetBuildLog(t *testing.T) {
	expectedLog := "Build log content\nLine 2\nLine 3"

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/job/test-job/123/consoleText" {
			t.Errorf("Expected path '/job/test-job/123/consoleText', got '%s'", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(expectedLog))
	}))
	defer server.Close()

	cfg := &config.JenkinsConfig{
		URL:      server.URL,
		Username: "testuser",
		Token:    "testtoken",
	}

	client := NewClient(cfg)
	log, err := client.GetBuildLog("test-job", 123)

	if err != nil {
		t.Fatalf("GetBuildLog failed: %v", err)
	}

	if log != expectedLog {
		t.Errorf("Expected log '%s', got '%s'", expectedLog, log)
	}
}
