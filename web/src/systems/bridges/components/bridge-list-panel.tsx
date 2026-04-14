import { Search } from "lucide-react";

import { Pill } from "@/components/design-system";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import {
  bridgeScopeTone,
  bridgeStatusTone,
  formatBridgeRelativeTime,
} from "@/systems/bridges/lib/bridge-formatters";
import type { BridgeHealthMap, BridgeSummary } from "@/systems/bridges/types";

interface BridgeListPanelProps {
  bridgeHealth: BridgeHealthMap;
  bridges: BridgeSummary[];
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
  return (
    <button
      className={cn(
        "relative flex w-full flex-col gap-2 border-b border-[color:rgba(58,58,60,0.45)] px-3 py-3 text-left transition-colors",
        "hover:bg-[color:var(--color-surface)]",
        isSelected && "bg-[color:var(--color-surface)]"
      )}
      data-testid={`bridge-item-${bridge.id}`}
      onClick={onSelect}
      type="button"
    >
      {isSelected ? (
        <span
          className="absolute left-0 top-1 bottom-1 w-[3px] rounded-r bg-[color:var(--color-accent)]"
          data-testid="bridge-active-indicator"
        />
      ) : null}

      <div className="flex items-start justify-between gap-3">
        <div className="space-y-1">
          <p className="text-sm font-medium text-[color:var(--color-text-primary)]">
            {bridge.display_name}
          </p>
          <p className="font-mono text-[0.64rem] uppercase tracking-[0.14em] text-[color:var(--color-text-label)]">
            {bridge.platform} / {bridge.extension_name}
          </p>
        </div>
        <span className="text-[0.72rem] text-[color:var(--color-text-tertiary)]">
          {formatBridgeRelativeTime(health?.last_success_at)}
        </span>
      </div>

      <div className="flex items-center justify-between gap-3 text-xs text-[color:var(--color-text-secondary)]">
        <span>{health?.route_count ?? 0} routes</span>
        <span>{health?.delivery_backlog ?? 0} backlog</span>
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <Pill emphasis="strong" kind="state" tone={bridgeStatusTone(bridge.status)}>
          {bridge.status}
        </Pill>
        <Pill kind="tag" tone={bridgeScopeTone(bridge.scope)}>
          {bridge.scope}
        </Pill>
      </div>
    </button>
  );
}

export function BridgeListPanel({
  bridgeHealth,
  bridges,
  onSearchChange,
  onSelectBridge,
  searchQuery,
  selectedBridgeId,
  summary,
}: BridgeListPanelProps) {
  return (
    <div
      className="flex w-[300px] shrink-0 flex-col border-r border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)]"
      data-testid="bridge-list-panel"
    >
      <div className="space-y-3 border-b border-[color:var(--color-divider)] p-3">
        <div className="relative">
          <Search className="pointer-events-none absolute left-2.5 top-1/2 size-3.5 -translate-y-1/2 text-[color:var(--color-text-tertiary)]" />
          <Input
            className="pl-8"
            data-testid="bridge-search-input"
            onChange={event => onSearchChange(event.target.value)}
            placeholder="Search bridges..."
            value={searchQuery}
          />
        </div>
        <p className="text-xs leading-relaxed text-[color:var(--color-text-tertiary)]">{summary}</p>
      </div>

      <div className="flex-1 overflow-y-auto">
        {bridges.length === 0 ? (
          <div
            className="px-4 py-8 text-center text-sm text-[color:var(--color-text-tertiary)]"
            data-testid="bridge-list-empty"
          >
            No bridges match the current filters.
          </div>
        ) : (
          bridges.map(bridge => (
            <BridgeListItem
              key={bridge.id}
              bridge={bridge}
              health={bridgeHealth[bridge.id]}
              isSelected={bridge.id === selectedBridgeId}
              onSelect={() => onSelectBridge(bridge.id)}
            />
          ))
        )}
      </div>
    </div>
  );
}
