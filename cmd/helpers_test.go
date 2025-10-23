package cmd

import (
	"devflow/internal/bitbucket"
	"fmt"
	"testing"
	"time"
)

func TestFirstLine(t *testing.T) {
	cases := []struct {
		in  string
		out string
	}{
		{"single line", "single line"},
		{"first line\nsecond line", "first line"},
		{"with\rcarriage", "with"},
	}
	for _, c := range cases {
		if got := firstLine(c.in); got != c.out {
			t.Fatalf("firstLine(%q) = %q, want %q", c.in, got, c.out)
		}
	}
}

func TestStatusStateIcon(t *testing.T) {
	cases := map[string]string{
		"SUCCESSFUL": "✅",
		"FAILED":     "❌",
		"ERROR":      "❌",
		"INPROGRESS": "🔄",
		"PENDING":    "🔄",
		"STOPPED":    "🚫",
		"CANCELLED":  "🚫",
		"UNKNOWN":    "📝",
	}
	for in, want := range cases {
		if got := statusStateIcon(in); got != want {
			t.Fatalf("statusStateIcon(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestShortHash(t *testing.T) {
	if got := shortHash("abcdef1234567890"); got != "abcdef123456" {
		t.Fatalf("shortHash mismatch: %s", got)
	}
	if got := shortHash("short"); got != "short" {
		t.Fatalf("shortHash mismatch: %s", got)
	}
}

func TestDisplayName(t *testing.T) {
	st := &bitbucket.CommitStatus{Key: "ci/build", Name: ""}
	if got := displayName(st); got != "ci/build" {
		t.Fatalf("displayName got %s", got)
	}
	st.Name = "CI Build"
	if got := displayName(st); got != "CI Build" {
		t.Fatalf("displayName got %s", got)
	}
}

func TestValidateSetStatusInputs(t *testing.T) {
	// Valid input
	if err := validateSetStatusInputs("SUCCESSFUL", "ci/key", "name", "https://example.com"); err != nil {
		t.Fatalf("expected valid inputs, got %v", err)
	}
	// Missing state
	if err := validateSetStatusInputs("", "k", "n", ""); err == nil {
		t.Fatalf("expected error for empty state")
	}
	// Invalid state
	if err := validateSetStatusInputs("BAD", "k", "", ""); err == nil {
		t.Fatalf("expected error for invalid state")
	}
	// Long key
	longKey := make([]byte, 101)
	for i := range longKey {
		longKey[i] = 'a'
	}
	if err := validateSetStatusInputs("SUCCESSFUL", string(longKey), "", ""); err == nil {
		t.Fatalf("expected error for long key")
	}
	// Invalid url
	if err := validateSetStatusInputs("SUCCESSFUL", "k", "", "ht!tp"); err == nil {
		t.Fatalf("expected error for invalid url")
	}
}

func TestCleanDescriptionAndADF(t *testing.T) {
	html := "<p>Hello <strong>World</strong><br/>Line2</p>"
	if got := cleanDescription(html); got != "Hello World\nLine2" {
		t.Fatalf("cleanDescription html mismatch: %q", got)
	}
	adf := `type: doc content: [ { type: paragraph, content: [ { type: text, text: "Hello" } ] } ]`
	if got := extractTextFromADF(adf); got == "" {
		t.Fatalf("expected non-empty ADF extraction")
	}
}

func TestFormatFileSize(t *testing.T) {
	cases := []struct {
		in  int64
		out string
	}{
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1024 * 1024, "1.0 MB"},
	}
	for _, c := range cases {
		if got := formatFileSize(c.in); got != c.out {
			t.Fatalf("formatFileSize(%d) = %q, want %q", c.in, got, c.out)
		}
	}
}

func TestRelativeTime(t *testing.T) {
	// Known recent time: now - 30s -> "just now"
	now := time.Now()
	recent := now.Format(time.RFC3339)
	if got := relativeTime(recent); got != "just now" {
		// allow small timing differences by accepting either just now or 0m ago
		if got != "0m ago" {
			t.Fatalf("relativeTime recent got %q", got)
		}
	}
	// older: 2 hours ago
	old := now.Add(-2 * time.Hour).Format(time.RFC3339)
	if got := relativeTime(old); got != "2h ago" {
		t.Fatalf("relativeTime 2h got %q", got)
	}
	// fallback unknown
	if got := relativeTime(""); got != "unknown" {
		t.Fatalf("expected unknown, got %s", got)
	}
}

// Simple helper to avoid flakiness in CI when time.Now() shifts between calls.
func TestRelativeTimeMillisFallback(t *testing.T) {
	// provide milliseconds since epoch as string
	ms := fmt.Sprintf("%d", time.Now().Add(-3*time.Hour).Unix()*1000)
	if got := relativeTime(ms); got != "3h ago" {
		t.Fatalf("expected 3h ago, got %s", got)
	}
}
