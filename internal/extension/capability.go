// Package extension enforces capability grants for extension security checks.
package extensionpkg

import (
	"fmt"
	"slices"
	"strings"
	"sync"

	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/extension/surfaces"
	"github.com/pedronauck/agh/internal/resources"
)

const (
	// CapabilityDeniedCode is the protocol-equivalent code for denied extension
	// capabilities and Host API actions.
	CapabilityDeniedCode = -32001
)

var (
	hostAPIMethodSecurityCapability = map[string]string{
		"automation/jobs":                "automation.read",
		"automation/jobs/get":            "automation.read",
		"automation/jobs/create":         "automation.write",
		"automation/jobs/update":         "automation.write",
		"automation/jobs/delete":         "automation.write",
		"automation/jobs/trigger":        "automation.write",
		"automation/jobs/runs":           "automation.read",
		"automation/triggers":            "automation.read",
		"automation/triggers/get":        "automation.read",
		"automation/triggers/create":     "automation.write",
		"automation/triggers/update":     "automation.write",
		"automation/triggers/delete":     "automation.write",
		"automation/triggers/runs":       "automation.read",
		"automation/triggers/fire":       "automation.write",
		"automation/runs":                "automation.read",
		"agents/heartbeat/delete":        "heartbeat.write",
		"agents/heartbeat/get":           "heartbeat.read",
		"agents/heartbeat/history":       "heartbeat.read",
		"agents/heartbeat/put":           "heartbeat.write",
		"agents/heartbeat/rollback":      "heartbeat.write",
		"agents/heartbeat/status":        "heartbeat.read",
		"agents/heartbeat/validate":      "heartbeat.read",
		"agents/heartbeat/wake":          "heartbeat.write",
		"agents/soul/delete":             "soul.write",
		"agents/soul/get":                "soul.read",
		"agents/soul/history":            "soul.read",
		"agents/soul/put":                "soul.write",
		"agents/soul/rollback":           "soul.write",
		"agents/soul/validate":           "soul.read",
		"tasks":                          "task.read",
		"tasks/get":                      "task.read",
		"tasks/timeline":                 "task.read",
		"tasks/tree":                     "task.read",
		"tasks/dashboard":                "task.read",
		"tasks/inbox":                    "task.read",
		"tasks/create":                   "task.write",
		"tasks/update":                   "task.write",
		"tasks/cancel":                   "task.write",
		"tasks/runs":                     "task.read",
		"tasks/runs/get":                 "task.read",
		"tasks/runs/enqueue":             "task.write",
		"tasks/runs/claim":               "task.write",
		"tasks/runs/start":               "task.write",
		"tasks/runs/attach_session":      "task.write",
		"tasks/runs/complete":            "task.write",
		"tasks/runs/fail":                "task.write",
		"tasks/runs/cancel":              "task.write",
		"resources/list":                 "resource.read",
		"resources/get":                  "resource.read",
		"resources/snapshot":             "resource.write",
		"bridges/instances/list":         "bridge.read",
		"bridges/instances/get":          "bridge.read",
		"bridges/instances/report_state": "bridge.write",
		"bridges/messages/ingest":        "bridge.write",
		"memory/forget":                  "memory.write",
		"memory/recall":                  "memory.read",
		"memory/store":                   "memory.write",
		"network/status":                 "network.read",
		"network/channels":               "network.read",
		"network/peers":                  "network.read",
		"network/threads":                "network.read",
		"network/thread/get":             "network.read",
		"network/thread/messages":        "network.read",
		"network/directs":                "network.read",
		"network/direct/resolve":         "network.write",
		"network/direct/messages":        "network.read",
		"network/work/get":               "network.read",
		"network/send":                   "network.write",
		"observe/events":                 "observe.read",
		"observe/health":                 "observe.read",
		"sandbox/list":                   "",
		"sandbox/info":                   "",
		"sandbox/exec":                   "sandbox.exec",
		"sessions/create":                "session.write",
		"sessions/events":                "session.read",
		"sessions/health/get":            "session.read",
		"sessions/list":                  "session.read",
		"sessions/prompt":                "session.write",
		"sessions/soul/refresh":          "soul.write",
		"sessions/status":                "session.read",
		"sessions/status/get":            "session.read",
		"sessions/stop":                  "session.write",
		"skills/list":                    "skills.read",
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
	mu             sync.RWMutex
	grants         map[string]capabilityGrant
	resourcePolicy aghconfig.ExtensionsResourcesConfig
}

type capabilityGrant struct {
	source         ExtensionSource
	actions        []string
	security       []string
	resourceKinds  []resources.ResourceKind
	resourceScopes []resources.ResourceScopeKind
}

// EffectiveGrant is the daemon-derived grant snapshot for one extension session.
type EffectiveGrant struct {
	Actions        []string
	Security       []string
	ResourceKinds  []resources.ResourceKind
	ResourceScopes []resources.ResourceScopeKind
}

// Register records one extension's effective grants by applying the source-tier
// ceiling before intersecting it with the manifest requests.
func (c *CapabilityChecker) Register(extName string, source ExtensionSource, manifest *Manifest) {
	if _, err := c.RegisterForSession(extName, source, manifest, resources.ResourceScopeKindGlobal); err != nil {
		return
	}
}

// RegisterForSession records one extension's effective grants for the supplied session scope ceiling.
func (c *CapabilityChecker) RegisterForSession(
	extName string,
	source ExtensionSource,
	manifest *Manifest,
	sessionMaxScope resources.ResourceScopeKind,
) (EffectiveGrant, error) {
	if c == nil {
		return EffectiveGrant{}, nil
	}

	name := strings.TrimSpace(extName)
	if name == "" {
		return EffectiveGrant{}, nil
	}

	grant, err := c.resolve(source, manifest, sessionMaxScope)
	if err != nil {
		return EffectiveGrant{}, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.grants == nil {
		c.grants = make(map[string]capabilityGrant)
	}
	c.grants[name] = grant
	return grant.snapshot(), nil
}

// Resolve computes one daemon-derived grant snapshot without storing it.
func (c *CapabilityChecker) Resolve(
	source ExtensionSource,
	manifest *Manifest,
	sessionMaxScope resources.ResourceScopeKind,
) (EffectiveGrant, error) {
	if c == nil {
		return EffectiveGrant{}, nil
	}

	grant, err := c.resolve(source, manifest, sessionMaxScope)
	if err != nil {
		return EffectiveGrant{}, err
	}
	return grant.snapshot(), nil
}

// Grant returns the stored effective grant snapshot for one extension.
func (c *CapabilityChecker) Grant(extName string) EffectiveGrant {
	return c.lookup(extName).snapshot()
}

// SetResourcePolicy installs the operator-configured extension resource policy.
func (c *CapabilityChecker) SetResourcePolicy(policy aghconfig.ExtensionsResourcesConfig) {
	if c == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.resourcePolicy = cloneResourcePolicy(policy)
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
	if strings.TrimSpace(requiredSecurity) == "" {
		return nil
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

func (c *CapabilityChecker) resolve(
	source ExtensionSource,
	manifest *Manifest,
	sessionMaxScope resources.ResourceScopeKind,
) (capabilityGrant, error) {
	c.mu.RLock()
	policy := cloneResourcePolicy(c.resourcePolicy)
	c.mu.RUnlock()

	var requestedActions []string
	var requestedSecurity []string
	var requestedResources surfaces.GrantRequest
	var err error
	if manifest != nil {
		requestedActions = normalizeUniqueStrings(manifest.Actions.Requires)
		requestedSecurity = normalizeUniqueStrings(manifest.Security.Capabilities)
		requestedResources, err = surfaces.ResolveManifestRequest(
			manifest.Resources.Publish.Families,
			manifest.Resources.Publish.MaxScope,
		)
		if err != nil {
			return capabilityGrant{}, err
		}
	}

	resourceKinds, resourceScopes, err := effectiveResourceGrants(
		source,
		policy,
		requestedResources,
		sessionMaxScope,
	)
	if err != nil {
		return capabilityGrant{}, err
	}

	return capabilityGrant{
		source:         source,
		actions:        effectiveActionGrants(source, requestedActions),
		security:       effectiveSecurityGrants(source, requestedSecurity),
		resourceKinds:  resourceKinds,
		resourceScopes: resourceScopes,
	}, nil
}

func (g capabilityGrant) snapshot() EffectiveGrant {
	return EffectiveGrant{
		Actions:        slices.Clone(g.actions),
		Security:       slices.Clone(g.security),
		ResourceKinds:  slices.Clone(g.resourceKinds),
		ResourceScopes: slices.Clone(g.resourceScopes),
	}
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
	maxResourceScope resources.ResourceScopeKind
}

func sourcePolicy(source ExtensionSource) sourceTierPolicy {
	switch source {
	case SourceBundled, SourceUser, SourceWorkspace:
		return sourceTierPolicy{
			allowAllActions:  true,
			allowAllSecurity: true,
			maxResourceScope: sourceTierMaxScope(source),
		}
	case SourceMarketplace:
		return sourceTierPolicy{
			allowedActions:   marketplaceActionCeiling(),
			allowedSecurity:  slices.Clone(marketplaceSecurityCeiling),
			maxResourceScope: sourceTierMaxScope(source),
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

func effectiveResourceGrants(
	source ExtensionSource,
	operatorPolicy aghconfig.ExtensionsResourcesConfig,
	requested surfaces.GrantRequest,
	sessionMaxScope resources.ResourceScopeKind,
) ([]resources.ResourceKind, []resources.ResourceScopeKind, error) {
	if len(requested.Kinds) == 0 {
		return nil, nil, nil
	}

	grantedKinds := slices.Clone(requested.Kinds)
	if len(operatorPolicy.AllowedKinds) > 0 {
		allowedKinds, err := surfaces.NormalizeAllowedKinds(operatorPolicy.AllowedKinds)
		if err != nil {
			return nil, nil, err
		}
		grantedKinds = intersectKinds(grantedKinds, allowedKinds)
	}
	if len(grantedKinds) == 0 {
		return nil, nil, nil
	}

	finalMaxScope, err := narrowScopeCeiling(
		requested.MaxScope,
		sourceTierMaxScope(source),
		operatorPolicy.MaxScope,
		sessionMaxScope,
	)
	if err != nil {
		return nil, nil, err
	}
	grantedScopes := intersectScopes(requested.Scopes, scopesThrough(finalMaxScope))
	if len(grantedScopes) == 0 {
		return nil, nil, nil
	}
	return grantedKinds, grantedScopes, nil
}

func sourceTierMaxScope(source ExtensionSource) resources.ResourceScopeKind {
	switch source {
	case SourceWorkspace, SourceMarketplace:
		return resources.ResourceScopeKindWorkspace
	case SourceBundled, SourceUser:
		return resources.ResourceScopeKindGlobal
	default:
		return ""
	}
}

func narrowScopeCeiling(
	requested resources.ResourceScopeKind,
	sourceTier resources.ResourceScopeKind,
	operator resources.ResourceScopeKind,
	session resources.ResourceScopeKind,
) (resources.ResourceScopeKind, error) {
	candidates := []resources.ResourceScopeKind{
		requested.Normalize(),
		sourceTier.Normalize(),
		operator.Normalize(),
		session.Normalize(),
	}
	result := resources.ResourceScopeKindGlobal
	seen := false
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if err := candidate.Validate("resource scope"); err != nil {
			return "", err
		}
		if !seen || scopeRank(candidate) < scopeRank(result) {
			result = candidate
			seen = true
		}
	}
	if !seen {
		return "", nil
	}
	return result, nil
}

func scopeRank(scope resources.ResourceScopeKind) int {
	switch scope.Normalize() {
	case resources.ResourceScopeKindWorkspace:
		return 0
	case resources.ResourceScopeKindGlobal:
		return 1
	default:
		return 2
	}
}

func scopesThrough(maxScope resources.ResourceScopeKind) []resources.ResourceScopeKind {
	switch maxScope.Normalize() {
	case resources.ResourceScopeKindGlobal:
		return []resources.ResourceScopeKind{
			resources.ResourceScopeKindGlobal,
			resources.ResourceScopeKindWorkspace,
		}
	case resources.ResourceScopeKindWorkspace:
		return []resources.ResourceScopeKind{resources.ResourceScopeKindWorkspace}
	default:
		return nil
	}
}

func intersectKinds(
	left []resources.ResourceKind,
	right []resources.ResourceKind,
) []resources.ResourceKind {
	if len(left) == 0 || len(right) == 0 {
		return nil
	}
	index := make(map[resources.ResourceKind]struct{}, len(right))
	for _, kind := range right {
		index[kind.Normalize()] = struct{}{}
	}
	var kinds []resources.ResourceKind
	for _, kind := range left {
		normalized := kind.Normalize()
		if _, ok := index[normalized]; ok {
			kinds = append(kinds, normalized)
		}
	}
	if len(kinds) == 0 {
		return nil
	}
	slices.Sort(kinds)
	return kinds
}

func intersectScopes(
	left []resources.ResourceScopeKind,
	right []resources.ResourceScopeKind,
) []resources.ResourceScopeKind {
	if len(left) == 0 || len(right) == 0 {
		return nil
	}
	index := make(map[resources.ResourceScopeKind]struct{}, len(right))
	for _, scope := range right {
		index[scope.Normalize()] = struct{}{}
	}
	var scopes []resources.ResourceScopeKind
	for _, scope := range left {
		normalized := scope.Normalize()
		if _, ok := index[normalized]; ok {
			scopes = append(scopes, normalized)
		}
	}
	if len(scopes) == 0 {
		return nil
	}
	slices.Sort(scopes)
	return scopes
}

func cloneResourcePolicy(policy aghconfig.ExtensionsResourcesConfig) aghconfig.ExtensionsResourcesConfig {
	return aghconfig.ExtensionsResourcesConfig{
		AllowedKinds: append([]resources.ResourceKind(nil), policy.AllowedKinds...),
		MaxScope:     policy.MaxScope,
		SnapshotRateLimit: aghconfig.ExtensionsResourceRateLimitConfig{
			Requests: policy.SnapshotRateLimit.Requests,
			Window:   policy.SnapshotRateLimit.Window,
			Queue:    policy.SnapshotRateLimit.Queue,
		},
		OperatorWriteRateLimit: aghconfig.ExtensionsResourceRateLimitConfig{
			Requests: policy.OperatorWriteRateLimit.Requests,
			Window:   policy.OperatorWriteRateLimit.Window,
			Queue:    policy.OperatorWriteRateLimit.Queue,
		},
	}
}
