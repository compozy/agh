package openapits

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Artifact describes one checked-in OpenAPI document and its generated type output.
type Artifact struct {
	SpecPath   string
	OutputPath string
}

// ErrStaleGeneratedFile reports that the committed generated file no longer matches the source spec.
var ErrStaleGeneratedFile = errors.New("generated file is stale")

// ErrMissingGeneratedFile reports that the committed generated file does not exist.
var ErrMissingGeneratedFile = errors.New("generated file is missing")

// Generate runs openapi-typescript for one artifact and formats the generated output with oxfmt.
func Generate(ctx context.Context, artifact Artifact) error {
	if err := os.MkdirAll(filepath.Dir(artifact.OutputPath), 0o755); err != nil {
		return fmt.Errorf("create output directory %q: %w", filepath.Dir(artifact.OutputPath), err)
	}

	if err := runCommand(ctx, "bunx", "openapi-typescript", artifact.SpecPath, "-o", artifact.OutputPath); err != nil {
		return fmt.Errorf("generate %q from %q: %w", artifact.OutputPath, artifact.SpecPath, err)
	}

	if err := runCommand(ctx, "bunx", "oxfmt", artifact.OutputPath); err != nil {
		return fmt.Errorf("format %q: %w", artifact.OutputPath, err)
	}

	return nil
}

// Check regenerates one artifact into a temporary file and fails when the checked-in output differs.
func Check(ctx context.Context, artifact Artifact) (err error) {
	file, err := os.CreateTemp("", "openapi-types-*.d.ts")
	if err != nil {
		return fmt.Errorf("create temporary output for %q: %w", artifact.OutputPath, err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close temporary output for %q: %w", artifact.OutputPath, err)
	}
	defer func() {
		removeErr := os.Remove(file.Name())
		if removeErr == nil || errors.Is(removeErr, os.ErrNotExist) {
			return
		}
		err = errors.Join(err, fmt.Errorf("remove temporary output %q: %w", file.Name(), removeErr))
	}()

	if err := Generate(ctx, Artifact{
		SpecPath:   artifact.SpecPath,
		OutputPath: file.Name(),
	}); err != nil {
		return err
	}

	want, err := os.ReadFile(file.Name())
	if err != nil {
		return fmt.Errorf("read generated output %q: %w", file.Name(), err)
	}

	return checkGeneratedFile(artifact.OutputPath, want)
}

func checkGeneratedFile(path string, want []byte) error {
	got, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%s: %w; run codegen", path, ErrMissingGeneratedFile)
		}
		return fmt.Errorf("read %q: %w", path, err)
	}

	if !bytes.Equal(got, want) {
		return fmt.Errorf("%s: %w; run codegen", path, ErrStaleGeneratedFile)
	}

	return nil
}

func runCommand(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = strings.TrimSpace(stdout.String())
		}
		if detail == "" {
			return fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
		}
		return fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, detail)
	}

	return nil
}
