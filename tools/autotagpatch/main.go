package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"strings"
	"time"

	"github.com/google/go-github/v67/github"
)

// `autotagpatch` is a command-line application that automatically tags the most recent commit that matches the following criteria:
// - The commit is on the default branch.
// - The commit is not already tagged.
// - All the commits up until the commit are made by Dependabot, or its commit message contains the string "Merge pull request #".
//
// You usually run this application as a GitHub Actions workflow, daily or weekly, to automatically tag the most recent commit that matches the criteria.

func main() {
	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, os.Interrupt)
	ctx := contextWithSignal(context.Background(), sigch)

	dryRun := flag.Bool("dry-run", false, "dry run")

	flag.Parse()

	log := log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)
	client, err := newClient(log)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	lastTag, err := client.getLastTag(ctx)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	allCommitsSinceLastTag, err := client.getCommitsSinceTag(ctx, lastTag)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	log.Printf("=== Checking commits from the oldest to the most recent, to figure out the commit to tag")
	var mostRecentEligibleCommit *github.RepositoryCommit
	for i := 0; i < len(allCommitsSinceLastTag); i++ {
		commit := allCommitsSinceLastTag[i]
		if client.isEligibleCommit(commit) {
			mostRecentEligibleCommit = commit
		} else {
			break
		}
	}

	if mostRecentEligibleCommit == nil {
		fmt.Println("No eligible commit found")
		os.Exit(0)
	}

	semver, err := parseSemver(*lastTag.Name)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	newTag := fmt.Sprintf("v%d.%d.%d", semver.Major, semver.Minor, semver.Patch+1)

	fmt.Printf("Tag %s would be created for commit %s\n", newTag, *mostRecentEligibleCommit.SHA)
	if !*dryRun {
		if err := client.createAndPushTag(ctx, newTag, mostRecentEligibleCommit); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		fmt.Printf("Tag %s created for commit %s\n", newTag, *mostRecentEligibleCommit.SHA)
	} else {
		fmt.Printf("Dry run: Tag %s created for commit %s\n", newTag, *mostRecentEligibleCommit.SHA)
	}
}

func contextWithSignal(ctx context.Context, sigch chan os.Signal) context.Context {
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		sig := <-sigch
		fmt.Printf("Received signal: %s\n", sig)
		cancel()
	}()

	return ctx
}

type semver struct {
	Major int
	Minor int
	Patch int
}

func parseSemver(tag string) (*semver, error) {
	var major, minor, patch int
	_, err := fmt.Sscanf(tag, "v%d.%d.%d", &major, &minor, &patch)
	if err != nil {
		return nil, fmt.Errorf("failed to parse semver: %w", err)
	}
	return &semver{Major: major, Minor: minor, Patch: patch}, nil
}

type client struct {
	url          string
	owner, repo  string
	githubClient *github.Client
	Log          *log.Logger
}

func newClient(log *log.Logger) (*client, error) {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("GITHUB_TOKEN is not set")
	}

	g := github.NewClient(&http.Client{})
	g = g.WithAuthToken(token)

	ownerRepo := os.Getenv("GITHUB_REPOSITORY")
	if ownerRepo == "" {
		return nil, fmt.Errorf("GITHUB_REPOSITORY is not set")
	}

	ownerRepoParts := strings.Split(ownerRepo, "/")
	if len(ownerRepoParts) != 2 {
		return nil, fmt.Errorf("invalid GITHUB_REPOSITORY format")
	}

	owner, repo := ownerRepoParts[0], ownerRepoParts[1]

	return &client{
		url:          "https://api.github.com",
		owner:        owner,
		repo:         repo,
		githubClient: g,
		Log:          log,
	}, nil
}

func (c *client) getLastTag(ctx context.Context) (*github.RepositoryTag, error) {
	tags, _, err := c.githubClient.Repositories.ListTags(ctx, c.owner, c.repo, &github.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}

	if len(tags) == 0 {
		return nil, fmt.Errorf("no tags found")
	}

	return tags[0], nil
}

// getCommitsSinceTag returns all the commits since the given tag, in an oldest-first order.
func (c *client) getCommitsSinceTag(ctx context.Context, tag *github.RepositoryTag) ([]*github.RepositoryCommit, error) {
	listOptions := github.ListOptions{
		Page:    1,
		PerPage: 100,
	}

	var (
		commits []*github.RepositoryCommit
	)

LOOP:
	for {
		c.Log.Printf("=== Fetching commits since tag %s (%s), page %d", *tag.Name, *tag.Commit.SHA, listOptions.Page)
		cs, res, err := c.githubClient.Repositories.ListCommits(ctx, c.owner, c.repo, &github.CommitsListOptions{
			ListOptions: listOptions,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list commits: %w", err)
		}

		for _, commit := range cs {
			c.Log.Printf("%s", summarizeCommit(commit))
			if *commit.SHA == *tag.Commit.SHA {
				break LOOP
			}
			commits = append(commits, commit)
		}

		if res.NextPage == 0 {
			break
		}

		listOptions.Page = res.NextPage

		time.Sleep(1 * time.Second)
	}

	if len(commits) == 0 {
		return nil, fmt.Errorf("no new commits found since tag %s", *tag.Name)
	}

	slices.Reverse(commits)

	return commits, nil
}

func summarizeCommit(commit *github.RepositoryCommit) string {
	shortSHA := (*commit.SHA)[:7]
	shortMsgLen := 30

	timestamp := commit.Commit.Committer.Date.Format("2006-01-02 15:04:05")

	msgFirstLine := strings.Split(*commit.Commit.Message, "\n")[0]
	if l := len(msgFirstLine); l < shortMsgLen {
		shortMsgLen = l
	}
	shortMessage := msgFirstLine[:shortMsgLen]
	summary := fmt.Sprintf("%s %s %s", shortSHA, timestamp, shortMessage)
	for i := len(summary); i < 38; i++ {
		summary += " "
	}
	return summary
}

func (c *client) isEligibleCommit(commit *github.RepositoryCommit) bool {
	summary := summarizeCommit(commit)

	if commit.Commit.Author.Name != nil && *commit.Commit.Author.Name == "dependabot[bot]" {
		c.Log.Printf("%s: Eligible, because this is created by Dependabot", summary)
		return true
	}

	if strings.Contains(*commit.Commit.Message, "Merge pull request #") && strings.Contains(*commit.Commit.Message, "/dependabot/") {
		c.Log.Printf("%s: Eligible, because this this is merging a dependabot update", summary)
		return true
	}

	c.Log.Printf("%s: Not eligible", summary)

	return false
}

// createAndPushTag creates a lightweight tag and pushes it to the remote repository.
func (c *client) createAndPushTag(ctx context.Context, tag string, commit *github.RepositoryCommit) error {
	_, _, err := c.githubClient.Git.CreateRef(ctx, c.owner, c.repo, &github.Reference{
		Ref: github.String("refs/tags/" + tag),
		Object: &github.GitObject{
			SHA: commit.SHA,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create tag reference: %w", err)
	}

	return nil
}
