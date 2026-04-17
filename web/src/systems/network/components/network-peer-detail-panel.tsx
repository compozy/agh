import { AlertCircle, Hash, Link2, Loader2, Network, Radio, Workflow } from "lucide-react";
import { Link } from "@tanstack/react-router";
import type { ReactNode } from "react";

import { Pill } from "@/components/design-system";

import {
  formatNetworkDateTime,
  formatNetworkNumber,
  getPeerDeliveredRate,
  getPeerDisplayName,
  getPeerHeartbeatLabel,
  getPeerTypeLabel,
} from "../lib/network-formatters";
import type { NetworkPeerDetail } from "../types";

interface NetworkPeerDetailPanelProps {
  error: Error | null;
  isLoading: boolean;
  peer: NetworkPeerDetail | undefined;
}

function DetailRow({ icon, label, value }: { icon: ReactNode; label: string; value: ReactNode }) {
  return (
    <div className="flex items-center gap-3 border-b border-[color:var(--color-divider)] px-4 py-3 last:border-b-0">
      <span className="flex size-4 shrink-0 items-center justify-center text-[color:var(--color-text-tertiary)]">
        {icon}
      </span>
      <span className="w-28 shrink-0 font-mono text-[0.66rem] uppercase tracking-[0.14em] text-[color:var(--color-text-label)]">
        {label}
      </span>
      <div className="min-w-0 flex-1 text-sm text-[color:var(--color-text-primary)]">{value}</div>
    </div>
  );
}

function PeerMetric({ detail, label, value }: { detail: string; label: string; value: string }) {
  const metricSlug = label.toLowerCase().replaceAll(" ", "-");

  return (
    <div
      className="rounded-xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] p-4"
      data-testid={`network-peer-metric-${metricSlug}`}
    >
      <p className="font-mono text-[0.66rem] uppercase tracking-[0.16em] text-[color:var(--color-text-label)]">
        {label}
      </p>
      <p className="mt-3 text-3xl font-semibold tracking-[-0.04em] text-[color:var(--color-text-primary)]">
        {value}
      </p>
      <p className="mt-1 text-sm text-[color:var(--color-text-secondary)]">{detail}</p>
    </div>
  );
}

export function NetworkPeerDetailPanel({ error, isLoading, peer }: NetworkPeerDetailPanelProps) {
  if (isLoading) {
    return (
      <div className="flex flex-1 items-center justify-center" data-testid="network-peer-loading">
        <Loader2 className="size-5 animate-spin text-[color:var(--color-text-tertiary)]" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex flex-1 items-center justify-center" data-testid="network-peer-error">
        <div className="flex flex-col items-center gap-2 text-center">
          <AlertCircle className="size-6 text-[color:var(--color-danger)]" />
          <p className="text-sm text-[color:var(--color-text-tertiary)]">
            {error.message ?? "Failed to load peer details"}
          </p>
        </div>
      </div>
    );
  }

  if (!peer) {
    return (
      <div className="flex flex-1 items-center justify-center" data-testid="network-peer-empty">
        <p className="max-w-md text-center text-sm leading-relaxed text-[color:var(--color-text-tertiary)]">
          Select a peer to inspect identity, current channel, and message metrics.
        </p>
      </div>
    );
  }

  const displayName = getPeerDisplayName(peer);
  const typeLabel = getPeerTypeLabel({ local: peer.local ?? false });

  return (
    <section
      className="flex min-h-0 flex-1 flex-col overflow-y-auto p-6"
      data-testid="network-peer-detail-panel"
    >
      <div className="space-y-5">
        <section className="border-b border-[color:var(--color-divider)] pb-5">
          <div className="flex flex-wrap items-center gap-3">
            <div className="flex size-8 items-center justify-center rounded-lg border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] text-[color:var(--color-text-secondary)]">
              <Network className="size-4" />
            </div>
            <h2 className="text-xl font-semibold text-[color:var(--color-text-primary)]">
              {displayName}
            </h2>
            <Pill emphasis="strong" kind="state" tone={peer.local ? "amber" : "neutral"}>
              {typeLabel}
            </Pill>
            {peer.session_id ? (
              <Link
                className="ml-auto inline-flex h-8 items-center rounded-lg border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] px-3 text-sm font-medium text-[color:var(--color-text-primary)] transition-colors hover:bg-[color:var(--color-surface-panel)]"
                params={{ id: peer.session_id }}
                to="/session/$id"
              >
                View Session
              </Link>
            ) : null}
          </div>
          <p className="mt-3 text-sm text-[color:var(--color-text-secondary)]">
            {getPeerHeartbeatLabel(peer)}
          </p>
        </section>

        <section className="space-y-3">
          <p className="font-mono text-[0.66rem] uppercase tracking-[0.16em] text-[color:var(--color-text-label)]">
            Peer Identity
          </p>
          <div className="overflow-hidden rounded-xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface)]">
            <DetailRow icon={<Hash className="size-3.5" />} label="peer id" value={peer.peer_id} />
            <DetailRow
              icon={<Radio className="size-3.5" />}
              label="display name"
              value={displayName}
            />
            <DetailRow icon={<Workflow className="size-3.5" />} label="type" value={typeLabel} />
            <DetailRow
              icon={<Link2 className="size-3.5" />}
              label="session"
              value={peer.session_id ?? "No local session bound"}
            />
          </div>
        </section>

        <section className="space-y-3">
          <div className="flex items-center gap-2">
            <p className="font-mono text-[0.66rem] uppercase tracking-[0.16em] text-[color:var(--color-text-label)]">
              Channel
            </p>
            <Pill emphasis="strong" kind="state" tone="neutral">
              {peer.channel ? "1" : "0"}
            </Pill>
          </div>
          {peer.channel ? (
            <div className="inline-flex w-fit items-center gap-2 rounded-xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] px-4 py-2 text-sm text-[color:var(--color-text-primary)]">
              <Hash className="size-4 text-[color:var(--color-text-tertiary)]" />
              <span>{peer.channel}</span>
            </div>
          ) : (
            <div className="rounded-xl border border-dashed border-[color:var(--color-divider)] bg-[color:var(--color-surface)] px-4 py-4 text-sm text-[color:var(--color-text-secondary)]">
              This peer is visible but did not report an active channel membership.
            </div>
          )}
        </section>

        <section className="space-y-3">
          <p className="font-mono text-[0.66rem] uppercase tracking-[0.16em] text-[color:var(--color-text-label)]">
            Message Statistics
          </p>
          <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
            <PeerMetric
              detail="total"
              label="sent"
              value={formatNetworkNumber(peer.metrics.sent ?? 0)}
            />
            <PeerMetric
              detail="total"
              label="received"
              value={formatNetworkNumber(peer.metrics.received ?? 0)}
            />
            <PeerMetric
              detail="all time"
              label="rejected"
              value={formatNetworkNumber(peer.metrics.rejected ?? 0)}
            />
            <PeerMetric
              detail={getPeerDeliveredRate(peer)}
              label="delivered"
              value={formatNetworkNumber(peer.metrics.delivered ?? 0)}
            />
          </div>
        </section>

        <p className="text-xs text-[color:var(--color-text-tertiary)]">
          Last updated: {formatNetworkDateTime(peer.last_seen)}
        </p>
      </div>
    </section>
  );
}
