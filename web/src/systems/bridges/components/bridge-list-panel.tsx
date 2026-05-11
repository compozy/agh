import { AlertCircle, Waypoints } from "lucide-react";

import {
  BlockLoading,
  Empty,
  Eyebrow,
  Item,
  ItemFooter,
  ItemHeader,
  ItemTitle,
  KindChip,
  ListGroup,
  Pill,
  SearchInput,
} from "@agh/ui";

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

function BridgeListItem({ bridge, health, isSelected, onSelect }: BridgeListItemProps) {
  const tone = bridgeStatusTone(bridge.status);
  const pulse = bridge.status === "starting";
  const effectiveStatus = health?.status ?? bridge.status;

  return (
    <Item
      as="button"
      className={cn("rounded-none border-x-0 border-t-0 border-b border-(--line) px-4 py-3")}
      data-testid={`bridge-item-${bridge.id}`}
      indicator={isSelected ? "rail" : "none"}
      onClick={onSelect}
      selected={isSelected}
      selectable
    >
      <ItemHeader>
        <ItemTitle>
          <Pill.Dot pulse={pulse} tone={tone} />
          <span className="truncate text-small-body font-medium text-(--fg)">
            {bridge.display_name}
          </span>
        </ItemTitle>
        <Eyebrow className="shrink-0">{formatBridgeRelativeTime(health?.last_success_at)}</Eyebrow>
      </ItemHeader>

      <ItemFooter className="flex-wrap justify-start gap-1.5">
        <KindChip kind={bridge.platform} />
        <Pill mono tone="neutral">
          {effectiveStatus}
        </Pill>
        {health?.route_count !== undefined ? (
          <span className="ml-auto font-mono text-badge text-(--subtle)">
            {health.route_count} routes
          </span>
        ) : null}
      </ItemFooter>
    </Item>
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
      <div className="space-y-2 border-b border-(--line) p-3">
        <SearchInput
          data-testid="bridge-search-input"
          onChange={onSearchChange}
          placeholder="Search bridges..."
          value={searchQuery}
        />
        <p className="text-xs text-(--muted)" data-testid="bridge-list-summary">
          {summary}
        </p>
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto">
        {isLoading && isEmpty ? (
          <BlockLoading
            className="min-h-full rounded-none border-0"
            data-testid="bridge-list-loading"
            label="Loading bridges"
            surface="bare"
          />
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
              <ListGroup
                count={group.items.length}
                data-testid={`bridge-list-group-${group.extensionName}-${group.platform}`}
                headerProps={{
                  "data-testid": `bridge-list-group-header-${group.extensionName}-${group.platform}`,
                }}
                key={`${group.extensionName}::${group.platform}`}
                label={group.platform}
              >
                {group.items.map(bridge => (
                  <BridgeListItem
                    bridge={bridge}
                    health={bridgeHealth[bridge.id]}
                    isSelected={bridge.id === selectedBridgeId}
                    key={bridge.id}
                    onSelect={() => onSelectBridge(bridge.id)}
                  />
                ))}
              </ListGroup>
            ))}
          </div>
        )}
      </div>
    </aside>
  );
}
