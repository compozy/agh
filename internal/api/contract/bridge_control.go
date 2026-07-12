package contract

import (
	"strings"

	bridgepkg "github.com/compozy/agh/internal/bridges"
)

// BridgeVerifyResponse returns the live, provider-owned checks for one bridge.
type BridgeVerifyResponse struct {
	BridgeInstanceID string                        `json:"bridge_instance_id"`
	Checks           []bridgepkg.BridgeCheckRecord `json:"checks"`
}

// BridgeWebhookRegistrationResponse reports the provider-owned registration outcome.
type BridgeWebhookRegistrationResponse struct {
	BridgeInstanceID string                      `json:"bridge_instance_id"`
	Status           bridgepkg.BridgeCheckStatus `json:"status"`
	Remediation      string                      `json:"remediation"`
}

// BridgeSendTestRequest requests one real provider delivery. Unlike
// BridgeTestDeliveryRequest, this request reaches the installed adapter.
type BridgeSendTestRequest struct {
	Message string                    `json:"message"`
	Target  BridgeDeliveryTargetInput `json:"target"`
}

// ToResolveDeliveryTargetRequest validates the body/path ownership boundary
// and normalizes the explicit target overrides.
func (r BridgeSendTestRequest) ToResolveDeliveryTargetRequest(
	bridgeInstanceID string,
) (bridgepkg.ResolveDeliveryTargetRequest, error) {
	return (BridgeTestDeliveryRequest{Target: r.Target}).ToResolveDeliveryTargetRequest(bridgeInstanceID)
}

// NormalizedMessage returns the provider-bound text without transport padding.
func (r BridgeSendTestRequest) NormalizedMessage() string {
	return strings.TrimSpace(r.Message)
}

// BridgeSendTestResponse reports the provider acknowledgment for a real send.
type BridgeSendTestResponse struct {
	Status           string                   `json:"status"`
	BridgeInstanceID string                   `json:"bridge_instance_id"`
	DeliveryID       string                   `json:"delivery_id"`
	RemoteMessageID  string                   `json:"remote_message_id,omitempty"`
	DeliveryTarget   bridgepkg.DeliveryTarget `json:"delivery_target"`
}
