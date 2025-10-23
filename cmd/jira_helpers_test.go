package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseLabels(t *testing.T) {
	cases := []struct {
		in  string
		out []string
	}{
		{"", nil},
		{"  ", nil},
		{"a,b,c", []string{"a", "b", "c"}},
		{"a, b, ,c ", []string{"a", "b", "c"}},
	}
	for _, c := range cases {
		got := parseLabels(c.in)
		if len(got) != len(c.out) {
			t.Fatalf("parseLabels(%q) => %v, want %v", c.in, got, c.out)
		}
		for i := range got {
			if got[i] != c.out[i] {
				t.Fatalf("parseLabels(%q)[%d] => %q, want %q", c.in, i, got[i], c.out[i])
			}
		}
	}
}

func TestResolveDescriptionInlineAndFile(t *testing.T) {
	// inline only
	s, err := resolveDescription("inline", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s != "inline" {
		t.Fatalf("unexpected value: %q", s)
	}

	// file only
	d := "file content"
	dir := t.TempDir()
	p := filepath.Join(dir, "desc.txt")
	if err := os.WriteFile(p, []byte(d), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	s2, err := resolveDescription("", p)
	if err != nil {
		t.Fatalf("unexpected error reading file: %v", err)
	}
	if s2 != d {
		t.Fatalf("unexpected file content: %q", s2)
	}

	// both provided -> error
	_, err = resolveDescription("x", "y")
	if err == nil {
		t.Fatalf("expected error when both inline and file provided")
	}
}

func TestResolveCommentBody(t *testing.T) {
	// inline only
	s, err := resolveCommentBody("hey", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s != "hey" {
		t.Fatalf("unexpected: %q", s)
	}

	// file only
	d := "comment body"
	dir := t.TempDir()
	p := filepath.Join(dir, "c.txt")
	if err := os.WriteFile(p, []byte(d), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	s2, err := resolveCommentBody("", p)
	if err != nil {
		t.Fatalf("unexpected error reading file: %v", err)
	}
	if s2 != d {
		t.Fatalf("unexpected file content: %q", s2)
	}

	// both -> error
	_, err = resolveCommentBody("a", "b")
	if err == nil {
		t.Fatalf("expected error when both inline and file provided")
	}
}
