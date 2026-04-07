// Package session orchestrates AGH session lifecycle around ACP-backed agents.
package session

import (
	"context"
	"errors"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	aghconfig "github.com/pedronauck/agh/internal/config"
	skillspkg "github.com/pedronauck/agh/internal/skills"
	"github.com/pedronauck/agh/internal/store"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

// AgentProcess is the session-owned handle for a running agent process.
type AgentProcess struct {
	PID       int
	AgentName string
	Command   string
	Args      []string
	Cwd       string
	SessionID string
	Caps      acp.ACPCaps
	StartedAt time.Time

	done                <-chan struct{}
	waitFn              func() error
	stderrFn            func() string
	approvePermissionFn func(context.Context, acp.ApproveRequest) error
	native              any
}

// AgentProcessOptions defines the exported fields and lifecycle hooks needed to construct an AgentProcess.
type AgentProcessOptions struct {
	PID               int
	AgentName         string
	Command           string
	Args              []string
	Cwd               string
	SessionID         string
	Caps              acp.ACPCaps
	StartedAt         time.Time
	Done              <-chan struct{}
	Wait              func() error
	Stderr            func() string
	ApprovePermission func(context.Context, acp.ApproveRequest) error
}

// NewAgentProcess constructs an AgentProcess for custom AgentDriver implementations.
func NewAgentProcess(opts AgentProcessOptions) *AgentProcess {
	done := opts.Done
	if done == nil {
		ch := make(chan struct{})
		close(ch)
		done = ch
	}

	waitFn := opts.Wait
	if waitFn == nil {
		waitFn = func() error {
			<-done
			return nil
		}
	}

	stderrFn := opts.Stderr
	if stderrFn == nil {
		stderrFn = func() string { return "" }
	}

	return &AgentProcess{
		PID:                 opts.PID,
		AgentName:           opts.AgentName,
		Command:             opts.Command,
		Args:                append([]string(nil), opts.Args...),
		Cwd:                 opts.Cwd,
		SessionID:           opts.SessionID,
		Caps:                opts.Caps,
		StartedAt:           opts.StartedAt,
		done:                done,
		waitFn:              waitFn,
		stderrFn:            stderrFn,
		approvePermissionFn: opts.ApprovePermission,
	}
}

// Done reports when the underlying runtime process exits.
func (p *AgentProcess) Done() <-chan struct{} {
	return p.done
}

// Wait blocks until the runtime process exits and returns its terminal error state.
func (p *AgentProcess) Wait() error {
	<-p.Done()
	return p.waitFn()
}

// Stderr returns any captured stderr output for the runtime process.
func (p *AgentProcess) Stderr() string {
	return p.stderrFn()
}

// ApprovePermission resolves one pending interactive permission request.
func (p *AgentProcess) ApprovePermission(ctx context.Context, req acp.ApproveRequest) error {
	if p.approvePermissionFn == nil {
		return errors.New("session: permission approval is not supported")
	}
	return p.approvePermissionFn(ctx, req)
}

func wrapACPProcess(proc *acp.AgentProcess) *AgentProcess {
	if proc == nil {
		return nil
	}

	return &AgentProcess{
		PID:       proc.PID,
		AgentName: proc.AgentName,
		Command:   proc.Command,
		Args:      append([]string(nil), proc.Args...),
		Cwd:       proc.Cwd,
		SessionID: proc.SessionID,
		Caps:      proc.Caps,
		StartedAt: proc.StartedAt,
		done:      proc.Done(),
		waitFn:    proc.Wait,
		stderrFn:  proc.Stderr,
		approvePermissionFn: func(ctx context.Context, req acp.ApproveRequest) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			return proc.ResolvePermission(req)
		},
		native: proc,
	}
}

// AgentDriver defines the ACP functionality consumed by the session manager.
type AgentDriver interface {
	Start(ctx context.Context, opts acp.StartOpts) (*AgentProcess, error)
	Prompt(ctx context.Context, proc *AgentProcess, req acp.PromptRequest) (<-chan acp.AgentEvent, error)
	Cancel(ctx context.Context, proc *AgentProcess) error
	Stop(ctx context.Context, proc *AgentProcess) error
}

// EventRecorder is the per-session storage surface consumed by session/.
type EventRecorder = store.EventRecorder

// Notifier fans out session lifecycle and prompt events to downstream observers.
type Notifier interface {
	OnSessionCreated(ctx context.Context, session *Session)
	OnSessionStopped(ctx context.Context, session *Session)
	OnAgentEvent(ctx context.Context, sessionID string, event acp.AgentEvent)
}

// PromptAssembler assembles the prompt context for a new session start.
type PromptAssembler interface {
	Assemble(ctx context.Context, agent aghconfig.AgentDef, workspace workspacepkg.ResolvedWorkspace) (string, error)
}

// SkillRegistry resolves the active skill set for a workspace during session start.
type SkillRegistry interface {
	ForWorkspace(ctx context.Context, resolved workspacepkg.ResolvedWorkspace) ([]*skillspkg.Skill, error)
}

// MCPResolver resolves skill-declared MCP servers into runtime config entries.
type MCPResolver interface {
	Resolve(skills []*skillspkg.Skill) []aghconfig.MCPServer
}

// ACPDriverAdapter adapts the concrete ACP driver to the session-local interface.
type ACPDriverAdapter struct {
	driver *acp.Driver
}

var _ AgentDriver = (*ACPDriverAdapter)(nil)

// NewACPDriverAdapter wraps the provided ACP driver for use by the session manager.
func NewACPDriverAdapter(driver *acp.Driver) *ACPDriverAdapter {
	return &ACPDriverAdapter{driver: driver}
}

// Start launches a new ACP-backed runtime process.
func (a *ACPDriverAdapter) Start(ctx context.Context, opts acp.StartOpts) (*AgentProcess, error) {
	proc, err := a.driver.Start(ctx, opts)
	if err != nil {
		return nil, err
	}
	return wrapACPProcess(proc), nil
}

// Prompt streams prompt events from the wrapped ACP runtime.
func (a *ACPDriverAdapter) Prompt(ctx context.Context, proc *AgentProcess, req acp.PromptRequest) (<-chan acp.AgentEvent, error) {
	native, err := a.nativeProcess(proc)
	if err != nil {
		return nil, err
	}
	return a.driver.Prompt(ctx, native, req)
}

// Cancel cooperatively cancels the active ACP prompt.
func (a *ACPDriverAdapter) Cancel(ctx context.Context, proc *AgentProcess) error {
	native, err := a.nativeProcess(proc)
	if err != nil {
		return err
	}
	return a.driver.Cancel(ctx, native)
}

// Stop stops the wrapped ACP runtime process.
func (a *ACPDriverAdapter) Stop(ctx context.Context, proc *AgentProcess) error {
	native, err := a.nativeProcess(proc)
	if err != nil {
		return err
	}
	return a.driver.Stop(ctx, native)
}

func (a *ACPDriverAdapter) nativeProcess(proc *AgentProcess) (*acp.AgentProcess, error) {
	if proc == nil {
		return nil, errors.New("session: agent process is required")
	}

	native, ok := proc.native.(*acp.AgentProcess)
	if !ok || native == nil {
		return nil, errors.New("session: unsupported agent process implementation")
	}
	return native, nil
}
