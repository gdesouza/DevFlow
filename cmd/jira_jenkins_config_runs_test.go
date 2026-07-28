package cmd

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"

	"devflow/internal/config"
	"devflow/internal/httpx"
	"devflow/internal/jira"
)

func registerHost(t *testing.T, host string, handler http.HandlerFunc) {
	t.Helper()
	httpx.RegisterTestServer(host, handler)
	t.Cleanup(func() { httpx.UnregisterTestServer(host) })
}

func setJiraCmdConfig(t *testing.T, cfg *config.Config) {
	t.Helper()
	orig := loadConfig
	loadConfig = func() (*config.Config, error) { return cfg, nil }
	t.Cleanup(func() { loadConfig = orig })
}

func TestJiraCommentLinkAndSpacesCmds(t *testing.T) {
	setJiraCmdConfig(t, &config.Config{
		Jira: config.JiraConfig{URL: "https://jira.example", Username: "alice", Token: "token"},
	})

	registerHost(t, "jira.example", func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/rest/api/3/issue/ABC-1/comment":
			w.WriteHeader(http.StatusCreated)
		case r.Method == http.MethodPost && r.URL.Path == "/rest/api/3/issue/ABC-1/remotelink":
			w.WriteHeader(http.StatusCreated)
		case r.Method == http.MethodGet && r.URL.Path == "/rest/api/3/project":
			_ = json.NewEncoder(w).Encode([]jira.Project{{Key: "ABC", Name: "Project ABC", Lead: struct {
				DisplayName string `json:"displayName"`
			}{DisplayName: "Lead"}}})
		default:
			t.Fatalf("unexpected jira request: %s %s", r.Method, r.URL.Path)
		}
	})

	origBody := commentBody
	origBodyFile := commentBodyFile
	origTitle := linkTitle
	origSummary := linkSummary
	defer func() {
		commentBody = origBody
		commentBodyFile = origBodyFile
		linkTitle = origTitle
		linkSummary = origSummary
	}()

	commentBody = "hello"
	commentBodyFile = ""
	out := captureStdout(func() {
		commentCmd.Run(commentCmd, []string{"ABC-1"})
	})
	if !strings.Contains(out, "Added comment to ABC-1") {
		t.Fatalf("unexpected jira comment output: %q", out)
	}

	linkTitle = "Docs"
	linkSummary = "Summary"
	out = captureStdout(func() {
		linkCmd.Run(linkCmd, []string{"ABC-1", "https://example.com"})
	})
	if !strings.Contains(out, "Added link to ABC-1") {
		t.Fatalf("unexpected jira link output: %q", out)
	}

	out = captureStdout(func() {
		spacesCmd.Run(spacesCmd, nil)
	})
	if !strings.Contains(out, "Found 1 Jira projects") || !strings.Contains(out, "ABC - Project ABC") {
		t.Fatalf("unexpected jira spaces output: %q", out)
	}
}

func TestJiraCreateUpdateAndMentionCmds(t *testing.T) {
	setJiraCmdConfig(t, &config.Config{
		Jira: config.JiraConfig{URL: "https://jira.example", Username: "alice", Token: "token"},
	})

	registerHost(t, "jira.example", func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/rest/api/3/issue":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(jira.Issue{Key: "ABC-1"})
		case r.Method == http.MethodPut && r.URL.Path == "/rest/api/3/issue/ABC-1":
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/rest/api/3/search/jql"):
			_ = json.NewEncoder(w).Encode(jira.SearchResponse{
				Issues: []jira.Issue{{Key: "ABC-2"}},
			})
		default:
			t.Fatalf("unexpected jira request: %s %s", r.Method, r.URL.Path)
		}
	})

	origProject := createProjectKey
	origType := createIssueType
	origPriority := createPriority
	origAssignee := createAssignee
	origLabels := createLabels
	origEpic := createEpic
	origStory := createStoryPoints
	origSprint := createSprint
	origTeam := createTeam
	origDesc := createDescription
	origDescFile := createDescriptionFile
	origUpdAssignee := updateAssignee
	origUpdPriority := updatePriority
	origUpdLabels := updateLabels
	origUpdSummary := updateSummary
	origUpdTitle := updateTitle
	origUpdDesc := updateDescription
	origUpdDescFile := updateDescriptionFile
	origUpdEpic := updateEpic
	origUpdStory := updateStoryPoints
	origUpdSprint := updateSprint
	origUpdTeam := updateTeam
	defer func() {
		createProjectKey = origProject
		createIssueType = origType
		createPriority = origPriority
		createAssignee = origAssignee
		createLabels = origLabels
		createEpic = origEpic
		createStoryPoints = origStory
		createSprint = origSprint
		createTeam = origTeam
		createDescription = origDesc
		createDescriptionFile = origDescFile
		updateAssignee = origUpdAssignee
		updatePriority = origUpdPriority
		updateLabels = origUpdLabels
		updateSummary = origUpdSummary
		updateTitle = origUpdTitle
		updateDescription = origUpdDesc
		updateDescriptionFile = origUpdDescFile
		updateEpic = origUpdEpic
		updateStoryPoints = origUpdStory
		updateSprint = origUpdSprint
		updateTeam = origUpdTeam
	}()

	createProjectKey = "ABC"
	createIssueType = "Task"
	createPriority = "High"
	createAssignee = "alice"
	createLabels = "a,b"
	createEpic = "EPIC-1"
	createStoryPoints = 3
	createSprint = "Sprint 1"
	createTeam = "team-a"
	createDescription = "line1\nline2"
	createDescriptionFile = ""

	out := captureStdout(func() {
		createTaskCmd.Run(createTaskCmd, []string{"Title"})
	})
	if !strings.Contains(out, "Created Jira issue ABC-1") {
		t.Fatalf("unexpected jira create output: %q", out)
	}

	updateAssignee = "alice"
	updatePriority = "High"
	updateLabels = "x,y"
	updateSummary = "Updated title"
	updateTitle = ""
	updateDescription = "body"
	updateDescriptionFile = ""
	updateEpic = "EPIC-2"
	updateStoryPoints = 5
	updateSprint = "Sprint 2"
	updateTeam = "team-b"

	out = captureStdout(func() {
		updateTaskCmd.Run(updateTaskCmd, []string{"ABC-1"})
	})
	if !strings.Contains(out, "Updated ABC-1") {
		t.Fatalf("unexpected jira update output: %q", out)
	}

	out = captureStdout(func() {
		mentionedCmd.Run(mentionedCmd, nil)
	})
	if !strings.Contains(out, "Found 1 Jira issues where you are mentioned") || !strings.Contains(out, "ABC-2") {
		t.Fatalf("unexpected jira mentioned output: %q", out)
	}
}

func TestJenkinsLogsAndConfigSetupCmd(t *testing.T) {
	registerHost(t, "jenkins.example", func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/job/my-job/42/consoleText":
			_, _ = w.Write([]byte("full log"))
		case "/job/my-job/42/wfapi/describe":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"stages": []map[string]any{
					{"id": "stage-1", "name": "Build", "status": "FAILED"},
				},
			})
		case "/job/my-job/42/execution/node/stage-1/wfapi/log":
			_, _ = w.Write([]byte("failed stage log"))
		default:
			t.Fatalf("unexpected jenkins request: %s", r.URL.Path)
		}
	})

	origLoad := loadConfig
	origSave := saveConfig
	defer func() {
		loadConfig = origLoad
		saveConfig = origSave
	}()
	loadConfig = func() (*config.Config, error) {
		return &config.Config{
			Jenkins: config.JenkinsConfig{URL: "https://jenkins.example"},
		}, nil
	}

	out := captureStdout(func() {
		jenkinsLogsCmd.Run(jenkinsLogsCmd, []string{"my-job", "42"})
	})
	if !strings.Contains(out, "full log") {
		t.Fatalf("unexpected jenkins logs output: %q", out)
	}

	origFailedStep, _ := jenkinsLogsCmd.Flags().GetBool("failed-step")
	t.Cleanup(func() {
		_ = jenkinsLogsCmd.Flags().Set("failed-step", map[bool]string{true: "true", false: "false"}[origFailedStep])
	})
	_ = jenkinsLogsCmd.Flags().Set("failed-step", "true")
	out = captureStdout(func() {
		jenkinsLogsCmd.Run(jenkinsLogsCmd, []string{"my-job", "42"})
	})
	if !strings.Contains(out, "failed stage log") {
		t.Fatalf("unexpected failed-step logs output: %q", out)
	}

	var saved *config.Config
	saveConfig = func(cfg *config.Config) error {
		saved = cfg
		return nil
	}
	input := strings.Join([]string{
		"https://jira.example",
		"jira-user",
		"jira-token",
		"workspace",
		"bb-user",
		"bb-api-user",
		"bb-token",
	}, "\n") + "\n"
	origStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	if _, err := w.Write([]byte(input)); err != nil {
		t.Fatalf("write input: %v", err)
	}
	_ = w.Close()
	os.Stdin = r
	defer func() { os.Stdin = origStdin }()
	captureStdout(func() {
		setupConfigCmd.Run(setupConfigCmd, nil)
	})
	if saved == nil || saved.Jira.URL == "" || saved.Bitbucket.Workspace == "" {
		t.Fatalf("expected config to be saved, got %+v", saved)
	}
}
