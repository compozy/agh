package daytona

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/pedronauck/agh/internal/environment"
)

func (p *daytonaProvider) SyncToRuntime(
	ctx context.Context,
	state environment.SessionState,
	opts environment.SyncOptions,
) (environment.SyncResult, error) {
	if ctx == nil {
		return environment.SyncResult{}, fmt.Errorf("environment/daytona: sync to runtime context is required")
	}
	if state.Backend != environment.BackendDaytona {
		return environment.SyncResult{}, fmt.Errorf("environment/daytona: sync to runtime backend = %q", state.Backend)
	}
	providerState, err := decodeProviderState(state.ProviderState)
	if err != nil {
		return environment.SyncResult{}, err
	}
	if providerState.LocalRootDir == "" || providerState.RuntimeRootDir == "" {
		return environment.SyncResult{}, fmt.Errorf("environment/daytona: sync to runtime missing root mapping")
	}
	roots := append(
		[]syncRoot{{local: providerState.LocalRootDir, runtime: providerState.RuntimeRootDir}},
		additionalSyncRoots(providerState.LocalAdditionalDirs, providerState.RuntimeAdditionalDirs)...,
	)
	sandbox := sandboxInfo{
		ID:                 providerState.SandboxID,
		APIURL:             providerState.APIURL,
		SSHHost:            providerState.SSHHost,
		SSHAccessExpiresAt: state.SSHAccessExpiresAt,
	}
	result := environment.SyncResult{}
	for _, root := range roots {
		stats, err := p.syncOneToRuntime(ctx, sandbox, root, opts)
		result.FilesSynced += stats.FilesSynced
		result.BytesTransferred += stats.BytesTransferred
		if err != nil {
			result.Errors = append(result.Errors, err.Error())
			return result, err
		}
	}
	return result, nil
}

func (p *daytonaProvider) SyncFromRuntime(
	ctx context.Context,
	state environment.SessionState,
	opts environment.SyncOptions,
) (environment.SyncResult, error) {
	if ctx == nil {
		return environment.SyncResult{}, fmt.Errorf("environment/daytona: sync from runtime context is required")
	}
	if state.Backend != environment.BackendDaytona {
		return environment.SyncResult{}, fmt.Errorf(
			"environment/daytona: sync from runtime backend = %q",
			state.Backend,
		)
	}
	providerState, err := decodeProviderState(state.ProviderState)
	if err != nil {
		return environment.SyncResult{}, err
	}
	if providerState.LocalRootDir == "" || providerState.RuntimeRootDir == "" {
		return environment.SyncResult{}, fmt.Errorf("environment/daytona: sync from runtime missing root mapping")
	}
	roots := append(
		[]syncRoot{{local: providerState.LocalRootDir, runtime: providerState.RuntimeRootDir}},
		additionalSyncRoots(providerState.LocalAdditionalDirs, providerState.RuntimeAdditionalDirs)...,
	)
	sandbox := sandboxInfo{
		ID:                 providerState.SandboxID,
		APIURL:             providerState.APIURL,
		SSHHost:            providerState.SSHHost,
		SSHAccessExpiresAt: state.SSHAccessExpiresAt,
	}
	result := environment.SyncResult{}
	for _, root := range roots {
		stats, err := p.syncOneFromRuntime(ctx, sandbox, root, opts)
		result.FilesSynced += stats.FilesSynced
		result.BytesTransferred += stats.BytesTransferred
		if err != nil {
			result.Errors = append(result.Errors, err.Error())
			return result, err
		}
	}
	return result, nil
}

type syncRoot struct {
	local   string
	runtime string
}

func additionalSyncRoots(localDirs []string, runtimeDirs []string) []syncRoot {
	limit := min(len(runtimeDirs), len(localDirs))
	roots := make([]syncRoot, 0, limit)
	for i := range limit {
		roots = append(roots, syncRoot{local: localDirs[i], runtime: runtimeDirs[i]})
	}
	return roots
}

func (p *daytonaProvider) syncOneToRuntime(
	ctx context.Context,
	sandbox sandboxInfo,
	root syncRoot,
	opts environment.SyncOptions,
) (environment.SyncResult, error) {
	session, err := p.transport.Dial(ctx, sandbox, remoteExtractCommand(root.runtime))
	if err != nil {
		return environment.SyncResult{}, fmt.Errorf(
			"environment/daytona: open sync-to-runtime SSH stream for %q: %w",
			root.runtime,
			err,
		)
	}
	stats, writeErr := writeTar(ctx, filepath.Clean(root.local), session, opts.ExcludePatterns)
	closeErr := session.CloseWrite()
	waitErr := session.Wait()
	if closeErr != nil {
		closeErr = fmt.Errorf("environment/daytona: close sync-to-runtime stream for %q: %w", root.runtime, closeErr)
	}
	if waitErr != nil {
		waitErr = fmt.Errorf(
			"environment/daytona: remote extract %q failed: %w stderr=%q",
			root.runtime,
			waitErr,
			session.Stderr(),
		)
	}
	if err := session.Close(); err != nil {
		p.logger.Warn("environment/daytona: close sync-to-runtime SSH session failed", "error", err)
	}
	result := environment.SyncResult{FilesSynced: stats.Files, BytesTransferred: stats.Bytes}
	if err := joinSyncErrors(writeErr, closeErr, waitErr); err != nil {
		result.Errors = append(result.Errors, err.Error())
		return result, err
	}
	p.logger.Info(
		"environment/daytona: synced workspace to runtime",
		"reason",
		string(opts.Reason),
		"local",
		root.local,
		"runtime",
		root.runtime,
		"files",
		stats.Files,
		"bytes",
		stats.Bytes,
	)
	return result, nil
}

func (p *daytonaProvider) syncOneFromRuntime(
	ctx context.Context,
	sandbox sandboxInfo,
	root syncRoot,
	opts environment.SyncOptions,
) (environment.SyncResult, error) {
	session, err := p.transport.Dial(ctx, sandbox, remoteArchiveCommand(root.runtime))
	if err != nil {
		return environment.SyncResult{}, fmt.Errorf(
			"environment/daytona: open sync-from-runtime SSH stream for %q: %w",
			root.runtime,
			err,
		)
	}
	stats, extractErr := extractTar(filepath.Clean(root.local), session)
	waitErr := session.Wait()
	if waitErr != nil {
		waitErr = fmt.Errorf(
			"environment/daytona: remote archive %q failed: %w stderr=%q",
			root.runtime,
			waitErr,
			session.Stderr(),
		)
	}
	if err := session.Close(); err != nil {
		p.logger.Warn("environment/daytona: close sync-from-runtime SSH session failed", "error", err)
	}
	result := environment.SyncResult{FilesSynced: stats.Files, BytesTransferred: stats.Bytes}
	if err := joinSyncErrors(extractErr, waitErr); err != nil {
		result.Errors = append(result.Errors, err.Error())
		return result, err
	}
	p.logger.Info(
		"environment/daytona: synced runtime to workspace",
		"reason",
		string(opts.Reason),
		"local",
		root.local,
		"runtime",
		root.runtime,
		"files",
		stats.Files,
		"bytes",
		stats.Bytes,
	)
	return result, nil
}

func joinSyncErrors(errs ...error) error {
	return errors.Join(errs...)
}
