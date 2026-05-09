import { useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { Hash, MoreHorizontal, PanelRight, RefreshCw, Search } from "lucide-react";

import {
  Button,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
  PageHeader,
} from "@agh/ui";

import { cn } from "@/lib/utils";

import { useChannelMembers } from "../../hooks/use-channel-members";
import { networkKeys } from "../../lib/query-keys";
import type { NetworkChannel, NetworkChannelSummary } from "../../types";
import { ChannelTabs, type ChannelTab } from "./channel-tabs";

export interface ChannelHeaderProps {
  channel: NetworkChannelSummary;
  detail: NetworkChannel | null;
  activeTab: ChannelTab;
  threadCount: number | null;
  directCount: number | null;
  openWorkCount: number;
  inspectorOpen: boolean;
  onInspectorToggle: () => void;
}

interface MetaInputs {
  channel: NetworkChannelSummary;
  detail: NetworkChannel | null;
  openWorkCount: number;
  agentCount: number;
  humanCount: number;
}

function buildMetaSegments({
  channel,
  detail,
  openWorkCount,
  agentCount,
  humanCount,
}: MetaInputs): string[] {
  const segments: string[] = [];

  const totalCount = agentCount + humanCount;
  const fallbackPeerCount =
    detail?.peer_count ??
    channel.peer_count ??
    (channel.local_peer_count ?? 0) + (channel.remote_peer_count ?? 0);

  if (totalCount > 0) {
    if (agentCount > 0) {
      segments.push(`${agentCount} ${agentCount === 1 ? "agent" : "agents"}`);
    }
    if (humanCount > 0) {
      segments.push(`${humanCount} ${humanCount === 1 ? "human" : "humans"}`);
    }
  } else if (fallbackPeerCount > 0) {
    segments.push(`${fallbackPeerCount} ${fallbackPeerCount === 1 ? "peer" : "peers"}`);
  } else {
    segments.push("no peers yet");
  }

  if (openWorkCount > 0) {
    segments.push(`${openWorkCount} active work`);
  }

  const purpose = detail?.purpose?.trim() || channel.purpose?.trim();
  if (purpose) {
    segments.push(purpose);
  }

  return segments;
}

export function ChannelHeader({
  channel,
  detail,
  activeTab,
  threadCount,
  directCount,
  openWorkCount,
  inspectorOpen,
  onInspectorToggle,
}: ChannelHeaderProps) {
  const queryClient = useQueryClient();
  const [overflowOpen, setOverflowOpen] = useState(false);
  const members = useChannelMembers(channel.channel);
  const metaSegments = buildMetaSegments({
    channel,
    detail,
    openWorkCount,
    agentCount: members.agentCount,
    humanCount: members.humanCount,
  });

  const handleRefresh = () => {
    void queryClient.invalidateQueries({ queryKey: networkKeys.channelScope(channel.channel) });
    setOverflowOpen(false);
  };

  return (
    <header className="flex flex-col" data-testid="network-channel-header">
      <PageHeader
        className="px-5 py-3"
        icon={Hash}
        subtitle={
          <span className="truncate" data-testid="network-channel-meta">
            {metaSegments.map(segment => (
              <span key={segment}>
                {segment !== metaSegments[0] ? (
                  <span aria-hidden="true" className="mx-2 text-(--color-text-tertiary)">
                    /
                  </span>
                ) : null}
                <span data-testid={`network-channel-meta-${segment}`}>{segment}</span>
              </span>
            ))}
          </span>
        }
        title={
          <span className="truncate" data-testid="network-channel-title">
            {channel.channel}
          </span>
        }
        controls={
          <div className="ml-auto flex shrink-0 items-center gap-1.5">
            <Button
              aria-disabled="true"
              aria-label="Search channel - coming soon"
              data-testid="network-channel-search"
              onClick={event => event.preventDefault()}
              size="icon-sm"
              tabIndex={-1}
              title="Search · Coming soon"
              type="button"
              variant="ghost"
            >
              <Search aria-hidden="true" className="size-4" />
            </Button>

            <Button
              aria-label={inspectorOpen ? "Close channel inspector" : "Open channel inspector"}
              aria-pressed={inspectorOpen}
              className={cn(
                inspectorOpen ? "bg-(--color-surface-elevated) text-(--color-text-primary)" : null
              )}
              data-state={inspectorOpen ? "open" : "closed"}
              data-testid="network-channel-inspector-toggle"
              onClick={onInspectorToggle}
              size="icon-sm"
              type="button"
              variant="ghost"
            >
              <PanelRight aria-hidden="true" className="size-4" />
            </Button>

            <DropdownMenu onOpenChange={setOverflowOpen} open={overflowOpen}>
              <DropdownMenuTrigger
                render={
                  <Button
                    aria-label="Channel actions"
                    data-testid="network-channel-kebab"
                    size="icon-sm"
                    type="button"
                    variant="ghost"
                  />
                }
              >
                <MoreHorizontal aria-hidden="true" className="size-4" />
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem
                  data-testid="network-channel-refresh"
                  onSelect={event => {
                    event.preventDefault();
                    handleRefresh();
                  }}
                >
                  <RefreshCw aria-hidden="true" className="size-3.5" />
                  Refresh data
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        }
      />

      <ChannelTabs
        activeTab={activeTab}
        channel={channel.channel}
        directCount={directCount}
        threadCount={threadCount}
      />
    </header>
  );
}
