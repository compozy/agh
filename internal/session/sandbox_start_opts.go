package session

import (
	"fmt"
	"strings"

	"github.com/compozy/agh/internal/acp"
	envpkg "github.com/compozy/agh/internal/sandbox"
)

func sandboxStartOpts(
	opts acp.StartOpts,
	prepared envpkg.Prepared,
	state envpkg.SessionState,
) (acp.StartOpts, error) {
	next := opts
	if command := strings.TrimSpace(prepared.Launch.Command); command != "" {
		next.Command = command
	}
	if prepared.Launch.Env != nil {
		next.Env = append([]string(nil), prepared.Launch.Env...)
	}
	cwd, err := sandboxRuntimeCWD(opts.Cwd, prepared, state)
	if err != nil {
		return acp.StartOpts{}, err
	}
	next.Cwd = cwd
	next.AdditionalDirs = append([]string(nil), prepared.RuntimeAdditionalDirs...)
	if next.AdditionalDirs == nil {
		next.AdditionalDirs = append([]string(nil), state.RuntimeAdditionalDirs...)
	}
	next.Launcher = prepared.Launcher
	next.ToolHost = prepared.ToolHost
	return next, nil
}

func sandboxRuntimeCWD(
	requested string,
	prepared envpkg.Prepared,
	state envpkg.SessionState,
) (string, error) {
	runtimeRoot := strings.TrimSpace(prepared.RuntimeRootDir)
	if runtimeRoot == "" {
		runtimeRoot = strings.TrimSpace(state.RuntimeRootDir)
	}
	requested = strings.TrimSpace(requested)
	if state.Backend == envpkg.BackendLocal {
		canonicalRoot, err := canonicalDirectory(runtimeRoot)
		if err != nil {
			return "", fmt.Errorf("%w: resolve local sandbox root: %w", ErrValidation, err)
		}
		if requested == "" {
			return canonicalRoot, nil
		}
		canonicalTarget, err := canonicalDirectory(requested)
		if err != nil {
			return "", fmt.Errorf("%w: resolve local sandbox cwd: %w", ErrValidation, err)
		}
		contained, err := directoryContains(canonicalRoot, canonicalTarget)
		if err != nil {
			return "", fmt.Errorf("%w: compare local sandbox cwd: %w", ErrValidation, err)
		}
		if !contained {
			return canonicalRoot, nil
		}
		cwd, err := resolveContainedDirectory(canonicalRoot, canonicalTarget)
		if err != nil {
			return "", fmt.Errorf("%w: resolve local sandbox cwd: %w", ErrValidation, err)
		}
		return cwd, nil
	}
	return runtimeRoot, nil
}
