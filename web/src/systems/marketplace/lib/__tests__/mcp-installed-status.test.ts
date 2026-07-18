import { describe, expect, it } from "vitest";

import type { SettingsMCPServerEntry } from "@/systems/settings";

import { marketplaceMCPInstalledStatus } from "../mcp-installed-status";

function server(
  overrides: Partial<SettingsMCPServerEntry> & Pick<SettingsMCPServerEntry, "name" | "transport">
): SettingsMCPServerEntry {
  return {
    scope: "workspace",
    source_metadata: {
      available_targets: [],
      effective_source: { kind: "workspace-config", scope: "workspace" },
      shadowed_sources: [],
    },
    ...overrides,
  };
}

describe("marketplaceMCPInstalledStatus", () => {
  it("Should mark OAuth remotes that need login as authorize", () => {
    expect(
      marketplaceMCPInstalledStatus(
        server({
          name: "linear",
          transport: "sse",
          auth: { type: "oauth2_pkce", client_id: "x", client_secret_configured: false },
          auth_status: {
            server_name: "linear",
            scope: "workspace",
            status: "needs_login",
            token_present: false,
            refreshable: true,
          },
          runtime_status: {
            configured: true,
            initialized: false,
            state: "auth_required",
            probe: "skipped",
            tool_count: 0,
          },
        })
      )
    ).toBe("authorize");
  });

  it("Should mark ready authenticated remotes as running", () => {
    expect(
      marketplaceMCPInstalledStatus(
        server({
          name: "sentry",
          transport: "sse",
          auth: { type: "oauth2_pkce", client_id: "x", client_secret_configured: false },
          auth_status: {
            server_name: "sentry",
            scope: "workspace",
            status: "authenticated",
            token_present: true,
            refreshable: true,
          },
          runtime_status: {
            configured: true,
            initialized: true,
            state: "ready",
            probe: "succeeded",
            tool_count: 4,
          },
        })
      )
    ).toBe("running");
  });

  it("Should mark ready stdio servers as running", () => {
    expect(
      marketplaceMCPInstalledStatus(
        server({
          name: "filesystem",
          transport: "stdio",
          runtime_status: {
            configured: true,
            initialized: true,
            state: "ready",
            probe: "succeeded",
            tool_count: 3,
          },
        })
      )
    ).toBe("running");
  });
});
