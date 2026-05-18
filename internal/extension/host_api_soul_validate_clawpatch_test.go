package extensionpkg

import (
	"context"
	"testing"

	extensioncontract "github.com/pedronauck/agh/internal/extension/contract"
	"github.com/pedronauck/agh/internal/soul"
)

func TestHostAPIHandlerSoulValidateBodyPresenceClawpatch(t *testing.T) {
	t.Parallel()

	t.Run("Should preserve an omitted soul validate body as current-file validation", func(t *testing.T) {
		t.Parallel()

		env, authoring := newSoulValidateBodyPresenceEnvClawpatch(t)
		_, err := env.handler.Handle(
			t.Context(),
			"ext-soul-read",
			string(extensioncontract.HostAPIMethodAgentsSoulValidate),
			mustHostAPIAuthoredJSON(t, map[string]any{
				"workspace_id": env.workspaceID,
				"agent_name":   "coder",
			}),
		)
		if err != nil {
			t.Fatalf("Handle(agents/soul/validate omitted body) error = %v", err)
		}
		if authoring.validateCalls != 1 {
			t.Fatalf("soul validate calls = %d, want 1", authoring.validateCalls)
		}
		if authoring.lastValidate.Body != nil {
			t.Fatalf("soul validate body = %q, want nil for omitted body", *authoring.lastValidate.Body)
		}
	})

	t.Run("Should preserve an explicit empty soul validate body as proposed content", func(t *testing.T) {
		t.Parallel()

		env, authoring := newSoulValidateBodyPresenceEnvClawpatch(t)
		_, err := env.handler.Handle(
			t.Context(),
			"ext-soul-read",
			string(extensioncontract.HostAPIMethodAgentsSoulValidate),
			mustHostAPIAuthoredJSON(t, map[string]any{
				"workspace_id": env.workspaceID,
				"agent_name":   "coder",
				"body":         "",
			}),
		)
		if err != nil {
			t.Fatalf("Handle(agents/soul/validate empty body) error = %v", err)
		}
		if authoring.validateCalls != 1 {
			t.Fatalf("soul validate calls = %d, want 1", authoring.validateCalls)
		}
		if authoring.lastValidate.Body == nil {
			t.Fatal("soul validate body = nil, want explicit empty string pointer")
		}
		if *authoring.lastValidate.Body != "" {
			t.Fatalf("soul validate body = %q, want empty string", *authoring.lastValidate.Body)
		}
	})
}

func newSoulValidateBodyPresenceEnvClawpatch(
	t *testing.T,
) (*hostAPITestEnv, *recordingSoulAuthoringClawpatch) {
	t.Helper()

	env := newHostAPITestEnv(t)
	authoring := &recordingSoulAuthoringClawpatch{
		result: hostAPITestSoulMutationResult(env.workspaceID, "coder", env.workspace.RootDir),
	}
	env.handler.soulAuthoring = authoring
	env.grant("ext-soul-read", []string{
		string(extensioncontract.HostAPIMethodAgentsSoulValidate),
	}, []string{"soul.read"})
	return env, authoring
}

type recordingSoulAuthoringClawpatch struct {
	result        soul.MutationResult
	validateCalls int
	lastValidate  soul.ValidateRequest
}

func (s *recordingSoulAuthoringClawpatch) Validate(
	_ context.Context,
	req soul.ValidateRequest,
) (soul.ValidateResult, error) {
	s.validateCalls++
	s.lastValidate = req
	return soul.ValidateResult{Soul: s.result.Soul}, nil
}

func (s *recordingSoulAuthoringClawpatch) Put(context.Context, soul.PutRequest) (soul.MutationResult, error) {
	return s.result, nil
}

func (s *recordingSoulAuthoringClawpatch) Delete(context.Context, soul.DeleteRequest) (soul.MutationResult, error) {
	return s.result, nil
}

func (s *recordingSoulAuthoringClawpatch) History(context.Context, soul.HistoryRequest) (soul.HistoryResult, error) {
	return soul.HistoryResult{Revisions: []soul.Revision{s.result.Revision}}, nil
}

func (s *recordingSoulAuthoringClawpatch) Rollback(
	context.Context,
	soul.RollbackRequest,
) (soul.MutationResult, error) {
	return s.result, nil
}

var _ hostAPISoulAuthoringService = (*recordingSoulAuthoringClawpatch)(nil)
