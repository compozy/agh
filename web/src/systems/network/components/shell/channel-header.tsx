import { MoreHorizontal } from "lucide-react";

import { Button } from "@agh/ui";

import type { NetworkChannel, NetworkChannelSummary } from "../../types";
import { ChannelTabs, type ChannelTab } from "./channel-tabs";

export interface ChannelHeaderProps {
  channel: NetworkChannelSummary;
  detail: NetworkChannel | null;
  activeTab: ChannelTab;
  threadCount: number | null;
  directCount: number | null;
  openWorkCount: number;
}

function buildIdentityMixLabel(
  channel: NetworkChannelSummary,
  detail: NetworkChannel | null
): string {
  const localCount = detail?.local_peer_count ?? channel.local_peer_count ?? 0;
  const remoteCount = detail?.remote_peer_count ?? channel.remote_peer_count ?? 0;
  if (localCount === 0 && remoteCount === 0) {
    return "no peers yet";
  }

  const parts: string[] = [];
  if (localCount > 0) {
    parts.push(`${localCount} local`);
  }
  if (remoteCount > 0) {
    parts.push(`${remoteCount} remote`);
  }
  return parts.join(" · ");
}

export function ChannelHeader({
  channel,
  detail,
  activeTab,
  threadCount,
  directCount,
  openWorkCount,
}: ChannelHeaderProps) {
  const identityLabel = buildIdentityMixLabel(channel, detail);

  return (
    <header className="flex flex-col" data-testid="network-channel-header">
      <div className="flex items-center gap-3 border-b border-[color:var(--color-divider)] px-5 py-3">
        <h1 className="truncate text-[18px] font-semibold text-[color:var(--color-text-primary)]">
          #{channel.channel}
        </h1>
        <span
          className="font-mono text-[10px] uppercase tracking-[0.06em] text-[color:var(--color-text-tertiary)]"
          data-testid="network-channel-identity-mix"
        >
          {identityLabel}
        </span>
        {openWorkCount > 0 ? (
          <span
            className="font-mono text-[10px] uppercase tracking-[0.06em] text-[color:var(--color-text-tertiary)]"
            data-testid="network-channel-active-work"
          >
            · {openWorkCount} active work
          </span>
        ) : null}
        <div className="ml-auto">
          <Button
            aria-label="Channel actions"
            data-testid="network-channel-kebab"
            size="icon-sm"
            type="button"
            variant="outline"
          >
            <MoreHorizontal aria-hidden="true" className="size-4" />
          </Button>
        </div>
      </div>

      <ChannelTabs
        activeTab={activeTab}
        channel={channel.channel}
        directCount={directCount}
        threadCount={threadCount}
      />
    </header>
  );
}
