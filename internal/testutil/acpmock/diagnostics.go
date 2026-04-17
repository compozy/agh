package acpmock

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/pedronauck/agh/internal/acp"
)

// DiagnosticsRecord captures one prompt execution emitted by the ACP mock driver.
type DiagnosticsRecord struct {
	AgentName   string            `json:"agent_name"`
	SessionID   string            `json:"session_id"`
	PromptIndex int               `json:"prompt_index"`
	Prompt      string            `json:"prompt"`
	PromptMeta  acp.PromptMeta    `json:"prompt_meta"`
	TurnName    string            `json:"turn_name,omitempty"`
	Match       TurnMatch         `json:"match"`
	Steps       []DiagnosticsStep `json:"steps"`
}

// DiagnosticsStep captures one executed fixture step with any observed runtime outputs.
type DiagnosticsStep struct {
	Kind         StepKind            `json:"kind"`
	Text         string              `json:"text,omitempty"`
	ToolCallID   string              `json:"tool_call_id,omitempty"`
	Decision     string              `json:"decision,omitempty"`
	Command      string              `json:"command,omitempty"`
	Args         []string            `json:"args,omitempty"`
	ExitCode     *int                `json:"exit_code,omitempty"`
	Output       string              `json:"output,omitempty"`
	Error        string              `json:"error,omitempty"`
	DriverAction DriverControlAction `json:"driver_action,omitempty"`
}

// ReadDiagnostics decodes newline-delimited diagnostics written by the mock driver.
func ReadDiagnostics(path string) ([]DiagnosticsRecord, error) {
	file, err := os.Open(strings.TrimSpace(path))
	if err != nil {
		return nil, fmt.Errorf("acpmock: open diagnostics %q: %w", path, err)
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

	records := make([]DiagnosticsRecord, 0, 4)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var record DiagnosticsRecord
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			return nil, fmt.Errorf("acpmock: decode diagnostics line %d: %w", lineNo, err)
		}
		records = append(records, record)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("acpmock: scan diagnostics %q: %w", path, err)
	}
	return records, nil
}
