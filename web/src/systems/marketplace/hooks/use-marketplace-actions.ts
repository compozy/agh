import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { reconcileInstalledExtensionCaches } from "@/integrations/tanstack-query/reconcile-installed-extension";
import { updateExtension } from "@/systems/extensions/adapters/extensions-api";
import { settingsKeys } from "@/systems/settings";
import { skillKeys } from "@/systems/skill";

import {
  activateMarketplaceBundle,
  installMarketplaceExtension,
  installMarketplaceMCP,
  installMarketplaceSkill,
  previewMarketplaceBundle,
  refreshMarketplaceCatalog,
  updateMarketplaceSkill,
} from "../adapters/marketplace-actions-api";
import { marketplaceKeys } from "../lib/query-keys";
import type {
  BundleActivationRequest,
  BundlePreviewRequest,
  ExtensionInstallRequest,
  ExtensionUpdateRequest,
  MarketplaceKind,
  MCPInstallRequest,
  SkillInstallRequest,
  SkillUpdateRequest,
} from "../types";

function invalidateMarketplace(queryClient: ReturnType<typeof useQueryClient>) {
  return queryClient.invalidateQueries({ queryKey: marketplaceKeys.all });
}

// Deep-link the post-install toast to the exact installed scope + server so the
// operator lands on /mcp with the row preselected, matching the marketplace
// Manage path producer (internal/api/core/marketplace_list.go).
function mcpManagePath(scope: string, server: string): string {
  const params = new URLSearchParams({ scope });
  if (server) params.set("server", server);
  return `/mcp?${params.toString()}`;
}

function mcpToastAction(label: "Authorize →" | "Manage →", path: string) {
  return {
    action: {
      label,
      onClick: () => globalThis.location.assign(path),
    },
  };
}

export function useRefreshMarketplaceCatalog() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (kind: Exclude<MarketplaceKind, "bundle"> | undefined) =>
      refreshMarketplaceCatalog(kind),
    onSettled: () => invalidateMarketplace(queryClient),
  });
}

export function useInstallMarketplaceMCP() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: MCPInstallRequest) => installMarketplaceMCP(body),
    onSuccess: (result, variables) => {
      const server = result.mcp_server?.name ?? variables.name ?? "";
      const scope = result.mcp_server?.scope ?? variables.scope;
      const path = mcpManagePath(scope, server);
      if (result.next_step === "authorize") {
        toast.success(
          "MCP server installed. Authorization is required.",
          mcpToastAction("Authorize →", path)
        );
        return;
      }
      toast.success("MCP server installed.", mcpToastAction("Manage →", path));
    },
    onSettled: () =>
      Promise.all([
        invalidateMarketplace(queryClient),
        queryClient.invalidateQueries({ queryKey: settingsKeys.mcpRoot() }),
      ]),
  });
}

export function useInstallMarketplaceSkill() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: SkillInstallRequest) => installMarketplaceSkill(body),
    onSettled: () =>
      Promise.all([
        invalidateMarketplace(queryClient),
        queryClient.invalidateQueries({ queryKey: skillKeys.all }),
      ]),
  });
}

export function useUpdateMarketplaceSkill() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: SkillUpdateRequest) => updateMarketplaceSkill(body),
    onSettled: () =>
      Promise.all([
        invalidateMarketplace(queryClient),
        queryClient.invalidateQueries({ queryKey: skillKeys.all }),
      ]),
  });
}

export function useInstallMarketplaceExtension() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: ExtensionInstallRequest) => installMarketplaceExtension(body),
    onSettled: () => reconcileInstalledExtensionCaches(queryClient),
  });
}

export function useUpdateMarketplaceExtension() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ name, body }: { name: string; body: ExtensionUpdateRequest }) =>
      updateExtension(name, body),
    onSettled: () => reconcileInstalledExtensionCaches(queryClient),
  });
}

export function usePreviewMarketplaceBundle() {
  return useMutation({
    mutationFn: (body: BundlePreviewRequest) => previewMarketplaceBundle(body),
  });
}

export function useActivateMarketplaceBundle() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: BundleActivationRequest) => activateMarketplaceBundle(body),
    onSettled: () => invalidateMarketplace(queryClient),
  });
}
