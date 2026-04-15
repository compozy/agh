package bridgesdk

import (
	"context"
	"errors"

	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	extensioncontract "github.com/pedronauck/agh/internal/extension/contract"
)

// CallFunc issues one typed Host API request.
type CallFunc func(context.Context, string, any, any) error

// HostAPIClient is the typed provider-side client for bridge Host API calls.
type HostAPIClient struct {
	call CallFunc
}

// NewHostAPIClient constructs a Host API client over the shared runtime peer.
func NewHostAPIClient(peer *Peer) *HostAPIClient {
	if peer == nil {
		return nil
	}
	return &HostAPIClient{
		call: peer.Call,
	}
}

// NewHostAPIClientFromCall constructs a Host API client from an arbitrary call
// function, mainly for tests.
func NewHostAPIClientFromCall(call CallFunc) *HostAPIClient {
	if call == nil {
		return nil
	}
	return &HostAPIClient{call: call}
}

// Call issues one raw Host API request.
func (c *HostAPIClient) Call(ctx context.Context, method string, params any, result any) error {
	if c == nil || c.call == nil {
		return errors.New("bridgesdk: host api client is required")
	}
	return c.call(ctx, method, params, result)
}

// ListBridgeInstances returns every bridge instance currently assigned to the provider runtime.
func (c *HostAPIClient) ListBridgeInstances(ctx context.Context) ([]bridgepkg.BridgeInstance, error) {
	var result []bridgepkg.BridgeInstance
	if err := c.Call(ctx, "bridges/instances/list", struct{}{}, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetBridgeInstance returns one provider-owned bridge instance.
func (c *HostAPIClient) GetBridgeInstance(ctx context.Context, bridgeInstanceID string) (*bridgepkg.BridgeInstance, error) {
	var result bridgepkg.BridgeInstance
	if err := c.Call(ctx, "bridges/instances/get", extensioncontract.BridgeInstanceTargetParams{
		BridgeInstanceID: bridgeInstanceID,
	}, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ReportBridgeInstanceState reports one provider-observed bridge status change.
func (c *HostAPIClient) ReportBridgeInstanceState(
	ctx context.Context,
	params extensioncontract.BridgesInstancesReportStateParams,
) (*bridgepkg.BridgeInstance, error) {
	var result bridgepkg.BridgeInstance
	if err := c.Call(ctx, "bridges/instances/report_state", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// IngestBridgeMessage ingests one normalized inbound bridge event.
func (c *HostAPIClient) IngestBridgeMessage(
	ctx context.Context,
	envelope bridgepkg.InboundMessageEnvelope,
) (*extensioncontract.BridgesMessagesIngestResult, error) {
	var result extensioncontract.BridgesMessagesIngestResult
	if err := c.Call(ctx, "bridges/messages/ingest", envelope, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
