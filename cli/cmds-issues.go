package cli

import (
	"fmt"
	"strings"
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

			// TODO write GetProjectFields
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
			for _, repo := range f.Repos {

				// todo support repos that are owner/repo by overriding the param one
				r := gh.NewRepo(f.Owner, repo, f.Token)

				// get all issues
				c.Printf("Retrieving all issues for <white>%s</>/<cyan>%s</>...", r.Owner, r.Name)
				issues, err := r.GetAllIssues("open")

				if err != nil {
					c.Printf("\n\n <red>ERROR!!</> %s\n", err)
					return nil
				}
				c.Printf(" found <yellow>%d</>\n", len(*issues))

				//Currently not interested in the username of the author for issues
				//if len(f.Authors) > 0 {
				//	c.Printf(" filtering on: <yellow>%s:</>\n", f.Authors)
				//
				//	// map of users
				//	msUserMap := map[string]bool{}
				//	for _, u := range f.Authors {
				//		msUserMap[u] = true
				//	}
				//
				//	var filteredIssues []github.Issue
				//	for _, issue := range *issues {
				//		if msUserMap[issue.User.GetLogin()] {
				//			filteredIssues = append(filteredIssues, issue)
				//		}
				//	}
				//
				//	sort.Slice(filteredIssues, func(i, j int) bool {
				//		return filteredIssues[i].GetNumber() < filteredIssues[j].GetNumber()
				//	})
				//
				//	c.Printf("  Found <lightBlue>%d</> filtered PRs: ", len(filteredIssues))
				//	for _, pr := range filteredIssues {
				//		c.Printf("<white>%d</>,", pr.GetNumber())
				//	}
				//	c.Printf("\n\n")
				//
				//	issues = &filteredIssues
				//}

				byStatus := map[string][]int{}

				totalWaiting := 0
				totalDaysWaiting := 0

				for _, issue := range *issues {
					issueNode := *issue.NodeID

					// flat := strings.Replace(strings.Replace(q, "\n", " ", -1), "\t", "", -1)
					//only put issues labeled bug onto the project
					var iid *string
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
							}
						}
					}

					// figure out status
					// TODO if approved make sure it stays approved
					reviews, err := r.PRReviewDecision(issue.GetNumber())
					if err != nil {
						c.Printf("\n\n <red>ERROR!!</> %s", err)
						continue
					}

					daysOpen := int(time.Now().Sub(issue.GetCreatedAt()) / (time.Hour * 24))
					daysWaiting := 0

					status := ""
					statusText := ""
					// nolint: gocritic
					if *reviews == "APPROVED" {
						statusText = "Approved"
						c.Printf("  <blue>Approved</> <gray>(reviews)</>\n")
					} else if issue.GetState() == "closed" {
						statusText = "Closed"
						daysOpen = int(issue.GetClosedAt().Sub(issue.GetCreatedAt()) / (time.Hour * 24))
						c.Printf("  <darkred>Closed</> <gray>(state)</>\n")
					} else if issue.Milestone != nil && *issue.Milestone.Title == "Blocked" {
						statusText = "Blocked"
						c.Printf("  <red>Blocked</> <gray>(milestone)</>\n")
						//} else if issue.GetDraft() {
						//	statusText = "In Progress"
						//	c.Printf("  <yellow>In Progress</> <gray>(draft)</>\n")
					} else if issue.GetState() == "" {
						statusText = "In Progress"
						c.Printf("  <yellow>In Progress</> <gray>(state)</>\n")
					} else {
						for _, l := range issue.Labels {
							if l != nil {
								if *l.Name == "waiting-response" {
									statusText = "Waiting for Response"
									c.Printf("  <lightGreen>Waiting for Response</> <gray>(label)</>\n")
									break
								}
							}
						}

						if statusText == "" {
							statusText = "Waiting for Review"
							c.Printf("  <green>Waiting for Review</> <gray>(default)</>")

							// calculate days waiting
							daysWaiting = daysOpen
							totalWaiting++

							events, err := r.GetAllIssueEvents(*issue.Number)
							if err != nil {
								c.Printf("\n\n <red>ERROR!!</> %s\n", err)
								return nil
							}
							c.Printf(" with <magenta>%d</> events\n", len(*events))

							for _, t := range *events {

								// check for waiting response label removed
								if t.GetEvent() == "unlabeled" {
									if t.Label.GetName() == "waiting-response" {
										daysWaiting = int(time.Now().Sub(t.GetCreatedAt()) / (time.Hour * 24))
										break
									}
								}

								// check for blocked milestone removal
								if t.GetEvent() == "unlabeled" {
									if t.Milestone.GetTitle() == "Blocked" {
										daysWaiting = int(time.Now().Sub(t.GetCreatedAt()) / (time.Hour * 24))
										break
									}
								}
							}

							totalDaysWaiting = totalDaysWaiting + daysWaiting
						}
					}

					status = statuses[statusText]
					byStatus[statusText] = append(byStatus[statusText], issue.GetNumber())

					c.Printf("  open %d days, waiting %d days\n", daysOpen, daysWaiting)

					q := `query=
					mutation (
                      $project:ID!, $item:ID!, 
                      $status_field:ID!, $status_value:String!, 
                      $pr_field:ID!, $pr_value:String!, 
                      $user_field:ID!, $user_value:String!, 
                      $daysOpen_field:ID!, $daysOpen_value:Float!, 
                      $daysWait_field:ID!, $daysWait_value:Float!,
					) {
					  set_status: updateProjectV2ItemFieldValue(input: {
						projectId: $project
						itemId: $item
						fieldId: $status_field
						value: { 
						  singleSelectOptionId: $status_value
						  }
					  }) {
						projectV2Item {
						  id
						  }
					  }
					  set_pr: updateProjectV2ItemFieldValue(input: {
						projectId: $project
						itemId: $item
						fieldId: $pr_field
						value: { 
						  text: $pr_value
						}
					  }) {
						projectV2Item {
						  id
						  }
					  }
                      set_user: updateProjectV2ItemFieldValue(input: {
						projectId: $project
						itemId: $item
						fieldId: $user_field
						value: { 
						  text: $user_value
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
					  set_dwait: updateProjectV2ItemFieldValue(input: {
						projectId: $project
						itemId: $item
						fieldId: $daysWait_field
						value: { 
						  number: $daysWait_value
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
						{"-f", "status_field=" + fields["Status"]},
						{"-f", "status_value=" + status},
						{"-f", "pr_field=" + fields["PR#"]},
						{"-f", fmt.Sprintf("pr_value=%d", *issue.Number)}, // todo string + value
						{"-f", "user_field=" + fields["User"]},
						{"-f", fmt.Sprintf("user_value=%s", issue.User.GetLogin())},
						{"-f", "daysOpen_field=" + fields["Open Days"]},
						{"-F", fmt.Sprintf("daysOpen_value=%d", daysOpen)},
						{"-f", "daysWait_field=" + fields["Waiting Days"]},
						{"-F", fmt.Sprintf("daysWait_value=%d", daysWaiting)},
					}

					out, err := r.GraphQLQuery(q, p)
					if err != nil {
						c.Printf("\n\n <red>ERROR!!</> %s\n%s", err, *out)
						return nil
					}

					c.Printf("\n")

					// TODO remove closed PRs? move them to closed status?
				}

				// output
				for k := range byStatus { // todo sort? format as table? https://github.com/jedib0t/go-pretty
					c.Printf("<cyan>%s</><gray>x%d -</> %s\n", k, len(byStatus[k]), strings.Trim(strings.ReplaceAll(fmt.Sprint(byStatus[k]), " ", ","), "[]"))
				}
				c.Printf("\n")

				c.Printf("Total of %d waiting for on average %d days\n", totalWaiting, totalDaysWaiting/totalWaiting)
			}

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
