package participation

import (
	"fmt"
	"strings"
)

// OwnerKey returns the canonical budget-owner identity for one execution.
func OwnerKey(owner OwnerRef) string {
	return string(owner.Kind) + ":" + strings.TrimSpace(owner.ID)
}

// ValidateOwner verifies that an execution owner has a supported kind and stable identity.
func ValidateOwner(owner OwnerRef) error {
	owner.Kind = OwnerKind(strings.TrimSpace(string(owner.Kind)))
	owner.ID = strings.TrimSpace(owner.ID)
	switch owner.Kind {
	case OwnerKindSession, OwnerKindTaskRun, OwnerKindLoopRun, OwnerKindAutomationRun:
	default:
		return fmt.Errorf("network participation: unsupported owner kind %q", owner.Kind)
	}
	if owner.ID == "" {
		return fmt.Errorf("network participation: %s owner id is required", owner.Kind)
	}
	return nil
}

// ValidateOwnerKey verifies a canonical kind-prefixed budget-owner identity.
func ValidateOwnerKey(value string) error {
	parts := strings.SplitN(strings.TrimSpace(value), ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("network participation: owner key must use kind:id form")
	}
	owner := OwnerRef{Kind: OwnerKind(parts[0]), ID: parts[1]}
	if err := ValidateOwner(owner); err != nil {
		return err
	}
	if OwnerKey(owner) != strings.TrimSpace(value) {
		return fmt.Errorf("network participation: owner key is not canonical")
	}
	return nil
}
