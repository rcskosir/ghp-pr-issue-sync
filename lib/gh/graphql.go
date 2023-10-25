package gh

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
)

func (t Token) GraphQLQueryUnmarshal(query string, params [][]string, data interface{}) error {
	out, err := t.GraphQLQuery(query, params)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(*out), data)
}

func (t Token) GraphQLQuery(query string, params [][]string) (*string, error) {
	args := []string{"api", "graphql", "-f", query}

	for _, p := range params {
		args = append(args, p[0])
		args = append(args, p[1])
	}

	ghc := exec.Command("gh", args...)
	if t.Token != nil {
		ghc.Env = []string{"GITHUB_TOKEN=" + *t.Token}
	}

	out, err := ghc.CombinedOutput()
	s := string(out)

	if err != nil {
		return &s, fmt.Errorf("graph ql query error: %s\n\n %s\n\n%s", err, query, out)
	}

	return &s, nil
}

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

func (p Project) GetProjectDetails() (*ProjectDetails, error) {
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

	params := [][]string{
		{"-f", "org=" + p.Owner},
		{"-F", "number=" + strconv.Itoa(p.Number)},
	}

	var project ProjectDetails
	if err := p.GraphQLQueryUnmarshal(q, params, &project); err != nil {
		return nil, err
	}

	return &project, nil
}

func (t Token) AddToProject(projectID, nodeID string) (*string, error) {
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

func (r Repo) PRReviewDecision(pr int) (*string, error) {
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
		{"-f", "owner=" + r.Owner},
		{"-f", "repo=" + r.Name},
		{"-F", "pr=" + strconv.Itoa(pr)},
	}

	var approved PRApproval
	if err := r.GraphQLQueryUnmarshal(q, p, &approved); err != nil {
		return nil, err
	}

	return &approved.Data.Repository.PullRequest.ReviewDecision, nil
}
