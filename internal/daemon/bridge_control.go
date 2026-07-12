package daemon

import (
	"context"
	"errors"
	"fmt"
	"strings"

	bridgepkg "github.com/compozy/agh/internal/bridges"
	extensionpkg "github.com/compozy/agh/internal/extension"
	"github.com/compozy/agh/internal/subprocess"
)

var (
	_ extensionpkg.BridgeControlRuntimeResolver = (*bridgeRuntime)(nil)
	_ bridgepkg.BridgeControlTransport          = (*bridgeRuntime)(nil)
)

// ResolveBridgeControlRuntime returns one fresh disabled-or-enabled instance snapshot without mutating lifecycle state.
func (r *bridgeRuntime) ResolveBridgeControlRuntime(
	ctx context.Context,
	extensionName string,
	bridgeInstanceID string,
	method bridgepkg.ControlMethod,
) (*subprocess.InitializeBridgeRuntime, error) {
	if r == nil {
		return nil, errors.New("daemon: bridge runtime is required")
	}
	if ctx == nil {
		return nil, errors.New("daemon: bridge control runtime context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := method.Validate(); err != nil {
		return nil, err
	}
	name := strings.TrimSpace(extensionName)
	instanceID := strings.TrimSpace(bridgeInstanceID)
	if name == "" {
		return nil, errors.New("daemon: bridge control extension name is required")
	}
	if instanceID == "" {
		return nil, errors.New("daemon: bridge control instance id is required")
	}

	ctx, unlock, err := r.lockBridgeControlLifecycleContext(ctx, name, instanceID)
	if err != nil {
		return nil, err
	}
	defer unlock()

	instance, err := r.GetInstance(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("daemon: resolve bridge control instance %q: %w", instanceID, err)
	}
	if strings.TrimSpace(instance.ExtensionName) != name {
		return nil, fmt.Errorf(
			"daemon: bridge instance %q belongs to extension %q, not %q",
			instanceID,
			instance.ExtensionName,
			name,
		)
	}
	boundSecrets, err := r.resolveBoundSecrets(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("daemon: resolve bridge control secrets for %q: %w", instanceID, err)
	}
	runtime := subprocess.InitializeBridgeRuntime{
		RuntimeVersion: subprocess.InitializeBridgeRuntimeVersion2,
		Purpose:        subprocess.BridgeRuntimePurposeControl,
		Provider:       name,
		Platform:       strings.TrimSpace(instance.Platform),
		AllowedMethods: []string{string(method)},
		ManagedInstances: []subprocess.InitializeBridgeManagedInstance{{
			Instance:     *instance,
			BoundSecrets: boundSecrets,
		}},
	}
	if err := runtime.Validate(); err != nil {
		return nil, fmt.Errorf("daemon: build bridge control runtime for %q: %w", instanceID, err)
	}
	return subprocess.CloneInitializeBridgeRuntime(&runtime), nil
}

// CheckBridge holds the instance lifecycle lock until the transient provider process is reaped.
func (r *bridgeRuntime) CheckBridge(
	ctx context.Context,
	extensionName string,
	req bridgepkg.BridgeCheckRequest,
) (bridgepkg.BridgeCheckResponse, error) {
	if err := req.Validate(); err != nil {
		return bridgepkg.BridgeCheckResponse{}, err
	}
	var response bridgepkg.BridgeCheckResponse
	err := r.withBridgeControlTransport(
		ctx,
		extensionName,
		req.BridgeInstanceID,
		func(callCtx context.Context, transport bridgepkg.BridgeControlTransport, name string) error {
			var callErr error
			response, callErr = transport.CheckBridge(callCtx, name, req)
			if callErr != nil {
				return callErr
			}
			return response.Validate()
		},
	)
	if err != nil {
		return bridgepkg.BridgeCheckResponse{}, err
	}
	return response, nil
}

// RegisterBridgeWebhook holds the instance lifecycle lock through provider registration and process reap.
func (r *bridgeRuntime) RegisterBridgeWebhook(
	ctx context.Context,
	extensionName string,
	req bridgepkg.BridgeWebhookRegistrationRequest,
) (bridgepkg.BridgeWebhookRegistrationResponse, error) {
	if err := req.Validate(); err != nil {
		return bridgepkg.BridgeWebhookRegistrationResponse{}, err
	}
	var response bridgepkg.BridgeWebhookRegistrationResponse
	err := r.withBridgeControlTransport(
		ctx,
		extensionName,
		req.BridgeInstanceID,
		func(callCtx context.Context, transport bridgepkg.BridgeControlTransport, name string) error {
			var callErr error
			response, callErr = transport.RegisterBridgeWebhook(callCtx, name, req)
			if callErr != nil {
				return callErr
			}
			return response.Validate()
		},
	)
	if err != nil {
		return bridgepkg.BridgeWebhookRegistrationResponse{}, err
	}
	return response, nil
}

type bridgeControlTransportCall func(context.Context, bridgepkg.BridgeControlTransport, string) error

func (r *bridgeRuntime) withBridgeControlTransport(
	ctx context.Context,
	extensionName string,
	bridgeInstanceID string,
	call bridgeControlTransportCall,
) error {
	if r == nil {
		return errors.New("daemon: bridge runtime is required")
	}
	if ctx == nil {
		return errors.New("daemon: bridge control context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	name := strings.TrimSpace(extensionName)
	instanceID := strings.TrimSpace(bridgeInstanceID)
	if name == "" {
		return errors.New("daemon: bridge control extension name is required")
	}

	ctx, unlock, err := r.lockBridgeControlLifecycleContext(ctx, name, instanceID)
	if err != nil {
		return err
	}
	defer unlock()
	instance, err := r.GetInstance(ctx, instanceID)
	if err != nil {
		return fmt.Errorf("daemon: load bridge control instance %q: %w", instanceID, err)
	}
	if strings.TrimSpace(instance.ExtensionName) != name {
		return fmt.Errorf(
			"daemon: bridge instance %q belongs to extension %q, not %q",
			instanceID,
			instance.ExtensionName,
			name,
		)
	}
	transport, ok := r.extensionRuntime().(bridgepkg.BridgeControlTransport)
	if !ok || transport == nil {
		return bridgepkg.ErrBridgeControlTransportUnavailable
	}
	return call(ctx, transport, name)
}
