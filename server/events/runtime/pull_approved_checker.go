package runtime

import (
	"github.com/runatlantis/atlantis/server/events/models"
)

//go:generate pegomock generate -m --use-experimental-model-gen --package mocks -o mocks/mock_pull_approved_checker.go PullApprovedChecker

type PullApprovedChecker interface {
	PullIsMergable(baseRepo models.Repo, pullNum int) (bool, error)
	PullIsApproved(baseRepo models.Repo, pull models.PullRequest) (bool, error)
}
