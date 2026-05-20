package contract

// HardCutTarget records a public contract that the prod-ready hard-cut work
// must delete without compatibility aliases or deprecated shims.
type HardCutTarget struct {
	Kind        string `json:"kind"`
	Identifier  string `json:"identifier"`
	Path        string `json:"path,omitempty"`
	Replacement string `json:"replacement,omitempty"`
	Owner       string `json:"owner"`
}

const (
	HardCutKindCLI       = "cli"
	HardCutKindConfig    = "config"
	HardCutKindEvent     = "event"
	HardCutKindHTTPRoute = "http_route"
	HardCutKindWebHook   = "web_hook"

	hardCutOwnerLogs   = "logs"
	hardCutOwnerStatus = "status"
)

// ProdReadyHardCutTargets returns the cross-slice delete targets accepted by
// the prod-ready TechSpec and cross-cut contract amendments.
func ProdReadyHardCutTargets() []HardCutTarget {
	return []HardCutTarget{
		{
			Kind:        HardCutKindCLI,
			Identifier:  "agh daemon status",
			Replacement: "agh status",
			Owner:       hardCutOwnerStatus,
		},
		{
			Kind:        HardCutKindCLI,
			Identifier:  "agh observe health",
			Replacement: "agh status",
			Owner:       hardCutOwnerStatus,
		},
		{
			Kind:        HardCutKindCLI,
			Identifier:  "agh observe events",
			Replacement: "agh logs",
			Owner:       hardCutOwnerLogs,
		},
		{
			Kind:        HardCutKindHTTPRoute,
			Identifier:  "GET /api/daemon/status",
			Replacement: "GET /api/status",
			Owner:       hardCutOwnerStatus,
		},
		{
			Kind:        HardCutKindHTTPRoute,
			Identifier:  "GET /api/observe/health",
			Replacement: "GET /api/status",
			Owner:       hardCutOwnerStatus,
		},
		{
			Kind:        HardCutKindHTTPRoute,
			Identifier:  "GET /api/observe/events",
			Replacement: "GET /api/logs",
			Owner:       hardCutOwnerLogs,
		},
		{
			Kind:        HardCutKindHTTPRoute,
			Identifier:  "GET /api/observe/events/stream",
			Replacement: "GET /api/logs/stream",
			Owner:       hardCutOwnerLogs,
		},
		{
			Kind:        HardCutKindHTTPRoute,
			Identifier:  "GET /api/providers/{provider_id}/*catalog_path",
			Replacement: "GET /api/model-catalog/providers/{provider_id}/models",
			Owner:       "provider-auth",
		},
		{
			Kind:        HardCutKindEvent,
			Identifier:  "skills.shadow",
			Replacement: "skill.shadowed",
			Owner:       "skills",
		},
		{
			Kind:       HardCutKindConfig,
			Identifier: "ProviderConfig.Aliases",
			Owner:      "provider-auth",
		},
		{
			Kind:        HardCutKindConfig,
			Identifier:  "notifications.presets.<name>",
			Replacement: "notification_presets",
			Owner:       "notifications",
		},
		{
			Kind:       HardCutKindConfig,
			Identifier: "network.presence.active_window_minutes",
			Owner:      "presence",
		},
		{
			Kind:       HardCutKindWebHook,
			Identifier: "useNetworkPresence",
			Path:       "web/src/hooks/use-network-presence.ts",
			Owner:      "presence",
		},
	}
}
