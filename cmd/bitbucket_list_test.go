package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"devflow/internal/bitbucket"
)

func captureStdout(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	f()
	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = old
	return buf.String()
}

func TestPrintPRsJSON_Empty(t *testing.T) {
	out := captureStdout(func() {
		printPRsJSON("w", "repo", nil)
	})
	var obj struct {
		Workspace    string                  `json:"workspace"`
		Repository   string                  `json:"repository"`
		PullRequests []bitbucket.PullRequest `json:"pull_requests"`
		Total        int                     `json:"total"`
	}
	if err := json.Unmarshal([]byte(out), &obj); err != nil {
		t.Fatalf("unmarshal failed: %v (output was: %q)", err, out)
	}
	if obj.Workspace != "w" || obj.Repository != "repo" || obj.Total != 0 || len(obj.PullRequests) != 0 {
		t.Errorf("unexpected output fields: %+v", obj)
	}
}

func TestPrintPRsJSON_One(t *testing.T) {
	pr := bitbucket.PullRequest{
		ID:    1,
		Title: "Test PR",
		State: "OPEN",
	}
	out := captureStdout(func() {
		printPRsJSON("space", "s", []bitbucket.PullRequest{pr})
	})
	if !strings.Contains(out, "Test PR") || !strings.Contains(out, "space") || !strings.Contains(out, "pull_requests") {
		t.Errorf("missing expected content in output: %q", out)
	}
	var obj struct {
		Total int `json:"total"`
	}
	json.Unmarshal([]byte(out), &obj)
	if obj.Total != 1 {
		t.Errorf("expected total=1 got %d (%q)", obj.Total, out)
	}
}
