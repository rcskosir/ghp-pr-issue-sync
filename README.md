# ghp-pr-sync

This is an application for dealing with GitHub Project Boards. This is intended to make sorting through issues and PRs in large repos faster and easier via the GitHub Project Board.

It can automatically add Issues or PRs to Project Boards based on the repo, organization, and project number. 

When adding Issues to Project Boards with the Issue # and number of Days Open. 

ghp-pr-sync is still a work in progress with more automation to come in future!

## Installation

To install ghp-pr-sync from the command line, you can run:

`go install ghp-pr-sync`

## Commands

## Add issues labeled bug to a GitHub project board and fill the Issue # and Open Days columns
Adds issues labeled bug to a GitHub project board and fill the Issue # and Open Days columns

### Examples

- Add issues labeled bug based on the organization name, repo name, and project number of the board to add it to:
```
go run main.go -o GITHUB_ORG -p GITHUB_PROJECT_NUMBER -r GITHUB_REPO -t GITHUB_TOKEN
```

## Notes

- A GitHub access token is required to make the requests and is set via the environment variable `GITHUB_TOKEN`
- The GitHub CLI tool gh needs to be installed 
