package cmd

import (
	"strings"
	"testing"

	"devflow/internal/bitbucket"
)

func TestDisplayThread(t *testing.T) {
	parentID := 1
	thread := CommentThread{
		RootComment: &bitbucket.Comment{
			ID: 1,
			User: struct {
				DisplayName string `json:"display_name"`
				UUID        string `json:"uuid"`
			}{DisplayName: "Alice"},
			CreatedOn: "2026-07-01T10:00:00Z",
			Inline: &struct {
				Path string `json:"path"`
				From int    `json:"from,omitempty"`
				To   int    `json:"to,omitempty"`
			}{Path: "main.go", From: 3, To: 5},
		},
		Replies: []*bitbucket.Comment{
			{
				ID: 2,
				User: struct {
					DisplayName string `json:"display_name"`
					UUID        string `json:"uuid"`
				}{DisplayName: "Bob"},
				CreatedOn: "2026-07-01T11:00:00Z",
				Parent:    &struct{ ID int `json:"id"` }{ID: parentID},
				Content: struct {
					Raw string `json:"raw"`
				}{Raw: "Reply body"},
			},
		},
		Resolved: true,
	}
	thread.RootComment.Content.Raw = "Root body"

	out := captureStdout(func() {
		displayThread(&thread, 2)
	})

	if !strings.Contains(out, "[Thread 2] ID: 1 ✅ RESOLVED") {
		t.Fatalf("unexpected thread header: %q", out)
	}
	if !strings.Contains(out, "📄 File: main.go (lines 3-5)") {
		t.Fatalf("expected inline range in output: %q", out)
	}
	if !strings.Contains(out, "Root body") || !strings.Contains(out, "Reply body") {
		t.Fatalf("expected thread bodies in output: %q", out)
	}
	if !strings.Contains(out, "Bob (ID: 2)") {
		t.Fatalf("expected reply metadata in output: %q", out)
	}
}

func TestDisplayThreadUnresolvedSingleLine(t *testing.T) {
	thread := CommentThread{
		RootComment: &bitbucket.Comment{
			ID: 7,
			User: struct {
				DisplayName string `json:"display_name"`
				UUID        string `json:"uuid"`
			}{DisplayName: "Carol"},
			CreatedOn: "2026-07-01T10:00:00Z",
			Inline: &struct {
				Path string `json:"path"`
				From int    `json:"from,omitempty"`
				To   int    `json:"to,omitempty"`
			}{Path: "main.go", To: 9},
		},
		Resolved: false,
	}
	thread.RootComment.Content.Raw = "Another body"

	out := captureStdout(func() {
		displayThread(&thread, 1)
	})

	if !strings.Contains(out, "⚠️  UNRESOLVED") {
		t.Fatalf("expected unresolved marker: %q", out)
	}
	if !strings.Contains(out, "📄 File: main.go (line 9)") {
		t.Fatalf("expected single-line inline annotation: %q", out)
	}
}
