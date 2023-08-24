package cli

import (
	"fmt"
	"time"

	"github.com/katbyte/ghp-pr-sync/lib/gh"
	"github.com/spf13/cobra"
	//nolint:misspell
	c "github.com/gookit/color"
)

func CmdIssues(_ *cobra.Command, _ []string) error {

	// For each repo get all issues and add to project only bugs
	//Can't add all issues with current limit on number of issues on a project
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
	if totalBugs > 0 {
		c.Printf("Total of %d bugs for on average %d days\n", totalBugs, totalDaysOpen/totalBugs)
	} else {
		c.Printf("Total of 0 bugs\n")
	}

	return nil
}
