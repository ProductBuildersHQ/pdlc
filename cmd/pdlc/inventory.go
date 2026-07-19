package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ProductBuildersHQ/pdlc"
)

func newInventoryCmd() *cobra.Command {
	var (
		movesOnly bool
		verbose   bool
	)

	cmd := &cobra.Command{
		Use:   "inventory",
		Short: "Classify repository files against the layout contract",
		Long: "inventory walks the repository, classifies files and directories against the\n" +
			"PDLC layout contract, and reports which are already canonical, which are\n" +
			"misplaced, and which are ambiguous. It never modifies anything; it is the\n" +
			"input to an adoption move plan.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			root, err := cmd.Flags().GetString("root")
			if err != nil {
				return err
			}

			manifest, err := pdlc.Layout()
			if err != nil {
				return err
			}

			inv, err := manifest.Classify(root)
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			moves := inv.Moves()
			ambiguities := inv.Ambiguities()

			if len(moves) > 0 {
				fmt.Fprintf(out, "Proposed moves (%d):\n", len(moves))
				for _, e := range moves {
					fmt.Fprintf(out, "  %-40s → %-32s (%s: %s)\n", e.Path, e.Canonical, e.Confidence, e.Reason)
				}
			} else {
				fmt.Fprintf(out, "No moves needed: every classified artifact is at its canonical path.\n")
			}

			if len(ambiguities) > 0 {
				fmt.Fprintf(out, "\nNeeds human resolution (%d):\n", len(ambiguities))
				for _, e := range ambiguities {
					fmt.Fprintf(out, "  %-40s matched %s", e.Path, e.ArtifactID)
					if len(e.Alternatives) > 0 {
						fmt.Fprintf(out, ", also %v", e.Alternatives)
					}
					if len(e.Conflicts) > 0 {
						fmt.Fprintf(out, ", contests %s with %v", e.Canonical, e.Conflicts)
					}
					fmt.Fprintln(out)
				}
			}

			if possibles := inv.Possibles(); len(possibles) > 0 {
				fmt.Fprintf(out, "\nWeak matches, not proposed as moves (%d)", len(possibles))
				if !verbose {
					fmt.Fprintf(out, " — use --verbose to list\n")
				} else {
					fmt.Fprintln(out, ":")
					for _, e := range possibles {
						fmt.Fprintf(out, "  %-40s looks like %s (%s)\n", e.Path, e.ArtifactID, e.Reason)
					}
				}
			}

			if movesOnly {
				return nil
			}

			fmt.Fprintf(out, "\nConformant artifacts:\n")
			var conformant int
			for _, e := range inv.Entries {
				if e.Conformant {
					fmt.Fprintf(out, "  %-40s %s\n", e.Path, e.ArtifactID)
					conformant++
				}
			}
			if conformant == 0 {
				fmt.Fprintf(out, "  (none)\n")
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&movesOnly, "moves-only", false, "show only proposed moves and ambiguities")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "list weak matches individually")
	return cmd
}
