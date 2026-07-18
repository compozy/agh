import { useEffect } from "react";

import { Button, Pill, Section } from "@agh/ui";

import { useMCPAuthorize } from "@/hooks/routes/use-mcp-authorize";
import {
  authorizeLabel,
  composeMCPRowStatus,
  deriveMCPAuthFilter,
  MCPAuthorizeDialog,
  SETTINGS_QUERY_INTERVALS,
  useSettingsMCPServers,
  type SettingsMCPServerEntry,
} from "@/systems/settings";
import { useActiveWorkspace } from "@/systems/workspace";

import type { MarketplaceEntryResponse } from "../types";

interface MarketplaceDetailMCPManageProps {
  entry: MarketplaceEntryResponse["entry"];
}

function findInstalledMCPServer(
  entry: MarketplaceEntryResponse["entry"],
  servers: readonly SettingsMCPServerEntry[]
): SettingsMCPServerEntry | undefined {
  return servers.find(
    server =>
      server.catalog_entry === entry.entry_id ||
      server.name === entry.installed_name ||
      server.name === entry.name
  );
}

function MarketplaceDetailMCPManage({ entry }: MarketplaceDetailMCPManageProps) {
  const { activeWorkspaceId } = useActiveWorkspace();
  const pollInterval = SETTINGS_QUERY_INTERVALS.collectionRefetchInterval;
  const globalQuery = useSettingsMCPServers({ scope: "global" }, { refetchInterval: pollInterval });
  const workspaceQuery = useSettingsMCPServers(
    { scope: "workspace", workspace_id: activeWorkspaceId ?? undefined },
    { enabled: Boolean(activeWorkspaceId), refetchInterval: pollInterval }
  );
  const server = findInstalledMCPServer(entry, [
    ...(globalQuery.data?.mcp_servers ?? []),
    ...(workspaceQuery.data?.mcp_servers ?? []),
  ]);

  const authFilter = server ? deriveMCPAuthFilter(server) : null;
  const authorize = useMCPAuthorize(authFilter);
  const { acknowledgeStatus, isAwaiting } = authorize;

  useEffect(() => {
    const status = server?.auth_status;
    if (isAwaiting && status?.status) {
      acknowledgeStatus(status.status, Boolean(status.token_present));
    }
  }, [acknowledgeStatus, isAwaiting, server?.auth_status]);

  if (!server) return null;

  const status = composeMCPRowStatus(server);
  const label = authorizeLabel(server);

  return (
    <>
      <Section label="Manage">
        <div className="flex flex-col gap-3 rounded-lg bg-canvas-soft px-4 py-3">
          <div className="flex flex-wrap items-center gap-1.5">
            <Pill mono tone={status.auth.tone}>
              {status.auth.label}
            </Pill>
            <Pill mono tone={status.runtime.tone}>
              {status.runtime.label}
            </Pill>
            <Pill mono tone={status.probe.tone}>
              {status.probe.label}
            </Pill>
          </div>
          {label ? (
            <div>
              <Button
                data-testid="mcp-authorize-btn"
                disabled={!authFilter || authorize.isAwaiting}
                onClick={() => {
                  if (!authFilter) return;
                  void authorize.beginAuthorize(server.name, {
                    status: server.auth_status?.status ?? "needs_login",
                    tokenPresent: Boolean(server.auth_status?.token_present),
                  });
                }}
                size="sm"
                type="button"
                variant="outline"
              >
                {label}
              </Button>
            </div>
          ) : null}
        </div>
      </Section>
      <MCPAuthorizeDialog
        authorize={authorize}
        scope={server.scope === "workspace" ? "workspace" : "global"}
        server={server}
      />
    </>
  );
}

export { MarketplaceDetailMCPManage };
export type { MarketplaceDetailMCPManageProps };
