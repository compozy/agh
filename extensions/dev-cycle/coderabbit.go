package devcycle

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	watchpkg "github.com/compozy/agh/internal/loop/watch"
)

const codeRabbitBotLogin = "coderabbitai"

type codeRabbitProvider struct {
	runner commandRunner
}

func newCodeRabbitProvider(runner commandRunner) *codeRabbitProvider {
	return &codeRabbitProvider{runner: runner}
}

func (p *codeRabbitProvider) FetchUnresolved(
	ctx context.Context,
	input codeRabbitFetchInput,
) (codeRabbitFetchOutput, error) {
	if err := p.requireGH(); err != nil {
		return codeRabbitFetchOutput{}, err
	}
	pr, err := normalizePR(input.PR)
	if err != nil {
		return codeRabbitFetchOutput{}, err
	}
	repo, err := p.currentRepository(ctx)
	if err != nil {
		return codeRabbitFetchOutput{}, err
	}
	payload, err := p.fetchPRGraph(ctx, repo, pr)
	if err != nil {
		return codeRabbitFetchOutput{}, err
	}
	issues := unresolvedCodeRabbitIssues(payload.Data.Repository.PullRequest.ReviewThreads.Nodes)
	return codeRabbitFetchOutput{
		PR:              pr,
		UnresolvedCount: len(issues),
		Issues:          issues,
	}, nil
}

func (p *codeRabbitProvider) ResolveThreads(
	ctx context.Context,
	input codeRabbitResolveInput,
) (codeRabbitResolveOutput, error) {
	if err := p.requireGH(); err != nil {
		return codeRabbitResolveOutput{}, err
	}
	pr, err := normalizePR(input.PR)
	if err != nil {
		return codeRabbitResolveOutput{}, err
	}
	threadIDs := resolveThreadIDs(input.Issues, input.Results)
	for _, threadID := range threadIDs {
		if _, err := p.runner.Run(ctx, "gh", []string{
			"api", "graphql", "-f",
			"query=mutation($id:ID!){resolveReviewThread(input:{threadId:$id}){thread{id isResolved}}}",
			"-F", "id=" + threadID,
		}, ""); err != nil {
			return codeRabbitResolveOutput{}, err
		}
	}
	return codeRabbitResolveOutput{PR: pr, ResolvedCount: len(threadIDs), ResolvedThreads: threadIDs}, nil
}

func (p *codeRabbitProvider) Poll(
	ctx context.Context,
	req watchpkg.PollRequest,
	spec codeRabbitWatchSpec,
) (watchpkg.PollResponse, error) {
	if err := p.requireGH(); err != nil {
		return watchpkg.PollResponse{}, err
	}
	pr, err := normalizePR(spec.PR)
	if err != nil {
		return watchpkg.PollResponse{}, err
	}
	repo, err := p.currentRepository(ctx)
	if err != nil {
		return watchpkg.PollResponse{}, err
	}
	payload, err := p.fetchPRGraph(ctx, repo, pr)
	if err != nil {
		return watchpkg.PollResponse{}, err
	}
	responsePayload := buildWatchPayload(pr, payload.Data.Repository.PullRequest)
	encoded, err := json.Marshal(responsePayload)
	if err != nil {
		return watchpkg.PollResponse{}, fmt.Errorf("dev-cycle: encode watch payload: %w", err)
	}
	digest := digestBytes(encoded)
	if req.ExpectedStateDigest == digest {
		return watchpkg.PollResponse{Ready: false, StateDigest: digest}, nil
	}
	ready := responsePayload.ProviderState.State == "current_reviewed" ||
		responsePayload.ProviderState.State == "current_settled"
	var settledAt *time.Time
	if ready {
		settledAt = quietPeriodDeadline(responsePayload, spec.QuietPeriod)
	}
	return watchpkg.PollResponse{
		Ready:       ready,
		StateDigest: digest,
		Payload:     json.RawMessage(encoded),
		SettledAt:   settledAt,
	}, nil
}

func (p *codeRabbitProvider) requireGH() error {
	if _, err := p.runner.LookPath("gh"); err != nil {
		return fmt.Errorf("dev-cycle: gh executable is required for CodeRabbit review tools: %w", err)
	}
	return nil
}

func (p *codeRabbitProvider) currentRepository(ctx context.Context) (string, error) {
	output, err := p.runner.Run(ctx, "gh", []string{
		"repo", "view", "--json", "nameWithOwner", "-q", ".nameWithOwner",
	}, "")
	if err == nil {
		repo := strings.TrimSpace(string(output))
		if repo != "" {
			return repo, nil
		}
	}
	remote, remoteErr := p.runner.Run(ctx, "git", []string{"remote", "get-url", defaultGitRemote}, "")
	if remoteErr != nil {
		return "", errors.Join(err, remoteErr)
	}
	return parseGitHubRemote(strings.TrimSpace(string(remote)))
}

func (p *codeRabbitProvider) fetchPRGraph(ctx context.Context, repo string, pr string) (graphQLResponse, error) {
	owner, name, err := splitRepo(repo)
	if err != nil {
		return graphQLResponse{}, err
	}
	query := `
query($owner:String!,$repo:String!,$pr:Int!){
  repository(owner:$owner,name:$repo){
    pullRequest(number:$pr){
      headRefOid
      reviewThreads(first:100){nodes{id isResolved comments(first:20){nodes{body path line author{login} createdAt}}}}
      reviews(first:100){nodes{id state submittedAt author{login}}}
    }
  }
}`
	output, err := p.runner.Run(ctx, "gh", []string{
		"api", "graphql",
		"-f", "query=" + query,
		"-F", "owner=" + owner,
		"-F", "repo=" + name,
		"-F", "pr=" + pr,
	}, "")
	if err != nil {
		return graphQLResponse{}, err
	}
	var response graphQLResponse
	if err := json.Unmarshal(output, &response); err != nil {
		return graphQLResponse{}, fmt.Errorf("dev-cycle: decode GitHub GraphQL response: %w", err)
	}
	if len(response.Errors) > 0 {
		return graphQLResponse{}, fmt.Errorf("dev-cycle: GitHub GraphQL error: %s", response.Errors[0].Message)
	}
	return response, nil
}

func unresolvedCodeRabbitIssues(threads []graphQLThread) []codeRabbitIssue {
	issues := make([]codeRabbitIssue, 0)
	for _, thread := range threads {
		if thread.IsResolved {
			continue
		}
		comment, ok := firstCodeRabbitComment(thread.Comments.Nodes)
		if !ok {
			continue
		}
		issue := codeRabbitIssue{
			ID:          stableIssueID(thread.ID),
			Title:       summarizeTitle(comment.Body),
			Body:        strings.TrimSpace(comment.Body),
			BodyRef:     digestString(comment.Body),
			File:        strings.TrimSpace(comment.Path),
			Severity:    "review",
			ProviderRef: thread.ID,
		}
		if comment.Line != nil {
			issue.Line = *comment.Line
		}
		issues = append(issues, issue)
	}
	return issues
}

func firstCodeRabbitComment(comments []graphQLComment) (graphQLComment, bool) {
	for _, comment := range comments {
		if strings.EqualFold(strings.TrimSpace(comment.Author.Login), codeRabbitBotLogin) {
			return comment, true
		}
	}
	return graphQLComment{}, false
}

func buildWatchPayload(pr string, pullRequest graphQLPullRequest) codeRabbitWatchPayload {
	review := latestCodeRabbitReview(pullRequest.Reviews.Nodes)
	state := "current_settled"
	switch strings.ToUpper(strings.TrimSpace(review.State)) {
	case "COMMENTED", "CHANGES_REQUESTED", "APPROVED":
		state = "current_reviewed"
	case "":
		state = "pending"
	}
	return codeRabbitWatchPayload{
		PR: pr,
		Review: codeRabbitReview{
			HeadSHA:     pullRequest.HeadRefOid,
			ReviewID:    review.ID,
			ReviewState: strings.ToLower(strings.TrimSpace(review.State)),
		},
		ProviderState: codeRabbitStatus{
			State:     state,
			UpdatedAt: review.SubmittedAt,
		},
		SubmittedAt: review.SubmittedAt,
		Metadata: map[string]string{
			"provider": "coderabbit",
		},
	}
}

func latestCodeRabbitReview(reviews []graphQLReviewNode) graphQLReviewNode {
	matches := make([]graphQLReviewNode, 0)
	for _, review := range reviews {
		if strings.EqualFold(strings.TrimSpace(review.Author.Login), codeRabbitBotLogin) {
			matches = append(matches, review)
		}
	}
	slices.SortFunc(matches, func(a, b graphQLReviewNode) int {
		if a.SubmittedAt == nil && b.SubmittedAt == nil {
			return 0
		}
		if a.SubmittedAt == nil {
			return -1
		}
		if b.SubmittedAt == nil {
			return 1
		}
		return a.SubmittedAt.Compare(*b.SubmittedAt)
	})
	if len(matches) == 0 {
		return graphQLReviewNode{}
	}
	return matches[len(matches)-1]
}

func resolveThreadIDs(issues []codeRabbitIssue, results []codeRabbitFixEntry) []string {
	accepted := make(map[string]bool)
	for _, result := range results {
		resolution := strings.ToLower(strings.TrimSpace(result.Resolution))
		triage := strings.ToLower(strings.TrimSpace(result.Triage))
		if result.ProviderRef != "" && codeRabbitResolutionAccepted(resolution, triage) {
			accepted[result.ProviderRef] = true
		}
		if result.ID != "" && codeRabbitResolutionAccepted(resolution, triage) {
			accepted[result.ID] = true
		}
	}
	threadIDs := make([]string, 0)
	for _, issue := range issues {
		if issue.ProviderRef == "" {
			continue
		}
		if len(results) == 0 || accepted[issue.ID] || accepted[issue.ProviderRef] {
			threadIDs = append(threadIDs, issue.ProviderRef)
		}
	}
	slices.Sort(threadIDs)
	return slices.Compact(threadIDs)
}

func codeRabbitResolutionAccepted(resolution string, triage string) bool {
	return resolution == codeRabbitFixed ||
		resolution == codeRabbitDocumented ||
		triage == codeRabbitTriageValid
}

func normalizePR(value any) (string, error) {
	switch typed := value.(type) {
	case float64:
		if typed <= 0 || typed != float64(int64(typed)) {
			return "", fmt.Errorf("dev-cycle: pr must be a positive integer")
		}
		return strconv.FormatInt(int64(typed), 10), nil
	case int:
		if typed <= 0 {
			return "", fmt.Errorf("dev-cycle: pr must be a positive integer")
		}
		return strconv.Itoa(typed), nil
	case string:
		pr := strings.TrimSpace(strings.TrimPrefix(typed, "#"))
		if pr == "" {
			return "", fmt.Errorf("dev-cycle: pr is required")
		}
		if _, err := strconv.Atoi(pr); err != nil {
			return "", fmt.Errorf("dev-cycle: pr must be numeric: %w", err)
		}
		return pr, nil
	default:
		return "", fmt.Errorf("dev-cycle: unsupported pr value %T", value)
	}
}

func splitRepo(repo string) (string, string, error) {
	parts := strings.Split(strings.TrimSpace(repo), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("dev-cycle: GitHub repository must be owner/name, got %q", repo)
	}
	return parts[0], parts[1], nil
}

func parseGitHubRemote(remote string) (string, error) {
	if repo, ok := strings.CutPrefix(remote, "git@github.com:"); ok {
		return strings.TrimSuffix(repo, ".git"), nil
	}
	parsed, err := url.Parse(remote)
	if err != nil {
		return "", fmt.Errorf("dev-cycle: parse git remote %q: %w", remote, err)
	}
	if parsed.Host != "github.com" {
		return "", fmt.Errorf("dev-cycle: origin remote is not github.com: %q", remote)
	}
	return strings.TrimSuffix(strings.TrimPrefix(parsed.Path, "/"), ".git"), nil
}

func summarizeTitle(body string) string {
	text := strings.TrimSpace(regexp.MustCompile(`\s+`).ReplaceAllString(body, " "))
	if text == "" {
		return "CodeRabbit review comment"
	}
	if len(text) <= 96 {
		return text
	}
	return strings.TrimSpace(text[:96]) + "..."
}

func stableIssueID(providerRef string) string {
	sum := sha256.Sum256([]byte(providerRef))
	return "cr-" + hex.EncodeToString(sum[:])[:12]
}

func digestString(value string) string {
	return digestBytes([]byte(value))
}

func digestBytes(value []byte) string {
	sum := sha256.Sum256(value)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func quietPeriodDeadline(payload codeRabbitWatchPayload, raw string) *time.Time {
	quietPeriod, err := time.ParseDuration(strings.TrimSpace(raw))
	if err != nil || quietPeriod <= 0 {
		return nil
	}
	base := time.Now().UTC()
	if payload.SubmittedAt != nil {
		base = payload.SubmittedAt.UTC()
	} else if payload.ProviderState.UpdatedAt != nil {
		base = payload.ProviderState.UpdatedAt.UTC()
	}
	settled := base.Add(quietPeriod)
	return &settled
}
