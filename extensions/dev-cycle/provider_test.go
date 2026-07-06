package devcycle

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
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
      reviews(first:100){nodes{id state submittedAt author{login}}}
    }
  }
}`
}
