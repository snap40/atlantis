// Copyright 2017 HootSuite Media Inc.
//
// Licensed under the Apache License, Version 2.0 (the License);
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an AS IS BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// Modified hereafter by contributors to runatlantis/atlantis.

package vcs

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"strings"

	"github.com/google/go-github/github"
	"github.com/pkg/errors"
	"github.com/runatlantis/atlantis/server/events/models"
)

// maxCommentBodySize is derived from the error message when you go over
// this limit.
const maxCommentBodySize = 65536

// GithubClient is used to perform GitHub actions.
type GithubClient struct {
	client *github.Client
	ctx    context.Context
}

// NewGithubClient returns a valid GitHub client.
func NewGithubClient(hostname string, user string, pass string) (*GithubClient, error) {
	tp := github.BasicAuthTransport{
		Username: strings.TrimSpace(user),
		Password: strings.TrimSpace(pass),
	}
	client := github.NewClient(tp.Client())
	// If we're using github.com then we don't need to do any additional configuration
	// for the client. It we're using Github Enterprise, then we need to manually
	// set the base url for the API.
	if hostname != "github.com" {
		baseURL := fmt.Sprintf("https://%s/api/v3/", hostname)
		base, err := url.Parse(baseURL)
		if err != nil {
			return nil, errors.Wrapf(err, "Invalid github hostname trying to parse %s", baseURL)
		}
		client.BaseURL = base
	}

	return &GithubClient{
		client: client,
		ctx:    context.Background(),
	}, nil
}

// GetModifiedFiles returns the names of files that were modified in the pull request.
// The names include the path to the file from the repo root, ex. parent/child/file.txt.
func (g *GithubClient) GetModifiedFiles(repo models.Repo, pull models.PullRequest) ([]string, error) {
	var files []string
	nextPage := 0
	for {
		opts := github.ListOptions{
			PerPage: 300,
		}
		if nextPage != 0 {
			opts.Page = nextPage
		}
		pageFiles, resp, err := g.client.PullRequests.ListFiles(g.ctx, repo.Owner, repo.Name, pull.Num, &opts)
		if err != nil {
			return files, err
		}
		for _, f := range pageFiles {
			files = append(files, f.GetFilename())
		}
		if resp.NextPage == 0 {
			break
		}
		nextPage = resp.NextPage
	}
	return files, nil
}

// CreateComment creates a comment on the pull request.
// If comment length is greater than the max comment length we split into
// multiple comments.
func (g *GithubClient) CreateComment(repo models.Repo, pullNum int, comment string) error {
	comments := g.splitAtMaxChars(comment, maxCommentBodySize, "\ncontinued...\n")
	for _, c := range comments {
		_, _, err := g.client.Issues.CreateComment(g.ctx, repo.Owner, repo.Name, pullNum, &github.IssueComment{Body: &c})
		if err != nil {
			return err
		}
	}
	return nil
}

// PullIsMergeable returns true if the pull request is in a mergable state
func (g *GithubClient) PullIsMergeable(repo models.Repo, pullNum int) (bool, error) {
	pull, _, err := g.client.PullRequests.Get(g.ctx, repo.Owner, repo.Name, pullNum)
	if err != nil {
		return false, errors.Wrap(err, "getting pull request")
	}

	// As far as I can tell possible values for this are 'clean' and 'blocked'
	// but can't find an authoritative source for all values.
	if *pull.MergeableState == "clean" {
		return true, nil
	} else {
		return false, nil
	}
}

// PullIsApproved returns true if the pull request was approved.
func (g *GithubClient) PullIsApproved(repo models.Repo, pull models.PullRequest) (bool, error) {
	reviews, _, err := g.client.PullRequests.ListReviews(g.ctx, repo.Owner, repo.Name, pull.Num, nil)
	if err != nil {
		return false, errors.Wrap(err, "getting reviews")
	}
	for _, review := range reviews {
		if review != nil && review.GetState() == "APPROVED" {
			return true, nil
		}
	}
	return false, nil
}

// GetPullRequest returns the pull request.
func (g *GithubClient) GetPullRequest(repo models.Repo, num int) (*github.PullRequest, error) {
	pull, _, err := g.client.PullRequests.Get(g.ctx, repo.Owner, repo.Name, num)
	return pull, err
}

// UpdateStatus updates the status badge on the pull request.
// See https://github.com/blog/1227-commit-status-api.
func (g *GithubClient) UpdateStatus(repo models.Repo, pull models.PullRequest, state models.CommitStatus, description string) error {
	const statusContext = "Atlantis"
	ghState := "error"
	switch state {
	case models.PendingCommitStatus:
		ghState = "pending"
	case models.SuccessCommitStatus:
		ghState = "success"
	case models.FailedCommitStatus:
		ghState = "failure"
	}
	status := &github.RepoStatus{
		State:       github.String(ghState),
		Description: github.String(description),
		Context:     github.String(statusContext)}
	_, _, err := g.client.Repositories.CreateStatus(g.ctx, repo.Owner, repo.Name, pull.HeadCommit, status)
	return err
}

// splitAtMaxChars splits comment into a slice with string up to max
// len separated by join which gets appended to the ends of the middle strings.
// If max <= len(join) we return an empty slice since this is an edge case we
// don't want to handle.
// nolint: unparam
func (g *GithubClient) splitAtMaxChars(comment string, max int, join string) []string {
	// If we're under the limit then no need to split.
	if len(comment) <= max {
		return []string{comment}
	}

	// If we can't fit the joining string in then this doesn't make sense.
	if max <= len(join) {
		return nil
	}

	var comments []string
	maxSize := max - len(join)
	numComments := int(math.Ceil(float64(len(comment)) / float64(maxSize)))
	for i := 0; i < numComments; i++ {
		upTo := g.min(len(comment), (i+1)*maxSize)
		portion := comment[i*maxSize : upTo]
		if i < numComments-1 {
			portion += join
		}
		comments = append(comments, portion)
	}
	return comments
}

func (g *GithubClient) min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
