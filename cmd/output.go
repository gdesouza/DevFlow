package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const (
	formatJSON     = "json"
	formatRaw      = "raw"
	formatTabular  = "tabular"
	formatDetailed = "detailed"
)

var outputFormat = formatDetailed

func formatFor(cmd *cobra.Command) string {
	format, err := cmd.Flags().GetString("format")
	if err != nil || format == "" {
		return formatDetailed
	}
	return format
}

func wantsJSON(cmd *cobra.Command) bool {
	if cmd.Flags().Changed("format") {
		return formatFor(cmd) == formatJSON || formatFor(cmd) == formatRaw
	}
	if formatFor(cmd) == formatJSON || formatFor(cmd) == formatRaw {
		return true
	}
	// Keep the existing flag as a backwards-compatible alias.
	legacyJSON, err := cmd.Flags().GetBool("json")
	return err == nil && legacyJSON
}

func wantsRaw(cmd *cobra.Command) bool {
	if cmd.Flags().Changed("format") {
		return formatFor(cmd) == formatRaw
	}
	if formatFor(cmd) == formatRaw {
		return true
	}
	legacyJSON, err := cmd.Flags().GetBool("json")
	return err == nil && legacyJSON
}

func wantsTabular(cmd *cobra.Command) bool {
	if cmd.Flags().Changed("format") {
		return formatFor(cmd) == formatTabular
	}
	return false
}

func printJSON(value any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func validateFormat(cmd *cobra.Command, _ []string) error {
	// Keep legacy switches available for compatibility while directing users to
	// the canonical format selector. Marking them here also covers commands
	// declared in separate files without duplicating setup code.
	if cmd.Flags().Lookup("json") != nil {
		_ = cmd.Flags().MarkDeprecated("json", "use --format raw instead")
	}
	if cmd.Flags().Lookup("tabular") != nil {
		_ = cmd.Flags().MarkDeprecated("tabular", "use --format tabular instead")
	}
	switch formatFor(cmd) {
	case formatJSON, formatRaw, formatTabular, formatDetailed:
		return nil
	default:
		return fmt.Errorf("invalid format %q: must be one of json, raw, tabular, or detailed", formatFor(cmd))
	}
}
