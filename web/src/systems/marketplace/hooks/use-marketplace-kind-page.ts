import { useEffect, useRef, useState } from "react";
import { useNavigate } from "@tanstack/react-router";

import {
  useBundleActivations,
  useExtensionInventory,
  type BundleActivation,
} from "@/systems/extensions";
import {
  SETTINGS_QUERY_INTERVALS,
  useSettingsMCPServers,
  type SettingsMCPServerEntry,
} from "@/systems/settings";
import { useSkills, type SkillPayload } from "@/systems/skill";
import { useActiveWorkspace } from "@/systems/workspace";
import { normalizeListingSearchValue } from "@/lib/listing-search";

import { useMarketplaceKind } from "./use-marketplace";
import type { MarketplaceKind, MarketplaceListing, MarketplaceRouteKind } from "../types";
import { marketplaceRouteKindFor } from "../types";
import {
  marketplaceKindScopeFromSearch,
  type MarketplaceKindScope,
  type MarketplaceKindSearch,
} from "../lib/marketplace-kind-search";

const SEARCH_DEBOUNCE_MS = 180;

export interface MarketplaceInstalledItem {
  entry: MarketplaceListing;
  skill?: SkillPayload;
  extensionEnabled?: boolean;
  viaBundle?: string | null;
  activationId?: string;
  activationVersion?: number;
  profileName?: string;
  scopeLabel?: string;
  mcpServer?: SettingsMCPServerEntry;
}

export interface MarketplaceKindPageModel {
  kind: MarketplaceKind;
  routeKind: MarketplaceRouteKind;
  scope: MarketplaceKindScope;
  query: string;
  draftQuery: string;
  setDraftQuery: (value: string) => void;
  setScope: (scope: MarketplaceKindScope) => void;
  clearSearch: () => void;
  marketplaceTotal: number;
  installedCount: number;
  updatesAvailable: number;
  marketEntries: readonly MarketplaceListing[];
  installedItems: readonly MarketplaceInstalledItem[];
  isLoading: boolean;
  isFetching: boolean;
  error: Error | null;
  refetch: () => void;
  workspaceId: string | null | undefined;
}

function matchesQuery(haystack: string, query: string): boolean {
  if (!query) return true;
  return haystack.toLowerCase().includes(query.toLowerCase());
}

function listingHaystack(entry: MarketplaceListing): string {
  return [entry.name, entry.description, entry.author, entry.transport, entry.source]
    .filter(Boolean)
    .join(" ");
}

function useMarketplaceKindPage(
  kind: MarketplaceKind,
  search: MarketplaceKindSearch
): MarketplaceKindPageModel {
  const navigate = useNavigate();
  const routeKind = marketplaceRouteKindFor(kind);
  const scope = marketplaceKindScopeFromSearch(search);
  const routeQuery = search.q ?? "";
  const { activeWorkspaceId } = useActiveWorkspace();
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const [draftState, setDraftState] = useState({ routeQuery, value: routeQuery });
  const draftQuery = draftState.routeQuery === routeQuery ? draftState.value : routeQuery;

  const marketQuery = useMarketplaceKind({
    kind,
    limit: 100,
    q: scope === "market" ? routeQuery || null : null,
    workspaceId: activeWorkspaceId,
  });

  const skillsQuery = useSkills(activeWorkspaceId ?? "");
  const extensionsQuery = useExtensionInventory();
  const bundlesQuery = useBundleActivations();
  const mcpPollInterval = SETTINGS_QUERY_INTERVALS.collectionRefetchInterval;
  const mcpGlobalQuery = useSettingsMCPServers(
    { scope: "global" },
    { enabled: kind === "mcp", refetchInterval: mcpPollInterval }
  );
  const mcpWorkspaceQuery = useSettingsMCPServers(
    { scope: "workspace", workspace_id: activeWorkspaceId ?? undefined },
    {
      enabled: kind === "mcp" && Boolean(activeWorkspaceId),
      refetchInterval: mcpPollInterval,
    }
  );

  useEffect(() => {
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, []);

  const updateSearch = (updater: (current: MarketplaceKindSearch) => MarketplaceKindSearch) => {
    void navigate({
      search: current => updater(current as MarketplaceKindSearch),
      to: `/marketplace/${routeKind}`,
    });
  };

  const setDraftQuery = (nextQuery: string) => {
    setDraftState({ routeQuery, value: nextQuery });
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => {
      debounceRef.current = null;
      updateSearch(current => ({
        ...current,
        q: normalizeListingSearchValue(nextQuery),
      }));
    }, SEARCH_DEBOUNCE_MS);
  };

  const setScope = (next: MarketplaceKindScope) => {
    updateSearch(current => ({
      ...current,
      tab: next === "installed" ? "installed" : undefined,
    }));
  };

  const clearSearch = () => {
    setDraftState({ routeQuery: "", value: "" });
    updateSearch(current => ({ ...current, q: undefined }));
  };

  const marketItems = marketQuery.data?.items ?? [];
  const marketplaceTotal = marketQuery.data?.total ?? marketItems.length;

  const listingBySlug = new Map<string, MarketplaceListing>();
  const listingByEntryId = new Map<string, MarketplaceListing>();
  for (const entry of marketItems) {
    listingByEntryId.set(entry.entry_id, entry);
    if (entry.install_slug) listingBySlug.set(entry.install_slug, entry);
  }

  const mcpServers = mergeMCPServers(
    mcpGlobalQuery.data?.mcp_servers ?? [],
    mcpWorkspaceQuery.data?.mcp_servers ?? []
  );

  const installedItems = buildInstalledItems({
    kind,
    query: routeQuery,
    marketItems,
    skills: skillsQuery.data ?? [],
    extensions: extensionsQuery.data ?? [],
    activations: bundlesQuery.data ?? [],
    mcpServers,
    listingBySlug,
    listingByEntryId,
  });

  const installedCount = installedItems.length;
  const updatesAvailable = installedItems.filter(item => item.entry.update_available).length;

  const marketEntries =
    scope === "market"
      ? marketItems.filter(entry => matchesQuery(listingHaystack(entry), routeQuery))
      : [];

  const inventoryLoading =
    kind === "skill"
      ? skillsQuery.isLoading
      : kind === "extension"
        ? extensionsQuery.isLoading
        : kind === "bundle"
          ? bundlesQuery.isLoading
          : kind === "mcp"
            ? mcpGlobalQuery.isLoading ||
              (Boolean(activeWorkspaceId) && mcpWorkspaceQuery.isLoading)
            : false;

  const isLoading =
    scope === "market" ? marketQuery.isLoading : marketQuery.isLoading || inventoryLoading;
  const inventoryError =
    kind === "skill"
      ? (skillsQuery.error ?? null)
      : kind === "extension"
        ? (extensionsQuery.error ?? null)
        : kind === "bundle"
          ? (bundlesQuery.error ?? null)
          : kind === "mcp"
            ? (mcpGlobalQuery.error ?? mcpWorkspaceQuery.error ?? null)
            : null;

  return {
    kind,
    routeKind,
    scope,
    query: routeQuery,
    draftQuery,
    setDraftQuery,
    setScope,
    clearSearch,
    marketplaceTotal,
    installedCount,
    updatesAvailable,
    marketEntries,
    installedItems: scope === "installed" ? installedItems : [],
    isLoading,
    isFetching: marketQuery.isFetching,
    error: (scope === "market" ? marketQuery.error : (inventoryError ?? marketQuery.error)) ?? null,
    refetch: () => {
      void marketQuery.refetch();
      if (kind === "skill") void skillsQuery.refetch();
      if (kind === "extension") void extensionsQuery.refetch();
      if (kind === "bundle") void bundlesQuery.refetch();
      if (kind === "mcp") {
        void mcpGlobalQuery.refetch();
        if (activeWorkspaceId) void mcpWorkspaceQuery.refetch();
      }
    },
    workspaceId: activeWorkspaceId,
  };
}

function mergeMCPServers(
  globalServers: readonly SettingsMCPServerEntry[],
  workspaceServers: readonly SettingsMCPServerEntry[]
): SettingsMCPServerEntry[] {
  const byName = new Map<string, SettingsMCPServerEntry>();
  for (const server of globalServers) byName.set(server.name, server);
  for (const server of workspaceServers) byName.set(server.name, server);
  return Array.from(byName.values());
}

function buildInstalledItems(input: {
  kind: MarketplaceKind;
  query: string;
  marketItems: readonly MarketplaceListing[];
  skills: readonly SkillPayload[];
  extensions: readonly {
    extension: {
      name: string;
      version: string;
      enabled: boolean;
      marketplace?: MarketplaceListing | null;
    };
    listing: MarketplaceListing | null;
    updateAvailable: boolean;
  }[];
  activations: readonly BundleActivation[];
  mcpServers: readonly SettingsMCPServerEntry[];
  listingBySlug: Map<string, MarketplaceListing>;
  listingByEntryId: Map<string, MarketplaceListing>;
}): MarketplaceInstalledItem[] {
  if (input.kind === "mcp") {
    const items: MarketplaceInstalledItem[] = [];
    for (const server of input.mcpServers) {
      const catalogEntry = server.catalog_entry?.trim();
      const listing =
        (catalogEntry ? input.listingByEntryId.get(catalogEntry) : undefined) ??
        input.marketItems.find(
          entry =>
            entry.entry_id === catalogEntry ||
            entry.installed_name === server.name ||
            entry.name === server.name
        );
      const entry: MarketplaceListing = listing
        ? {
            ...listing,
            installed: true,
            installed_name: server.name,
            installed_version: server.catalog_version ?? listing.installed_version,
            transport: server.transport || listing.transport,
            update_available: listing.update_available,
          }
        : {
            entry_id: catalogEntry || server.name,
            kind: "mcp",
            name: server.name,
            description: "",
            installed: true,
            installed_name: server.name,
            installed_version: server.catalog_version,
            update_available: false,
            transport: server.transport,
            source: "installed",
            version: server.catalog_version,
          };
      const installed: MarketplaceInstalledItem = {
        entry,
        mcpServer: server,
        scopeLabel: server.workspace_id ? `${server.scope} · ${server.workspace_id}` : server.scope,
      };
      if (
        matchesQuery(
          [
            installed.entry.name,
            installed.entry.description,
            installed.entry.transport,
            installed.scopeLabel,
          ]
            .filter(Boolean)
            .join(" "),
          input.query
        )
      ) {
        items.push(installed);
      }
    }
    return items;
  }

  if (input.kind === "skill") {
    const items: MarketplaceInstalledItem[] = [];
    for (const skill of input.skills) {
      const slug = skill.provenance?.slug?.trim();
      const listing =
        (slug ? input.listingBySlug.get(slug) : undefined) ??
        input.marketItems.find(
          entry =>
            entry.installed_name === skill.name ||
            entry.name === skill.name ||
            entry.entry_id === skill.name
        );
      const entry: MarketplaceListing = listing
        ? {
            ...listing,
            installed: true,
            installed_name: skill.name,
            installed_version: skill.version ?? listing.installed_version,
            update_available: listing.update_available,
          }
        : {
            entry_id: skill.name,
            kind: "skill",
            name: skill.name,
            description: skill.description ?? "",
            installed: true,
            installed_name: skill.name,
            installed_version: skill.version,
            update_available: false,
            version: skill.version,
            source: skill.source,
          };
      const item: MarketplaceInstalledItem = {
        entry,
        skill,
        viaBundle: skill.provenance?.installed_from_bundle ?? null,
        activationId: skill.provenance?.installed_from_bundle
          ? input.activations.find(activation => {
              const bundleRef = skill.provenance?.installed_from_bundle;
              return (
                activation.bundle_name === bundleRef ||
                `${activation.bundle_name}/${activation.profile_name}` === bundleRef
              );
            })?.id
          : undefined,
      };
      if (
        matchesQuery(
          [item.entry.name, item.entry.description, item.viaBundle].filter(Boolean).join(" "),
          input.query
        )
      ) {
        items.push(item);
      }
    }
    return items;
  }

  if (input.kind === "extension") {
    const items: MarketplaceInstalledItem[] = [];
    for (const item of input.extensions) {
      const listing = item.listing;
      const entry: MarketplaceListing = listing
        ? {
            ...listing,
            installed: true,
            installed_name: item.extension.name,
            installed_version: item.extension.version,
            update_available: item.updateAvailable,
          }
        : {
            entry_id: item.extension.name,
            kind: "extension",
            name: item.extension.name,
            description: "",
            installed: true,
            installed_name: item.extension.name,
            installed_version: item.extension.version,
            update_available: item.updateAvailable,
            version: item.extension.version,
            source: "",
          };
      const installed: MarketplaceInstalledItem = {
        entry,
        extensionEnabled: item.extension.enabled,
      };
      if (matchesQuery(listingHaystack(installed.entry), input.query)) {
        items.push(installed);
      }
    }
    return items;
  }

  const items: MarketplaceInstalledItem[] = [];
  for (const activation of input.activations) {
    const listing =
      input.marketItems.find(entry => entry.name === activation.bundle_name) ??
      input.listingByEntryId.get(activation.bundle_name);
    const entry: MarketplaceListing = listing
      ? {
          ...listing,
          installed: true,
          installed_name: activation.bundle_name,
          update_available: activation.spec_drift === true,
        }
      : {
          entry_id: activation.bundle_name,
          kind: "bundle",
          name: activation.bundle_name,
          description: "",
          installed: true,
          installed_name: activation.bundle_name,
          update_available: activation.spec_drift === true,
          source: activation.extension_name,
        };
    const installed: MarketplaceInstalledItem = {
      entry,
      activationId: activation.id,
      activationVersion: activation.version,
      profileName: activation.profile_name,
      scopeLabel: activation.workspace_id
        ? `${activation.scope} · ${activation.workspace_id}`
        : activation.scope,
    };
    if (
      matchesQuery(
        [
          installed.entry.name,
          installed.entry.description,
          installed.profileName,
          installed.scopeLabel,
        ]
          .filter(Boolean)
          .join(" "),
        input.query
      )
    ) {
      items.push(installed);
    }
  }
  return items;
}

export { useMarketplaceKindPage };
