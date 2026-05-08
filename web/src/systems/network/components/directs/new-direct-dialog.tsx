import { useEffect, useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";

import {
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@agh/ui";

import { useResolveNetworkDirectRoom } from "../../hooks/use-network-actions";
import { networkPeersOptions } from "../../lib/query-options";
import type { NetworkPeerSummary } from "../../types";
import { getPeerDisplayName } from "../../lib/network-formatters";

export interface NewDirectDialogProps {
  open: boolean;
  onOpenChange: (next: boolean) => void;
  channel: string;
  /** Local peer id used as the requesting side of the resolve request. */
  selfPeerId?: string;
  /** Local session id used for the resolve request. */
  sessionId: string;
}

interface PickerProps {
  channel: string;
  selfPeerId?: string;
  onSelect: (peer: NetworkPeerSummary) => void;
  selectedPeerId: string | null;
  disabled: boolean;
}

function PeerPickerList({ channel, selfPeerId, onSelect, selectedPeerId, disabled }: PickerProps) {
  const peersQuery = useQuery(networkPeersOptions(channel));
  const candidates = useMemo(() => {
    const peers = peersQuery.data ?? [];
    return peers.filter(peer => peer.peer_id !== selfPeerId);
  }, [peersQuery.data, selfPeerId]);
  const isLoading = peersQuery.isLoading;

  if (isLoading) {
    return <p className="px-2 py-3 text-xs text-(--color-text-tertiary)">Loading peers…</p>;
  }

  if (candidates.length === 0) {
    return (
      <p
        className="px-2 py-3 text-xs text-(--color-text-tertiary)"
        data-testid="network-new-direct-no-peers"
      >
        No other peers in this channel yet.
      </p>
    );
  }

  return (
    <ul aria-label="Channel peers" className="flex flex-col gap-1" role="listbox">
      {candidates.map(peer => (
        <li key={peer.peer_id}>
          <button
            aria-selected={peer.peer_id === selectedPeerId ? "true" : "false"}
            className="flex w-full items-baseline justify-between rounded-chip px-2 py-2 text-left hover:bg-(--color-hover) focus-visible:bg-(--color-hover) focus-visible:outline-none disabled:opacity-50"
            data-testid={`network-new-direct-peer-${peer.peer_id}`}
            disabled={disabled}
            onClick={() => onSelect(peer)}
            role="option"
            type="button"
          >
            <span className="truncate text-small-body text-(--color-text-primary)">
              @{getPeerDisplayName(peer)}
            </span>
            <span className="font-mono text-badge uppercase tracking-mono text-(--color-text-tertiary)">
              {peer.peer_id}
            </span>
          </button>
        </li>
      ))}
    </ul>
  );
}

export function NewDirectDialog({
  open,
  onOpenChange,
  channel,
  selfPeerId,
  sessionId,
}: NewDirectDialogProps) {
  const navigate = useNavigate();
  const { resolveRoom, isResolving, error } = useResolveNetworkDirectRoom();
  const [selectedPeerId, setSelectedPeerId] = useState<string | null>(null);

  useEffect(() => {
    if (!open) {
      setSelectedPeerId(null);
    }
  }, [open]);

  const handleSelect = async (peer: NetworkPeerSummary) => {
    setSelectedPeerId(peer.peer_id);
    try {
      const direct = await resolveRoom({
        channel,
        body: {
          peer_id: peer.peer_id,
          session_id: sessionId,
        },
      });
      onOpenChange(false);
      void navigate({
        to: "/network/$channel/directs/$directId",
        params: { channel, directId: direct.direct_id },
      });
    } catch {
      // Error message is rendered inline below; the dialog stays open.
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md" data-testid="network-new-direct-dialog">
        <DialogHeader>
          <DialogTitle>New direct room</DialogTitle>
          <DialogDescription>
            Pick a peer in #{channel} to open a restricted bilateral conversation.
          </DialogDescription>
        </DialogHeader>

        <PeerPickerList
          channel={channel}
          disabled={isResolving}
          onSelect={handleSelect}
          selectedPeerId={selectedPeerId}
          selfPeerId={selfPeerId}
        />

        {error ? (
          <p
            className="text-xs text-(--color-danger)"
            data-testid="network-new-direct-error"
            role="alert"
          >
            {error.message}
          </p>
        ) : null}

        <DialogFooter>
          <Button onClick={() => onOpenChange(false)} type="button" variant="outline">
            Cancel
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
