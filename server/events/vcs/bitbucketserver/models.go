package bitbucketserver

const (
	PullCreatedHeader        = "pr:opened"
	PullMergedHeader         = "pr:merged"
	PullDeclinedHeader       = "pr:declined"
	PullCommentCreatedHeader = "pr:comment:added"
)

type CommentEvent struct {
	CommonEventData
	Comment *Comment `json:"comment,omitempty" validate:"required"`
}

type PullRequestEvent struct {
	CommonEventData
}

type CommonEventData struct {
	Actor       *Actor       `json:"actor,omitempty" validate:"required"`
	PullRequest *PullRequest `json:"pullRequest,omitempty" validate:"required"`
}

type PullRequest struct {
	ID        *int    `json:"id,omitempty" validate:"required"`
	FromRef   *Ref    `json:"fromRef,omitempty" validate:"required"`
	ToRef     *Ref    `json:"toRef,omitempty" validate:"required"`
	State     *string `json:"state,omitempty" validate:"required"`
	Reviewers []struct {
		Approved *bool `json:"approved,omitempty" validate:"required"`
	} `json:"reviewers,omitempty" validate:"required"`
}

type Ref struct {
	Repository   *Repository `json:"repository,omitempty" validate:"required"`
	DisplayID    *string     `json:"displayId,omitempty" validate:"required"`
	LatestCommit *string     `json:"latestCommit,omitempty" validate:"required"`
}

type Repository struct {
	Slug    *string  `json:"slug,omitempty" validate:"required"`
	Project *Project `json:"project,omitempty" validate:"required"`
}

type Project struct {
	Name *string `json:"name,omitempty" validate:"required"`
	Key  *string `json:"key,omitempty" validate:"required"`
}

type Actor struct {
	Username *string `json:"name,omitempty" validate:"required"`
}

type Comment struct {
	Text *string `json:"text,omitempty" validate:"required"`
}

type Changes struct {
	Values []struct {
		Path struct {
			ToString *string `json:"toString,omitempty" validate:"required"`
		} `json:"path,omitempty" validate:"required"`
	} `json:"values,omitempty" validate:"required"`
	NextPageStart *int  `json:"nextPageStart,omitempty"`
	IsLastPage    *bool `json:"isLastPage,omitempty" validate:"required"`
}
