package bridges

import (
	"context"
	"errors"
	"fmt"
	"strings"

	redactpkg "github.com/compozy/agh/internal/redact"
)

var (
	// ErrBridgeControlTransportUnavailable reports that no extension runtime can execute a provider control call.
	ErrBridgeControlTransportUnavailable = errors.New("bridges: control transport unavailable")
)

// ControlMethod identifies one typed provider control operation.
type ControlMethod string

const (
	// ControlMethodCheck asks the adapter to run read-only provider checks.
	ControlMethodCheck ControlMethod = "bridges/check"
	// ControlMethodWebhookRegister asks the adapter to register its configured webhook.
	ControlMethodWebhookRegister ControlMethod = "bridges/webhook/register"
)

// ControlMethodValues returns the closed control method family in stable order.
func ControlMethodValues() []string {
	return []string{string(ControlMethodCheck), string(ControlMethodWebhookRegister)}
}

// Validate enforces the closed provider control method family.
func (m ControlMethod) Validate() error {
	trimmed := strings.TrimSpace(string(m))
	if string(m) != trimmed {
		return fmt.Errorf("bridges: unsupported control method %q", string(m))
	}
	switch ControlMethod(trimmed) {
	case ControlMethodCheck, ControlMethodWebhookRegister:
		return nil
	default:
		return fmt.Errorf("bridges: unsupported control method %q", strings.TrimSpace(string(m)))
	}
}

// BridgeCheckStatus is the closed result state for one provider check.
type BridgeCheckStatus string

const (
	BridgeCheckStatusPass    BridgeCheckStatus = "pass"
	BridgeCheckStatusWarn    BridgeCheckStatus = "warn"
	BridgeCheckStatusFail    BridgeCheckStatus = "fail"
	BridgeCheckStatusSkipped BridgeCheckStatus = "skipped"
)

// BridgeCheckStatusValues returns the closed check status family in stable order.
func BridgeCheckStatusValues() []string {
	return []string{
		string(BridgeCheckStatusPass),
		string(BridgeCheckStatusWarn),
		string(BridgeCheckStatusFail),
		string(BridgeCheckStatusSkipped),
	}
}

// Validate enforces the closed public check status family.
func (s BridgeCheckStatus) Validate() error {
	trimmed := strings.TrimSpace(string(s))
	if string(s) != trimmed {
		return fmt.Errorf("bridges: unsupported bridge check status %q", string(s))
	}
	switch BridgeCheckStatus(trimmed) {
	case BridgeCheckStatusPass, BridgeCheckStatusWarn, BridgeCheckStatusFail, BridgeCheckStatusSkipped:
		return nil
	default:
		return fmt.Errorf("bridges: unsupported bridge check status %q", strings.TrimSpace(string(s)))
	}
}

// BridgeCheckRecord is one stable, redacted provider check result.
// Provider response bodies and raw errors are deliberately absent from this contract.
type BridgeCheckRecord struct {
	Check       string            `json:"check"`
	Status      BridgeCheckStatus `json:"status"`
	Remediation string            `json:"remediation"`
}

// Validate enforces a named, closed, actionable provider check result.
func (r BridgeCheckRecord) Validate() error {
	if strings.TrimSpace(r.Check) == "" {
		return errors.New("bridges: bridge check name is required")
	}
	if redactpkg.String(r.Check) != r.Check {
		return errors.New("bridges: bridge check name contains sensitive data")
	}
	status := BridgeCheckStatus(strings.TrimSpace(string(r.Status)))
	if err := status.Validate(); err != nil {
		return err
	}
	if status != BridgeCheckStatusPass && strings.TrimSpace(r.Remediation) == "" {
		return errors.New("bridges: bridge check remediation is required unless status is pass")
	}
	if redactpkg.String(r.Remediation) != r.Remediation {
		return errors.New("bridges: bridge check remediation contains sensitive data")
	}
	return nil
}

// BridgeCheckRequest selects the daemon-owned bridge instance to inspect.
type BridgeCheckRequest struct {
	BridgeInstanceID string `json:"bridge_instance_id"`
}

// Validate requires an explicit bridge instance.
func (r BridgeCheckRequest) Validate() error {
	return requireField(r.BridgeInstanceID, "bridge check instance id")
}

// BridgeCheckResponse returns every explicit provider check record.
type BridgeCheckResponse struct {
	Checks []BridgeCheckRecord `json:"checks"`
}

// Validate rejects empty or malformed provider responses.
func (r BridgeCheckResponse) Validate() error {
	if len(r.Checks) == 0 {
		return errors.New("bridges: bridge check response checks are required")
	}
	seen := make(map[string]struct{}, len(r.Checks))
	for index, check := range r.Checks {
		if err := check.Validate(); err != nil {
			return fmt.Errorf("bridges: bridge check response check %d: %w", index, err)
		}
		name := strings.TrimSpace(check.Check)
		if _, ok := seen[name]; ok {
			return fmt.Errorf("bridges: bridge check response check %q is duplicated", name)
		}
		seen[name] = struct{}{}
	}
	return nil
}

// BridgeWebhookRegistrationRequest selects the instance whose configured webhook is registered.
type BridgeWebhookRegistrationRequest struct {
	BridgeInstanceID string `json:"bridge_instance_id"`
}

// Validate requires an explicit bridge instance.
func (r BridgeWebhookRegistrationRequest) Validate() error {
	return requireField(r.BridgeInstanceID, "bridge webhook registration instance id")
}

// BridgeWebhookRegistrationResponse is the stable result of provider webhook registration.
type BridgeWebhookRegistrationResponse struct {
	Status      BridgeCheckStatus `json:"status"`
	Remediation string            `json:"remediation"`
}

// Validate enforces the same closed, actionable status contract as provider checks.
func (r BridgeWebhookRegistrationResponse) Validate() error {
	return (BridgeCheckRecord{
		Check:       "webhook.registration",
		Status:      r.Status,
		Remediation: r.Remediation,
	}).Validate()
}

// BridgeControlTransport executes provider-owned control calls through isolated extension runtimes.
type BridgeControlTransport interface {
	CheckBridge(
		ctx context.Context,
		extensionName string,
		req BridgeCheckRequest,
	) (BridgeCheckResponse, error)
	RegisterBridgeWebhook(
		ctx context.Context,
		extensionName string,
		req BridgeWebhookRegistrationRequest,
	) (BridgeWebhookRegistrationResponse, error)
}

// PassCheck builds a successful public check without diagnostic prose.
func PassCheck(check string) BridgeCheckRecord {
	return BridgeCheckRecord{
		Check:       strings.TrimSpace(check),
		Status:      BridgeCheckStatusPass,
		Remediation: "",
	}
}

// SkippedCheck builds an explicit skipped check with the operator's next action.
func SkippedCheck(check string, remediation string) BridgeCheckRecord {
	return BridgeCheckRecord{
		Check:       strings.TrimSpace(check),
		Status:      BridgeCheckStatusSkipped,
		Remediation: strings.TrimSpace(remediation),
	}
}

// FailedSecretCheck reports a missing binding without accepting or exposing a raw provider error.
func FailedSecretCheck(check string, binding string) BridgeCheckRecord {
	return BridgeCheckRecord{
		Check:  strings.TrimSpace(check),
		Status: BridgeCheckStatusFail,
		Remediation: fmt.Sprintf(
			"Bind the required %q bridge secret and retry.",
			strings.TrimSpace(binding),
		),
	}
}
