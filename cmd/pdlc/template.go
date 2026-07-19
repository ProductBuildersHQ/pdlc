package main

import (
	"github.com/spf13/cobra"

	"github.com/ProductBuildersHQ/pdlc/specplan"
)

func newTemplateCmd() *cobra.Command {
	var profileName string

	cmd := &cobra.Command{
		Use:   "template <spec-type>",
		Short: "Print the template for a spec type, to author from",
		Long: "template emits the template content for a spec type (for example prd or uxd),\n" +
			"resolved for the given spec profile with fallback to the embedded defaults.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			content, err := specplan.Template(profileName, args[0])
			if err != nil {
				return err
			}
			_, err = cmd.OutOrStdout().Write([]byte(content))
			return err
		},
	}

	cmd.Flags().StringVar(&profileName, "spec-profile", "big-tech-product", "spec profile to resolve the template under")
	return cmd
}
