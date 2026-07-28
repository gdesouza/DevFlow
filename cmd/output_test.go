package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"devflow/internal/bitbucket"
	"github.com/spf13/cobra"
)

func TestValidateFormat(t *testing.T) {
	tests := []struct {
		name    string
		format  string
		wantErr bool
	}{
		{name: "json", format: formatJSON},
		{name: "raw", format: formatRaw},
		{name: "tabular", format: formatTabular},
		{name: "detailed", format: formatDetailed},
		{name: "invalid", format: "yaml", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.Flags().String("format", tt.format, "")
			if err := validateFormat(cmd, nil); (err != nil) != tt.wantErr {
				t.Fatalf("validateFormat() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWantsJSONSupportsFormatAndLegacyFlag(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("format", formatDetailed, "")
	cmd.Flags().Bool("json", false, "")

	if wantsJSON(cmd) {
		t.Fatal("detailed format should not select JSON")
	}
	if err := cmd.Flags().Set("format", formatJSON); err != nil {
		t.Fatal(err)
	}
	if !wantsJSON(cmd) {
		t.Fatal("json format should select JSON")
	}
	if err := cmd.Flags().Set("format", formatDetailed); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("json", "true"); err != nil {
		t.Fatal(err)
	}
	if wantsJSON(cmd) {
		t.Fatal("explicit detailed format should take precedence over legacy json flag")
	}

	legacyCmd := &cobra.Command{}
	legacyCmd.Flags().String("format", formatDetailed, "")
	legacyCmd.Flags().Bool("json", false, "")
	if err := legacyCmd.Flags().Set("json", "true"); err != nil {
		t.Fatal(err)
	}
	if !wantsJSON(legacyCmd) {
		t.Fatal("legacy json flag should remain supported when format is not explicit")
	}
}

func TestWantsTabularHonorsExplicitFormat(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("format", formatDetailed, "")
	if wantsTabular(cmd) {
		t.Fatal("detailed format should not select tabular")
	}
	if err := cmd.Flags().Set("format", formatTabular); err != nil {
		t.Fatal(err)
	}
	if !wantsTabular(cmd) {
		t.Fatal("tabular format should select tabular output")
	}
}

func TestWantsRawAndPrintJSON(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("format", formatDetailed, "")
	cmd.Flags().Bool("json", false, "")
	if err := cmd.Flags().Set("format", formatRaw); err != nil {
		t.Fatal(err)
	}
	if !wantsRaw(cmd) || !wantsJSON(cmd) {
		t.Fatal("raw format should select raw JSON output")
	}

	out := captureStdout(func() {
		if err := printJSON(map[string]string{"status": "ok"}); err != nil {
			t.Fatalf("printJSON failed: %v", err)
		}
	})
	var value map[string]string
	if err := json.Unmarshal([]byte(out), &value); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	if value["status"] != "ok" {
		t.Fatalf("unexpected JSON output: %v", value)
	}
}

func TestRenderKeyValueTableAndPRsTabular(t *testing.T) {
	keyValue := captureStdout(func() {
		renderKeyValueTable([][2]string{{"Key", "Value"}})
	})
	if !strings.Contains(keyValue, "FIELD") || !strings.Contains(keyValue, "Value") {
		t.Fatalf("unexpected key-value table: %s", keyValue)
	}

	pr := bitbucket.PullRequest{ID: 7, Title: "Improve output", State: "OPEN"}
	pr.Author.DisplayName = "Alice"
	pr.Source.Branch.Name = "feature"
	pr.Destination.Branch.Name = "main"
	table := captureStdout(func() {
		printPRsTabular("workspace", "repo", []bitbucket.PullRequest{pr})
	})
	for _, want := range []string{"REPOSITORY", "TITLE", "Improve output", "feature", "main"} {
		if !strings.Contains(table, want) {
			t.Fatalf("tabular PR output missing %q: %s", want, table)
		}
	}
}
