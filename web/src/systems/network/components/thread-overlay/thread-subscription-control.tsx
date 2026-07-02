import { useQuery } from "@tanstack/react-query";
import { Bell, BellOff, ChevronDown } from "lucide-react";

import {
  Button,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
  Spinner,
} from "@agh/ui";

import {
  useDeleteNetworkSubscription,
  useUpsertNetworkSubscription,
} from "../../hooks/use-network-actions";
import { networkSubscriptionsOptions } from "../../lib/query-options";
import type { NetworkSubscriptionMode } from "../../types";

const MODES: ReadonlyArray<{
  value: NetworkSubscriptionMode;
  label: string;
}> = [
  { value: "full", label: "Full delivery" },
  { value: "digest", label: "Digest" },
  { value: "mute", label: "Mute" },
];

interface ThreadSubscriptionControlProps {
  workspaceId: string;
  channel: string;
  threadId: string;
  peerId?: string | null;
}

function modeLabel(mode?: string | null): string {
  switch (mode) {
    case "full":
      return "Full";
    case "digest":
      return "Digest";
    case "mute":
      return "Muted";
    default:
      return "Default";
  }
}

export function ThreadSubscriptionControl({
  workspaceId,
  channel,
  threadId,
  peerId,
}: ThreadSubscriptionControlProps) {
  const enabled = workspaceId !== "" && channel !== "" && threadId !== "" && Boolean(peerId);
  const subscriptions = useQuery(
    networkSubscriptionsOptions(
      workspaceId,
      channel,
      { peer_id: peerId ?? undefined, thread_id: threadId },
      enabled
    )
  );
  const upsert = useUpsertNetworkSubscription();
  const remove = useDeleteNetworkSubscription();
  const current = subscriptions.data?.[0] ?? null;
  const isBusy = upsert.isPending || remove.isPending || subscriptions.isFetching;
  const label = modeLabel(current?.mode);
  const Icon = current?.mode === "mute" ? BellOff : Bell;

  const setMode = (mode: NetworkSubscriptionMode) => {
    if (!peerId) {
      return;
    }
    void upsert.mutateAsync({
      workspaceId,
      channel,
      data: {
        peer_id: peerId,
        thread_id: threadId,
        mode,
      },
    });
  };

  const resetMode = () => {
    if (!peerId) {
      return;
    }
    void remove.mutateAsync({ workspaceId, channel, peerId, threadId });
  };

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <Button
            aria-label="Thread delivery mode"
            data-testid="network-thread-subscription-trigger"
            disabled={!enabled || isBusy}
            size="sm"
            type="button"
            variant="ghost"
          />
        }
      >
        {isBusy ? (
          <Spinner aria-hidden="true" className="size-3" />
        ) : (
          <Icon aria-hidden="true" className="size-3" />
        )}
        {label}
        <ChevronDown aria-hidden="true" className="size-3 text-subtle" />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        {MODES.map(mode => (
          <DropdownMenuItem
            data-testid={`network-thread-subscription-${mode.value}`}
            key={mode.value}
            onSelect={event => {
              event.preventDefault();
              setMode(mode.value);
            }}
          >
            {mode.label}
          </DropdownMenuItem>
        ))}
        <DropdownMenuItem
          data-testid="network-thread-subscription-default"
          onSelect={event => {
            event.preventDefault();
            resetMode();
          }}
        >
          Default routing
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
