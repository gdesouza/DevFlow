package jenkins

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"devflow/internal/config"
	"fmt"
)

// Existing tests omitted for brevity...

func TestGetJobBuilds_ErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		if _, err := fmt.Fprint(w, `{"error":"Not found"}`); err != nil {
			t.Fatalf("write failed: %v", err)
		}
	}))
	defer server.Close()

	cfg := &config.JenkinsConfig{URL: server.URL, Username: "u", Token: "t"}
	client := NewClient(cfg)
	_, err := client.GetJobBuilds("badjob", 10)
	if err == nil {
		t.Errorf("Expected error for non-OK status, got nil")
	}
}

func TestGetJobBuilds_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := fmt.Fprint(w, "not-json"); err != nil {
			t.Fatalf("write failed: %v", err)
		}
	}))
	defer server.Close()
	cfg := &config.JenkinsConfig{URL: server.URL, Username: "u", Token: "t"}
	client := NewClient(cfg)
	_, err := client.GetJobBuilds("badjob", 10)
	if err == nil {
		t.Errorf("Expected decode error, got nil")
	}
}

func TestGetBuildLog_ErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		if _, err := fmt.Fprint(w, "fail"); err != nil {
			t.Fatalf("write failed: %v", err)
		}
	}))
	defer server.Close()
	cfg := &config.JenkinsConfig{URL: server.URL, Username: "u", Token: "t"}
	client := NewClient(cfg)
	_, err := client.GetBuildLog("badjob", 10)
	if err == nil {
		t.Errorf("Expected log error, got nil")
	}
}

func TestGetBuildStages_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()
	cfg := &config.JenkinsConfig{URL: server.URL, Username: "u", Token: "t"}
	client := NewClient(cfg)
	stages, err := client.GetBuildStages("job", 1)
	if err != nil || stages != nil {
		t.Errorf("Expected nil stages and no error for not found, got %v %v", stages, err)
	}
}

func TestGetBuildStages_ErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		if _, err := fmt.Fprint(w, "fail"); err != nil {
			t.Fatalf("write failed: %v", err)
		}
	}))
	defer server.Close()
	cfg := &config.JenkinsConfig{URL: server.URL, Username: "u", Token: "t"}
	client := NewClient(cfg)
	_, err := client.GetBuildStages("job", 1)
	if err == nil {
		t.Error("Expected error for bad status, got nil")
	}
}

func TestGetBuildStages_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := fmt.Fprint(w, "not-json"); err != nil {
			t.Fatalf("write failed: %v", err)
		}
	}))
	defer server.Close()
	cfg := &config.JenkinsConfig{URL: server.URL, Username: "u", Token: "t"}
	client := NewClient(cfg)
	_, err := client.GetBuildStages("job", 1)
	if err == nil {
		t.Error("Expected decode error, got nil")
	}
}

func TestGetBuildStages_Success(t *testing.T) {
	stagesJson := `{ "stages": [ { "id": "99", "name": "Test", "status": "FAILED", "startTimeMillis": 100, "durationMillis": 200, "error": "" } ]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := fmt.Fprint(w, stagesJson); err != nil {
			t.Fatalf("write failed: %v", err)
		}
	}))
	defer server.Close()
	cfg := &config.JenkinsConfig{URL: server.URL, Username: "u", Token: "t"}
	client := NewClient(cfg)
	stages, err := client.GetBuildStages("job", 1)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(stages) != 1 || stages[0].ID != "99" || stages[0].Status != "FAILED" {
		t.Errorf("Stage parse error, got %v", stages)
	}
}

func TestGetFailedStepLog_PipelineFailedStage(t *testing.T) {
	// stub for GetBuildStages and then stage log, fallback
	stageJson := `{ "stages": [ { "id": "1234", "name": "Deploy", "status": "FAILED", "startTimeMillis": 10, "durationMillis": 20 } ]}`
	stageLog := "FAILURE LOG"
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		switch r.URL.Path {
		case "/job/job/1/wfapi/describe":
			w.WriteHeader(http.StatusOK)
			if _, err := fmt.Fprint(w, stageJson); err != nil {
				t.Fatalf("write failed: %v", err)
			}
		case "/job/job/1/execution/node/1234/wfapi/log":
			w.WriteHeader(http.StatusOK)
			if _, err := fmt.Fprint(w, stageLog); err != nil {
				t.Fatalf("write failed: %v", err)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	cfg := &config.JenkinsConfig{URL: server.URL, Username: "u", Token: "t"}
	client := NewClient(cfg)
	log, err := client.GetFailedStepLog("job", 1)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if log == "" || log != "=== Failed Stage: Deploy ===\nFAILURE LOG" {
		t.Errorf("Expected pipeline failed stage log, got %q", log)
	}
}

func TestGetFailedStepLog_PipelineFallback(t *testing.T) {
	// stage log endpoint returns error, fallback to full log
	stageJson := `{ "stages": [ { "id": "456", "name": "Unit", "status": "FAILED" } ]}`
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		switch r.URL.Path {
		case "/job/job/1/wfapi/describe":
			w.WriteHeader(http.StatusOK)
			if _, err := fmt.Fprint(w, stageJson); err != nil {
				t.Fatalf("write failed: %v", err)
			}
		case "/job/job/1/execution/node/456/wfapi/log":
			w.WriteHeader(http.StatusInternalServerError)
		case "/job/job/1/consoleText":
			w.WriteHeader(http.StatusOK)
			if _, err := fmt.Fprint(w, "main log"); err != nil {
				t.Fatalf("write failed: %v", err)
			}
		}
	}))
	defer server.Close()
	cfg := &config.JenkinsConfig{URL: server.URL, Username: "u", Token: "t"}
	client := NewClient(cfg)
	log, err := client.GetFailedStepLog("job", 1)
	if err != nil {
		t.Fatalf("Pipeline fallback error: %v", err)
	}
	if log == "" || log != "main log" {
		t.Errorf("Expected fallback to main log, got %q", log)
	}
}

func TestGetFailedStepLog_NonPipelineJob(t *testing.T) {
	// wfapi/describe returns nil stages, fallback to full log
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		switch r.URL.Path {
		case "/job/job/1/wfapi/describe":
			w.WriteHeader(http.StatusNotFound)
		case "/job/job/1/consoleText":
			w.WriteHeader(http.StatusOK)
			if _, err := fmt.Fprint(w, "full log"); err != nil {
				t.Fatalf("write failed: %v", err)
			}
		}
	}))
	defer server.Close()
	cfg := &config.JenkinsConfig{URL: server.URL, Username: "u", Token: "t"}
	client := NewClient(cfg)
	log, err := client.GetFailedStepLog("job", 1)
	if err != nil {
		t.Fatalf("Non-pipeline fallback error: %v", err)
	}
	if log == "" || log != "full log" {
		t.Errorf("Expected full log fallback, got %q", log)
	}
}
