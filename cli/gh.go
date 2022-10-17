package cli

import (
	"strconv"

	"github.com/katbyte/ghp-pr-sync/lib/gh"
)

type ProjectDetails struct {
	Data struct {
		Organization struct { // nolint: misspell
			ProjectV2 struct {
				Id     string
				Fields struct {
					Nodes []struct {
						Id      string
						Name    string
						Options []struct {
							Id   string
							Name string
						}
					}
				}
			}
		}
	}
}

func (f FlagData) GetProjectDetails() (*ProjectDetails, error) {
	t := gh.Token{Token: &f.Token}

	// nolint: misspell
	q := `query=
        query($org: String!, $number: Int!) {
            organization(login: $org){
                projectV2(number: $number) {
                    id
                    fields(first:20) {
                        nodes {
                            ... on ProjectV2Field {
                                id
                                name
                            }
                            ... on ProjectV2SingleSelectField {
                                id
                                name
                                options {
                                    id
                                    name
                                }
                            }
                        }
                    }
                }
            }
        }
    `

	p := [][]string{
		{"-f", "org=" + f.Org},
		{"-F", "number=" + strconv.Itoa(f.ProjectNumber)},
	}

	var project ProjectDetails
	if err := t.GraphQLQueryUnmarshal(q, p, &project); err != nil {
		return nil, err
	}

	return &project, nil
}

func (f FlagData) AddToProject(projectID, nodeID string) (*string, error) {
	t := gh.Token{Token: &f.Token}

	q := `query=
        mutation($project:ID!, $pr:ID!) {
          addProjectV2ItemById(input: {projectId: $project, contentId: $pr}) {
            item {
              id
            }
          }
        }
    `

	p := [][]string{
		{"-f", "project=" + projectID},
		{"-f", "pr=" + nodeID},
		{"--jq", ".data.addProjectV2ItemById.item.id"},
	}

	return t.GraphQLQuery(q, p)
}

type PRApproval struct {
	Data struct {
		Repository struct {
			PullRequest struct {
				Title          string
				ReviewDecision string
			}
		}
	}
}

func (f FlagData) PRReviewDecision(pr int) (*string, error) {
	t := gh.Token{Token: &f.Token}

	q := `query=
        query($owner: String!, $repo: String!, $pr: Int!) {
            repository(name: $repo, owner: $owner) {
                pullRequest(number: $pr) {
                    title
                    reviewDecision
                    state
                    reviews(first: 100) {
                        nodes {
                            state
                            author {
                                login
                            }
                        }
                    }
                }
            }
        }
    `

	p := [][]string{
		{"-f", "owner=" + f.Owner},
		{"-f", "repo=" + f.Repo},
		{"-F", "pr=" + strconv.Itoa(pr)},
	}

	var approved PRApproval
	if err := t.GraphQLQueryUnmarshal(q, p, &approved); err != nil {
		return nil, err
	}

	return &approved.Data.Repository.PullRequest.ReviewDecision, nil
}
