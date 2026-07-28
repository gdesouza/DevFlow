package cmd

import (
	"testing"

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
	if !wantsJSON(cmd) {
		t.Fatal("legacy json flag should remain supported")
	}
}
