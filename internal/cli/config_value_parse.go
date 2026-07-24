package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	shellquote "github.com/kballard/go-shellquote"
	"github.com/spf13/cobra"
)

func parseConfigSetValue(kind configSetValueKind, raw string) (any, error) {
	trimmed := strings.TrimSpace(raw)
	switch kind {
	case configSetString:
		return raw, nil
	case configSetBool:
		value, err := strconv.ParseBool(trimmed)
		if err != nil {
			return nil, fmt.Errorf("cli: parse bool value %q: %w", raw, err)
		}
		return value, nil
	case configSetInt:
		value, err := strconv.Atoi(trimmed)
		if err != nil {
			return nil, fmt.Errorf("cli: parse integer value %q: %w", raw, err)
		}
		return value, nil
	case configSetInt64:
		value, err := strconv.ParseInt(trimmed, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("cli: parse integer value %q: %w", raw, err)
		}
		return value, nil
	case configSetFloat:
		value, err := strconv.ParseFloat(trimmed, 64)
		if err != nil {
			return nil, fmt.Errorf("cli: parse float value %q: %w", raw, err)
		}
		return value, nil
	case configSetDuration:
		if _, err := time.ParseDuration(trimmed); err != nil {
			return nil, fmt.Errorf("cli: parse duration value %q: %w", raw, err)
		}
		return trimmed, nil
	case configSetStringSlice:
		return parseStringSliceValue(trimmed)
	case configSetFloatSlice:
		return parseFloatSliceValue(trimmed)
	default:
		return nil, fmt.Errorf("cli: unsupported config value kind %d", kind)
	}
}

func parseStringSliceValue(raw string) ([]string, error) {
	if strings.TrimSpace(raw) == "" {
		return []string{}, nil
	}
	if strings.HasPrefix(strings.TrimSpace(raw), "[") {
		var values []string
		if err := json.Unmarshal([]byte(raw), &values); err != nil {
			return nil, fmt.Errorf("cli: parse string array %q: %w", raw, err)
		}
		return values, nil
	}
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return values, nil
}

func parseFloatSliceValue(raw string) ([]float64, error) {
	var values []float64
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil, fmt.Errorf("cli: parse float array %q: %w", raw, err)
	}
	return values, nil
}

func runConfigEditor(cmd *cobra.Command, deps commandDeps, path string) error {
	editor := strings.TrimSpace(deps.getenv("VISUAL"))
	if editor == "" {
		editor = strings.TrimSpace(deps.getenv("EDITOR"))
	}
	if editor == "" {
		return errors.New("cli: config edit requires VISUAL or EDITOR")
	}
	parts, err := shellquote.Split(editor)
	if err != nil {
		return fmt.Errorf("cli: parse config editor command: %w", err)
	}
	if len(parts) == 0 {
		return errors.New("cli: config editor command is empty")
	}
	//nolint:gosec // VISUAL/EDITOR intentionally selects the local editor for config edit.
	editorCmd := exec.CommandContext(cmd.Context(), parts[0], append(parts[1:], path)...)
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = cmd.OutOrStdout()
	editorCmd.Stderr = cmd.ErrOrStderr()
	if err := editorCmd.Run(); err != nil {
		return fmt.Errorf("cli: run config editor %q: %w", parts[0], err)
	}
	return nil
}
