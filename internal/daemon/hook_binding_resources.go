package daemon

import (
	"context"
	"errors"
	"fmt"
	"strings"

	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/resources"
)

const (
	hookBindingResourceKind     resources.ResourceKind = "hook.binding"
	hookBindingResourceMaxBytes                        = 256 << 10
)

type hookBindingProjectionPlan struct {
	revision   int64
	operations int
	state      *hookspkg.BindingState
}

func (p *hookBindingProjectionPlan) Kind() resources.ResourceKind {
	return hookBindingResourceKind
}

func (p *hookBindingProjectionPlan) Revision() int64 {
	if p == nil {
		return 0
	}
	return p.revision
}

func (p *hookBindingProjectionPlan) OperationCount() int {
	if p == nil {
		return 0
	}
	return p.operations
}

type hookBindingProjector struct {
	runtime *hookspkg.Hooks
}

var _ resources.TypedProjector[hookspkg.HookDecl] = (*hookBindingProjector)(nil)

func newHookBindingCodec() (resources.KindCodec[hookspkg.HookDecl], error) {
	return resources.NewJSONCodec(
		hookBindingResourceKind,
		hookBindingResourceMaxBytes,
		validateHookBindingSpec,
	)
}

func newHookBindingStore(
	raw resources.RawStore,
	codec resources.KindCodec[hookspkg.HookDecl],
) (resources.Store[hookspkg.HookDecl], error) {
	return resources.NewStore(raw, codec)
}

func newHookBindingProjector(runtime *hookspkg.Hooks) resources.TypedProjector[hookspkg.HookDecl] {
	if runtime == nil {
		return nil
	}
	return &hookBindingProjector{runtime: runtime}
}

func validateHookBindingSpec(
	_ context.Context,
	scope resources.ResourceScope,
	spec hookspkg.HookDecl,
) (hookspkg.HookDecl, error) {
	normalizedScope := scope.Normalize()
	if err := normalizedScope.Validate("scope"); err != nil {
		return hookspkg.HookDecl{}, err
	}

	normalized, err := hookspkg.CanonicalizeHookDecl(spec)
	if err != nil {
		return hookspkg.HookDecl{}, err
	}

	if normalizedScope.Kind == resources.ResourceScopeKindWorkspace {
		workspaceID := strings.TrimSpace(normalized.Matcher.WorkspaceID)
		switch {
		case workspaceID == "":
			normalized.Matcher.WorkspaceID = normalizedScope.ID
		case workspaceID != normalizedScope.ID:
			return hookspkg.HookDecl{}, fmt.Errorf(
				"%w: hook workspace matcher %q does not match resource scope %q",
				resources.ErrInvalidScopeBinding,
				workspaceID,
				normalizedScope.ID,
			)
		}
	}

	return normalized, nil
}

func (p *hookBindingProjector) Kind() resources.ResourceKind {
	return hookBindingResourceKind
}

func (p *hookBindingProjector) DependsOn() []resources.ResourceKind {
	return nil
}

func (p *hookBindingProjector) Build(
	_ context.Context,
	records []resources.Record[hookspkg.HookDecl],
) (resources.ProjectionPlan, error) {
	if p == nil || p.runtime == nil {
		return nil, errors.New("daemon: hook binding projector runtime is required")
	}

	decls := make([]hookspkg.HookDecl, 0, len(records))
	var revision int64
	for _, record := range records {
		if record.Version > revision {
			revision = record.Version
		}
		decls = append(decls, cloneDaemonHookDecl(record.Spec))
	}

	state, err := p.runtime.BuildBindingState(decls)
	if err != nil {
		return nil, err
	}

	return &hookBindingProjectionPlan{
		revision:   revision,
		operations: state.HookCount(),
		state:      state,
	}, nil
}

func (p *hookBindingProjector) Apply(ctx context.Context, plan resources.ProjectionPlan) error {
	if p == nil || p.runtime == nil {
		return errors.New("daemon: hook binding projector runtime is required")
	}
	if ctx == nil {
		return errors.New("daemon: hook binding projector apply context is required")
	}

	typed, ok := plan.(*hookBindingProjectionPlan)
	if !ok {
		return fmt.Errorf("daemon: hook binding projector plan has type %T", plan)
	}

	return p.runtime.ApplyBindingState(typed.state, typed.revision)
}
