package main

import (
	"fmt"
	"sort"

	"github.com/plexusone/structured-evaluation/rubric"
	"github.com/spf13/cobra"

	"github.com/ProductBuildersHQ/pdlc"
	"github.com/ProductBuildersHQ/pdlc/project"
	"github.com/ProductBuildersHQ/pdlc/readiness"
)

func newCheckCmd() *cobra.Command {
	var (
		write  bool
		strict bool
	)

	cmd := &cobra.Command{
		Use:   "check",
		Short: "Evaluate project readiness and report gaps",
		Long: "check evaluates every in-scope domain for artifact presence and, where a\n" +
			"tool has produced a report, for quality. Domains excluded by the project's\n" +
			"profile are reported as excluded rather than as failures.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			root, err := cmd.Flags().GetString("root")
			if err != nil {
				return err
			}

			manifest, err := pdlc.Layout()
			if err != nil {
				return err
			}
			proj, err := project.Load(root)
			if err != nil {
				return err
			}

			rep, err := readiness.Evaluate(manifest, proj, readiness.Options{
				Root:        root,
				GeneratedBy: "pdlc/" + pdlc.SpecVersion,
			})
			if err != nil {
				return err
			}

			if write {
				dest, err := readiness.Write(root, rep)
				if err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "wrote %s\n\n", dest)
			}

			printReport(cmd, rep)

			if strict && !rep.Pass {
				return fmt.Errorf("readiness check failed")
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&write, "write", true, "write the report to quality/readiness.json")
	cmd.Flags().BoolVar(&strict, "strict", false, "exit non-zero when the project is not ready")
	return cmd
}

func printReport(cmd *cobra.Command, rep *rubric.Rubric) {
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "%s\n\n", rep.Summary)

	for _, c := range rep.Categories {
		fmt.Fprintf(out, "  %s %-18s %s\n", c.Score.Icon(), c.Category, c.Reasoning)
	}

	if excluded, ok := rep.Extensions["excluded"].([]string); ok && len(excluded) > 0 {
		fmt.Fprintf(out, "\n  excluded by profile: %v\n", excluded)
	}

	if cov, ok := rep.Extensions["localeCoverage"].(map[string]float64); ok && len(cov) > 0 {
		locales := make([]string, 0, len(cov))
		for l := range cov {
			locales = append(locales, l)
		}
		sort.Strings(locales)
		fmt.Fprintf(out, "\n  locale coverage:\n")
		for _, l := range locales {
			fmt.Fprintf(out, "    %-8s %.0f%%\n", l, cov[l]*100)
		}
	}

	if len(rep.Findings) == 0 {
		return
	}
	fmt.Fprintf(out, "\nGaps to close:\n")
	for _, f := range rep.Findings {
		fmt.Fprintf(out, "  [%s] %s\n      %s\n", f.Severity, f.Title, f.Recommendation)
	}
}
