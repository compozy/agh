package acpmock

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pedronauck/agh/internal/acp"
)

const FixtureVersion = 2

const (
	aghSituationContextOpen             = "<agh-situation-context>"
	aghSituationContextClose            = "</agh-situation-context>"
	aghCurrentSkillsOpen                = "<current-available-skills>"
	aghCurrentSkillsClose               = "</current-available-skills>"
	aghCurrentSkillsLastInstructionLine = "If current tool policy denies `agh__skill_view`, use `agh skill view <name>` as an operator fallback."
	aghDurableMemoryOpen                = "Relevant durable memory for this turn:"
	aghDurableMemoryUserMessageMarker   = "\n\nUser message:\n"
)

type StepKind string

const (
	StepKindAssistant     StepKind = "assistant"
	StepKindThought       StepKind = "thought"
	StepKindToolCall      StepKind = "tool_call"
	StepKindPermission    StepKind = "permission"
	StepKindSandbox       StepKind = "sandbox_exec"
	StepKindBridgeContent StepKind = "bridge_response"
	StepKindDriverControl StepKind = "driver_control"
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

// TurnMatch routes a prompt to a turn fixture using exact stable fields only.
type TurnMatch struct {
	TurnSource string            `json:"turn_source,omitempty"`
	UserText   string            `json:"user_text,omitempty"`
	Occurrence int               `json:"occurrence,omitempty"`
	Network    *TurnMatchNetwork `json:"network,omitempty"`
}

// TurnMatchNetwork captures exact AGH network envelope field matching.
type TurnMatchNetwork struct {
	MessageID   string `json:"message_id,omitempty"`
	Kind        string `json:"kind,omitempty"`
	Channel     string `json:"channel,omitempty"`
	Surface     string `json:"surface,omitempty"`
	ThreadID    string `json:"thread_id,omitempty"`
	DirectID    string `json:"direct_id,omitempty"`
	From        string `json:"from,omitempty"`
	To          string `json:"to,omitempty"`
	WorkID      string `json:"work_id,omitempty"`
	ReplyTo     string `json:"reply_to,omitempty"`
	TraceID     string `json:"trace_id,omitempty"`
	CausationID string `json:"causation_id,omitempty"`
	Trust       string `json:"trust,omitempty"`
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

	Command              string             `json:"command,omitempty"`
	Args                 []string           `json:"args,omitempty"`
	Cwd                  string             `json:"cwd,omitempty"`
	ExpectExitCode       *int               `json:"expect_exit_code,omitempty"`
	ExpectOutputContains string             `json:"expect_output_contains,omitempty"`
	ExpectErrorContains  string             `json:"expect_error_contains,omitempty"`
	EmitOutput           bool               `json:"emit_output,omitempty"`
	DriverControl        *DriverControlStep `json:"driver_control,omitempty"`
}

// DriverControlStep injects driver-level protocol or lifecycle faults.
type DriverControlStep struct {
	Action     DriverControlAction `json:"action"`
	RawJSONRPC string              `json:"raw_jsonrpc,omitempty"`
	Async      bool                `json:"async,omitempty"`
	DelayMS    int                 `json:"delay_ms,omitempty"`
}

// DriverControlAction identifies one supported driver fault injection action.
type DriverControlAction string

const (
	DriverControlDisconnect       DriverControlAction = "disconnect"
	DriverControlWriteRawJSONRPC  DriverControlAction = "write_raw_jsonrpc"
	DriverControlBlockUntilCancel DriverControlAction = "block_until_cancel"
)

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
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&fixture); err != nil {
		return Fixture{}, err
	}
	if decoder.More() {
		return Fixture{}, errors.New("acpmock: fixture JSON must contain exactly one document")
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
	for idx := range f.Agents {
		name := strings.TrimSpace(f.Agents[idx].Name)
		if name == "" {
			return fmt.Errorf("acpmock: agents[%d].name is required", idx)
		}
		f.Agents[idx].Name = name
		if _, ok := seen[name]; ok {
			return fmt.Errorf("acpmock: duplicate agent %q", name)
		}
		seen[name] = struct{}{}
		agent := f.Agents[idx]
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
func (a AgentFixture) SelectTurn(prompt string, occurrence int, meta ...acp.PromptMeta) (TurnFixture, error) {
	if occurrence <= 0 {
		return TurnFixture{}, fmt.Errorf("acpmock: prompt occurrence %d must be >= 1", occurrence)
	}

	input := turnMatchInput{
		UserText: strings.TrimSpace(prompt),
	}
	if len(meta) > 0 {
		input.Meta = meta[0].Normalize()
	}

	for _, turn := range a.Turns {
		if turn.Match.matches(input, occurrence) {
			return turn, nil
		}
	}

	return TurnFixture{}, fmt.Errorf(
		"acpmock: no turn matched agent %q prompt %q at occurrence %d with meta %#v",
		a.Name,
		input.UserText,
		occurrence,
		input.Meta,
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

// Validate ensures the turn match contains only supported exact selectors.
func (m TurnMatch) Validate(path string) error {
	if m.Occurrence < 0 {
		return fmt.Errorf("acpmock: %s.occurrence must be >= 0", path)
	}
	normalized := m.Normalize()
	switch normalized.TurnSource {
	case "", acp.PromptTurnSourceUser, acp.PromptTurnSourceNetwork:
	default:
		return fmt.Errorf("acpmock: %s.turn_source %q is invalid", path, normalized.TurnSource)
	}
	if normalized.TurnSource == "" && normalized.UserText == "" && normalized.Network == nil {
		return fmt.Errorf("acpmock: %s requires at least one exact selector", path)
	}
	if normalized.Network != nil {
		if err := normalized.Network.Validate(path + ".network"); err != nil {
			return err
		}
	}
	return nil
}

// Normalize returns a trimmed copy of the turn matcher.
func (m TurnMatch) Normalize() TurnMatch {
	normalized := TurnMatch{
		TurnSource: strings.TrimSpace(m.TurnSource),
		UserText:   strings.TrimSpace(m.UserText),
		Occurrence: m.Occurrence,
	}
	if m.Network != nil {
		network := m.Network.Normalize()
		if !network.IsZero() {
			normalized.Network = &network
		}
	}
	return normalized
}

type turnMatchInput struct {
	UserText string
	Meta     acp.PromptMeta
}

func (m TurnMatch) matches(input turnMatchInput, occurrence int) bool {
	normalized := m.Normalize()
	if normalized.Occurrence > 0 && normalized.Occurrence != occurrence {
		return false
	}
	if normalized.TurnSource != "" && input.Meta.Normalize().TurnSource != normalized.TurnSource {
		return false
	}
	if normalized.UserText != "" && canonicalUserText(input.UserText) != normalized.UserText {
		return false
	}
	if normalized.Network != nil {
		if input.Meta.Network == nil {
			return false
		}
		if !normalized.Network.matches(*input.Meta.Network) {
			return false
		}
	}
	return true
}

func canonicalUserText(prompt string) string {
	current := strings.TrimSpace(prompt)
	for {
		next := stripLeadingSituationContext(current)
		next = stripLeadingCurrentSkillsCatalog(next)
		next = stripLeadingDurableMemory(next)
		next = strings.TrimSpace(next)
		if next == current {
			return current
		}
		current = next
	}
}

func stripLeadingSituationContext(prompt string) string {
	trimmed := strings.TrimSpace(prompt)
	if !strings.HasPrefix(trimmed, aghSituationContextOpen) {
		return trimmed
	}

	_, after, ok := strings.Cut(trimmed, aghSituationContextClose)
	if !ok {
		return trimmed
	}

	return strings.TrimSpace(after)
}

func stripLeadingCurrentSkillsCatalog(prompt string) string {
	trimmed := strings.TrimSpace(prompt)
	if !strings.HasPrefix(trimmed, aghCurrentSkillsOpen) {
		return trimmed
	}

	_, afterClose, ok := strings.Cut(trimmed, aghCurrentSkillsClose)
	if !ok {
		return trimmed
	}

	afterClose = strings.TrimSpace(afterClose)
	_, afterInstructions, ok := strings.Cut(afterClose, aghCurrentSkillsLastInstructionLine)
	if ok {
		return strings.TrimSpace(afterInstructions)
	}
	return afterClose
}

func stripLeadingDurableMemory(prompt string) string {
	trimmed := strings.TrimSpace(prompt)
	if !strings.HasPrefix(trimmed, aghDurableMemoryOpen) {
		return trimmed
	}

	_, after, ok := strings.Cut(trimmed, aghDurableMemoryUserMessageMarker)
	if !ok {
		return trimmed
	}
	return strings.TrimSpace(after)
}

// Normalize returns a trimmed copy of the network matcher.
func (m TurnMatchNetwork) Normalize() TurnMatchNetwork {
	return TurnMatchNetwork{
		MessageID:   strings.TrimSpace(m.MessageID),
		Kind:        strings.TrimSpace(m.Kind),
		Channel:     strings.TrimSpace(m.Channel),
		Surface:     strings.TrimSpace(m.Surface),
		ThreadID:    strings.TrimSpace(m.ThreadID),
		DirectID:    strings.TrimSpace(m.DirectID),
		From:        strings.TrimSpace(m.From),
		To:          strings.TrimSpace(m.To),
		WorkID:      strings.TrimSpace(m.WorkID),
		ReplyTo:     strings.TrimSpace(m.ReplyTo),
		TraceID:     strings.TrimSpace(m.TraceID),
		CausationID: strings.TrimSpace(m.CausationID),
		Trust:       strings.TrimSpace(m.Trust),
	}
}

// IsZero reports whether the network matcher carries any fields.
func (m TurnMatchNetwork) IsZero() bool {
	return m.Normalize() == (TurnMatchNetwork{})
}

// Validate ensures only exact-match network selectors are configured.
func (m TurnMatchNetwork) Validate(path string) error {
	if m.IsZero() {
		return fmt.Errorf("acpmock: %s requires at least one network selector", path)
	}
	return nil
}

func (m TurnMatchNetwork) matches(meta acp.PromptNetworkMeta) bool {
	want := m.Normalize()
	got := meta.Normalize()
	return exactStringMatch(want.MessageID, got.MessageID) &&
		exactStringMatch(want.Kind, got.Kind) &&
		exactStringMatch(want.Channel, got.Channel) &&
		exactStringMatch(want.Surface, got.Surface) &&
		exactStringMatch(want.ThreadID, got.ThreadID) &&
		exactStringMatch(want.DirectID, got.DirectID) &&
		exactStringMatch(want.From, got.From) &&
		exactStringMatch(want.To, got.To) &&
		exactStringMatch(want.WorkID, got.WorkID) &&
		exactStringMatch(want.ReplyTo, got.ReplyTo) &&
		exactStringMatch(want.TraceID, got.TraceID) &&
		exactStringMatch(want.CausationID, got.CausationID) &&
		exactStringMatch(want.Trust, got.Trust)
}

func exactStringMatch(want string, got string) bool {
	if strings.TrimSpace(want) == "" {
		return true
	}
	return strings.TrimSpace(got) == strings.TrimSpace(want)
}

// Validate ensures the step kind and payload are internally consistent.
func (s Step) Validate(path string) error {
	if err := s.validateKindPayload(path); err != nil {
		return err
	}
	if strings.TrimSpace(s.Cwd) != "" && !filepath.IsAbs(strings.TrimSpace(s.Cwd)) {
		return fmt.Errorf("acpmock: %s.cwd must be absolute when set", path)
	}

	return nil
}

func (s Step) validateKindPayload(path string) error {
	switch s.Kind {
	case StepKindAssistant, StepKindThought, StepKindBridgeContent:
		return validateTextStep(path, s)
	case StepKindToolCall:
		return validateToolCallStep(path, s)
	case StepKindPermission:
		return validatePermissionStep(path, s)
	case StepKindSandbox:
		return validateSandboxStep(path, s)
	case StepKindDriverControl:
		return validateDriverControlStep(path, s)
	default:
		return fmt.Errorf("acpmock: %s.kind %q is invalid", path, s.Kind)
	}
}

func validateTextStep(path string, step Step) error {
	if !hasTextPayload(step.Text, step.Chunks) {
		return fmt.Errorf("acpmock: %s requires text or chunks", path)
	}
	return nil
}

func validateToolCallStep(path string, step Step) error {
	if strings.TrimSpace(step.ToolCallID) == "" {
		return fmt.Errorf("acpmock: %s.tool_call_id is required", path)
	}
	if strings.TrimSpace(step.Title) == "" {
		return fmt.Errorf("acpmock: %s.title is required", path)
	}
	if err := validateToolKind(path+".tool_kind", step.ToolKind); err != nil {
		return err
	}
	return validateToolStatus(path+".status", step.Status)
}

func validatePermissionStep(path string, step Step) error {
	if strings.TrimSpace(step.ToolCallID) == "" {
		return fmt.Errorf("acpmock: %s.tool_call_id is required", path)
	}
	if err := validateToolKind(path+".tool_kind", step.ToolKind); err != nil {
		return err
	}
	if err := validateToolStatus(path+".status", step.Status); err != nil {
		return err
	}
	return validatePermissionDecision(path+".expect_decision", step.ExpectDecision)
}

func validateSandboxStep(path string, step Step) error {
	if strings.TrimSpace(step.Command) == "" {
		return fmt.Errorf("acpmock: %s.command is required", path)
	}
	if err := validateToolKind(path+".tool_kind", step.ToolKind); err != nil {
		return err
	}
	return validateToolStatus(path+".status", step.Status)
}

func validateDriverControlStep(path string, step Step) error {
	if step.DriverControl == nil {
		return fmt.Errorf("acpmock: %s.driver_control is required", path)
	}
	return step.DriverControl.Validate(path + ".driver_control")
}

// Validate ensures the driver-control payload is internally consistent.
func (d DriverControlStep) Validate(path string) error {
	if d.DelayMS < 0 {
		return fmt.Errorf("acpmock: %s.delay_ms must be >= 0", path)
	}
	switch d.Action {
	case DriverControlDisconnect, DriverControlBlockUntilCancel:
		if strings.TrimSpace(d.RawJSONRPC) != "" {
			return fmt.Errorf("acpmock: %s.raw_jsonrpc is only valid for write_raw_jsonrpc", path)
		}
	case DriverControlWriteRawJSONRPC:
		if strings.TrimSpace(d.RawJSONRPC) == "" {
			return fmt.Errorf("acpmock: %s.raw_jsonrpc is required", path)
		}
	default:
		return fmt.Errorf("acpmock: %s.action %q is invalid", path, d.Action)
	}
	if d.Async && d.Action == DriverControlBlockUntilCancel {
		return fmt.Errorf("acpmock: %s.async is invalid for block_until_cancel", path)
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
