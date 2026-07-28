package cmd

import (
	"testing"

	"devflow/internal/bitbucket"
)

func TestPipelineStateLabel(t *testing.T) {
	tests := []struct {
		name string
		p    bitbucket.Pipeline
		want string
	}{
		{
			name: "completed successful",
			p: func() bitbucket.Pipeline {
				var p bitbucket.Pipeline
				p.State.Name = "COMPLETED"
				p.State.Result = &struct {
					Name string `json:"name"`
				}{Name: "SUCCESSFUL"}
				return p
			}(),
			want: "✅ SUCCESSFUL",
		},
		{
			name: "completed without result",
			p: func() bitbucket.Pipeline {
				var p bitbucket.Pipeline
				p.State.Name = "COMPLETED"
				return p
			}(),
			want: "✅ COMPLETED",
		},
		{
			name: "in progress stage",
			p: func() bitbucket.Pipeline {
				var p bitbucket.Pipeline
				p.State.Name = "IN_PROGRESS"
				p.State.Stage = &struct {
					Name string `json:"name"`
				}{Name: "Build"}
				return p
			}(),
			want: "🔄 Build",
		},
		{
			name: "in progress no stage",
			p: func() bitbucket.Pipeline {
				var p bitbucket.Pipeline
				p.State.Name = "IN_PROGRESS"
				return p
			}(),
			want: "🔄 IN_PROGRESS",
		},
		{name: "pending", p: bitbucket.Pipeline{State: struct {
			Name  string `json:"name"`
			Stage *struct {
				Name string `json:"name"`
			} `json:"stage"`
			Result *struct {
				Name string `json:"name"`
			} `json:"result"`
		}{Name: "PENDING"}}, want: "⏳ PENDING"},
		{name: "paused", p: bitbucket.Pipeline{State: struct {
			Name  string `json:"name"`
			Stage *struct {
				Name string `json:"name"`
			} `json:"stage"`
			Result *struct {
				Name string `json:"name"`
			} `json:"result"`
		}{Name: "PAUSED"}}, want: "⏸  PAUSED"},
		{name: "halted", p: bitbucket.Pipeline{State: struct {
			Name  string `json:"name"`
			Stage *struct {
				Name string `json:"name"`
			} `json:"stage"`
			Result *struct {
				Name string `json:"name"`
			} `json:"result"`
		}{Name: "HALTED"}}, want: "🛑 HALTED"},
		{name: "unknown named", p: bitbucket.Pipeline{State: struct {
			Name  string `json:"name"`
			Stage *struct {
				Name string `json:"name"`
			} `json:"stage"`
			Result *struct {
				Name string `json:"name"`
			} `json:"result"`
		}{Name: "QUEUED"}}, want: "📝 QUEUED"},
		{name: "unknown empty", p: bitbucket.Pipeline{}, want: "❓ UNKNOWN"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := pipelineStateLabel(tc.p); got != tc.want {
				t.Fatalf("pipelineStateLabel() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestStepStateLabel(t *testing.T) {
	tests := []struct {
		name string
		step bitbucket.PipelineStep
		want string
	}{
		{
			name: "completed successful",
			step: func() bitbucket.PipelineStep {
				var s bitbucket.PipelineStep
				s.State.Name = "COMPLETED"
				s.State.Result = &struct {
					Name string `json:"name"`
				}{Name: "FAILED"}
				return s
			}(),
			want: "❌ FAILED",
		},
		{
			name: "completed without result",
			step: func() bitbucket.PipelineStep {
				var s bitbucket.PipelineStep
				s.State.Name = "COMPLETED"
				return s
			}(),
			want: "✅ COMPLETED",
		},
		{
			name: "in progress",
			step: func() bitbucket.PipelineStep {
				var s bitbucket.PipelineStep
				s.State.Name = "IN_PROGRESS"
				return s
			}(),
			want: "🔄 RUNNING",
		},
		{name: "pending", step: bitbucket.PipelineStep{State: struct {
			Name   string `json:"name"`
			Result *struct {
				Name string `json:"name"`
			} `json:"result"`
		}{Name: "PENDING"}}, want: "⏳ PENDING"},
		{name: "paused", step: bitbucket.PipelineStep{State: struct {
			Name   string `json:"name"`
			Result *struct {
				Name string `json:"name"`
			} `json:"result"`
		}{Name: "PAUSED"}}, want: "⏸  PAUSED"},
		{name: "halted", step: bitbucket.PipelineStep{State: struct {
			Name   string `json:"name"`
			Result *struct {
				Name string `json:"name"`
			} `json:"result"`
		}{Name: "HALTED"}}, want: "🛑 HALTED"},
		{name: "unknown named", step: bitbucket.PipelineStep{State: struct {
			Name   string `json:"name"`
			Result *struct {
				Name string `json:"name"`
			} `json:"result"`
		}{Name: "QUEUED"}}, want: "QUEUED"},
		{name: "unknown empty", step: bitbucket.PipelineStep{}, want: "❓ UNKNOWN"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := stepStateLabel(tc.step); got != tc.want {
				t.Fatalf("stepStateLabel() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestPipelineResultIcon(t *testing.T) {
	cases := map[string]string{
		"SUCCESSFUL": "✅",
		"FAILED":     "❌",
		"ERROR":      "💥",
		"STOPPED":    "🛑",
		"other":      "📝",
	}
	for in, want := range cases {
		if got := pipelineResultIcon(in); got != want {
			t.Fatalf("pipelineResultIcon(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPipelineAndStepDuration(t *testing.T) {
	var pipeline bitbucket.Pipeline
	var step bitbucket.PipelineStep

	if got := pipelineDuration(pipeline); got != "-" {
		t.Fatalf("pipelineDuration zero = %q", got)
	}
	if got := stepDuration(step); got != "-" {
		t.Fatalf("stepDuration zero = %q", got)
	}

	pipeline.BuildSecondsUsed = 61
	if got := pipelineDuration(pipeline); got != "1m1s" {
		t.Fatalf("pipelineDuration 61 = %q", got)
	}

	step.DurationInSeconds = 120
	if got := stepDuration(step); got != "2m" {
		t.Fatalf("stepDuration 120 = %q", got)
	}
}
