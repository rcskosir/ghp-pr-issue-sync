package cli

import (
	"fmt"
	"time"

	"github.com/katbyte/ghp-pr-sync/lib/gh"
	"github.com/katbyte/ghp-pr-sync/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	//nolint:misspell
	c "github.com/gookit/color"
)

func ValidateParamsIssues(params []string) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		for _, p := range params {
			if viper.GetString(p) == "" {
				return fmt.Errorf(p + " parameter can't be empty")
			}
		}

		return nil
	}
}

func MakeIssues(cmdName string) (*cobra.Command, error) {
	root := &cobra.Command{
		Use:           cmdName + " [command]",
		Short:         cmdName + "is a small utility to TODO",
		Long:          `TODO`,
		SilenceErrors: true,
		PreRunE:       ValidateParamsIssues([]string{"token", "org", "repo", "project-number"}),
		RunE: func(cmd *cobra.Command, args []string) error {
			/**GetFlags gets the Owner, Project Number and token, my Project number is 188
			Right now this runs for the PR project, so I need to either duplicate everything to pull
			my project number, or I need a loop that does PR, then Issue projects
			**/
			f := GetFlagsIssues()
			p := gh.NewProject(f.Owner, f.ProjectNumber, f.Token)

			c.Printf("Looking up project details for <green>%s</>/<lightGreen>%d</>...\n", f.Org, f.ProjectNumber)
			project, err := p.GetProjectDetails()
			if err != nil {
				c.Printf("\n\n <red>ERROR!!</> %s", err)
				return nil
			}
			pid := project.Data.Organization.ProjectV2.Id
			c.Printf("  ID: <magenta>%s</>\n", pid)

			statuses := map[string]string{}
			fields := map[string]string{}

			for _, f := range project.Data.Organization.ProjectV2.Fields.Nodes {
				fields[f.Name] = f.Id
				c.Printf("    <lightBlue>%s</> <> <lightCyan>%s</>\n", f.Name, f.Id)

				if f.Name == "Status" {
					for _, s := range f.Options {
						statuses[s.Name] = s.Id
						c.Printf("      <blue>%s</> <> <cyan>%s</>\n", s.Name, s.Id)
					}
				}
			}
			fmt.Println()

			// For each repo get all issues and add to project only bugs
			//Can't add all issues with current limit on number of issues on a project

			r := gh.NewRepo(f.Owner, f.Repo, f.Token)

			// get all issues
			c.Printf("Retrieving all issues for <white>%s</>/<cyan>%s</>...", r.Owner, r.Name)
			issues, err := r.GetAllIssues("open")

			if err != nil {
				c.Printf("\n\n <red>ERROR!!</> %s\n", err)
				return nil
			}
			c.Printf(" found <yellow>%d</>\n", len(*issues))

			//Currently not interested in the username of the author for issues, so I removed the code for now
			// TODO calculate days open for 365 average (number of bugs from last 365/number of days open for last 365 days)
			totalBugs := 0
			daysOpen := 0
			totalDaysOpen := 0

			for _, issue := range *issues {
				issueNode := *issue.NodeID

				//only put issues labeled bug into the project, therefore graphyQL is inside this loop
				for _, l := range issue.Labels {
					if l != nil {
						if *l.Name == "bug" {
							c.Printf("Syncing issue <lightCyan>%d</> (<cyan>%s</>) to project.. ", issue.GetNumber(), issueNode)
							iid, err := p.AddToProject(pid, issueNode)
							if err != nil {
								c.Printf("\n\n <red>ERROR!!</> %s", err)
								continue
							}
							c.Printf("<magenta>%s</>", *iid)

							totalBugs++
							daysOpen = int(time.Now().Sub(issue.GetCreatedAt()) / (time.Hour * 24))
							totalDaysOpen = totalDaysOpen + daysOpen

							//statuses and waiting days code removed

							c.Printf("  open %d days\n", daysOpen)
							//TODO issue with set_issue, number vs text vs int vs String
							q := `query=
								mutation (
								  $project:ID!, $item:ID!, 
								  $issue_field:ID!, $issue_value:String!, 
								  $daysOpen_field:ID!, $daysOpen_value:Float!, 
								) {
								  set_issue: updateProjectV2ItemFieldValue(input: {
									projectId: $project
									itemId: $item
									fieldId: $issue_field
									value: { 
									  text: $issue_value
									}
								  }) {
									projectV2Item {
									  id
									  }
								  }
								  set_dopen: updateProjectV2ItemFieldValue(input: {
									projectId: $project
									itemId: $item
									fieldId: $daysOpen_field
									value: { 
									  number: $daysOpen_value
									}
								  }) {
									projectV2Item {
									  id
									  }
								  }
								}
				`

							p := [][]string{
								{"-f", "project=" + pid},
								{"-f", "item=" + *iid},
								{"-f", "issue_field=" + fields["Issue#"]},
								{"-f", fmt.Sprintf("issue_value=%d", *issue.Number)},
								{"-f", "daysOpen_field=" + fields["Open Days"]},
								{"-F", fmt.Sprintf("daysOpen_value=%d", daysOpen)},
							}

							out, err := r.GraphQLQuery(q, p)
							if err != nil {
								c.Printf("\n\n <red>ERROR!!</> %s\n%s", err, *out)
								return nil
							}

							c.Printf("\n")
						}
					}
				}
				// no PR review decision for Issues, removed code
			}
			// output
			//totalDaysOpen is for ALL bugs, so this will not match the metrics that only track last 365 days.
			//TODO check for 0, because if there are 0 bugs this will cause a panic
			c.Printf("Total of %d bugs for on average %d days\n", totalBugs, totalDaysOpen/totalBugs)

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

	if err := configureFlagsIssues(root); err != nil {
		return nil, fmt.Errorf("unable to configure flags: %w", err)
	}

	return root, nil
}
