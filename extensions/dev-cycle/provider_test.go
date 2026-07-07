package devcycle

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	watchpkg "github.com/compozy/agh/internal/loop/watch"
)

func TestCodeRabbitProviderShouldSurfaceProviderFailures(t *testing.T) {
	t.Run("Should fail when gh is unavailable", func(t *testing.T) {
		t.Parallel()

		provider := newCodeRabbitProvider(&recordingCommandRunner{
			lookPathErrs: map[string]error{"gh": errors.New("not found")},
		})

		_, err := provider.FetchUnresolved(context.Background(), codeRabbitFetchInput{PR: 17})
		if err == nil || !strings.Contains(err.Error(), "gh executable is required") {
			t.Fatalf("FetchUnresolved() error = %v, want missing gh diagnostic", err)
		}
	})

	t.Run("Should surface GitHub GraphQL auth errors", func(t *testing.T) {
		t.Parallel()

		runner := &recordingCommandRunner{
			lookPathResults: map[string]string{"gh": "/usr/bin/gh"},
			runResults: map[string][]byte{
				commandKey(
					"gh",
					"repo",
					"view",
					"--json",
					"nameWithOwner",
					"-q",
					".nameWithOwner",
				): []byte("acme/repo\n"),
				commandKey(
					"gh",
					"api",
					"graphql",
					"-f",
					"query="+fetchPRQueryForTest(),
					"-F",
					"owner=acme",
					"-F",
					"repo=repo",
					"-F",
					"pr=17",
				): []byte(`{"errors":[{"message":"HTTP 401: Bad credentials"}]}`),
			},
		}
		provider := newCodeRabbitProvider(runner)

		_, err := provider.FetchUnresolved(context.Background(), codeRabbitFetchInput{PR: 17})
		if err == nil || !strings.Contains(err.Error(), "GitHub GraphQL error: HTTP 401") {
			t.Fatalf("FetchUnresolved() error = %v, want GraphQL auth diagnostic", err)
		}
	})

	t.Run("Should resolve only accepted provider threads once", func(t *testing.T) {
		t.Parallel()

		runner := &recordingCommandRunner{
			lookPathResults: map[string]string{"gh": "/usr/bin/gh"},
		}
		provider := newCodeRabbitProvider(runner)
		input := codeRabbitResolveInput{
			PR: "17",
			Issues: []codeRabbitIssue{
				{ID: "issue-a", ProviderRef: "thread-a"},
				{ID: "issue-b", ProviderRef: "thread-b"},
				{ID: "issue-c", ProviderRef: "thread-a"},
			},
			Results: []codeRabbitFixEntry{
				{ID: "issue-a", Resolution: "fixed"},
				{ProviderRef: "thread-b", Triage: "invalid"},
			},
		}

		output, err := provider.ResolveThreads(context.Background(), input)
		if err != nil {
			t.Fatalf("ResolveThreads() error = %v", err)
		}
		if output.ResolvedCount != 1 || !slices.Equal(output.ResolvedThreads, []string{"thread-a"}) {
			t.Fatalf("ResolveThreads() output = %#v, want only thread-a resolved once", output)
		}
		if got, want := len(runner.calls), 1; got != want {
			t.Fatalf("command calls = %d, want %d", got, want)
		}
		if got := strings.Join(runner.calls[0].args, " "); !strings.Contains(got, "id=thread-a") {
			t.Fatalf("resolve command args = %q, want thread-a mutation", got)
		}
	})

	t.Run("Should reject empty resolution results for fetched issues", func(t *testing.T) {
		t.Parallel()

		runner := &recordingCommandRunner{
			lookPathResults: map[string]string{"gh": "/usr/bin/gh"},
		}
		provider := newCodeRabbitProvider(runner)

		_, err := provider.ResolveThreads(context.Background(), codeRabbitResolveInput{
			PR: "17",
			Issues: []codeRabbitIssue{
				{ID: "issue-a", ProviderRef: "thread-a"},
			},
		})
		if !errors.Is(err, watchpkg.ErrSpecInvalid) {
			t.Fatalf("ResolveThreads() error = %v, want ErrSpecInvalid", err)
		}
		if runner.called("gh", "api") {
			t.Fatalf("gh api was called for empty results: %#v", runner.calls)
		}
	})

	t.Run("Should reject triage-only valid results", func(t *testing.T) {
		t.Parallel()

		runner := &recordingCommandRunner{
			lookPathResults: map[string]string{"gh": "/usr/bin/gh"},
		}
		provider := newCodeRabbitProvider(runner)

		_, err := provider.ResolveThreads(context.Background(), codeRabbitResolveInput{
			PR: "17",
			Issues: []codeRabbitIssue{
				{ID: "issue-a", ProviderRef: "thread-a"},
			},
			Results: []codeRabbitFixEntry{
				{ID: "issue-a", Triage: "valid"},
			},
		})
		if !errors.Is(err, watchpkg.ErrSpecInvalid) {
			t.Fatalf("ResolveThreads() error = %v, want ErrSpecInvalid", err)
		}
		if runner.called("gh", "api") {
			t.Fatalf("gh api was called for triage-only result: %#v", runner.calls)
		}
	})

	t.Run("Should resolve documented provider refs without issue ids", func(t *testing.T) {
		t.Parallel()

		runner := &recordingCommandRunner{
			lookPathResults: map[string]string{"gh": "/usr/bin/gh"},
		}
		provider := newCodeRabbitProvider(runner)

		output, err := provider.ResolveThreads(context.Background(), codeRabbitResolveInput{
			PR: "17",
			Issues: []codeRabbitIssue{
				{ID: "issue-a", ProviderRef: "thread-a"},
			},
			Results: []codeRabbitFixEntry{
				{ProviderRef: "thread-a", Resolution: "documented"},
			},
		})
		if err != nil {
			t.Fatalf("ResolveThreads() error = %v", err)
		}
		if output.ResolvedCount != 1 || !slices.Equal(output.ResolvedThreads, []string{"thread-a"}) {
			t.Fatalf("ResolveThreads() output = %#v, want thread-a resolved", output)
		}
	})

	t.Run("Should skip synthetic nitpick refs when resolving GitHub review threads", func(t *testing.T) {
		t.Parallel()

		runner := &recordingCommandRunner{
			lookPathResults: map[string]string{"gh": "/usr/bin/gh"},
		}
		provider := newCodeRabbitProvider(runner)

		output, err := provider.ResolveThreads(context.Background(), codeRabbitResolveInput{
			PR: "17",
			Issues: []codeRabbitIssue{
				{ID: "issue-thread", ProviderRef: "thread-a"},
				{ID: "issue-nitpick", ProviderRef: "review:901,nitpick_hash:abc123"},
			},
			Results: []codeRabbitFixEntry{
				{ID: "issue-thread", Resolution: "fixed"},
				{ID: "issue-nitpick", Resolution: "fixed"},
			},
		})
		if err != nil {
			t.Fatalf("ResolveThreads() error = %v", err)
		}
		if output.ResolvedCount != 1 || !slices.Equal(output.ResolvedThreads, []string{"thread-a"}) {
			t.Fatalf("ResolveThreads() output = %#v, want only real review thread resolved", output)
		}
		if got, want := len(runner.calls), 1; got != want {
			t.Fatalf("command calls = %d, want %d", got, want)
		}
		args := strings.Join(runner.calls[0].args, " ")
		if !strings.Contains(args, "id=thread-a") || strings.Contains(args, "review:901") {
			t.Fatalf("resolve command args = %q, want only thread-a mutation", args)
		}
	})

	t.Run("Should accept nitpick-only remediation without GitHub thread mutation", func(t *testing.T) {
		t.Parallel()

		runner := &recordingCommandRunner{
			lookPathResults: map[string]string{"gh": "/usr/bin/gh"},
		}
		provider := newCodeRabbitProvider(runner)

		output, err := provider.ResolveThreads(context.Background(), codeRabbitResolveInput{
			PR: "17",
			Issues: []codeRabbitIssue{
				{ID: "issue-nitpick", ProviderRef: "review:901,nitpick_hash:abc123"},
			},
			Results: []codeRabbitFixEntry{
				{ID: "issue-nitpick", Resolution: "documented"},
			},
		})
		if err != nil {
			t.Fatalf("ResolveThreads() error = %v", err)
		}
		if output.ResolvedCount != 0 || len(output.ResolvedThreads) != 0 {
			t.Fatalf("ResolveThreads() output = %#v, want no GitHub thread resolutions", output)
		}
		if runner.called("gh", "api") {
			t.Fatalf("gh api was called for synthetic nitpick ref: %#v", runner.calls)
		}
	})
}

func TestCodeRabbitProviderShouldFetchReviewItems(t *testing.T) {
	t.Run("Should fetch unresolved review threads without opt-in nitpicks", func(t *testing.T) {
		t.Parallel()

		runner := codeRabbitFetchRunner(t, codeRabbitGraphQLWithThread(), codeRabbitReviewsWithNitpick(t))
		provider := newCodeRabbitProvider(runner)

		output, err := provider.FetchUnresolved(context.Background(), codeRabbitFetchInput{PR: 17})
		if err != nil {
			t.Fatalf("FetchUnresolved() error = %v", err)
		}
		if output.UnresolvedCount != 1 {
			t.Fatalf("FetchUnresolved() unresolved_count = %d, want 1", output.UnresolvedCount)
		}
		issue := output.Issues[0]
		if issue.ProviderRef != "thread-a" || issue.Author != codeRabbitBotLogin || issue.Severity != "review" {
			t.Fatalf("FetchUnresolved() issue = %#v, want CodeRabbit thread issue", issue)
		}
		if runner.calledWithArg("gh", "repos/acme/repo/pulls/17/reviews?per_page=100&page=1") {
			t.Fatalf("pull request reviews were fetched without include_nitpicks: %#v", runner.calls)
		}
	})

	t.Run("Should include review body nitpicks only when requested", func(t *testing.T) {
		t.Parallel()

		runner := codeRabbitFetchRunner(t, codeRabbitGraphQLWithThread(), codeRabbitReviewsWithNitpick(t))
		provider := newCodeRabbitProvider(runner)

		output, err := provider.FetchUnresolved(context.Background(), codeRabbitFetchInput{
			PR:              17,
			IncludeNitpicks: true,
		})
		if err != nil {
			t.Fatalf("FetchUnresolved() error = %v", err)
		}
		if output.UnresolvedCount != 2 {
			t.Fatalf("FetchUnresolved() unresolved_count = %d, want 2", output.UnresolvedCount)
		}
		nitpick := output.Issues[0]
		if output.Issues[1].Severity == reviewBodyCommentSeverityNitpick {
			nitpick = output.Issues[1]
		}
		if nitpick.Severity != reviewBodyCommentSeverityNitpick {
			t.Fatalf("FetchUnresolved() issues = %#v, want nitpick issue", output.Issues)
		}
		if nitpick.File != "internal/foo.go" || nitpick.Line != 12 {
			t.Fatalf("nitpick location = %s:%d, want internal/foo.go:12", nitpick.File, nitpick.Line)
		}
		if nitpick.SourceReviewID != "901" || nitpick.SourceReviewSubmittedAt != "2026-07-07T12:00:00Z" {
			t.Fatalf("nitpick source metadata = %#v, want review metadata", nitpick)
		}
		if !strings.HasPrefix(nitpick.ProviderRef, "review:901,nitpick_hash:") {
			t.Fatalf("nitpick provider_ref = %q, want review/hash provider ref", nitpick.ProviderRef)
		}
	})
}

func TestCodeRabbitProviderShouldPollCurrentReviewStatus(t *testing.T) {
	t.Run("Should mark current reviewed after successful CodeRabbit status on local HEAD", func(t *testing.T) {
		t.Parallel()

		runner := codeRabbitPollRunner(
			"17",
			"head-sha",
			"head-sha",
			codeRabbitStatuses("success", "finished"),
			codeRabbitReviews("901", "head-sha", "COMMENTED"),
		)
		provider := newCodeRabbitProvider(runner)

		response, err := provider.Poll(context.Background(), watchpkg.PollRequest{}, codeRabbitWatchSpec{
			PR:          17,
			QuietPeriod: "20s",
		})
		if err != nil {
			t.Fatalf("Poll() error = %v", err)
		}
		if !response.Ready {
			t.Fatalf("Poll() ready = false, want true")
		}
		payload := decodeCodeRabbitWatchPayload(t, response.Payload)
		if payload.ProviderState.State != codeRabbitWatchCurrentReviewed {
			t.Fatalf("provider state = %#v, want current reviewed", payload.ProviderState)
		}
		if payload.Review.HeadSHA != "head-sha" ||
			payload.Review.LocalHeadSHA != "head-sha" ||
			payload.Review.ReviewCommitSHA != "head-sha" {
			t.Fatalf("review payload = %#v, want matching head/local/review SHAs", payload.Review)
		}
		if response.SettledAt == nil {
			t.Fatalf("Poll() settled_at = nil, want quiet-period deadline")
		}
	})

	t.Run("Should confirm ready state even when digest did not change", func(t *testing.T) {
		t.Parallel()

		runner := codeRabbitPollRunner(
			"17",
			"head-sha",
			"head-sha",
			codeRabbitStatuses("success", "finished"),
			codeRabbitReviews("901", "head-sha", "COMMENTED"),
		)
		provider := newCodeRabbitProvider(runner)

		first, err := provider.Poll(context.Background(), watchpkg.PollRequest{}, codeRabbitWatchSpec{
			PR:          17,
			QuietPeriod: "20s",
		})
		if err != nil {
			t.Fatalf("Poll(first) error = %v", err)
		}
		if !first.Ready || first.StateDigest == "" {
			t.Fatalf("Poll(first) = %#v, want ready response with digest", first)
		}
		second, err := provider.Poll(
			context.Background(),
			watchpkg.PollRequest{ExpectedStateDigest: first.StateDigest},
			codeRabbitWatchSpec{PR: 17, QuietPeriod: "20s"},
		)
		if err != nil {
			t.Fatalf("Poll(second) error = %v", err)
		}
		if !second.Ready {
			t.Fatalf("Poll(second) ready = false, want ready confirmation for unchanged digest")
		}
		if len(second.Payload) == 0 {
			t.Fatalf("Poll(second) payload empty, want confirmation payload")
		}
		if second.SettledAt == nil {
			t.Fatalf("Poll(second) settled_at = nil, want quiet-period deadline")
		}
	})

	t.Run("Should keep stale review commits not ready", func(t *testing.T) {
		t.Parallel()

		runner := codeRabbitPollRunner(
			"17",
			"head-sha",
			"head-sha",
			codeRabbitStatuses("success", "finished"),
			codeRabbitReviews("901", "old-sha", "COMMENTED"),
		)
		provider := newCodeRabbitProvider(runner)

		response, err := provider.Poll(context.Background(), watchpkg.PollRequest{}, codeRabbitWatchSpec{PR: 17})
		if err != nil {
			t.Fatalf("Poll() error = %v", err)
		}
		if response.Ready {
			t.Fatalf("Poll() ready = true, want stale review to block readiness")
		}
		payload := decodeCodeRabbitWatchPayload(t, response.Payload)
		if payload.ProviderState.State != codeRabbitWatchStale {
			t.Fatalf("provider state = %#v, want stale", payload.ProviderState)
		}
	})

	t.Run("Should keep pending CodeRabbit status not ready", func(t *testing.T) {
		t.Parallel()

		runner := codeRabbitPollRunner(
			"17",
			"head-sha",
			"head-sha",
			codeRabbitStatuses("pending", "reviewing"),
			codeRabbitReviews("901", "head-sha", "COMMENTED"),
		)
		provider := newCodeRabbitProvider(runner)

		response, err := provider.Poll(context.Background(), watchpkg.PollRequest{}, codeRabbitWatchSpec{PR: 17})
		if err != nil {
			t.Fatalf("Poll() error = %v", err)
		}
		if response.Ready {
			t.Fatalf("Poll() ready = true, want pending status to block readiness")
		}
		payload := decodeCodeRabbitWatchPayload(t, response.Payload)
		if payload.ProviderState.State != codeRabbitWatchPending {
			t.Fatalf("provider state = %#v, want pending", payload.ProviderState)
		}
	})

	t.Run("Should surface failed CodeRabbit status as provider error", func(t *testing.T) {
		t.Parallel()

		runner := codeRabbitPollRunner(
			"17",
			"head-sha",
			"head-sha",
			codeRabbitStatuses("failure", "review failed"),
			codeRabbitReviews("901", "head-sha", "COMMENTED"),
		)
		provider := newCodeRabbitProvider(runner)

		_, err := provider.Poll(context.Background(), watchpkg.PollRequest{}, codeRabbitWatchSpec{PR: 17})
		if err == nil || !strings.Contains(err.Error(), "coderabbit status \"failure\"") {
			t.Fatalf("Poll() error = %v, want failed status diagnostic", err)
		}
	})

	t.Run("Should block readiness when local HEAD differs from PR head", func(t *testing.T) {
		t.Parallel()

		runner := codeRabbitPollRunner(
			"17",
			"head-sha",
			"local-sha",
			codeRabbitStatuses("success", "finished"),
			codeRabbitReviews("901", "head-sha", "COMMENTED"),
		)
		provider := newCodeRabbitProvider(runner)

		response, err := provider.Poll(context.Background(), watchpkg.PollRequest{}, codeRabbitWatchSpec{PR: 17})
		if err != nil {
			t.Fatalf("Poll() error = %v", err)
		}
		if response.Ready {
			t.Fatalf("Poll() ready = true, want local HEAD mismatch to block readiness")
		}
		payload := decodeCodeRabbitWatchPayload(t, response.Payload)
		if payload.ProviderState.State != codeRabbitWatchLocalHeadMismatch {
			t.Fatalf("provider state = %#v, want local head mismatch", payload.ProviderState)
		}
	})
}

func TestGitProviderShouldGuardPushes(t *testing.T) {
	t.Run("Should require a changed HEAD before guarded push", func(t *testing.T) {
		t.Parallel()

		runner := gitRunnerForPushTest("feature", "abc123")
		provider := newGitProvider(runner)

		_, err := provider.Push(context.Background(), gitPushInput{
			Remote:              "origin",
			RequireHeadAdvanced: true,
			ExpectedHead:        "abc123",
		})
		if err == nil || !strings.Contains(err.Error(), "HEAD did not advance") {
			t.Fatalf("Push() error = %v, want head-advance guard", err)
		}
		if runner.called("git", "push") {
			t.Fatalf("git push was called despite unchanged HEAD: %#v", runner.calls)
		}
	})

	t.Run("Should require expected_head when the head guard is enabled", func(t *testing.T) {
		t.Parallel()

		runner := gitRunnerForPushTest("feature", "def456")
		provider := newGitProvider(runner)

		_, err := provider.Push(context.Background(), gitPushInput{
			Remote:              "origin",
			RequireHeadAdvanced: true,
		})
		if err == nil || !strings.Contains(err.Error(), "expected_head is required") {
			t.Fatalf("Push() error = %v, want expected_head validation", err)
		}
		if runner.called("git", "push") {
			t.Fatalf("git push was called without expected_head: %#v", runner.calls)
		}
	})

	t.Run("Should separate push options from remote and branch", func(t *testing.T) {
		t.Parallel()

		runner := gitRunnerForPushTest("feature", "def456")
		provider := newGitProvider(runner)

		output, err := provider.Push(context.Background(), gitPushInput{Remote: "origin"})
		if err != nil {
			t.Fatalf("Push() error = %v", err)
		}
		if !output.Pushed || output.Branch != "feature" || output.Head != "def456" {
			t.Fatalf("Push() output = %#v, want pushed feature at def456", output)
		}
		if len(runner.calls) != 3 {
			t.Fatalf("command calls = %#v, want branch/head/push", runner.calls)
		}
		if got, want := strings.Join(runner.calls[2].args, " "), "push -- origin feature"; got != want {
			t.Fatalf("git push args = %q, want %q", got, want)
		}
	})
}

type recordedCommand struct {
	name string
	args []string
}

type recordingCommandRunner struct {
	lookPathResults map[string]string
	lookPathErrs    map[string]error
	runResults      map[string][]byte
	runErrs         map[string]error
	calls           []recordedCommand
}

func (r *recordingCommandRunner) LookPath(file string) (string, error) {
	if err := r.lookPathErrs[file]; err != nil {
		return "", err
	}
	if path := strings.TrimSpace(r.lookPathResults[file]); path != "" {
		return path, nil
	}
	return "/usr/bin/" + file, nil
}

func (r *recordingCommandRunner) Run(
	_ context.Context,
	name string,
	args []string,
	_ string,
) ([]byte, error) {
	r.calls = append(r.calls, recordedCommand{name: name, args: append([]string(nil), args...)})
	key := commandKey(name, args...)
	if err := r.runErrs[key]; err != nil {
		return nil, err
	}
	if output, ok := r.runResults[key]; ok {
		return append([]byte(nil), output...), nil
	}
	return []byte{}, nil
}

func (r *recordingCommandRunner) called(name string, firstArg string) bool {
	for _, call := range r.calls {
		if call.name == name && len(call.args) > 0 && call.args[0] == firstArg {
			return true
		}
	}
	return false
}

func (r *recordingCommandRunner) calledWithArg(name string, arg string) bool {
	for _, call := range r.calls {
		if call.name != name {
			continue
		}
		if slices.Contains(call.args, arg) {
			return true
		}
	}
	return false
}

func gitRunnerForPushTest(branch string, head string) *recordingCommandRunner {
	return &recordingCommandRunner{
		lookPathResults: map[string]string{"git": "/usr/bin/git"},
		runResults: map[string][]byte{
			commandKey("git", "rev-parse", "--abbrev-ref", gitHeadRef): []byte(branch + "\n"),
			commandKey("git", "rev-parse", gitHeadRef):                 []byte(head + "\n"),
		},
	}
}

func commandKey(name string, args ...string) string {
	return name + "\x00" + strings.Join(args, "\x00")
}

func fetchPRQueryForTest() string {
	return `
query($owner:String!,$repo:String!,$pr:Int!){
  repository(owner:$owner,name:$repo){
    pullRequest(number:$pr){
      headRefOid
      reviewThreads(first:100){nodes{id isResolved comments(first:20){nodes{body path line author{login} createdAt}}}}
    }
  }
}`
}

func codeRabbitRepoViewKey() string {
	return commandKey("gh", "repo", "view", "--json", "nameWithOwner", "-q", ".nameWithOwner")
}

func codeRabbitGraphQLKey(pr string) string {
	return commandKey(
		"gh",
		"api",
		"graphql",
		"-f",
		"query="+fetchPRQueryForTest(),
		"-F",
		"owner=acme",
		"-F",
		"repo=repo",
		"-F",
		"pr="+pr,
	)
}

func codeRabbitFetchRunner(t *testing.T, graphQL string, reviews string) *recordingCommandRunner {
	t.Helper()
	return &recordingCommandRunner{
		lookPathResults: map[string]string{"gh": "/usr/bin/gh"},
		runResults: map[string][]byte{
			codeRabbitRepoViewKey():    []byte("acme/repo\n"),
			codeRabbitGraphQLKey("17"): []byte(graphQL),
			commandKey(
				"gh",
				"api",
				"repos/acme/repo/pulls/17/reviews?per_page=100&page=1",
			): []byte(reviews),
		},
	}
}

func codeRabbitPollRunner(
	pr string,
	head string,
	localHead string,
	statuses string,
	reviews string,
) *recordingCommandRunner {
	return &recordingCommandRunner{
		lookPathResults: map[string]string{"gh": "/usr/bin/gh", "git": "/usr/bin/git"},
		runResults: map[string][]byte{
			codeRabbitRepoViewKey(): []byte("acme/repo\n"),
			codeRabbitGraphQLKey(pr): []byte(fmt.Sprintf(
				`{"data":{"repository":{"pullRequest":{"headRefOid":%q,"reviewThreads":{"nodes":[]}}}}}`,
				head,
			)),
			commandKey("git", "rev-parse", "HEAD"): []byte(localHead + "\n"),
			commandKey(
				"gh",
				"api",
				fmt.Sprintf("repos/acme/repo/commits/%s/statuses?per_page=100&page=1", head),
			): []byte(statuses),
			commandKey(
				"gh",
				"api",
				fmt.Sprintf("repos/acme/repo/pulls/%s/reviews?per_page=100&page=1", pr),
			): []byte(reviews),
		},
	}
}

func codeRabbitGraphQLWithThread() string {
	return `{"data":{"repository":{"pullRequest":{"headRefOid":"head-sha","reviewThreads":{"nodes":[{"id":"thread-a","isResolved":false,"comments":{"nodes":[{"body":"Fix the production path","path":"internal/foo.go","line":9,"author":{"login":"coderabbitai[bot]"},"createdAt":"2026-07-07T12:00:00Z"}]}}]}}}}}`
}

func codeRabbitReviewsWithNitpick(t *testing.T) string {
	t.Helper()
	payload := []map[string]any{
		{
			"id": 901,
			"body": "<details>\n<summary>1 Nitpick comment</summary>\n<blockquote>\n<details>\n" +
				"<summary>internal/foo.go (1)</summary>\n<blockquote>\n" +
				"`12`: **Prefer wrapped errors**\nUse `%w` when returning the underlying error.\n" +
				"</blockquote>\n</details>\n</blockquote>\n</details>",
			"commit_id":    "head-sha",
			"state":        "COMMENTED",
			"submitted_at": "2026-07-07T12:00:00Z",
			"user": map[string]string{
				"login": codeRabbitBotLogin,
			},
		},
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal(codeRabbitReviewsWithNitpick) error = %v", err)
	}
	return string(encoded)
}

func codeRabbitStatuses(state string, description string) string {
	return fmt.Sprintf(
		`[{"state":%q,"description":%q,"context":"CodeRabbit","updated_at":"2026-07-07T12:00:00Z"}]`,
		state,
		description,
	)
}

func codeRabbitReviews(id string, commitID string, state string) string {
	return fmt.Sprintf(
		`[{"id":%s,"body":"review body","commit_id":%q,"state":%q,"submitted_at":"2026-07-07T12:00:00Z","user":{"login":"coderabbitai[bot]"}}]`,
		id,
		commitID,
		state,
	)
}

func decodeCodeRabbitWatchPayload(t *testing.T, raw json.RawMessage) codeRabbitWatchPayload {
	t.Helper()
	var payload codeRabbitWatchPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("Unmarshal(watch payload) error = %v; raw = %s", err, string(raw))
	}
	return payload
}
