package cli

import (
	"fmt"
	"github.com/katbyte/ghp-pr-sync/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	//nolint:misspell
)

func ValidateParams(params []string) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		for _, p := range params {
			if viper.GetString(p) == "" {
				return fmt.Errorf(p + " parameter can't be empty")
			}
		}

		return nil
	}
}

func Make(cmdName string) (*cobra.Command, error) {
	root := &cobra.Command{
		Use:           cmdName + " [command]",
		Short:         cmdName + "is a small utility to TODO",
		Long:          `TODO`,
		SilenceErrors: true,
		PreRunE:       ValidateParams([]string{"token", "org", "repo", "project-number"}),
		RunE: func(cmd *cobra.Command, args []string) error {
			// call subcommand for PRs, and subcommand for issues
			fmt.Println("Error: must also specify a resource of prs or issues")

			return nil
		},
	}

	root.AddCommand(&cobra.Command{
		Use:           "version",
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(cmdName + " v" + version.Version + "-" + version.GitCommit)
		},
	})
	// TODO validate the Params and args
	root.AddCommand(&cobra.Command{
		Use:           "issues",
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		PreRunE:       ValidateParams([]string{"token", "org", "repo", "project-number"}),
		RunE:          CmdIssues,
	})
	// TODO validate the Params and args
	root.AddCommand(&cobra.Command{
		Use:           "prs",
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		PreRunE:       ValidateParams([]string{"token", "org", "repo", "project-number"}),
		RunE:          CmdPRs,
	})

	if err := configureFlags(root); err != nil {
		return nil, fmt.Errorf("unable to configure flags: %w", err)
	}

	return root, nil
}
