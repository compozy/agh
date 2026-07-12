package bridges

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// InboundEventFamily identifies the typed inbound bridge event family.
type InboundEventFamily string

const (
	// InboundEventFamilyMessage identifies a text-and-attachment message event.
	InboundEventFamilyMessage InboundEventFamily = "message"
	// InboundEventFamilyCommand identifies a typed slash-command style event.
	InboundEventFamilyCommand InboundEventFamily = "command"
	// InboundEventFamilyAction identifies a typed button/action event.
	InboundEventFamilyAction InboundEventFamily = "action"
	// InboundEventFamilyReaction identifies a typed reaction add/remove event.
	InboundEventFamilyReaction InboundEventFamily = "reaction"
	// InboundEventFamilyEdit identifies an update or deletion of an existing message.
	InboundEventFamilyEdit InboundEventFamily = "edit"
)

// InboundEventFamilyValues returns the closed wire values in stable order.
func InboundEventFamilyValues() []string {
	return []string{
		string(InboundEventFamilyMessage),
		string(InboundEventFamilyCommand),
		string(InboundEventFamilyAction),
		string(InboundEventFamilyReaction),
		string(InboundEventFamilyEdit),
	}
}

// Normalize returns the canonical inbound event-family representation.
func (f InboundEventFamily) Normalize() InboundEventFamily {
	return InboundEventFamily(strings.ToLower(strings.TrimSpace(string(f))))
}

// Validate reports whether the inbound event family belongs to the supported set.
func (f InboundEventFamily) Validate() error {
	switch f.Normalize() {
	case InboundEventFamilyMessage,
		InboundEventFamilyCommand,
		InboundEventFamilyAction,
		InboundEventFamilyReaction,
		InboundEventFamilyEdit:
		return nil
	case "":
		return errors.New("bridges: inbound event family is required")
	default:
		return fmt.Errorf("bridges: unsupported inbound event family %q", strings.TrimSpace(string(f)))
	}
}

// InboundCommand captures a typed slash-command style inbound interaction.
type InboundCommand struct {
	Command   string `json:"command"`
	Text      string `json:"text,omitempty"`
	TriggerID string `json:"trigger_id,omitempty"`
}

// Validate reports whether the command payload contains the required identity.
func (c InboundCommand) Validate() error {
	return requireField(strings.TrimSpace(c.Command), "inbound command")
}

func (c InboundCommand) normalize() InboundCommand {
	return InboundCommand{
		Command:   strings.TrimSpace(c.Command),
		Text:      strings.TrimSpace(c.Text),
		TriggerID: strings.TrimSpace(c.TriggerID),
	}
}

// InboundAction captures a typed button/action inbound interaction.
type InboundAction struct {
	ActionID  string `json:"action_id"`
	MessageID string `json:"message_id,omitempty"`
	Value     string `json:"value,omitempty"`
	TriggerID string `json:"trigger_id,omitempty"`
}

// Validate reports whether the action payload contains the required identity.
func (a InboundAction) Validate() error {
	return requireField(strings.TrimSpace(a.ActionID), "inbound action id")
}

func (a InboundAction) normalize() InboundAction {
	return InboundAction{
		ActionID:  strings.TrimSpace(a.ActionID),
		MessageID: strings.TrimSpace(a.MessageID),
		Value:     strings.TrimSpace(a.Value),
		TriggerID: strings.TrimSpace(a.TriggerID),
	}
}

// InboundReaction captures a typed reaction add/remove inbound interaction.
type InboundReaction struct {
	MessageID string `json:"message_id"`
	Emoji     string `json:"emoji"`
	RawEmoji  string `json:"raw_emoji,omitempty"`
	Added     bool   `json:"added"`
}

// Validate reports whether the reaction payload contains the required identity.
func (r InboundReaction) Validate() error {
	if err := requireField(strings.TrimSpace(r.MessageID), "inbound reaction message id"); err != nil {
		return err
	}
	return requireField(strings.TrimSpace(r.Emoji), "inbound reaction emoji")
}

func (r InboundReaction) normalize() InboundReaction {
	return InboundReaction{
		MessageID: strings.TrimSpace(r.MessageID),
		Emoji:     strings.TrimSpace(r.Emoji),
		RawEmoji:  strings.TrimSpace(r.RawEmoji),
		Added:     r.Added,
	}
}

// InboundEditOperation distinguishes replacement text from message deletion.
type InboundEditOperation string

const (
	// InboundEditOperationUpdated replaces the existing message text.
	InboundEditOperationUpdated InboundEditOperation = "updated"
	// InboundEditOperationDeleted removes the existing message.
	InboundEditOperationDeleted InboundEditOperation = "deleted"
)

// InboundEditOperationValues returns the closed wire values in stable order.
func InboundEditOperationValues() []string {
	return []string{string(InboundEditOperationUpdated), string(InboundEditOperationDeleted)}
}

// Normalize returns the canonical inbound edit operation.
func (o InboundEditOperation) Normalize() InboundEditOperation {
	return InboundEditOperation(strings.ToLower(strings.TrimSpace(string(o))))
}

// Validate reports whether the edit operation belongs to the supported set.
func (o InboundEditOperation) Validate() error {
	switch o.Normalize() {
	case InboundEditOperationUpdated, InboundEditOperationDeleted:
		return nil
	case "":
		return errors.New("bridges: inbound edit operation is required")
	default:
		return fmt.Errorf("bridges: unsupported inbound edit operation %q", strings.TrimSpace(string(o)))
	}
}

// InboundEdit captures one typed update or deletion of an existing platform message.
type InboundEdit struct {
	MessageID         string               `json:"message_id"`
	NewText           string               `json:"new_text"`
	OriginalTimestamp time.Time            `json:"original_timestamp"`
	Operation         InboundEditOperation `json:"operation"`
}

// Validate reports whether the edit carries an unambiguous operation payload.
func (e InboundEdit) Validate() error {
	normalized := e.normalize()
	if err := requireField(normalized.MessageID, "inbound edit message id"); err != nil {
		return err
	}
	if normalized.OriginalTimestamp.IsZero() {
		return errors.New("bridges: inbound edit original timestamp is required")
	}
	if err := normalized.Operation.Validate(); err != nil {
		return err
	}
	switch normalized.Operation {
	case InboundEditOperationUpdated:
		return requireField(normalized.NewText, "inbound edit new text")
	case InboundEditOperationDeleted:
		if normalized.NewText != "" {
			return errors.New("bridges: inbound deleted message cannot include new text")
		}
		return nil
	default:
		return errors.New("bridges: inbound edit operation is required")
	}
}

func (e InboundEdit) normalize() InboundEdit {
	normalized := e
	normalized.MessageID = strings.TrimSpace(normalized.MessageID)
	normalized.NewText = strings.TrimSpace(normalized.NewText)
	normalized.Operation = normalized.Operation.Normalize()
	if !normalized.OriginalTimestamp.IsZero() {
		normalized.OriginalTimestamp = normalized.OriginalTimestamp.UTC()
	}
	return normalized
}
