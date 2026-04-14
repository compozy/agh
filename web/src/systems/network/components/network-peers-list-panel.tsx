import { Search } from "lucide-react";

import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";

import {
  formatNetworkRelativeTime,
  getPeerDisplayName,
  getPeerPresenceTone,
  getPeerTypeLabel,
} from "../lib/network-formatters";
import type { NetworkPeerSummary } from "../types";

interface NetworkPeersListPanelProps {
  onSearchChange: (query: string) => void;
  onSelectPeer: (peerId: string) => void;
  peers: NetworkPeerSummary[];
  searchQuery: string;
  selectedPeerId: string | null;
}

function PeerListItem({
  isSelected,
  onSelect,
  peer,
}: {
  isSelected: boolean;
  onSelect: () => void;
  peer: NetworkPeerSummary;
}) {
  const displayName = getPeerDisplayName(peer);
  const meta = peer.local ? getPeerTypeLabel(peer) : formatNetworkRelativeTime(peer.last_seen);

  return (
    <button
      className={cn(
        "relative flex w-full items-center gap-3 border-b border-[color:rgba(58,58,60,0.45)] px-4 py-2.5 text-left transition-colors",
        "hover:bg-[color:var(--color-surface)]",
        isSelected && "bg-[color:var(--color-surface)]"
      )}
      data-testid={`network-peer-item-${peer.peer_id}`}
      onClick={onSelect}
      type="button"
    >
      {isSelected ? (
        <span className="absolute left-0 top-1 bottom-1 w-[3px] rounded-r bg-[color:var(--color-accent)]" />
      ) : null}
      <span className={cn("size-2 shrink-0 rounded-full", getPeerPresenceTone(peer))} />
      <span className="min-w-0 flex-1 truncate text-sm font-medium text-[color:var(--color-text-primary)]">
        {displayName}
      </span>
      <span
        className={cn(
          "shrink-0 font-mono text-[0.64rem] uppercase tracking-[0.12em]",
          peer.local
            ? "text-[color:var(--color-accent)]"
            : "text-[color:var(--color-text-tertiary)]"
        )}
      >
        {meta}
      </span>
    </button>
  );
}

export function NetworkPeersListPanel({
  onSearchChange,
  onSelectPeer,
  peers,
  searchQuery,
  selectedPeerId,
}: NetworkPeersListPanelProps) {
  return (
    <aside
      className="flex w-[280px] shrink-0 flex-col border-r border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)]"
      data-testid="network-peers-list-panel"
    >
      <div className="border-b border-[color:var(--color-divider)] p-3">
        <div className="relative">
          <Search className="pointer-events-none absolute left-2.5 top-1/2 size-3.5 -translate-y-1/2 text-[color:var(--color-text-tertiary)]" />
          <Input
            className="pl-8"
            data-testid="network-peer-search-input"
            onChange={event => onSearchChange(event.target.value)}
            placeholder="Search peers..."
            value={searchQuery}
          />
        </div>
      </div>

      <div className="flex-1 overflow-y-auto">
        {peers.length === 0 ? (
          <div
            className="flex min-h-full items-center justify-center px-6 py-10 text-center text-sm text-[color:var(--color-text-secondary)]"
            data-testid="network-peers-list-empty"
          >
            No peers found
          </div>
        ) : (
          peers.map(peer => (
            <PeerListItem
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
