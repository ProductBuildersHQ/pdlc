package main

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/ProductBuildersHQ/pdlc/project"
	"github.com/ProductBuildersHQ/pdlc/specplan"
)

func newPlanCmd() *cobra.Command {
	var (
		profileName string
		specsRoot   string
		format      string
	)

	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Show the authoring plan for a spec profile",
		Long: "plan resolves a spec profile (a VisionSpec authoring methodology such as\n" +
			"big-tech-product) into the specs it requires: for each, the canonical location\n" +
			"in the PDLC layout and whether a template and rubric are available. With no\n" +
			"--spec-profile, it uses the authoring spec profiles declared in pdlc.yaml.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			root, err := cmd.Flags().GetString("root")
			if err != nil {
				return err
			}

			names, err := resolveProfileNames(root, profileName)
			if err != nil {
				return err
			}

			var plans []*specplan.Plan
			for _, name := range names {
				p, err := specplan.Resolve(name, specsRoot)
				if err != nil {
					return err
				}
				plans = append(plans, p)
			}

			return renderPlans(cmd, plans, format)
		},
	}

	cmd.Flags().StringVar(&profileName, "spec-profile", "", "spec profile to plan (defaults to pdlc.yaml authoring profiles)")
	cmd.Flags().StringVar(&specsRoot, "specs-root", specplan.DefaultSpecsRoot, "root for spec artifact paths")
	cmd.Flags().StringVarP(&format, "format", "f", "table", "output format: table, json, or yaml")
	return cmd
}

// resolveProfileNames determines which spec profiles to plan: the explicit flag
// if given, else the authoring spec profiles declared in the project manifest.
func resolveProfileNames(root, explicit string) ([]string, error) {
	if explicit != "" {
		return []string{explicit}, nil
	}

	proj, err := project.Load(root)
	if err != nil {
		return nil, fmt.Errorf("%w (or pass --spec-profile)", err)
	}

	var names []string
	for _, sp := range proj.Spec.SpecProfiles {
		if sp.ResolvedProvider() == project.ProviderVisionSpec && sp.ResolvedRole() == project.RoleAuthoring {
			names = append(names, sp.Name)
		}
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("no VisionSpec authoring profiles in %s; pass --spec-profile", project.ManifestFilename)
	}
	return names, nil
}

func renderPlans(cmd *cobra.Command, plans []*specplan.Plan, format string) error {
	out := cmd.OutOrStdout()

	switch format {
	case "json":
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(payload(plans))
	case "yaml":
		enc := yaml.NewEncoder(out)
		defer func() { _ = enc.Close() }()
		return enc.Encode(payload(plans))
	case "table":
		for _, p := range plans {
			fmt.Fprintf(out, "spec profile: %s   (specs root: %s)\n", p.SpecProfile, p.SpecsRoot)
			w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "  SPEC\tPATH\tTEMPLATE\tRUBRIC\n")
			for _, a := range p.Artifacts {
				fmt.Fprintf(w, "  %s\t%s\t%s\t%s\n", a.SpecType, a.Path, yesno(a.HasTemplate), yesno(a.HasRubric))
			}
			if err := w.Flush(); err != nil {
				return err
			}
			fmt.Fprintln(out)
		}
		return nil
	default:
		return fmt.Errorf("unknown format %q (want table, json, or yaml)", format)
	}
}

// payload unwraps a single plan so json/yaml output is an object for one profile
// and an array for several.
func payload(plans []*specplan.Plan) any {
	if len(plans) == 1 {
		return plans[0]
	}
	return plans
}

func yesno(b bool) string {
	if b {
		return "yes"
	}
	return "—"
}
