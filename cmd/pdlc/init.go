package main

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/ProductBuildersHQ/pdlc/layout"
	"github.com/ProductBuildersHQ/pdlc/project"
)

func newInitCmd() *cobra.Command {
	var (
		id       string
		title    string
		profile  string
		locales  []string
		srcLoc   string
		overRide bool
	)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create a pdlc.yaml project manifest",
		RunE: func(cmd *cobra.Command, _ []string) error {
			root, err := cmd.Flags().GetString("root")
			if err != nil {
				return err
			}

			if _, err := project.Load(root); err == nil && !overRide {
				return fmt.Errorf("%s already exists (use --force to overwrite)", project.ManifestFilename)
			} else if err != nil && !errors.Is(err, project.ErrNotFound) {
				return err
			}

			if id == "" {
				abs, err := filepath.Abs(root)
				if err != nil {
					return fmt.Errorf("resolve root %q: %w", root, err)
				}
				id = filepath.Base(abs)
			}

			p := layout.Profile(profile)
			if !p.Valid() {
				return fmt.Errorf("unknown profile %q (want minimal, standard, full, or custom)", profile)
			}

			proj := project.New(id, title, p)
			if srcLoc != "" {
				proj.Spec.Locales.Source = srcLoc
				proj.Spec.Locales.Targets = locales
			}

			if err := project.Save(root, "", proj); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "created %s (profile %q)\n",
				filepath.Join(root, project.ManifestFilename), p)
			return nil
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "product identifier (defaults to directory name)")
	cmd.Flags().StringVar(&title, "title", "", "product title")
	cmd.Flags().StringVar(&profile, "profile", string(layout.ProfileStandard), "profile: minimal, standard, full, or custom")
	cmd.Flags().StringVar(&srcLoc, "source-locale", "", "source locale, for example en-US")
	cmd.Flags().StringSliceVar(&locales, "target-locales", nil, "target locales, for example de-DE,ja-JP")
	cmd.Flags().BoolVar(&overRide, "force", false, "overwrite an existing manifest")
	return cmd
}
