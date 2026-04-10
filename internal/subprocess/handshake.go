package subprocess

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
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
	Provides        []string `json:"provides"`
	GrantedActions  []string `json:"granted_actions"`
	GrantedSecurity []string `json:"granted_security"`
}

// InitializeMethods lists callable method families for the session.
type InitializeMethods struct {
	DaemonRequests    []string `json:"daemon_requests"`
	ExtensionServices []string `json:"extension_services"`
}

// InitializeRuntime carries runtime intervals and deadlines negotiated during initialize.
type InitializeRuntime struct {
	HealthCheckIntervalMS int64 `json:"health_check_interval_ms"`
	HealthCheckTimeoutMS  int64 `json:"health_check_timeout_ms"`
	ShutdownTimeoutMS     int64 `json:"shutdown_timeout_ms"`
	DefaultHookTimeoutMS  int64 `json:"default_hook_timeout_ms"`
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
	Provides []string `json:"provides"`
	Actions  []string `json:"actions"`
	Security []string `json:"security"`
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

	return nil
}

func validateSubset(label string, accepted []string, granted []string) error {
	for _, value := range accepted {
		if !slices.Contains(granted, value) {
			return fmt.Errorf("subprocess: %s %q is not granted", label, value)
		}
	}
	return nil
}
