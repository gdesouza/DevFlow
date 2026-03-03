package cmd

import (
	"testing"
)

func TestFormatBuildStatus(t *testing.T) {
	tests := []struct {
		result   string
		building bool
		expect   string
	}{
		{"SUCCESS", false, "✅ SUCCESS"},
		{"FAILURE", false, "❌ FAILURE"},
		{"UNSTABLE", false, "⚠️  UNSTABLE"},
		{"ABORTED", false, "🛑 ABORTED"},
		{"", false, "❓ "},
		{"RANDOM", false, "❓ RANDOM"},
		{"SUCCESS", true, "🔄 BUILDING"},
		{"FAILURE", true, "🔄 BUILDING"},
	}

	for _, tt := range tests {
		got := formatBuildStatus(tt.result, tt.building)
		if got != tt.expect {
			t.Errorf("formatBuildStatus(%q, %v) = %q; want %q", tt.result, tt.building, got, tt.expect)
		}
	}
}

func TestFormatTimestamp(t *testing.T) {
	// 1640000000000 ms after epoch = 2021-12-20 09:13:20 UTC
	ts := int64(1640000000000)
	expect := "2021-12-20 06:33:20"
	got := formatTimestamp(ts)
	if got != expect {
		t.Errorf("formatTimestamp(%v) = %q; want %q", ts, got, expect)
	}
}

func TestFormatDuration(t *testing.T) {
	cases := []struct {
		dur    int64
		expect string
	}{
		{45000, "45.0s"},
		{61000, "1.0m"},
		{125000, "2.1m"},
		{1000, "1.0s"},
	}
	for _, c := range cases {
		got := formatDuration(c.dur)
		if got != c.expect {
			t.Errorf("formatDuration(%d) = %q; want %q", c.dur, got, c.expect)
		}
	}
}

func TestRepeat(t *testing.T) {
	if got := repeat("-", 5); got != "-----" {
		t.Errorf("repeat('-', 5) = %q; want '-----'", got)
	}
}
