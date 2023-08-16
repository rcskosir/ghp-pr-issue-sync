package gh

import (
	"fmt"
	"github.com/google/go-github/v45/github"
	"github.com/katbyte/ghp-pr-sync/lib/clog"
	"sort"
)

func (r Repo) ListAllIssues(state string, cb func([]*github.Issue, *github.Response) error) error {
	client, ctx := r.NewClient()

	opts := &github.IssueListOptions{
		State: state,
		ListOptions: github.ListOptions{
			Page:    1,
			PerPage: 100,
		},
	}

	for {
		clog.Log.Debugf("Listing all Issues for %s/%s (Page %d)...", r.Owner, r.Name, opts.ListOptions.Page)
		issues, resp, err := client.Issues.List(ctx, true, opts)
		if err != nil {
			return fmt.Errorf("unable to list Issues for %s/%s (Page %d): %w", r.Owner, r.Name, opts.ListOptions.Page, err)
		}

		if err = cb(issues, resp); err != nil {
			return fmt.Errorf("callback failed for %s/%s (Page %d): %w", r.Owner, r.Name, opts.ListOptions.Page, err)
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return nil
}

func (r Repo) GetAllIssues(state string) (*[]github.Issue, error) {
	var allIssues []github.Issue

	err := r.ListAllIssues(state, func(issues []*github.Issue, resp *github.Response) error {
		for index, i := range issues {
			if i == nil {
				clog.Log.Debugf("issues[%d] was nil, skipping", index)
				continue
			}

			n := i.GetNumber()
			if n == 0 {
				clog.Log.Debugf("issues[%d].Number was nil/0, skipping", index)
				continue
			}
			if label := i.GetLabelsURL(); label == "bug" {

			}

			allIssues = append(allIssues, *i)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get all issues for %s/%s: %w", r.Owner, r.Name, err)
	}

	sort.Slice(allIssues, func(i, j int) bool {
		return allIssues[i].GetNumber() < allIssues[j].GetNumber()
	})

	return &allIssues, nil
}
