// Package extension enforces capability grants for extension security checks.
package extension

import (
	"fmt"
	"slices"
	"strings"
	"sync"
)

const (
	// CapabilityDeniedCode is the protocol-equivalent code for denied extension
	// capabilities and Host API actions.
	CapabilityDeniedCode = -32001
)

var (
	hostAPIMethodSecurityCapability = map[string]string{
		"automation/jobs":                 "automation.read",
		"automation/jobs/get":             "automation.read",
		"automation/jobs/create":          "automation.write",
		"automation/jobs/update":          "automation.write",
		"automation/jobs/delete":          "automation.write",
		"automation/jobs/trigger":         "automation.write",
		"automation/jobs/runs":            "automation.read",
		"automation/triggers":             "automation.read",
		"automation/triggers/get":         "automation.read",
		"automation/triggers/create":      "automation.write",
		"automation/triggers/update":      "automation.write",
		"automation/triggers/delete":      "automation.write",
		"automation/triggers/runs":        "automation.read",
		"automation/triggers/fire":        "automation.write",
		"automation/runs":                 "automation.read",
		"channels/instances/get":          "channel.read",
		"channels/instances/report_state": "channel.write",
		"channels/messages/ingest":        "channel.write",
		"memory/forget":                   "memory.write",
		"memory/recall":                   "memory.read",
		"memory/store":                    "memory.write",
		"observe/events":                  "observe.read",
		"observe/health":                  "observe.read",
		"sessions/create":                 "session.write",
		"sessions/events":                 "session.read",
		"sessions/list":                   "session.read",
		"sessions/prompt":                 "session.write",
		"sessions/status":                 "session.read",
		"sessions/stop":                   "session.write",
		"skills/list":                     "skills.read",
	}

	marketplaceSecurityCeiling = []string{
		"memory.read",
		"observe.read",
		"session.read",
		"skills.read",
		"tool.read",
	}
)

// ExtensionSource identifies where an extension was installed from.
type ExtensionSource int

const (
	// SourceBundled identifies built-in extensions shipped with the daemon.
	SourceBundled ExtensionSource = iota
	// SourceUser identifies user-installed extensions trusted by the operator.
	SourceUser
	// SourceWorkspace identifies workspace-scoped extensions trusted by the project.
	SourceWorkspace
	// SourceMarketplace identifies marketplace-installed extensions subject to
	// restricted default grants until an explicit allowlist exists.
	SourceMarketplace
)

// String returns the persisted text form for one extension source tier.
func (s ExtensionSource) String() string {
	switch s {
	case SourceBundled:
		return "bundled"
	case SourceUser:
		return "user"
	case SourceWorkspace:
		return "workspace"
	case SourceMarketplace:
		return "marketplace"
	default:
		return ""
	}
}

// CapabilityDeniedData is the structured data for capability-denied failures.
type CapabilityDeniedData struct {
	Method   string   `json:"method"`
	Required []string `json:"required"`
	Granted  []string `json:"granted"`
}

// ErrCapabilityDenied reports that an extension attempted a method or
// capability outside its effective grants.
type ErrCapabilityDenied struct {
	Data CapabilityDeniedData
}

// Error returns the protocol-aligned capability denied message.
func (e *ErrCapabilityDenied) Error() string {
	if e == nil {
		return ""
	}
	if strings.TrimSpace(e.Data.Method) == "" {
		return "capability_denied"
	}
	if len(e.Data.Required) == 0 {
		return fmt.Sprintf("capability_denied: %s", e.Data.Method)
	}
	return fmt.Sprintf(
		"capability_denied: %s requires %v (granted %v)",
		e.Data.Method,
		e.Data.Required,
		e.Data.Granted,
	)
}

// Code returns the protocol-equivalent error code for capability denials.
func (e *ErrCapabilityDenied) Code() int {
	return CapabilityDeniedCode
}

// CapabilityChecker tracks effective grants per extension and evaluates
// capability checks for hook dispatch and Host API calls.
type CapabilityChecker struct {
	mu     sync.RWMutex
	grants map[string]capabilityGrant
}

type capabilityGrant struct {
	source   ExtensionSource
	actions  []string
	security []string
}

// Register records one extension's effective grants by applying the source-tier
// ceiling before intersecting it with the manifest requests.
func (c *CapabilityChecker) Register(extName string, source ExtensionSource, manifest *Manifest) {
	if c == nil {
		return
	}

	name := strings.TrimSpace(extName)
	if name == "" {
		return
	}

	var requestedActions []string
	var requestedSecurity []string
	if manifest != nil {
		requestedActions = normalizeUniqueStrings(manifest.Actions.Requires)
		requestedSecurity = normalizeUniqueStrings(manifest.Security.Capabilities)
	}

	grant := capabilityGrant{
		source:   source,
		actions:  effectiveActionGrants(source, requestedActions),
		security: effectiveSecurityGrants(source, requestedSecurity),
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.grants == nil {
		c.grants = make(map[string]capabilityGrant)
	}
	c.grants[name] = grant
}

// Unregister removes any effective grants tracked for one extension.
func (c *CapabilityChecker) Unregister(extName string) {
	if c == nil {
		return
	}

	name := strings.TrimSpace(extName)
	if name == "" {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.grants == nil {
		return
	}
	delete(c.grants, name)
}

// Check reports whether extName has the requested security capability.
func (c *CapabilityChecker) Check(extName, capability string) error {
	required := strings.TrimSpace(capability)
	if required == "" {
		return fmt.Errorf("extension: capability is required")
	}

	grant := c.lookup(extName)
	if capabilityGranted(grant.security, required) {
		return nil
	}
	return newCapabilityDeniedError(required, []string{required}, grant.security)
}

// CheckHostAPI reports whether extName may call the Host API method under both
// the granted_actions and granted_security gates.
func (c *CapabilityChecker) CheckHostAPI(extName, method string) error {
	method = strings.TrimSpace(method)
	if method == "" {
		return fmt.Errorf("extension: host api method is required")
	}

	requiredSecurity, ok := hostAPIMethodSecurityCapability[method]
	if !ok {
		return fmt.Errorf("extension: unknown host api method %q", method)
	}

	grant := c.lookup(extName)
	if !slices.Contains(grant.actions, method) {
		return newCapabilityDeniedError(method, []string{method}, grant.actions)
	}
	if !capabilityGranted(grant.security, requiredSecurity) {
		return newCapabilityDeniedError(method, []string{requiredSecurity}, grant.security)
	}
	return nil
}

func (c *CapabilityChecker) lookup(extName string) capabilityGrant {
	if c == nil {
		return capabilityGrant{}
	}

	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.grants == nil {
		return capabilityGrant{}
	}
	return c.grants[strings.TrimSpace(extName)]
}

func newCapabilityDeniedError(method string, required []string, granted []string) error {
	return &ErrCapabilityDenied{
		Data: CapabilityDeniedData{
			Method:   strings.TrimSpace(method),
			Required: normalizeUniqueStrings(required),
			Granted:  normalizeUniqueStrings(granted),
		},
	}
}

func effectiveActionGrants(source ExtensionSource, requested []string) []string {
	requested = normalizeUniqueStrings(requested)
	policy := sourcePolicy(source)
	if policy.allowAllActions {
		return requested
	}
	if len(requested) == 0 || len(policy.allowedActions) == 0 {
		return nil
	}

	var granted []string
	for _, method := range requested {
		if slices.Contains(policy.allowedActions, method) {
			granted = append(granted, method)
		}
	}
	return normalizeUniqueStrings(granted)
}

func effectiveSecurityGrants(source ExtensionSource, requested []string) []string {
	requested = normalizeUniqueStrings(requested)
	policy := sourcePolicy(source)
	if policy.allowAllSecurity {
		return requested
	}
	if len(requested) == 0 || len(policy.allowedSecurity) == 0 {
		return nil
	}

	var granted []string
	for _, request := range requested {
		if ceilingAllowsRequestedGrant(policy.allowedSecurity, request) {
			granted = append(granted, request)
			continue
		}

		for _, allowed := range policy.allowedSecurity {
			if capabilityGrantSuperset(request, allowed) {
				granted = append(granted, allowed)
			}
		}
	}
	return normalizeUniqueStrings(granted)
}

func capabilityGranted(grants []string, capability string) bool {
	required := strings.TrimSpace(capability)
	if required == "" {
		return false
	}
	for _, grant := range grants {
		if capabilityGrantSuperset(grant, required) {
			return true
		}
	}
	return false
}

func ceilingAllowsRequestedGrant(ceiling []string, requested string) bool {
	for _, allowed := range ceiling {
		if capabilityGrantSuperset(allowed, requested) {
			return true
		}
	}
	return false
}

func capabilityGrantSuperset(grant string, requested string) bool {
	grant = strings.TrimSpace(grant)
	requested = strings.TrimSpace(requested)
	switch {
	case grant == "", requested == "":
		return false
	case grant == "*":
		return true
	case requested == "*":
		return grant == "*"
	}

	grantParts := strings.Split(grant, ".")
	requestedParts := strings.Split(requested, ".")
	if len(grantParts) != len(requestedParts) {
		return false
	}

	for idx, part := range grantParts {
		if part == "*" {
			continue
		}
		if requestedParts[idx] == "*" {
			return false
		}
		if part != requestedParts[idx] {
			return false
		}
	}
	return true
}

func normalizeUniqueStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(values))
	dst := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		dst = append(dst, trimmed)
	}
	if len(dst) == 0 {
		return nil
	}
	slices.Sort(dst)
	return dst
}

type sourceTierPolicy struct {
	allowAllActions  bool
	allowAllSecurity bool
	allowedActions   []string
	allowedSecurity  []string
}

func sourcePolicy(source ExtensionSource) sourceTierPolicy {
	switch source {
	case SourceBundled, SourceUser, SourceWorkspace:
		return sourceTierPolicy{
			allowAllActions:  true,
			allowAllSecurity: true,
		}
	case SourceMarketplace:
		return sourceTierPolicy{
			allowedActions:  marketplaceActionCeiling(),
			allowedSecurity: slices.Clone(marketplaceSecurityCeiling),
		}
	default:
		return sourceTierPolicy{}
	}
}

func marketplaceActionCeiling() []string {
	actions := make([]string, 0, len(hostAPIMethodSecurityCapability))
	for method, capability := range hostAPIMethodSecurityCapability {
		if capabilityGranted(marketplaceSecurityCeiling, capability) {
			actions = append(actions, method)
		}
	}
	slices.Sort(actions)
	return actions
}
