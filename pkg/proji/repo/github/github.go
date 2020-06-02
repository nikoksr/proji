package github

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	gh "github.com/google/go-github/v31/github"
)

// GitHub struct holds important data about a github repo
type GitHub struct {
	baseURI     *url.URL
	OwnerName   string
	RepoName    string
	BranchName  string
	TreeEntries []*gh.TreeEntry
	repoSHA     string
	client      *gh.Client
}

// setRepoSHA sets the repoSHA attribute equal to the SHA-1 of the last commit in the current branch
func (g *GitHub) setRepoSHA(ctx context.Context) error {
	if g.BranchName == "" {
		/*
			r := &gh.Repository{}
			r, _, err := g.client.Repositories.Get(ctx, g.OwnerName, g.RepoName)
			if err != nil {
				return err
			}
			g.BranchName = r.GetDefaultBranch()
		*/
		// Default to master branch for now. The above uses too many API calls and Github's API limit gets exceeded
		// too quickly.
		g.BranchName = "master"
	}

	b, _, err := g.client.Repositories.GetBranch(ctx, g.OwnerName, g.RepoName, g.BranchName)
	if err != nil {
		return err
	}
	g.repoSHA = b.GetCommit().GetSHA()
	return nil
}

// New creates a new github repo object
func New(URL *url.URL) (repo.Importer, error) {
	if URL.Hostname() != "github.com" {
		return nil, fmt.Errorf("invalid host %s", URL.Hostname())
	}

	// Extract owner, repo and branch if given
	// Examples:
	//  - /[nikoksr]/[proji]				-> extracts owner and repo name; no branch name
	//  - /[nikoksr]/[proji]/tree/[master]	-> extracts owner, repo and branch name
	r := regexp.MustCompile(`/([^/]+)/([^/]+)(?:/tree/([^/]+))?`)
	specs := r.FindStringSubmatch(URL.Path)

	if specs == nil {
		return nil, fmt.Errorf("could not parse url")
	}

	OwnerName := specs[1]
	RepoName := specs[2]
	BranchName := specs[3]

	if OwnerName == "" || RepoName == "" {
		return nil, fmt.Errorf("could not extract user and/or repository name. Please check the URL")
	}

	g := &GitHub{
		baseURI:     URL,
		OwnerName:   OwnerName,
		RepoName:    RepoName,
		BranchName:  BranchName,
		TreeEntries: make([]*gh.TreeEntry, 0),
		repoSHA:     "",
		client:      gh.NewClient(&http.Client{Timeout: 10 * time.Second}),
	}

	err := g.setRepoSHA(context.Background())
	if err != nil {
		return nil, err
	}
	return g, nil
}

// GetBaseURI returns the base URI of the repo
// You can pass the relative path to a file of that repo to receive the complete raw url for said file.
// Or you pass an empty string resulting in the base of the raw url for files of this repo.
func (g *GitHub) FilePathToRawURI(filePath string) string {
	if strings.HasPrefix(filePath, "/") {
		filePath = filePath[1:]
	}
	return fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", g.OwnerName, g.RepoName, g.BranchName, filePath)
}

// GetTreeEntries gets the paths and types of the repo tree
func (g *GitHub) LoadTreeEntries() error {
	tree, _, err := g.client.Git.GetTree(context.Background(), g.OwnerName, g.RepoName, g.repoSHA, true)
	if err != nil {
		return err
	}
	if tree.GetTruncated() {
		return fmt.Errorf("the response was truncated by Github, which means that the number of items in the tree array exceeded the maximum limit.\n\nClone the repo manually with git and use 'proji class import --directory /path/to/repo' to import the local instance of that repo")
	}
	g.TreeEntries = tree.Entries
	return nil
}

// Owner returns the name of the repo owner
func (g *GitHub) Owner() string { return g.OwnerName }

// Repo returns the name of the repo
func (g *GitHub) Repo() string { return g.RepoName }
