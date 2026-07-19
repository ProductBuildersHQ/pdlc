package main

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/ProductBuildersHQ/pdlc"
	"github.com/ProductBuildersHQ/pdlc/layout"
	"github.com/ProductBuildersHQ/pdlc/project"
)

func newLayoutCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "layout",
		Short: "Show the canonical layout contract",
	}
	cmd.AddCommand(newLayoutShowCmd(), newLayoutExportCmd())
	return cmd
}

func newLayoutShowCmd() *cobra.Command {
	var profile string

	cmd := &cobra.Command{
		Use:   "show",
		Short: "List canonical artifact locations and their requirement levels",
		RunE: func(cmd *cobra.Command, _ []string) error {
			manifest, err := pdlc.Layout()
			if err != nil {
				return err
			}

			p := layout.Profile(profile)
			if !p.Valid() {
				return fmt.Errorf("unknown profile %q", profile)
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "DOMAIN\tARTIFACT\tCANONICAL PATH\tAUTHORITY\t%s\n", "LEVEL")
			for _, d := range manifest.Domains {
				artifacts := manifest.ArtifactsInDomain(d.ID)
				if d.QualityOnly {
					fmt.Fprintf(w, "%s\t(quality only)\t-\t-\t%s\n", d.ID, project.DomainLevel(manifest, d.ID, p))
					continue
				}
				for _, a := range artifacts {
					fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", d.ID, a.ID, a.Canonical, a.Authority, a.LevelFor(p))
				}
			}
			return w.Flush()
		},
	}

	cmd.Flags().StringVar(&profile, "profile", string(layout.ProfileFull), "profile to display levels for")
	return cmd
}

func newLayoutExportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "export",
		Short: "Print the embedded layout.yaml contract",
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := cmd.OutOrStdout().Write(pdlc.LayoutYAML())
			return err
		},
	}
}
