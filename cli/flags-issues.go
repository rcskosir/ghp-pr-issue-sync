package cli

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type FlagDataIssues struct {
	Token         string
	Org           string
	Owner         string
	Repo          string
	ProjectNumber int
}

// Pulling from one repo, so Repo is a String, not concerned with Authors for Issues
func configureFlagsIssues(root *cobra.Command) error {
	flags := FlagDataIssues{}
	pflags := root.PersistentFlags()

	pflags.StringVarP(&flags.Token, "token", "t", "", "github oauth token (GITHUB_TOKEN)")
	pflags.StringVarP(&flags.Org, "org", "o", "", "github organization (GITHUB_ORG)") // nolint: misspell
	pflags.StringVarP(&flags.Owner, "owner", "", "", "github repo owner, defaults to org (GITHUB_OWNER)")
	pflags.StringVarP(&flags.Repo, "repo", "r", "", "github repo name (GITHUB_REPO)")
	pflags.IntVarP(&flags.ProjectNumber, "project-number", "p", 0, "github project number (GITHUB_PROJECT_NUMBER)")

	// binding map for viper/pflag -> env
	m := map[string]string{
		"token":          "GITHUB_TOKEN",
		"org":            "GITHUB_ORG",
		"owner":          "GITHUB_OWNER",
		"repo":           "GITHUB_REPO",           // todo rename this to repos
		"project-number": "GITHUB_PROJECT_NUMBER", // will I need to add my own project number here?
	}

	for name, env := range m {
		if err := viper.BindPFlag(name, pflags.Lookup(name)); err != nil {
			return fmt.Errorf("error binding '%s' flag: %w", name, err)
		}

		if env != "" {
			if err := viper.BindEnv(name, env); err != nil {
				return fmt.Errorf("error binding '%s' to env '%s' : %w", name, env, err)
			}
		}
	}

	return nil
}

func GetFlagsIssues() FlagDataIssues {
	owner := viper.GetString("owner")
	if owner == "" {
		owner = viper.GetString("org")
	}

	// there has to be an easier way....
	return FlagDataIssues{
		Token:         viper.GetString("token"),
		Org:           viper.GetString("org"),
		Owner:         owner,
		Repo:          viper.GetString("repo"),
		ProjectNumber: viper.GetInt("project-number"),
	}
}
