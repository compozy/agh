package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/pedronauck/agh/internal/api/spec"
	"github.com/pedronauck/agh/internal/codegen/sdkts"
)

const defaultSDKContractsPath = "sdk/typescript/src/generated/contracts.ts"

var ErrStaleGeneratedFile = errors.New("generated file is stale")

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: agh-codegen <openapi|sdk-contracts|all|check>")
	}

	switch args[0] {
	case "openapi":
		return writeOpenAPI(spec.DefaultPath)
	case "sdk-contracts":
		return writeSDKContracts(defaultSDKContractsPath)
	case "all":
		if err := writeOpenAPI(spec.DefaultPath); err != nil {
			return err
		}
		return writeSDKContracts(defaultSDKContractsPath)
	case "check":
		if err := checkOpenAPI(spec.DefaultPath); err != nil {
			return err
		}
		return checkSDKContracts(defaultSDKContractsPath)
	default:
		return fmt.Errorf("unknown codegen target %q", args[0])
	}
}

func writeOpenAPI(path string) error {
	if err := spec.WriteFile(path); err != nil {
		return fmt.Errorf("write openapi to %q: %w", path, err)
	}
	return nil
}

func writeSDKContracts(path string) error {
	content, err := generateFormattedSDKContracts(path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create sdk contracts directory %q: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return fmt.Errorf("write sdk contracts to %q: %w", path, err)
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

func checkSDKContracts(path string) error {
	content, err := generateFormattedSDKContracts(path)
	if err != nil {
		return err
	}
	return checkFile(path, content)
}

func marshalOpenAPI() ([]byte, error) {
	file, err := os.CreateTemp("", "agh-openapi-*.json")
	if err != nil {
		return nil, fmt.Errorf("create temporary openapi file: %w", err)
	}
	if err := file.Close(); err != nil {
		return nil, fmt.Errorf("close temporary openapi file %q: %w", file.Name(), err)
	}
	defer func() {
		_ = os.Remove(file.Name())
	}()

	if err := spec.WriteFile(file.Name()); err != nil {
		return nil, fmt.Errorf("write openapi to temporary file %q: %w", file.Name(), err)
	}
	data, err := os.ReadFile(file.Name())
	if err != nil {
		return nil, fmt.Errorf("read temporary openapi file %q: %w", file.Name(), err)
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
	if !reflect.DeepEqual(gotCanonical, wantCanonical) {
		return fmt.Errorf("%s: %w; run codegen", path, ErrStaleGeneratedFile)
	}
	return nil
}

func canonicalJSON(data []byte) (any, error) {
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, err
	}
	return value, nil
}

func generateFormattedSDKContracts(path string) ([]byte, error) {
	content, err := sdkts.Generate()
	if err != nil {
		return nil, fmt.Errorf("generate sdk contracts: %w", err)
	}
	formatted, err := formatTypeScript(path, []byte(content))
	if err != nil {
		return nil, err
	}
	return formatted, nil
}

func formatTypeScript(path string, content []byte) ([]byte, error) {
	cmd := exec.Command("bunx", "oxfmt", "--stdin-filepath", path)
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
