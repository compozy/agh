package contract

import "testing"

func TestProdReadyHardCutTargets(t *testing.T) {
	t.Parallel()

	t.Run("Should codify cross-cut delete targets without aliases", func(t *testing.T) {
		t.Parallel()

		targets := ProdReadyHardCutTargets()
		if len(targets) == 0 {
			t.Fatal("ProdReadyHardCutTargets() returned no targets")
		}
		seen := make(map[string]HardCutTarget, len(targets))
		for _, target := range targets {
			if target.Kind == "" || target.Identifier == "" || target.Owner == "" {
				t.Fatalf("hard-cut target is incomplete: %#v", target)
			}
			key := target.Kind + "\x00" + target.Identifier
			if previous, ok := seen[key]; ok {
				t.Fatalf("hard-cut target duplicated: %#v and %#v", previous, target)
			}
			seen[key] = target
		}

		for _, target := range []HardCutTarget{
			{Kind: HardCutKindCLI, Identifier: "agh daemon status"},
			{Kind: HardCutKindCLI, Identifier: "agh observe health"},
			{Kind: HardCutKindCLI, Identifier: "agh observe events"},
			{Kind: HardCutKindHTTPRoute, Identifier: "GET /api/daemon/status"},
			{Kind: HardCutKindHTTPRoute, Identifier: "GET /api/observe/health"},
			{Kind: HardCutKindHTTPRoute, Identifier: "GET /api/observe/events"},
			{Kind: HardCutKindHTTPRoute, Identifier: "GET /api/providers/{provider_id}/*catalog_path"},
			{Kind: HardCutKindEvent, Identifier: "skills.shadow"},
			{Kind: HardCutKindConfig, Identifier: "ProviderConfig.Aliases"},
			{Kind: HardCutKindConfig, Identifier: "notifications.presets.<name>"},
			{Kind: HardCutKindWebHook, Identifier: "useNetworkPresence"},
		} {
			key := target.Kind + "\x00" + target.Identifier
			if _, ok := seen[key]; !ok {
				t.Fatalf("missing hard-cut target %s %q", target.Kind, target.Identifier)
			}
		}
		webHook := seen[HardCutKindWebHook+"\x00"+"useNetworkPresence"]
		if webHook.Path != "web/src/hooks/use-network-presence.ts" {
			t.Fatalf("useNetworkPresence Path = %q, want source path", webHook.Path)
		}
	})
}
