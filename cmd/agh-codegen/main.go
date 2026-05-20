package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/pedronauck/agh/internal/api/spec"
	"github.com/pedronauck/agh/internal/codegen/sdkts"
	"github.com/pedronauck/agh/internal/config/lifecycle"
	"github.com/pedronauck/agh/internal/fileutil"
)

const (
	subcommandCheck = "check"
)

const defaultSDKContractsPath = "sdk/typescript/src/generated/contracts.ts"
const defaultLifecycleMatrixPath = "packages/site/content/runtime/core/configuration/lifecycle-matrix.mdx"

var ErrStaleGeneratedFile = errors.New("generated file is stale")

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), shutdownSignals()...)
	err := run(ctx, os.Args[1:])
	stop()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	return runWithPaths(ctx, args, spec.DefaultPath, defaultSDKContractsPath)
}

func runWithPaths(ctx context.Context, args []string, openapiPath string, sdkContractsPath string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: agh-codegen <openapi|sdk-contracts|lifecycle-matrix|all|check>")
	}
	lifecycleMatrixPath := lifecycleMatrixPathFor(openapiPath)

	switch args[0] {
	case "openapi":
		return writeOpenAPI(openapiPath)
	case "sdk-contracts":
		return writeSDKContracts(ctx, sdkContractsPath)
	case "lifecycle-matrix":
		return writeLifecycleMatrix(lifecycleMatrixPath)
	case "all":
		return writeAll(ctx, openapiPath, sdkContractsPath, lifecycleMatrixPath)
	case subcommandCheck:
		if err := checkOpenAPI(openapiPath); err != nil {
			return err
		}
		if err := checkSDKContracts(ctx, sdkContractsPath); err != nil {
			return err
		}
		return checkLifecycleMatrix(lifecycleMatrixPath)
	default:
		return fmt.Errorf("unknown codegen target %q", args[0])
	}
}

func writeOpenAPI(path string) error {
	content, err := marshalOpenAPI()
	if err != nil {
		return fmt.Errorf("write openapi to %q: %w", path, err)
	}
	if err := publishGeneratedFile(path, content); err != nil {
		return fmt.Errorf("write openapi to %q: %w", path, err)
	}
	return nil
}

func writeSDKContracts(ctx context.Context, path string) error {
	content, err := generateFormattedSDKContracts(ctx, path)
	if err != nil {
		return err
	}
	if err := publishGeneratedFile(path, content); err != nil {
		return fmt.Errorf("write sdk contracts to %q: %w", path, err)
	}
	return nil
}

func writeAll(ctx context.Context, openapiPath string, sdkContractsPath string, lifecycleMatrixPath string) error {
	if err := writeAllWith(
		ctx,
		openapiPath,
		sdkContractsPath,
		marshalOpenAPI,
		generateFormattedSDKContracts,
		publishGeneratedFile,
	); err != nil {
		return err
	}
	return writeLifecycleMatrix(lifecycleMatrixPath)
}

func writeAllWith(
	ctx context.Context,
	openapiPath string,
	sdkContractsPath string,
	generateOpenAPI func() ([]byte, error),
	generateSDK func(context.Context, string) ([]byte, error),
	publishFile func(string, []byte) error,
) error {
	openapiContent, err := generateOpenAPI()
	if err != nil {
		return fmt.Errorf("write openapi to %q: %w", openapiPath, err)
	}

	sdkContent, err := generateSDK(ctx, sdkContractsPath)
	if err != nil {
		return err
	}

	previousOpenAPI, openapiExisted, err := readGeneratedFile(openapiPath)
	if err != nil {
		return fmt.Errorf("read existing openapi %q: %w", openapiPath, err)
	}

	if err := publishFile(openapiPath, openapiContent); err != nil {
		return fmt.Errorf("write openapi to %q: %w", openapiPath, err)
	}
	if err := publishFile(sdkContractsPath, sdkContent); err != nil {
		if restoreErr := restoreGeneratedFile(
			openapiPath,
			previousOpenAPI,
			openapiExisted,
			publishFile,
		); restoreErr != nil {
			return fmt.Errorf(
				"write sdk contracts to %q: %w; restore openapi %q: %v",
				sdkContractsPath,
				err,
				openapiPath,
				restoreErr,
			)
		}
		return fmt.Errorf("write sdk contracts to %q: %w", sdkContractsPath, err)
	}

	return nil
}

func publishGeneratedFile(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create output directory %q: %w", filepath.Dir(path), err)
	}
	if err := fileutil.AtomicWriteFile(path, content, 0o600); err != nil {
		return err
	}
	return nil
}

func readGeneratedFile(path string) ([]byte, bool, error) {
	content, err := os.ReadFile(path)
	if err == nil {
		return content, true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	return nil, false, err
}

func restoreGeneratedFile(path string, content []byte, existed bool, publishFile func(string, []byte) error) error {
	if existed {
		return publishFile(path, content)
	}
	if err := fileutil.AtomicRemoveFile(path); err != nil {
		return err
	}
	return nil
}

func checkOpenAPI(path string) error {
	want, err := marshalOpenAPI()
	if err != nil {
		return err
	}
	return checkJSONFile(path, want)
}

func checkSDKContracts(ctx context.Context, path string) error {
	content, err := generateFormattedSDKContracts(ctx, path)
	if err != nil {
		return err
	}
	return checkFile(path, content)
}

func writeLifecycleMatrix(path string) error {
	content := generateLifecycleMatrixMDX()
	if err := publishGeneratedFile(path, content); err != nil {
		return fmt.Errorf("write lifecycle matrix to %q: %w", path, err)
	}
	return nil
}

func checkLifecycleMatrix(path string) error {
	return checkFile(path, generateLifecycleMatrixMDX())
}

func marshalOpenAPI() ([]byte, error) {
	data, err := spec.Render()
	if err != nil {
		return nil, fmt.Errorf("render openapi: %w", err)
	}
	return data, nil
}

func checkFile(path string, want []byte) error {
	got, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s is missing; run codegen", path)
		}
		return fmt.Errorf("read %q: %w", path, err)
	}
	if !bytes.Equal(got, want) {
		return fmt.Errorf("%s: %w; run codegen", path, ErrStaleGeneratedFile)
	}
	return nil
}

func checkJSONFile(path string, want []byte) error {
	got, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s is missing; run codegen", path)
		}
		return fmt.Errorf("read %q: %w", path, err)
	}

	gotCanonical, err := canonicalJSON(got)
	if err != nil {
		return fmt.Errorf("decode %q: %w", path, err)
	}
	wantCanonical, err := canonicalJSON(want)
	if err != nil {
		return fmt.Errorf("decode generated json for %q: %w", path, err)
	}
	if !bytes.Equal(gotCanonical, wantCanonical) {
		return fmt.Errorf("%s: %w; run codegen", path, ErrStaleGeneratedFile)
	}
	return nil
}

func canonicalJSON(data []byte) ([]byte, error) {
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, err
	}
	return json.Marshal(value)
}

func shutdownSignals() []os.Signal {
	return []os.Signal{os.Interrupt, syscall.SIGTERM}
}

func generateFormattedSDKContracts(ctx context.Context, path string) ([]byte, error) {
	content, err := sdkts.Generate()
	if err != nil {
		return nil, fmt.Errorf("generate sdk contracts: %w", err)
	}
	formatted, err := formatTypeScript(ctx, path, []byte(content))
	if err != nil {
		return nil, err
	}
	return formatted, nil
}

func lifecycleMatrixPathFor(openapiPath string) string {
	if filepath.Clean(openapiPath) == filepath.Clean(spec.DefaultPath) {
		return defaultLifecycleMatrixPath
	}
	return filepath.Join(filepath.Dir(openapiPath), "lifecycle-matrix.mdx")
}

func generateLifecycleMatrixMDX() []byte {
	var out strings.Builder
	out.WriteString("---\n")
	out.WriteString("title: Config Lifecycle Matrix\n")
	out.WriteString(
		"description: Generated AGH config lifecycle matrix for live apply, session rebind, and restart-required changes.\n",
	)
	out.WriteString("---\n\n")
	out.WriteString("{/* Code generated by go run ./cmd/agh-codegen lifecycle-matrix. DO NOT EDIT. */}\n\n")
	out.WriteString(
		"`config.toml` is desired state. The daemon active generation advances only when `ConfigApplyService` can apply the desired change to runtime truth.\n\n",
	)
	out.WriteString("## Lifecycle Values\n\n")
	out.WriteString("| Lifecycle | Runtime effect |\n")
	out.WriteString("| --- | --- |\n")
	for _, value := range []lifecycle.Lifecycle{
		lifecycle.Live,
		lifecycle.LiveAdd,
		lifecycle.LiveRemoveIfUnused,
		lifecycle.SessionRebind,
		lifecycle.RestartRequired,
	} {
		out.WriteString("| `")
		out.WriteString(string(value))
		out.WriteString("` | ")
		out.WriteString(lifecycleDescription(value))
		out.WriteString(" |\n")
	}
	out.WriteString("\n## Matrix\n\n")
	out.WriteString("| Key path pattern | Lifecycle | Diff class | Next action when not immediately active |\n")
	out.WriteString("| --- | --- | --- | --- |\n")
	for _, rule := range lifecycle.SortedMatrix() {
		out.WriteString("| `")
		out.WriteString(rule.Pattern)
		out.WriteString("` | `")
		out.WriteString(string(rule.Lifecycle))
		out.WriteString("` | `")
		out.WriteString(string(rule.DiffClass))
		out.WriteString("` | `")
		out.WriteString(string(nextActionForDocs(rule.Lifecycle)))
		out.WriteString("` |\n")
	}
	out.WriteString("\n## New Live Reload Budgets\n\n")
	out.WriteString("| Key | Default | Validation |\n")
	out.WriteString("| --- | --- | --- |\n")
	out.WriteString("| `daemon.reload_timeouts.providers` | `5s` | At least `1s` and at most `60s`. |\n")
	out.WriteString("| `daemon.reload_timeouts.mcp` | `10s` | At least `1s` and at most `60s`. |\n")
	out.WriteString("| `daemon.reload_timeouts.bridges` | `30s` | At least `1s` and at most `300s`. |\n")
	return []byte(out.String())
}

func lifecycleDescription(value lifecycle.Lifecycle) string {
	switch value {
	case lifecycle.Live:
		return "Applies in place and advances the active config generation."
	case lifecycle.LiveAdd:
		return "Adds a new runtime entry in place and advances the active generation."
	case lifecycle.LiveRemoveIfUnused:
		return "Removes a runtime entry only when it has no active users."
	case lifecycle.SessionRebind:
		return "Applies to new sessions while existing sessions keep their bound values."
	case lifecycle.RestartRequired:
		return "Writes desired state and records a blocked apply until the daemon restarts."
	default:
		return "Unknown lifecycle."
	}
}

func nextActionForDocs(value lifecycle.Lifecycle) lifecycle.NextAction {
	switch value {
	case lifecycle.RestartRequired:
		return lifecycle.NextActionRestartDaemon
	case lifecycle.SessionRebind:
		return lifecycle.NextActionNewSession
	default:
		return lifecycle.NextActionNone
	}
}

func formatTypeScript(ctx context.Context, path string, content []byte) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "bunx", "oxfmt", "--stdin-filepath", path)
	cmd.Stdin = bytes.NewReader(content)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			return nil, fmt.Errorf("format typescript %q with oxfmt: %w", path, err)
		}
		return nil, fmt.Errorf("format typescript %q with oxfmt: %w: %s", path, err, detail)
	}
	return stdout.Bytes(), nil
}

func removeTemporaryFile(path string) error {
	err := os.Remove(path)
	if err == nil || errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return fmt.Errorf("remove temporary file %q: %w", path, err)
}
