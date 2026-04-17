package acpmock

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const FixtureVersion = 1

type StepKind string

const (
	StepKindAssistant     StepKind = "assistant"
	StepKindThought       StepKind = "thought"
	StepKindToolCall      StepKind = "tool_call"
	StepKindPermission    StepKind = "permission"
	StepKindEnvironment   StepKind = "environment_exec"
	StepKindBridgeContent StepKind = "bridge_response"
)

// Fixture describes one deterministic multi-agent ACP mock scenario.
type Fixture struct {
	Version int            `json:"version"`
	Agents  []AgentFixture `json:"agents"`
}

// AgentFixture describes one named ACP mock agent inside a fixture file.
type AgentFixture struct {
	Name        string        `json:"name"`
	Provider    string        `json:"provider"`
	Model       string        `json:"model,omitempty"`
	Permissions string        `json:"permissions,omitempty"`
	Prompt      string        `json:"prompt,omitempty"`
	Turns       []TurnFixture `json:"turns"`
}

// TurnFixture describes one deterministic prompt turn for an agent.
type TurnFixture struct {
	Name       string    `json:"name,omitempty"`
	Match      TurnMatch `json:"match"`
	Steps      []Step    `json:"steps"`
	StopReason string    `json:"stop_reason,omitempty"`
}

// TurnMatch routes a prompt to a turn fixture.
type TurnMatch struct {
	Equals     string `json:"equals,omitempty"`
	Contains   string `json:"contains,omitempty"`
	Occurrence int    `json:"occurrence,omitempty"`
}

// Step describes one deterministic ACP action emitted or executed by the driver.
type Step struct {
	Kind StepKind `json:"kind"`

	Text   string   `json:"text,omitempty"`
	Chunks []string `json:"chunks,omitempty"`

	ToolCallID  string          `json:"tool_call_id,omitempty"`
	Title       string          `json:"title,omitempty"`
	ToolKind    string          `json:"tool_kind,omitempty"`
	Path        string          `json:"path,omitempty"`
	Status      string          `json:"status,omitempty"`
	ContentText string          `json:"content_text,omitempty"`
	RawInput    json.RawMessage `json:"raw_input,omitempty"`
	RawOutput   json.RawMessage `json:"raw_output,omitempty"`

	ExpectDecision string `json:"expect_decision,omitempty"`
	EmitDecision   bool   `json:"emit_decision,omitempty"`
	EmitText       string `json:"emit_text,omitempty"`

	Command              string   `json:"command,omitempty"`
	Args                 []string `json:"args,omitempty"`
	Cwd                  string   `json:"cwd,omitempty"`
	ExpectExitCode       *int     `json:"expect_exit_code,omitempty"`
	ExpectOutputContains string   `json:"expect_output_contains,omitempty"`
	ExpectErrorContains  string   `json:"expect_error_contains,omitempty"`
	EmitOutput           bool     `json:"emit_output,omitempty"`
}

// LoadFixture parses and validates one fixture file.
func LoadFixture(path string) (Fixture, error) {
	target := strings.TrimSpace(path)
	if target == "" {
		return Fixture{}, errors.New("acpmock: fixture path is required")
	}

	data, err := os.ReadFile(target)
	if err != nil {
		return Fixture{}, fmt.Errorf("acpmock: read fixture %q: %w", target, err)
	}
	fixture, err := ParseFixture(data)
	if err != nil {
		return Fixture{}, fmt.Errorf("acpmock: parse fixture %q: %w", target, err)
	}
	return fixture, nil
}

// ParseFixture decodes and validates fixture JSON.
func ParseFixture(data []byte) (Fixture, error) {
	var fixture Fixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		return Fixture{}, err
	}
	if err := fixture.Validate(); err != nil {
		return Fixture{}, err
	}
	return fixture, nil
}

// Validate ensures the fixture can drive deterministic ACP scenarios.
func (f Fixture) Validate() error {
	if f.Version != FixtureVersion {
		return fmt.Errorf("acpmock: fixture version %d, want %d", f.Version, FixtureVersion)
	}
	if len(f.Agents) == 0 {
		return errors.New("acpmock: at least one fixture agent is required")
	}

	seen := make(map[string]struct{}, len(f.Agents))
	for idx, agent := range f.Agents {
		name := strings.TrimSpace(agent.Name)
		if name == "" {
			return fmt.Errorf("acpmock: agents[%d].name is required", idx)
		}
		if _, ok := seen[name]; ok {
			return fmt.Errorf("acpmock: duplicate agent %q", name)
		}
		seen[name] = struct{}{}
		if err := agent.Validate(fmt.Sprintf("agents[%d]", idx)); err != nil {
			return err
		}
	}

	return nil
}

// Agent returns one named fixture agent.
func (f Fixture) Agent(name string) (AgentFixture, error) {
	target := strings.TrimSpace(name)
	if target == "" {
		return AgentFixture{}, errors.New("acpmock: fixture agent name is required")
	}
	for _, agent := range f.Agents {
		if agent.Name == target {
			return agent, nil
		}
	}
	return AgentFixture{}, fmt.Errorf("acpmock: fixture agent %q not found", target)
}

// SelectTurn returns the first turn that matches the supplied prompt occurrence.
func (a AgentFixture) SelectTurn(prompt string, occurrence int) (TurnFixture, error) {
	if occurrence <= 0 {
		return TurnFixture{}, fmt.Errorf("acpmock: prompt occurrence %d must be >= 1", occurrence)
	}

	text := strings.TrimSpace(prompt)
	for _, turn := range a.Turns {
		if turn.Match.matches(text, occurrence) {
			return turn, nil
		}
	}

	return TurnFixture{}, fmt.Errorf(
		"acpmock: no turn matched agent %q prompt %q at occurrence %d",
		a.Name,
		text,
		occurrence,
	)
}

// Validate ensures the agent fixture is usable.
func (a AgentFixture) Validate(path string) error {
	if strings.TrimSpace(a.Provider) == "" {
		return fmt.Errorf("acpmock: %s.provider is required", path)
	}
	if len(a.Turns) == 0 {
		return fmt.Errorf("acpmock: %s.turns must contain at least one turn", path)
	}
	for idx, turn := range a.Turns {
		if err := turn.Validate(fmt.Sprintf("%s.turns[%d]", path, idx)); err != nil {
			return err
		}
	}
	return nil
}

// Validate ensures the turn fixture is usable.
func (t TurnFixture) Validate(path string) error {
	if err := t.Match.Validate(path + ".match"); err != nil {
		return err
	}
	if len(t.Steps) == 0 {
		return fmt.Errorf("acpmock: %s.steps must contain at least one step", path)
	}
	for idx, step := range t.Steps {
		if err := step.Validate(fmt.Sprintf("%s.steps[%d]", path, idx)); err != nil {
			return err
		}
	}
	if stopReason := strings.TrimSpace(t.StopReason); stopReason != "" {
		switch stopReason {
		case "end_turn", "canceled":
		default:
			return fmt.Errorf("acpmock: %s.stop_reason %q is invalid", path, stopReason)
		}
	}
	return nil
}

// Validate ensures the turn match contains only supported selectors.
func (m TurnMatch) Validate(path string) error {
	if m.Occurrence < 0 {
		return fmt.Errorf("acpmock: %s.occurrence must be >= 0", path)
	}
	if strings.TrimSpace(m.Equals) != "" && strings.TrimSpace(m.Contains) != "" {
		return fmt.Errorf("acpmock: %s cannot set both equals and contains", path)
	}
	return nil
}

func (m TurnMatch) matches(prompt string, occurrence int) bool {
	if m.Occurrence > 0 && m.Occurrence != occurrence {
		return false
	}

	switch {
	case strings.TrimSpace(m.Equals) != "":
		return prompt == strings.TrimSpace(m.Equals)
	case strings.TrimSpace(m.Contains) != "":
		return strings.Contains(prompt, strings.TrimSpace(m.Contains))
	default:
		return true
	}
}

// Validate ensures the step kind and payload are internally consistent.
func (s Step) Validate(path string) error {
	switch s.Kind {
	case StepKindAssistant, StepKindThought, StepKindBridgeContent:
		if !hasTextPayload(s.Text, s.Chunks) {
			return fmt.Errorf("acpmock: %s requires text or chunks", path)
		}
	case StepKindToolCall:
		if strings.TrimSpace(s.ToolCallID) == "" {
			return fmt.Errorf("acpmock: %s.tool_call_id is required", path)
		}
		if strings.TrimSpace(s.Title) == "" {
			return fmt.Errorf("acpmock: %s.title is required", path)
		}
		if err := validateToolKind(path+".tool_kind", s.ToolKind); err != nil {
			return err
		}
		if err := validateToolStatus(path+".status", s.Status); err != nil {
			return err
		}
	case StepKindPermission:
		if strings.TrimSpace(s.ToolCallID) == "" {
			return fmt.Errorf("acpmock: %s.tool_call_id is required", path)
		}
		if err := validateToolKind(path+".tool_kind", s.ToolKind); err != nil {
			return err
		}
		if err := validateToolStatus(path+".status", s.Status); err != nil {
			return err
		}
		if err := validatePermissionDecision(path+".expect_decision", s.ExpectDecision); err != nil {
			return err
		}
	case StepKindEnvironment:
		if strings.TrimSpace(s.Command) == "" {
			return fmt.Errorf("acpmock: %s.command is required", path)
		}
		if err := validateToolKind(path+".tool_kind", s.ToolKind); err != nil {
			return err
		}
		if err := validateToolStatus(path+".status", s.Status); err != nil {
			return err
		}
	default:
		return fmt.Errorf("acpmock: %s.kind %q is invalid", path, s.Kind)
	}

	if strings.TrimSpace(s.Cwd) != "" && !filepath.IsAbs(strings.TrimSpace(s.Cwd)) {
		return fmt.Errorf("acpmock: %s.cwd must be absolute when set", path)
	}

	return nil
}

func hasTextPayload(text string, chunks []string) bool {
	if strings.TrimSpace(text) != "" {
		return true
	}
	for _, chunk := range chunks {
		if strings.TrimSpace(chunk) != "" {
			return true
		}
	}
	return false
}

func validateToolKind(path string, raw string) error {
	switch strings.TrimSpace(raw) {
	case "", "read", "edit", "delete", "move", "search", "execute", "think", "fetch", "switch_mode", "other":
		return nil
	default:
		return fmt.Errorf("acpmock: %s %q is invalid", path, raw)
	}
}

func validateToolStatus(path string, raw string) error {
	switch strings.TrimSpace(raw) {
	case "", "pending", "in_progress", "completed", "failed":
		return nil
	default:
		return fmt.Errorf("acpmock: %s %q is invalid", path, raw)
	}
}

func validatePermissionDecision(path string, raw string) error {
	switch strings.TrimSpace(raw) {
	case "", "allow-once", "allow-always", "reject-once", "reject-always":
		return nil
	default:
		return fmt.Errorf("acpmock: %s %q is invalid", path, raw)
	}
}
