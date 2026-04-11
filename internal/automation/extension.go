package automation

import (
	"errors"
	"strings"
)

// ExtensionTriggerRequest describes one extension-originated trigger fire.
type ExtensionTriggerRequest struct {
	Event       string          `json:"event"`
	Scope       AutomationScope `json:"scope"`
	WorkspaceID string          `json:"workspace_id,omitempty"`
	Payload     map[string]any  `json:"payload,omitempty"`
}

// Validate ensures the extension trigger request matches the ext.* ingress contract.
func (r ExtensionTriggerRequest) Validate(path string) error {
	if strings.TrimSpace(r.Event) == "" {
		return errors.New(nestedPath(path, "event") + " is required")
	}
	if !strings.HasPrefix(strings.TrimSpace(r.Event), "ext.") {
		return errors.New(nestedPath(path, "event") + " must start with \"ext.\"")
	}
	if err := ValidateScopeBinding(r.Scope, r.WorkspaceID, path, "workspace_id"); err != nil {
		return err
	}
	return nil
}
