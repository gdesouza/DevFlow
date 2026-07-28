package cmd

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"devflow/internal/config"
)

type fakeAuthClient struct {
	testAuthErr error
	calls       int
}

func (f *fakeAuthClient) TestAuth() error {
	f.calls++
	return f.testAuthErr
}

func TestRunAuthStatus_SuccessWithWorkspace(t *testing.T) {
	var out bytes.Buffer
	client := &fakeAuthClient{}

	err := runAuthStatus(func() (*config.Config, error) {
		return &config.Config{
			Bitbucket: config.BitbucketConfig{
				Workspace: "workspace",
				Token:     "token",
			},
		}, nil
	}, func(cfg *config.BitbucketConfig) authChecker {
		if cfg.Workspace != "workspace" {
			t.Fatalf("expected workspace to be forwarded, got %q", cfg.Workspace)
		}
		return client
	}, &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.calls != 1 {
		t.Fatalf("expected TestAuth to be called once, got %d", client.calls)
	}
	if got := out.String(); !strings.Contains(got, `Bitbucket authentication is valid for workspace "workspace"`) {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestRunAuthStatus_SuccessWithoutWorkspace(t *testing.T) {
	var out bytes.Buffer
	client := &fakeAuthClient{}

	err := runAuthStatus(func() (*config.Config, error) {
		return &config.Config{
			Bitbucket: config.BitbucketConfig{
				Token: "token",
			},
		}, nil
	}, func(cfg *config.BitbucketConfig) authChecker {
		if cfg.Token != "token" {
			t.Fatalf("expected token to be forwarded, got %q", cfg.Token)
		}
		return client
	}, &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := strings.TrimSpace(out.String()); got != "Bitbucket authentication is valid" {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestRunAuthStatus_ConfigLoadFailure(t *testing.T) {
	var out bytes.Buffer

	err := runAuthStatus(func() (*config.Config, error) {
		return nil, errors.New("boom")
	}, func(cfg *config.BitbucketConfig) authChecker {
		t.Fatal("newClient should not be called on config load failure")
		return nil
	}, &out)
	if err == nil || !strings.Contains(err.Error(), "load config") {
		t.Fatalf("expected load config error, got %v", err)
	}
}

func TestRunAuthStatus_MissingToken(t *testing.T) {
	var out bytes.Buffer

	err := runAuthStatus(func() (*config.Config, error) {
		return &config.Config{
			Bitbucket: config.BitbucketConfig{
				Workspace: "workspace",
			},
		}, nil
	}, func(cfg *config.BitbucketConfig) authChecker {
		t.Fatal("newClient should not be called when token is missing")
		return nil
	}, &out)
	if err == nil || !strings.Contains(err.Error(), "bitbucket token not configured") {
		t.Fatalf("expected missing token error, got %v", err)
	}
}

func TestRunAuthStatus_AuthFailure(t *testing.T) {
	var out bytes.Buffer
	expectedErr := errors.New("unauthorized")
	client := &fakeAuthClient{testAuthErr: expectedErr}

	err := runAuthStatus(func() (*config.Config, error) {
		return &config.Config{
			Bitbucket: config.BitbucketConfig{
				Token: "token",
			},
		}, nil
	}, func(cfg *config.BitbucketConfig) authChecker {
		return client
	}, &out)
	if err == nil || !strings.Contains(err.Error(), "bitbucket authentication failed") {
		t.Fatalf("expected auth failure error, got %v", err)
	}
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected wrapped error %v, got %v", expectedErr, err)
	}
}

func TestRunAuthStatus_JSONOutput(t *testing.T) {
	for _, workspace := range []string{"workspace", ""} {
		t.Run(workspace, func(t *testing.T) {
			var out bytes.Buffer
			err := runAuthStatusWithFormat(func() (*config.Config, error) {
				return &config.Config{Bitbucket: config.BitbucketConfig{
					Workspace: workspace,
					Token:     "token",
				}}, nil
			}, func(cfg *config.BitbucketConfig) authChecker {
				return &fakeAuthClient{}
			}, &out, true)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.Contains(out.String(), `"authenticated": true`) || !strings.Contains(out.String(), workspace) {
				t.Fatalf("unexpected JSON output: %q", out.String())
			}
		})
	}
}
