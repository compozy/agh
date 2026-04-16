// Package local implements the daemon-host execution environment provider.
package local

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/environment"
)

var _ environment.Provider = (*localProvider)(nil)

// Option customizes the local provider.
type Option func(*localProvider)

type localProvider struct {
	logger         *slog.Logger
	stopTimeout    time.Duration
	permissionMode aghconfig.PermissionMode
}

// WithLogger directs provider-created launcher and tool-host diagnostics to logger.
func WithLogger(logger *slog.Logger) Option {
	return func(provider *localProvider) {
		provider.logger = logger
	}
}

// WithStopTimeout configures how long local launcher stop waits before escalation.
func WithStopTimeout(timeout time.Duration) Option {
	return func(provider *localProvider) {
		provider.stopTimeout = timeout
	}
}

// WithPermissionMode configures the local tool host permission policy.
func WithPermissionMode(mode aghconfig.PermissionMode) Option {
	return func(provider *localProvider) {
		provider.permissionMode = mode
	}
}

// NewProvider returns the local daemon-host environment provider.
func NewProvider(opts ...Option) environment.Provider {
	provider := &localProvider{
		logger:         slog.Default(),
		permissionMode: aghconfig.PermissionModeApproveReads,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(provider)
		}
	}
	if provider.logger == nil {
		provider.logger = slog.Default()
	}
	return provider
}

// NewRegistry returns a provider registry with local registered as the default backend.
func NewRegistry(opts ...Option) (*environment.Registry, error) {
	return environment.NewRegistry(NewProvider(opts...))
}

func (p *localProvider) Backend() environment.Backend {
	return environment.BackendLocal
}

func (p *localProvider) Prepare(
	ctx context.Context,
	req environment.PrepareRequest,
) (environment.Prepared, error) {
	if ctx == nil {
		return environment.Prepared{}, errors.New("environment/local: prepare context is required")
	}

	launcher := acp.NewLocalLauncher(p.logger, p.stopTimeout)
	toolHost, err := acp.NewLocalToolHost(ctx, req.LocalRootDir, p.permissionModeFor(req), p.logger)
	if err != nil {
		return environment.Prepared{}, fmt.Errorf("environment/local: create tool host: %w", err)
	}

	runtimeAdditionalDirs := cloneStrings(req.LocalAdditionalDirs)
	launchAdditionalDirs := cloneStrings(runtimeAdditionalDirs)
	agentEnv := cloneStrings(req.AgentEnv)
	preparedState := environment.SessionState{
		EnvironmentID:         req.EnvironmentID,
		Backend:               environment.BackendLocal,
		Profile:               req.Environment.Profile,
		InstanceID:            strings.TrimSpace(req.InstanceID),
		RuntimeRootDir:        req.LocalRootDir,
		RuntimeAdditionalDirs: cloneStrings(runtimeAdditionalDirs),
		ProviderState:         cloneRawMessage(req.ProviderState),
		PreparedAt:            time.Now().UTC(),
	}

	return environment.Prepared{
		State:                 preparedState,
		RuntimeRootDir:        req.LocalRootDir,
		RuntimeAdditionalDirs: runtimeAdditionalDirs,
		Launcher:              launcher,
		Launch: environment.LaunchSpec{
			Command:        req.AgentCommand,
			Cwd:            req.LocalRootDir,
			AdditionalDirs: launchAdditionalDirs,
			Env:            agentEnv,
		},
		ToolHost: toolHost,
	}, nil
}

func (p *localProvider) SyncToRuntime(
	ctx context.Context,
	_ environment.SessionState,
	_ environment.SyncOptions,
) (environment.SyncResult, error) {
	if ctx == nil {
		return environment.SyncResult{}, errors.New("environment/local: sync to runtime context is required")
	}
	return environment.SyncResult{}, nil
}

func (p *localProvider) SyncFromRuntime(
	ctx context.Context,
	_ environment.SessionState,
	_ environment.SyncOptions,
) (environment.SyncResult, error) {
	if ctx == nil {
		return environment.SyncResult{}, errors.New("environment/local: sync from runtime context is required")
	}
	return environment.SyncResult{}, nil
}

func (p *localProvider) Destroy(ctx context.Context, _ environment.SessionState) error {
	if ctx == nil {
		return errors.New("environment/local: destroy context is required")
	}
	return nil
}

func (p *localProvider) permissionModeFor(req environment.PrepareRequest) aghconfig.PermissionMode {
	if mode := strings.TrimSpace(req.Permissions); mode != "" {
		return aghconfig.PermissionMode(mode)
	}
	return p.permissionMode
}

func cloneStrings(values []string) []string {
	if values == nil {
		return nil
	}
	cloned := make([]string, len(values))
	copy(cloned, values)
	return cloned
}

func cloneRawMessage(value json.RawMessage) json.RawMessage {
	if value == nil {
		return nil
	}
	cloned := make(json.RawMessage, len(value))
	copy(cloned, value)
	return cloned
}
