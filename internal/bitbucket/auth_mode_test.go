package bitbucket

import "testing"

// helper replicating selection logic (username -> basic else bearer)
func selectAuthMode(username string) string {
	if username != "" {
		return "basic"
	}
	return "bearer"
}

func TestSelectAuthMode(t *testing.T) {
	cases := []struct {
		name     string
		username string
		expect   string
	}{
		{"WithUsernameUsesBasic", "user@example.com", "basic"},
		{"EmptyUsernameUsesBearer", "", "bearer"},
	}

	for _, tc := range cases {
		if got := selectAuthMode(tc.username); got != tc.expect {
			// provide descriptive failure
			if tc.username == "" {
				t.Fatalf("%s: expected bearer when username empty, got %s", tc.name, got)
			}
			f := tc.username
			t.Fatalf("%s: expected %s for username %s, got %s", tc.name, tc.expect, f, got)
		}
	}
}
