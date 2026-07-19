package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ProductBuildersHQ/pdlc/specplan"
)

func newSpecProfilesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "spec-profiles",
		Aliases: []string{"workflows"},
		Short:   "List available spec profiles (VisionSpec authoring methodologies)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := cmd.OutOrStdout()
			for _, name := range specplan.ListProfiles() {
				fmt.Fprintln(out, name)
			}
			return nil
		},
	}
	return cmd
}
