package devcycle

import "time"

type codeRabbitFetchInput struct {
	PR any `json:"pr"`
}

type codeRabbitResolveInput struct {
	PR      any                  `json:"pr"`
	Issues  []codeRabbitIssue    `json:"issues,omitempty"`
	Results []codeRabbitFixEntry `json:"results,omitempty"`
}

type codeRabbitWatchSpec struct {
	Kind        string `json:"kind"`
	PR          any    `json:"pr"`
	QuietPeriod string `json:"quiet_period,omitempty"`
}

type codeRabbitIssue struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Body        string `json:"body,omitempty"`
	BodyRef     string `json:"body_ref,omitempty"`
	File        string `json:"file,omitempty"`
	Line        int    `json:"line,omitempty"`
	Severity    string `json:"severity,omitempty"`
	ProviderRef string `json:"provider_ref,omitempty"`
}

type codeRabbitFixEntry struct {
	ID          string `json:"id"`
	Triage      string `json:"triage,omitempty"`
	Resolution  string `json:"resolution,omitempty"`
	ProviderRef string `json:"provider_ref,omitempty"`
}

type codeRabbitFetchOutput struct {
	PR              string            `json:"pr"`
	UnresolvedCount int               `json:"unresolved_count"`
	Issues          []codeRabbitIssue `json:"issues"`
}

type codeRabbitResolveOutput struct {
	PR              string   `json:"pr"`
	ResolvedCount   int      `json:"resolved_count"`
	ResolvedThreads []string `json:"resolved_threads"`
}

type codeRabbitWatchPayload struct {
	PR            string            `json:"pr"`
	Review        codeRabbitReview  `json:"review"`
	ProviderState codeRabbitStatus  `json:"provider_status"`
	SubmittedAt   *time.Time        `json:"submitted_at,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

type codeRabbitReview struct {
	HeadSHA     string `json:"head_sha"`
	ReviewID    string `json:"review_id,omitempty"`
	ReviewState string `json:"review_state"`
}

type codeRabbitStatus struct {
	State     string     `json:"state"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}

type graphQLResponse struct {
	Data   graphQLData    `json:"data"`
	Errors []graphQLError `json:"errors,omitempty"`
}

type graphQLData struct {
	Repository graphQLRepository `json:"repository"`
}

type graphQLRepository struct {
	PullRequest graphQLPullRequest `json:"pullRequest"`
}

type graphQLPullRequest struct {
	HeadRefOid    string            `json:"headRefOid"`
	ReviewThreads graphQLThreadList `json:"reviewThreads"`
	Reviews       graphQLReviewList `json:"reviews"`
}

type graphQLThreadList struct {
	Nodes []graphQLThread `json:"nodes"`
}

type graphQLThread struct {
	ID         string          `json:"id"`
	IsResolved bool            `json:"isResolved"`
	Comments   graphQLComments `json:"comments"`
}

type graphQLComments struct {
	Nodes []graphQLComment `json:"nodes"`
}

type graphQLComment struct {
	Body      string        `json:"body"`
	Path      string        `json:"path"`
	Line      *int          `json:"line"`
	Author    graphQLAuthor `json:"author"`
	CreatedAt *time.Time    `json:"createdAt"`
}

type graphQLAuthor struct {
	Login string `json:"login"`
}

type graphQLReviewList struct {
	Nodes []graphQLReviewNode `json:"nodes"`
}

type graphQLReviewNode struct {
	ID          string        `json:"id"`
	State       string        `json:"state"`
	SubmittedAt *time.Time    `json:"submittedAt"`
	Author      graphQLAuthor `json:"author"`
}

type graphQLError struct {
	Message string `json:"message"`
}
