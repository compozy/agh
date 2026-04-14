import { AlertCircle, Loader2, Search } from "lucide-react";

import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";

import { formatChannelPeerCount } from "../lib/network-formatters";
import type { NetworkChannelSummary } from "../types";

interface NetworkChannelsListPanelProps {
  channels: NetworkChannelSummary[];
  errorMessage?: string | null;
  isLoading?: boolean;
  onSearchChange: (query: string) => void;
  onSelectChannel: (channel: string) => void;
  searchQuery: string;
  selectedChannel: string | null;
}

interface ChannelListItemProps {
  channel: NetworkChannelSummary;
  isSelected: boolean;
  onSelect: () => void;
}

function ChannelListItem({ channel, isSelected, onSelect }: ChannelListItemProps) {
  return (
    <button
      className={cn(
        "relative flex w-full items-center gap-3 border-b border-[color:rgba(58,58,60,0.45)] px-4 py-2.5 text-left transition-colors",
        "hover:bg-[color:var(--color-surface)]",
        isSelected && "bg-[color:var(--color-surface)]"
      )}
      data-testid={`network-channel-item-${channel.channel}`}
      onClick={onSelect}
      type="button"
    >
      {isSelected ? (
        <span className="absolute left-0 top-1 bottom-1 w-[3px] rounded-r bg-[color:var(--color-accent)]" />
      ) : null}
      <span className="min-w-0 flex-1 truncate text-sm font-medium text-[color:var(--color-text-primary)]">
        {channel.channel}
      </span>
      <span className="shrink-0 text-xs text-[color:var(--color-text-secondary)]">
        {formatChannelPeerCount(channel.peer_count)}
      </span>
    </button>
  );
}

export function NetworkChannelsListPanel({
  channels,
  errorMessage = null,
  isLoading = false,
  onSearchChange,
  onSelectChannel,
  searchQuery,
  selectedChannel,
}: NetworkChannelsListPanelProps) {
  const isEmpty = channels.length === 0;

  return (
    <aside
      className="flex w-[280px] shrink-0 flex-col border-r border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)]"
      data-testid="network-channels-list-panel"
    >
      <div className="border-b border-[color:var(--color-divider)] p-3">
        <div className="relative">
          <Search className="pointer-events-none absolute left-2.5 top-1/2 size-3.5 -translate-y-1/2 text-[color:var(--color-text-tertiary)]" />
          <Input
            className="pl-8"
            data-testid="network-channel-search-input"
            onChange={event => onSearchChange(event.target.value)}
            placeholder="Search channels..."
            value={searchQuery}
          />
        </div>
      </div>

      <div className="flex-1 overflow-y-auto">
        {isLoading && isEmpty ? (
          <div
            className="flex min-h-full items-center justify-center px-6 py-10"
            data-testid="network-channels-list-loading"
          >
            <Loader2 className="size-5 animate-spin text-[color:var(--color-text-tertiary)]" />
          </div>
        ) : errorMessage && isEmpty ? (
          <div
            className="flex min-h-full items-center justify-center px-6 py-10"
            data-testid="network-channels-list-error"
          >
            <div className="flex max-w-xs flex-col items-center gap-2 text-center">
              <AlertCircle className="size-5 text-[color:var(--color-danger)]" />
              <p className="text-sm text-[color:var(--color-text-secondary)]">{errorMessage}</p>
            </div>
          </div>
        ) : isEmpty ? (
          <div
            className="flex min-h-full items-center justify-center px-6 py-10 text-center text-sm text-[color:var(--color-text-secondary)]"
            data-testid="network-channels-list-empty"
          >
            No channels found
          </div>
        ) : (
          channels.map(channel => (
            <ChannelListItem
              channel={channel}
              isSelected={channel.channel === selectedChannel}
              key={channel.channel}
              onSelect={() => onSelectChannel(channel.channel)}
            />
          ))
        )}
      </div>
    </aside>
  );
}
