package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/google/go-github/v45/github"
	"github.com/katbyte/ghp-pr-sync/lib/gh"
	"github.com/katbyte/ghp-pr-sync/version"
	"github.com/spf13/cobra"

	//nolint:misspell
	c "github.com/gookit/color"
)

func Make(cmdName string) (*cobra.Command, error) {
	// This is a no-op to avoid accidentally triggering broken builds on malformed commands
	root := &cobra.Command{
		Use:           cmdName + " [command]",
		Short:         cmdName + "is a small utility to TODO",
		Long:          `TODO`,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			f := GetFlags()
			r := gh.NewRepo(f.Owner, f.Repo, f.Token)

			c.Printf("Looking up project details for <green>%s</>/<lightGreen>%d</>...\n", f.Org, f.ProjectNumber)
			project, err := f.GetProjectDetails()
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

			// get all pull requests
			c.Printf("Retrieving all prs for <white>%s</>/<cyan>%s</>...", r.Owner, r.Name)
			prs, err := r.GetAllPullRequests("open")
			if err != nil {
				c.Printf("\n\n <red>ERROR!!</> %s\n", err)
				return nil
			}
			c.Printf(" found <yellow>%d</>\n", len(*prs))

			if len(f.Authors) > 0 {
				// map of microsoft users
				msUserMap := map[string]bool{}
				for _, u := range f.Authors {
					msUserMap[u] = true
				}

				var filteredPRs []github.PullRequest
				for _, pr := range *prs {
					if msUserMap[pr.User.GetLogin()] {
						filteredPRs = append(filteredPRs, pr)
					}
				}

				sort.Slice(filteredPRs, func(i, j int) bool {
					return filteredPRs[i].GetNumber() < filteredPRs[j].GetNumber()
				})

				c.Printf("  Found <lightBlue>%d</> filtered PRs: ", len(filteredPRs))
				for _, pr := range filteredPRs {
					c.Printf("<white>%d</>,", pr.GetNumber())
				}
				c.Printf("\n\n")

				prs = &filteredPRs
			}

			byStatus := map[string][]int{}

			for _, pr := range *prs {
				prNode := *pr.NodeID

				// flat := strings.Replace(strings.Replace(q, "\n", " ", -1), "\t", "", -1)
				c.Printf("Syncing pr <lightCyan>%d</> (<cyan>%s</>) to project.. ", pr.GetNumber(), prNode)
				iid, err := f.AddToProject(pid, prNode)
				if err != nil {
					c.Printf("\n\n <red>ERROR!!</> %s", err)
					continue
				}
				c.Printf("<magenta>%s</>", *iid)

				// figure out status
				// TODO if approved make sure it stays approved
				reviews, err := f.PRReviewDecision(pr.GetNumber())
				if err != nil {
					c.Printf("\n\n <red>ERROR!!</> %s", err)
					continue
				}

				status := ""
				statusText := ""
				// nolint: gocritic
				if *reviews == "APPROVED" {
					statusText = "Approved"
					c.Printf("  <blue>Approved</> <gray>(reviews)</>\n")
				} else if pr.GetState() == "closed" {
					statusText = "Closed"
					c.Printf("  <darkred>In Progress</> <gray>(state)</>\n")
				} else if pr.Milestone != nil && *pr.Milestone.Title == "Blocked" {
					statusText = "Blocked"
					c.Printf("  <red>Blocked</> <gray>(milestone)</>\n")
				} else if pr.GetDraft() {
					statusText = "In Progress"
					c.Printf("  <yellow>In Progress</> <gray>(draft)</>\n")
				} else if pr.GetState() == "" {
					statusText = "In Progress"
					c.Printf("  <yellow>In Progress</> <gray>(state)</>\n")
				} else {
					for _, l := range pr.Labels {
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
						c.Printf("  <green>Waiting for Review</> <gray>(default)</>\n")
					}
				}

				status = statuses[statusText]
				byStatus[statusText] = append(byStatus[statusText], pr.GetNumber())

				q := `query=
					mutation ($project: ID!, $item: ID!, $status_field: ID!, $status_value: String!, $pr_field: ID!, $pr_value: String!
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
					}
				`

				p := [][]string{
					{"-f", "project=" + pid},
					{"-f", "item=" + *iid},
					{"-f", "status_field=" + fields["Status"]},
					{"-f", "status_value=" + status},
					{"-f", "pr_field=" + fields["PR#"]},
					{"-f", fmt.Sprintf("pr_value=%d", *pr.Number)},
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

	if err := configureFlags(root); err != nil {
		return nil, fmt.Errorf("unable to configure flags: %w", err)
	}

	return root, nil
}
