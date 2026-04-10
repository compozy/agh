package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pedronauck/agh/internal/api/spec"
	"github.com/pedronauck/agh/internal/codegen/sdkts"
)

const defaultSDKContractsPath = "sdk/typescript/src/generated/contracts.ts"

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
	return spec.WriteFile(path)
}

func writeSDKContracts(path string) error {
	content, err := sdkts.Generate()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func checkOpenAPI(path string) error {
	doc, err := spec.Document()
	if err != nil {
		return err
	}
	want, err := marshalOpenAPI(doc)
	if err != nil {
		return err
	}
	return checkFile(path, want)
}

func checkSDKContracts(path string) error {
	content, err := sdkts.Generate()
	if err != nil {
		return err
	}
	return checkFile(path, []byte(content))
}

func marshalOpenAPI(doc any) ([]byte, error) {
	file, err := os.CreateTemp("", "agh-openapi-*.json")
	if err != nil {
		return nil, err
	}
	_ = file.Close()
	defer func() {
		_ = os.Remove(file.Name())
	}()

	if err := spec.WriteFile(file.Name()); err != nil {
		return nil, err
	}
	return os.ReadFile(file.Name())
}

func checkFile(path string, want []byte) error {
	got, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s is missing; run codegen", path)
		}
		return err
	}
	if !bytes.Equal(got, want) {
		return fmt.Errorf("%s is stale; run codegen", path)
	}
	return nil
}
