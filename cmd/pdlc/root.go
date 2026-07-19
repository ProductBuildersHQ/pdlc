package main

import (
	"github.com/spf13/cobra"

	"github.com/ProductBuildersHQ/pdlc"
)

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pdlc",
		Short: "Evaluate a product project against the PDLC layout contract",
		Long: "pdlc inspects a product project repository, reports where artifacts belong,\n" +
			"and evaluates readiness for handoff to a builder.",
		Version:       pdlc.SpecVersion,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.PersistentFlags().StringP("root", "C", ".", "project repository root")

	cmd.AddCommand(newCheckCmd(), newInventoryCmd(), newInitCmd(), newLayoutCmd())
	return cmd
}
