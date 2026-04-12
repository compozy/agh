package subprocess

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/pedronauck/agh/internal/channels"
	extensionprotocol "github.com/pedronauck/agh/internal/extension/protocol"
)

// InitializeRequest is the AGH -> extension session contract request.
type InitializeRequest struct {
	ProtocolVersion          string                 `json:"protocol_version"`
	SupportedProtocolVersion []string               `json:"supported_protocol_versions"`
	AGHVersion               string                 `json:"agh_version"`
	Extension                InitializeExtension    `json:"extension"`
	Capabilities             InitializeCapabilities `json:"capabilities"`
	Methods                  InitializeMethods      `json:"methods"`
	Runtime                  InitializeRuntime      `json:"runtime"`
}

// InitializeExtension identifies the launched extension.
type InitializeExtension struct {
	Name       string `json:"name"`
	Version    string `json:"version"`
	SourceTier string `json:"source_tier"`
}

// InitializeCapabilities carries runtime-granted capabilities.
type InitializeCapabilities struct {
	Provides        []string                          `json:"provides"`
	GrantedActions  []extensionprotocol.HostAPIMethod `json:"granted_actions"`
	GrantedSecurity []string                          `json:"granted_security"`
}

// InitializeMethods lists callable method families for the session.
type InitializeMethods struct {
	DaemonRequests    []string `json:"daemon_requests"`
	ExtensionServices []string `json:"extension_services"`
}

// InitializeRuntime carries runtime intervals and deadlines negotiated during initialize.
type InitializeRuntime struct {
	HealthCheckIntervalMS int64                     `json:"health_check_interval_ms"`
	HealthCheckTimeoutMS  int64                     `json:"health_check_timeout_ms"`
	ShutdownTimeoutMS     int64                     `json:"shutdown_timeout_ms"`
	DefaultHookTimeoutMS  int64                     `json:"default_hook_timeout_ms"`
	Channel               *InitializeChannelRuntime `json:"channel,omitempty"`
}

// InitializeChannelRuntime carries the instance-scoped channel launch material
// granted to one channel-capable extension session.
type InitializeChannelRuntime struct {
	Instance     channels.ChannelInstance       `json:"instance"`
	BoundSecrets []InitializeChannelBoundSecret `json:"bound_secrets,omitempty"`
}

// InitializeChannelBoundSecret is one launch-time channel secret resolved by AGH.
type InitializeChannelBoundSecret struct {
	BindingName string `json:"binding_name"`
	Kind        string `json:"kind"`
	Value       string `json:"value"`
}

// InitializeResponse is the extension -> AGH initialize acknowledgment.
type InitializeResponse struct {
	ProtocolVersion      string                  `json:"protocol_version"`
	ExtensionInfo        InitializeExtensionInfo `json:"extension_info"`
	AcceptedCapabilities AcceptedCapabilities    `json:"accepted_capabilities"`
	ImplementedMethods   []string                `json:"implemented_methods"`
	SupportedHookEvents  []string                `json:"supported_hook_events"`
	Supports             InitializeSupports      `json:"supports"`
}

// InitializeExtensionInfo identifies the running extension implementation.
type InitializeExtensionInfo struct {
	Name       string `json:"name"`
	Version    string `json:"version"`
	SDKName    string `json:"sdk_name,omitempty"`
	SDKVersion string `json:"sdk_version,omitempty"`
}

// AcceptedCapabilities is the subset the extension accepted for this session.
type AcceptedCapabilities struct {
	Provides []string                          `json:"provides"`
	Actions  []extensionprotocol.HostAPIMethod `json:"actions"`
	Security []string                          `json:"security"`
}

// InitializeSupports advertises optional protocol features.
type InitializeSupports struct {
	HealthCheck  bool `json:"health_check"`
	ProvideTools bool `json:"provide_tools"`
}

// ShutdownRequest is the cooperative drain request sent before signal escalation.
type ShutdownRequest struct {
	Reason     string `json:"reason"`
	DeadlineMS int64  `json:"deadline_ms"`
}

// ShutdownResponse acknowledges a cooperative drain request.
type ShutdownResponse struct {
	Acknowledged bool `json:"acknowledged"`
}

// Initialize performs the session initialize handshake and transitions the process to ready on success.
func (p *Process) Initialize(ctx context.Context, request InitializeRequest) (InitializeResponse, error) {
	if p == nil {
		return InitializeResponse{}, errors.New("subprocess: process is required")
	}
	if p.currentState() != processStateStarting {
		return InitializeResponse{}, errors.New("subprocess: initialize may only be called once")
	}

	if err := request.Validate(); err != nil {
		return InitializeResponse{}, err
	}

	var response InitializeResponse
	if err := p.Call(ctx, initializeMethod, request, &response); err != nil {
		return InitializeResponse{}, err
	}
	if err := validateInitializeResponse(request, response); err != nil {
		return InitializeResponse{}, err
	}

	p.markReady()
	p.maybeStartHealthMonitor(request.Runtime, response.Supports)

	return response, nil
}

// Validate checks that the initialize request carries the mandatory contract fields.
func (r InitializeRequest) Validate() error {
	if strings.TrimSpace(r.ProtocolVersion) == "" {
		return errors.New("subprocess: initialize protocol_version is required")
	}
	if len(r.SupportedProtocolVersion) == 0 {
		return errors.New("subprocess: initialize supported_protocol_versions is required")
	}
	if strings.TrimSpace(r.Extension.Name) == "" {
		return errors.New("subprocess: initialize extension.name is required")
	}
	if strings.TrimSpace(r.Extension.Version) == "" {
		return errors.New("subprocess: initialize extension.version is required")
	}
	if r.Runtime.HealthCheckIntervalMS <= 0 {
		return errors.New("subprocess: initialize health_check_interval_ms must be > 0")
	}
	if r.Runtime.HealthCheckTimeoutMS <= 0 {
		return errors.New("subprocess: initialize health_check_timeout_ms must be > 0")
	}
	if r.Runtime.ShutdownTimeoutMS <= 0 {
		return errors.New("subprocess: initialize shutdown_timeout_ms must be > 0")
	}
	if r.Runtime.DefaultHookTimeoutMS <= 0 {
		return errors.New("subprocess: initialize default_hook_timeout_ms must be > 0")
	}
	if r.Runtime.Channel != nil {
		if err := r.Runtime.Channel.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func validateInitializeResponse(request InitializeRequest, response InitializeResponse) error {
	if !slices.Contains(request.SupportedProtocolVersion, response.ProtocolVersion) {
		return fmt.Errorf("subprocess: initialize selected unsupported protocol version %q", response.ProtocolVersion)
	}
	if err := validateSubset("accepted actions", response.AcceptedCapabilities.Actions, request.Capabilities.GrantedActions); err != nil {
		return err
	}
	if err := validateSubset("accepted security", response.AcceptedCapabilities.Security, request.Capabilities.GrantedSecurity); err != nil {
		return err
	}
	if err := validateSubset("accepted provides", response.AcceptedCapabilities.Provides, request.Capabilities.Provides); err != nil {
		return err
	}

	if !slices.Contains(response.ImplementedMethods, shutdownMethod) {
		return errors.New("subprocess: initialize response missing required shutdown method")
	}
	if !response.Supports.HealthCheck || !slices.Contains(response.ImplementedMethods, "health_check") {
		return errors.New("subprocess: initialize response missing required health_check support")
	}
	for _, method := range extensionprotocol.CapabilityServiceMethods(response.AcceptedCapabilities.Provides) {
		if !slices.Contains(response.ImplementedMethods, method) {
			return fmt.Errorf("subprocess: initialize response missing required capability service method %q", method)
		}
	}

	return nil
}

// Validate checks that the granted channel launch payload is internally consistent.
func (r InitializeChannelRuntime) Validate() error {
	instance := r.Instance
	if err := instance.Validate(); err != nil {
		return fmt.Errorf("subprocess: initialize channel instance: %w", err)
	}

	seen := make(map[string]struct{}, len(r.BoundSecrets))
	for _, secret := range r.BoundSecrets {
		normalized := secret.normalize()
		if err := normalized.Validate(); err != nil {
			return fmt.Errorf("subprocess: initialize channel bound secret: %w", err)
		}
		if _, ok := seen[normalized.BindingName]; ok {
			return fmt.Errorf("subprocess: initialize channel bound secret %q is duplicated", normalized.BindingName)
		}
		seen[normalized.BindingName] = struct{}{}
	}

	return nil
}

// Validate checks that the bound secret payload is complete.
func (s InitializeChannelBoundSecret) Validate() error {
	normalized := s.normalize()
	if strings.TrimSpace(normalized.BindingName) == "" {
		return errors.New("subprocess: initialize channel bound secret binding_name is required")
	}
	if strings.TrimSpace(normalized.Kind) == "" {
		return errors.New("subprocess: initialize channel bound secret kind is required")
	}
	if strings.TrimSpace(normalized.Value) == "" {
		return errors.New("subprocess: initialize channel bound secret value is required")
	}
	return nil
}

func (r InitializeChannelRuntime) normalize() InitializeChannelRuntime {
	normalized := r
	if len(normalized.BoundSecrets) == 0 {
		normalized.BoundSecrets = nil
		return normalized
	}

	boundSecrets := make([]InitializeChannelBoundSecret, 0, len(normalized.BoundSecrets))
	for _, secret := range normalized.BoundSecrets {
		boundSecrets = append(boundSecrets, secret.normalize())
	}
	slices.SortFunc(boundSecrets, func(left InitializeChannelBoundSecret, right InitializeChannelBoundSecret) int {
		return strings.Compare(left.BindingName, right.BindingName)
	})
	normalized.BoundSecrets = boundSecrets
	return normalized
}

func (s InitializeChannelBoundSecret) normalize() InitializeChannelBoundSecret {
	normalized := s
	normalized.BindingName = strings.TrimSpace(normalized.BindingName)
	normalized.Kind = strings.TrimSpace(normalized.Kind)
	normalized.Value = strings.TrimSpace(normalized.Value)
	return normalized
}

// CloneInitializeChannelRuntime returns a deep copy safe to retain in manager state.
func CloneInitializeChannelRuntime(src *InitializeChannelRuntime) *InitializeChannelRuntime {
	if src == nil {
		return nil
	}

	cloned := src.normalize()
	cloned.Instance = cloneChannelInstance(cloned.Instance)
	cloned.BoundSecrets = append([]InitializeChannelBoundSecret(nil), cloned.BoundSecrets...)
	return &cloned
}

func cloneChannelInstance(instance channels.ChannelInstance) channels.ChannelInstance {
	cloned := instance
	if len(cloned.DeliveryDefaults) > 0 {
		cloned.DeliveryDefaults = append(json.RawMessage(nil), cloned.DeliveryDefaults...)
	}
	return cloned
}

func validateSubset[T ~string](label string, accepted []T, granted []T) error {
	for _, value := range accepted {
		if !slices.Contains(granted, value) {
			return fmt.Errorf("subprocess: %s %q is not granted", label, value)
		}
	}
	return nil
}
