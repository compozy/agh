import { AlertCircle, Hash, Loader2 } from "lucide-react";

import { Empty, MonoBadge, SearchInput } from "@agh/ui";
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

interface ChannelRowProps {
  channel: NetworkChannelSummary;
  isSelected: boolean;
  onSelect: () => void;
}

function ChannelRow({ channel, isSelected, onSelect }: ChannelRowProps) {
  return (
    <button
      aria-pressed={isSelected}
      className={cn(
        "relative flex w-full items-center gap-2 border-b border-[color:var(--color-divider)] px-4 py-2.5 text-left transition-colors",
        "hover:bg-[color:var(--color-hover)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[color:var(--color-accent)]",
        isSelected && "bg-[color:var(--color-surface)]"
      )}
      data-state={isSelected ? "selected" : undefined}
      data-testid={`network-channel-item-${channel.channel}`}
      onClick={onSelect}
      type="button"
    >
      {isSelected ? (
        <span
          aria-hidden="true"
          className="absolute left-0 top-1 bottom-1 w-[3px] rounded-r bg-[color:var(--color-accent)]"
        />
      ) : null}
      <Hash
        aria-hidden="true"
        className="size-3.5 shrink-0 text-[color:var(--color-text-tertiary)]"
      />
      <span className="min-w-0 flex-1 truncate font-mono text-[13px] text-[color:var(--color-text-primary)]">
        {channel.channel}
      </span>
      <MonoBadge className="shrink-0 normal-case">
        {formatChannelPeerCount(channel.peer_count)}
      </MonoBadge>
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
    <aside className="flex min-h-0 flex-1 flex-col" data-testid="network-channels-list-panel">
      <div className="border-b border-[color:var(--color-divider)] p-3">
        <SearchInput
          data-testid="network-channel-search-input"
          onChange={onSearchChange}
          placeholder="Search channels…"
          value={searchQuery}
        />
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto">
        {isLoading && isEmpty ? (
          <div
            className="flex min-h-full items-center justify-center px-6 py-10"
            data-testid="network-channels-list-loading"
          >
            <Loader2
              aria-hidden="true"
              className="size-5 animate-spin text-[color:var(--color-text-tertiary)]"
            />
          </div>
        ) : errorMessage && isEmpty ? (
          <div
            className="flex min-h-full items-center justify-center p-4"
            data-testid="network-channels-list-error"
          >
            <Empty
              className="max-w-sm"
              icon={AlertCircle}
              title="Unable to load channels"
              description={errorMessage}
            />
          </div>
        ) : isEmpty && searchQuery !== "" ? (
          <div
            className="flex min-h-full items-center justify-center p-4"
            data-testid="network-channels-list-empty"
          >
            <Empty
              className="max-w-sm"
              icon={Hash}
              title="No channels found"
              description="Try another search term to find a materialized channel."
            />
          </div>
        ) : isEmpty ? (
          <div
            className="flex min-h-full items-center justify-center p-4"
            data-testid="network-channels-list-empty"
          >
            <Empty
              className="max-w-sm"
              icon={Hash}
              title="No channels yet"
              description="Create your first channel to enable agent-to-agent coordination."
            />
          </div>
        ) : (
          channels.map(channel => (
            <ChannelRow
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
