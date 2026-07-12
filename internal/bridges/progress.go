package bridges

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	redactpkg "github.com/compozy/agh/internal/redact"
)

const maxToolProgressErrorRunes = 160

// ToolProgressPhase is one closed tool lifecycle phase sent to bridge adapters.
type ToolProgressPhase string

const (
	ToolProgressPhaseStarted   ToolProgressPhase = "started"
	ToolProgressPhaseCompleted ToolProgressPhase = "completed"
	ToolProgressPhaseFailed    ToolProgressPhase = "failed"
)

// ToolProgressPhaseValues returns the closed wire values in stable order.
func ToolProgressPhaseValues() []string {
	return []string{
		string(ToolProgressPhaseStarted),
		string(ToolProgressPhaseCompleted),
		string(ToolProgressPhaseFailed),
	}
}

const (
	projectionEventAgentMessage = "agent_message"
	projectionEventDone         = "done"
	projectionEventError        = "error"
	projectionEventToolCall     = "tool_call"
	projectionEventToolResult   = "tool_result"
)

// Normalize returns the canonical lifecycle phase.
func (p ToolProgressPhase) Normalize() ToolProgressPhase {
	return ToolProgressPhase(strings.ToLower(strings.TrimSpace(string(p))))
}

// Validate reports whether the phase belongs to the bridge progress contract.
func (p ToolProgressPhase) Validate() error {
	switch p.Normalize() {
	case ToolProgressPhaseStarted, ToolProgressPhaseCompleted, ToolProgressPhaseFailed:
		return nil
	case "":
		return errors.New("bridges: tool progress phase is required")
	default:
		return fmt.Errorf("bridges: unsupported tool progress phase %q", p)
	}
}

// ToolProgress is the presentation-only tool lifecycle payload delivered to adapters.
type ToolProgress struct {
	ToolCallID string            `json:"tool_call_id"`
	ToolID     string            `json:"tool_id"`
	Phase      ToolProgressPhase `json:"phase"`
	Label      string            `json:"label"`
	Preview    string            `json:"preview,omitempty"`
	Emoji      string            `json:"emoji,omitempty"`
	DurationMS int64             `json:"duration_ms,omitempty"`
	Error      string            `json:"error,omitempty"`
	Index      int               `json:"index"`
}

// Validate enforces the typed, redacted progress wire contract.
func (p ToolProgress) Validate() error {
	normalized := p.normalize()
	if err := requireField(normalized.ToolCallID, "tool progress tool call id"); err != nil {
		return err
	}
	if err := requireField(normalized.ToolID, "tool progress tool id"); err != nil {
		return err
	}
	if err := normalized.Phase.Validate(); err != nil {
		return err
	}
	if normalized.DurationMS < 0 {
		return errors.New("bridges: tool progress duration cannot be negative")
	}
	if normalized.Index <= 0 {
		return errors.New("bridges: tool progress index must be positive")
	}
	for label, value := range map[string]string{
		"label":   normalized.Label,
		"preview": normalized.Preview,
		"emoji":   normalized.Emoji,
		"error":   normalized.Error,
	} {
		if strings.ContainsAny(value, "\r\n") {
			return fmt.Errorf("bridges: tool progress %s must be one line", label)
		}
		if redactpkg.String(value) != value {
			return fmt.Errorf("bridges: tool progress %s contains secret-shaped material", label)
		}
	}
	if utf8.RuneCountInString(normalized.Error) > maxToolProgressErrorRunes {
		return fmt.Errorf("bridges: tool progress error exceeds %d runes", maxToolProgressErrorRunes)
	}
	return nil
}

func (p ToolProgress) normalize() ToolProgress {
	p.ToolCallID = strings.TrimSpace(p.ToolCallID)
	p.ToolID = strings.TrimSpace(p.ToolID)
	p.Phase = p.Phase.Normalize()
	p.Label = strings.TrimSpace(p.Label)
	p.Preview = strings.TrimSpace(p.Preview)
	p.Emoji = strings.TrimSpace(p.Emoji)
	p.Error = strings.TrimSpace(p.Error)
	return p
}

func sanitizeToolProgressError(value string) string {
	value = redactpkg.String(strings.Join(strings.Fields(value), " "))
	if utf8.RuneCountInString(value) <= maxToolProgressErrorRunes {
		return value
	}
	return strings.TrimSpace(string([]rune(value)[:maxToolProgressErrorRunes]))
}

func cloneToolProgress(progress *ToolProgress) *ToolProgress {
	if progress == nil {
		return nil
	}
	cloned := progress.normalize()
	return &cloned
}

// ProjectAgentEvent is the single presentation projection gate used by live
// and replay delivery paths. The returned event is a template: the broker adds
// delivery identity, accumulated text, sequence, index, and rendered chrome.
func ProjectAgentEvent(
	event DeliveryProjectionEvent,
	mode ProgressMode,
) (DeliveryEvent, bool, error) {
	normalized := event.normalize()
	switch normalized.Type {
	case projectionEventAgentMessage:
		if normalized.Text == "" {
			return DeliveryEvent{}, false, nil
		}
		return DeliveryEvent{
			EventType: DeliveryEventTypeDelta,
			Content:   MessageContent{Text: normalized.Text},
		}, true, nil
	case projectionEventDone:
		return DeliveryEvent{EventType: DeliveryEventTypeFinal, Final: true}, true, nil
	case projectionEventError:
		return DeliveryEvent{
			EventType: DeliveryEventTypeError,
			Final:     true,
			Error:     &DeliveryErrorDetail{Message: normalized.Error},
		}, true, nil
	case projectionEventToolCall, projectionEventToolResult:
		return projectToolAgentEvent(normalized, mode)
	default:
		return DeliveryEvent{}, false, nil
	}
}

func projectToolAgentEvent(
	event DeliveryProjectionEvent,
	mode ProgressMode,
) (DeliveryEvent, bool, error) {
	mode = mode.Normalize()
	if err := mode.Validate(); err != nil {
		return DeliveryEvent{}, false, err
	}
	if mode == ProgressModeOff {
		return DeliveryEvent{}, false, nil
	}
	if event.Tool == nil {
		return DeliveryEvent{}, false, fmt.Errorf("bridges: %s projection requires tool progress", event.Type)
	}

	progress := event.Tool.normalize()
	if progress.ToolCallID == "" {
		return DeliveryEvent{}, false, errors.New("bridges: projected tool call id is required")
	}
	if event.Type == projectionEventToolCall {
		if progress.ToolID == "" {
			return DeliveryEvent{}, false, errors.New("bridges: projected tool id is required")
		}
		progress.Phase = ToolProgressPhaseStarted
	} else if progress.Phase != ToolProgressPhaseFailed {
		progress.Phase = ToolProgressPhaseCompleted
	}
	if err := progress.Phase.Validate(); err != nil {
		return DeliveryEvent{}, false, err
	}
	if progress.DurationMS < 0 {
		return DeliveryEvent{}, false, errors.New("bridges: projected tool duration cannot be negative")
	}

	// ACP input is not a trusted presentation surface. Chrome is always rebuilt
	// daemon-side after redaction and the index is allocated only after enqueue.
	progress.Label = ""
	progress.Preview = ""
	progress.Emoji = ""
	progress.Error = sanitizeToolProgressError(progress.Error)
	progress.Index = 0
	return DeliveryEvent{
		EventType: DeliveryEventTypeProgress,
		Progress:  &progress,
	}, true, nil
}
