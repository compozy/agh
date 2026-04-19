import { AlertCircle, Loader2, Waypoints } from "lucide-react";

import { Empty, KindChip, MonoBadge, SearchInput, StatusDot, type StatusDotTone } from "@agh/ui";
import { cn } from "@/lib/utils";

import {
  bridgeStatusTone,
  formatBridgeRelativeTime,
} from "@/systems/bridges/lib/bridge-formatters";
import type { BridgeHealthMap, BridgeSummary } from "@/systems/bridges/types";

interface BridgeListPanelProps {
  bridgeHealth: BridgeHealthMap;
  bridges: BridgeSummary[];
  errorMessage?: string | null;
  isLoading?: boolean;
  onSearchChange: (query: string) => void;
  onSelectBridge: (bridgeId: string) => void;
  searchQuery: string;
  selectedBridgeId: string | null;
  summary: string;
}

interface BridgeListItemProps {
  bridge: BridgeSummary;
  health?: BridgeHealthMap[string];
  isSelected: boolean;
  onSelect: () => void;
}

function listItemTone(status: BridgeSummary["status"]): StatusDotTone {
  switch (bridgeStatusTone(status)) {
    case "green":
      return "success";
    case "amber":
      return "warning";
    case "danger":
      return "danger";
    case "violet":
      return "info";
    default:
      return "neutral";
  }
}

function BridgeListItem({ bridge, health, isSelected, onSelect }: BridgeListItemProps) {
  const tone = listItemTone(bridge.status);
  const pulse = bridge.status === "starting";
  const effectiveStatus = health?.status ?? bridge.status;

  return (
    <button
      aria-pressed={isSelected}
      className={cn(
        "relative flex w-full flex-col gap-2 border-b border-[color:var(--color-divider)] px-4 py-3 text-left transition-colors",
        "hover:bg-[color:var(--color-hover)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[color:var(--color-accent)]",
        isSelected && "bg-[color:var(--color-surface)]"
      )}
      data-state={isSelected ? "selected" : undefined}
      data-testid={`bridge-item-${bridge.id}`}
      onClick={onSelect}
      type="button"
    >
      {isSelected ? (
        <span
          aria-hidden="true"
          className="absolute left-0 top-2 bottom-2 w-[3px] rounded-r bg-[color:var(--color-accent)]"
          data-testid="bridge-active-indicator"
        />
      ) : null}

      <div className="flex items-start justify-between gap-3">
        <div className="flex min-w-0 items-center gap-2">
          <StatusDot pulse={pulse} tone={tone} />
          <span className="truncate text-[13px] font-medium text-[color:var(--color-text-primary)]">
            {bridge.display_name}
          </span>
        </div>
        <span className="shrink-0 font-mono text-[10px] uppercase tracking-[0.1em] text-[color:var(--color-text-tertiary)]">
          {formatBridgeRelativeTime(health?.last_success_at)}
        </span>
      </div>

      <div className="flex flex-wrap items-center gap-1.5">
        <KindChip kind={bridge.platform} />
        <MonoBadge tone="neutral">{effectiveStatus}</MonoBadge>
        {health?.route_count !== undefined ? (
          <span className="ml-auto font-mono text-[10px] text-[color:var(--color-text-tertiary)]">
            {health.route_count} routes
          </span>
        ) : null}
      </div>
    </button>
  );
}

interface BridgeProviderGroup {
  extensionName: string;
  items: BridgeSummary[];
  label: string;
  platform: string;
}

function groupBridgesByProvider(bridges: BridgeSummary[]): BridgeProviderGroup[] {
  const byKey = new Map<string, BridgeProviderGroup>();
  for (const bridge of bridges) {
    const key = `${bridge.extension_name}::${bridge.platform}`;
    const existing = byKey.get(key);
    if (existing) {
      existing.items.push(bridge);
      continue;
    }
    byKey.set(key, {
      extensionName: bridge.extension_name,
      items: [bridge],
      label: bridge.platform,
      platform: bridge.platform,
    });
  }
  return Array.from(byKey.values()).sort((left, right) =>
    left.platform.localeCompare(right.platform)
  );
}

export function BridgeListPanel({
  bridgeHealth,
  bridges,
  errorMessage = null,
  isLoading = false,
  onSearchChange,
  onSelectBridge,
  searchQuery,
  selectedBridgeId,
  summary,
}: BridgeListPanelProps) {
  const isEmpty = bridges.length === 0;
  const groups = groupBridgesByProvider(bridges);

  return (
    <aside className="flex min-h-0 flex-1 flex-col" data-testid="bridge-list-panel">
      <div className="space-y-2 border-b border-[color:var(--color-divider)] p-3">
        <SearchInput
          data-testid="bridge-search-input"
          onChange={onSearchChange}
          placeholder="Search bridges…"
          value={searchQuery}
        />
        <p
          className="text-[12px] text-[color:var(--color-text-secondary)]"
          data-testid="bridge-list-summary"
        >
          {summary}
        </p>
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto">
        {isLoading && isEmpty ? (
          <div
            className="flex min-h-full items-center justify-center px-6 py-10"
            data-testid="bridge-list-loading"
          >
            <Loader2
              aria-hidden="true"
              className="size-5 animate-spin text-[color:var(--color-text-tertiary)]"
            />
          </div>
        ) : errorMessage && isEmpty ? (
          <div
            className="flex min-h-full items-center justify-center p-4"
            data-testid="bridge-list-error"
          >
            <Empty
              className="max-w-sm"
              description={errorMessage}
              icon={AlertCircle}
              title="Unable to load bridges"
            />
          </div>
        ) : isEmpty ? (
          <div
            className="flex min-h-full items-center justify-center p-4"
            data-testid="bridge-list-empty"
          >
            <Empty
              className="max-w-sm"
              description={
                searchQuery.trim() !== ""
                  ? "Try a different search term or adjust the scope filter."
                  : "No bridges match the current filters."
              }
              icon={Waypoints}
              title="No bridges found"
            />
          </div>
        ) : (
          <div data-testid="bridge-list-groups">
            {groups.map(group => (
              <div
                data-testid={`bridge-list-group-${group.extensionName}-${group.platform}`}
                key={`${group.extensionName}::${group.platform}`}
              >
                <div
                  className="flex items-center justify-between gap-2 border-b border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)] px-4 py-2"
                  data-testid={`bridge-list-group-header-${group.extensionName}-${group.platform}`}
                >
                  <span className="font-mono text-[10px] uppercase tracking-[0.14em] text-[color:var(--color-text-label)]">
                    {group.platform}
                  </span>
                  <MonoBadge>{group.items.length}</MonoBadge>
                </div>
                {group.items.map(bridge => (
                  <BridgeListItem
                    bridge={bridge}
                    health={bridgeHealth[bridge.id]}
                    isSelected={bridge.id === selectedBridgeId}
                    key={bridge.id}
                    onSelect={() => onSelectBridge(bridge.id)}
                  />
                ))}
              </div>
            ))}
          </div>
        )}
      </div>
    </aside>
  );
}
