import { useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { Hash, MoreHorizontal, PanelRight, RefreshCw } from "lucide-react";

import {
  Button,
  DetailHeader,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@agh/ui";

import { cn } from "@/lib/utils";

import { useChannelMembers } from "../../hooks/use-channel-members";
import { networkKeys } from "../../lib/query-keys";
import type { NetworkChannel, NetworkChannelSummary } from "../../types";

export interface ChannelHeaderProps {
  channel: NetworkChannelSummary;
  detail: NetworkChannel | null;
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

  const meta = (
    <span className="truncate" data-testid="network-channel-meta">
      {metaSegments.map((segment, index) => (
        <span key={segment}>
          {index > 0 ? (
            <span aria-hidden="true" className="mx-2 text-subtle">
              /
            </span>
          ) : null}
          <span data-testid={`network-channel-meta-${segment}`}>{segment}</span>
        </span>
      ))}
    </span>
  );

  const actions = (
    <>
      <Button
        aria-label={inspectorOpen ? "Close channel inspector" : "Open channel inspector"}
        aria-pressed={inspectorOpen}
        className={cn(inspectorOpen ? "bg-elevated text-fg" : null)}
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
    </>
  );

  return (
    <header data-testid="network-channel-header">
      <DetailHeader
        actions={actions}
        className="px-5 py-3"
        meta={meta}
        title={
          <span className="flex min-w-0 items-center gap-2">
            <span
              aria-hidden="true"
              data-slot="page-header-icon"
              className="inline-flex size-6 shrink-0 items-center justify-center rounded-sm bg-elevated text-accent"
            >
              <Hash className="size-3.5" />
            </span>
            <span className="truncate" data-testid="network-channel-title">
              {channel.channel}
            </span>
          </span>
        }
      />
    </header>
  );
}
