import { AlertCircle, Loader2, Users } from "lucide-react";

import { Empty, MonoBadge, SearchInput, StatusDot, type StatusDotTone } from "@agh/ui";
import { cn } from "@/lib/utils";

import {
  formatNetworkRelativeTime,
  getPeerDisplayName,
  getPeerTypeLabel,
} from "../lib/network-formatters";
import type { NetworkPeerSummary } from "../types";

interface NetworkPeersListPanelProps {
  errorMessage?: string | null;
  isLoading?: boolean;
  onSearchChange: (query: string) => void;
  onSelectPeer: (peerId: string) => void;
  peers: NetworkPeerSummary[];
  searchQuery: string;
  selectedPeerId: string | null;
}

interface PeerRowProps {
  isSelected: boolean;
  onSelect: () => void;
  peer: NetworkPeerSummary;
}

function resolvePeerStatusTone(peer: NetworkPeerSummary): StatusDotTone {
  if (peer.local) {
    return "accent";
  }

  if (!peer.last_seen) {
    return "neutral";
  }

  const parsed = new Date(peer.last_seen);
  if (Number.isNaN(parsed.getTime())) {
    return "neutral";
  }

  return Date.now() - parsed.getTime() <= 60_000 ? "success" : "neutral";
}

function PeerRow({ isSelected, onSelect, peer }: PeerRowProps) {
  const displayName = getPeerDisplayName(peer);
  const trailingLabel = peer.local
    ? getPeerTypeLabel(peer)
    : formatNetworkRelativeTime(peer.last_seen);
  const trailingTone: "accent" | "default" = peer.local ? "accent" : "default";

  return (
    <button
      aria-pressed={isSelected}
      className={cn(
        "relative flex w-full items-center gap-2 border-b border-[color:var(--color-divider)] px-4 py-2.5 text-left transition-colors",
        "hover:bg-[color:var(--color-hover)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[color:var(--color-accent)]",
        isSelected && "bg-[color:var(--color-surface)]"
      )}
      data-state={isSelected ? "selected" : undefined}
      data-testid={`network-peer-item-${peer.peer_id}`}
      onClick={onSelect}
      type="button"
    >
      {isSelected ? (
        <span
          aria-hidden="true"
          className="absolute left-0 top-1 bottom-1 w-[3px] rounded-r bg-[color:var(--color-accent)]"
        />
      ) : null}
      <StatusDot size="md" tone={resolvePeerStatusTone(peer)} />
      <span className="min-w-0 flex-1 truncate text-[13px] font-medium text-[color:var(--color-text-primary)]">
        {displayName}
      </span>
      <MonoBadge className="shrink-0" tone={trailingTone}>
        {trailingLabel}
      </MonoBadge>
    </button>
  );
}

export function NetworkPeersListPanel({
  errorMessage = null,
  isLoading = false,
  onSearchChange,
  onSelectPeer,
  peers,
  searchQuery,
  selectedPeerId,
}: NetworkPeersListPanelProps) {
  const isEmpty = peers.length === 0;

  return (
    <aside className="flex min-h-0 flex-1 flex-col" data-testid="network-peers-list-panel">
      <div className="border-b border-[color:var(--color-divider)] p-3">
        <SearchInput
          data-testid="network-peer-search-input"
          onChange={onSearchChange}
          placeholder="Search peers…"
          value={searchQuery}
        />
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto">
        {isLoading && isEmpty ? (
          <div
            className="flex min-h-full items-center justify-center px-6 py-10"
            data-testid="network-peers-list-loading"
          >
            <Loader2
              aria-hidden="true"
              className="size-5 animate-spin text-[color:var(--color-text-tertiary)]"
            />
          </div>
        ) : errorMessage && isEmpty ? (
          <div
            className="flex min-h-full items-center justify-center p-4"
            data-testid="network-peers-list-error"
          >
            <Empty
              className="max-w-sm"
              icon={AlertCircle}
              title="Unable to load peers"
              description={errorMessage}
            />
          </div>
        ) : isEmpty && searchQuery !== "" ? (
          <div
            className="flex min-h-full items-center justify-center p-4"
            data-testid="network-peers-list-empty"
          >
            <Empty
              className="max-w-sm"
              icon={Users}
              title="No peers found"
              description="Try another search term to find a visible network peer."
            />
          </div>
        ) : isEmpty ? (
          <div
            className="flex min-h-full items-center justify-center p-4"
            data-testid="network-peers-list-empty"
          >
            <Empty
              className="max-w-sm"
              icon={Users}
              title="No peers connected"
              description="Peers are discovered automatically when agents join the network."
            />
          </div>
        ) : (
          peers.map(peer => (
            <PeerRow
              isSelected={peer.peer_id === selectedPeerId}
              key={peer.peer_id}
              onSelect={() => onSelectPeer(peer.peer_id)}
              peer={peer}
            />
          ))
        )}
      </div>
    </aside>
  );
}
