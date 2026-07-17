import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { toast } from "sonner";

import {
  useInstallMarketplaceExtension,
  useInstallMarketplaceMCP,
  useInstallMarketplaceSkill,
  useUpdateMarketplaceExtension,
  useUpdateMarketplaceSkill,
} from "../hooks/use-marketplace-actions";
import { marketplaceEntryOptions } from "../lib/query-options";
import type {
  MarketplaceEntryResponse,
  MarketplaceKind,
  MarketplaceListing,
  MCPInstallRequest,
} from "../types";
import { BundleActivationDialog } from "./bundle-activation-dialog";
import { ExtensionTrustDialog } from "./extension-trust-dialog";
import { MCPInstallDialog } from "./mcp-install-dialog";
import { marketplaceEntrySlug, marketplaceErrorMessage } from "./marketplace-ui";

interface MarketplaceActionController {
  dialogs: React.ReactNode;
  handleAction: (entry: MarketplaceListing) => void;
  isEntryPending: (entry: MarketplaceListing) => boolean;
}

function pendingKey(entry: MarketplaceListing): string {
  return `${entry.kind}:${entry.entry_id}`;
}

function installedName(entry: MarketplaceListing): string {
  if (!entry.installed_name) {
    throw new Error(`Installed identity is unavailable for ${entry.name}`);
  }
  return entry.installed_name;
}

function installedSkillPath(name: string): string {
  return `/skills/${encodeURIComponent(name)}`;
}

function installedExtensionPath(name: string): string {
  return `/extensions/${encodeURIComponent(name)}`;
}

function useMarketplaceActionController(workspaceId?: string | null): MarketplaceActionController {
  const queryClient = useQueryClient();
  const installSkill = useInstallMarketplaceSkill();
  const updateSkill = useUpdateMarketplaceSkill();
  const installExtension = useInstallMarketplaceExtension();
  const updateExtension = useUpdateMarketplaceExtension();
  const installMCP = useInstallMarketplaceMCP();
  const [pendingEntries, setPendingEntries] = useState<ReadonlyMap<string, number>>(
    () => new Map()
  );
  const [mcpDetail, setMCPDetail] = useState<MarketplaceEntryResponse | null>(null);
  const [bundleDetail, setBundleDetail] = useState<MarketplaceEntryResponse | null>(null);
  const [trustEntry, setTrustEntry] = useState<MarketplaceListing | null>(null);
  const [trustError, setTrustError] = useState<string | null>(null);

  const trackPendingEntry = async <T,>(
    entry: MarketplaceListing,
    action: () => Promise<T>
  ): Promise<T> => {
    const key = pendingKey(entry);
    setPendingEntries(current => {
      const next = new Map(current);
      next.set(key, (next.get(key) ?? 0) + 1);
      return next;
    });
    return Promise.resolve()
      .then(action)
      .finally(() => {
        setPendingEntries(current => {
          const next = new Map(current);
          const remaining = (next.get(key) ?? 1) - 1;
          if (remaining > 0) next.set(key, remaining);
          else next.delete(key);
          return next;
        });
      });
  };

  const withPendingEntry = async (entry: MarketplaceListing, action: () => Promise<void>) => {
    try {
      await trackPendingEntry(entry, action);
    } catch (error) {
      toast.error(marketplaceErrorMessage(error, `Failed to update ${entry.name}`));
    }
  };

  const fetchDetail = async (entry: MarketplaceListing) => {
    const kind = entry.kind as MarketplaceKind;
    return queryClient.fetchQuery(
      marketplaceEntryOptions({ entryId: entry.entry_id, kind, workspaceId })
    );
  };

  const handleAction = (entry: MarketplaceListing) => {
    if (entry.kind === "extension" && entry.trust?.decision === "blocked") return;
    if (entry.kind === "extension" && entry.trust?.decision === "allowed_unverified") {
      setTrustError(null);
      setTrustEntry(entry);
      return;
    }
    void withPendingEntry(entry, async () => {
      if (entry.kind === "skill") {
        if (entry.update_available) {
          await updateSkill.mutateAsync({ name: installedName(entry) });
          toast.success(`${entry.name} updated`);
        } else {
          const result = await installSkill.mutateAsync({
            slug: marketplaceEntrySlug(entry),
            version: entry.version,
          });
          toast.success(`${entry.name} installed`, {
            action: {
              label: "Manage →",
              onClick: () => globalThis.location.assign(installedSkillPath(result.skill.name)),
            },
          });
        }
        return;
      }
      if (entry.kind === "extension") {
        if (entry.update_available) {
          await updateExtension.mutateAsync({
            body: { allow_unverified: false, version: entry.version },
            name: installedName(entry),
          });
          toast.success(`${entry.name} updated`);
          return;
        }
        const result = await installExtension.mutateAsync({
          allow_unverified: false,
          slug: marketplaceEntrySlug(entry),
          version: entry.version,
        });
        toast.success(`${entry.name} installed`, {
          action: {
            label: "Manage →",
            onClick: () =>
              globalThis.location.assign(installedExtensionPath(result.extension.name)),
          },
        });
        return;
      }
      const detail = await fetchDetail(entry);
      if (entry.kind === "mcp") setMCPDetail(detail);
      if (entry.kind === "bundle") setBundleDetail(detail);
    });
  };

  const confirmUnverifiedExtension = async () => {
    if (!trustEntry) return;
    setTrustError(null);
    try {
      await trackPendingEntry(trustEntry, async () => {
        if (trustEntry.update_available) {
          await updateExtension.mutateAsync({
            body: { allow_unverified: true, version: trustEntry.version },
            name: installedName(trustEntry),
          });
          toast.success(`${trustEntry.name} updated`);
          return;
        }
        const result = await installExtension.mutateAsync({
          allow_unverified: true,
          slug: marketplaceEntrySlug(trustEntry),
          version: trustEntry.version,
        });
        toast.success(`${trustEntry.name} installed`, {
          action: {
            label: "Manage →",
            onClick: () =>
              globalThis.location.assign(installedExtensionPath(result.extension.name)),
          },
        });
      });
      setTrustEntry(null);
    } catch (error) {
      setTrustError(marketplaceErrorMessage(error, "Failed to install the extension"));
    }
  };

  const installSelectedMCP = async (request: MCPInstallRequest) => {
    return installMCP.mutateAsync(request);
  };

  const dialogs = (
    <>
      {mcpDetail ? (
        <MCPInstallDialog
          data={mcpDetail}
          key={mcpDetail.entry.entry_id}
          onInstall={installSelectedMCP}
          onOpenChange={open => {
            if (!open) setMCPDetail(null);
          }}
          open
          workspaceId={workspaceId}
        />
      ) : null}
      {bundleDetail ? (
        <BundleActivationDialog
          data={bundleDetail}
          key={bundleDetail.entry.entry_id}
          onOpenChange={open => {
            if (!open) setBundleDetail(null);
          }}
          open
          workspaceId={workspaceId}
        />
      ) : null}
      {trustEntry ? (
        <ExtensionTrustDialog
          entry={trustEntry}
          error={trustError}
          onConfirm={() => void confirmUnverifiedExtension()}
          onOpenChange={open => {
            if (!open) setTrustEntry(null);
          }}
          open
          pending={installExtension.isPending || updateExtension.isPending}
        />
      ) : null}
    </>
  );

  return {
    dialogs,
    handleAction,
    isEntryPending: entry => (pendingEntries.get(pendingKey(entry)) ?? 0) > 0,
  };
}

export { useMarketplaceActionController };
export type { MarketplaceActionController };
