import { AlertCircle, Loader2, Network } from "lucide-react";
import { Link } from "@tanstack/react-router";

import {
  Empty,
  Metric,
  MonoBadge,
  Pill,
  Section,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@agh/ui";

import {
  buildPeerCapabilityViews,
  formatNetworkDateTime,
  formatNetworkNumber,
  getPeerDeliveredRate,
  getPeerDisplayName,
  getPeerHeartbeatLabel,
  getPeerTypeLabel,
  hasCapabilityDetail,
} from "../lib/network-formatters";
import type { NetworkPeerCapabilityView, NetworkPeerDetail } from "../types";

interface NetworkPeerDetailPanelProps {
  error: Error | null;
  isLoading: boolean;
  peer: NetworkPeerDetail | undefined;
}

function DetailStateFallback({ children, testId }: { children: React.ReactNode; testId: string }) {
  return (
    <div className="flex min-h-0 flex-1 items-center justify-center p-6" data-testid={testId}>
      {children}
    </div>
  );
}

interface PeerMetricProps {
  detail: string;
  label: string;
  value: string;
}

function PeerMetric({ detail, label, value }: PeerMetricProps) {
  const slug = label.toLowerCase().replaceAll(" ", "-");

  return (
    <Metric
      data-testid={`network-peer-metric-${slug}`}
      detail={detail}
      label={label}
      value={value}
    />
  );
}

interface CapabilityDetailListProps {
  items: readonly string[];
  label: string;
  slug: string;
  testIdPrefix: string;
}

function CapabilityDetailList({ items, label, slug, testIdPrefix }: CapabilityDetailListProps) {
  if (items.length === 0) {
    return null;
  }

  return (
    <div className="flex flex-col gap-1" data-testid={`${testIdPrefix}-${slug}`}>
      <span className="font-mono text-[10px] uppercase tracking-[0.14em] text-[color:var(--color-text-label)]">
        {label}
      </span>
      <ul className="flex list-disc flex-col gap-0.5 pl-4 text-[12.5px] text-[color:var(--color-text-secondary)]">
        {items.map((item, index) => (
          <li key={`${slug}-${index}`}>{item}</li>
        ))}
      </ul>
    </div>
  );
}

interface CapabilityBadgeRowProps {
  items: readonly string[];
  label: string;
  slug: string;
  testIdPrefix: string;
}

function CapabilityBadgeRow({ items, label, slug, testIdPrefix }: CapabilityBadgeRowProps) {
  if (items.length === 0) {
    return null;
  }

  return (
    <div className="flex flex-wrap items-center gap-1.5" data-testid={`${testIdPrefix}-${slug}`}>
      <span className="font-mono text-[10px] uppercase tracking-[0.14em] text-[color:var(--color-text-label)]">
        {label}
      </span>
      {items.map(item => (
        <MonoBadge key={`${slug}-${item}`}>{item}</MonoBadge>
      ))}
    </div>
  );
}

interface CapabilityRowProps {
  view: NetworkPeerCapabilityView;
}

function CapabilityRow({ view }: CapabilityRowProps) {
  const testIdRoot = `network-peer-capability-${view.id}`;
  const testIdDetail = `${testIdRoot}-detail`;
  const detail = view.detail;
  const hasDetail = detail !== null && hasCapabilityDetail(view);

  return (
    <li
      className="flex flex-col gap-2 rounded-[var(--radius-md)] border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] px-3 py-3"
      data-testid={testIdRoot}
    >
      <div className="flex flex-wrap items-center gap-2">
        <MonoBadge tone="accent">{view.id}</MonoBadge>
        {detail?.version ? (
          <MonoBadge data-testid={`${testIdRoot}-version`}>v{detail.version}</MonoBadge>
        ) : null}
        <span
          className="min-w-0 flex-1 text-[13px] text-[color:var(--color-text-primary)]"
          data-testid={`${testIdRoot}-summary`}
        >
          {view.summary || "No summary provided."}
        </span>
      </div>

      {hasDetail && detail ? (
        <div className="flex flex-col gap-2" data-testid={testIdDetail}>
          {detail.outcome ? (
            <p
              className="text-[12.5px] leading-snug text-[color:var(--color-text-secondary)]"
              data-testid={`${testIdRoot}-outcome`}
            >
              <span className="font-mono text-[10px] uppercase tracking-[0.14em] text-[color:var(--color-text-label)]">
                outcome
              </span>{" "}
              {detail.outcome}
            </p>
          ) : null}
          <CapabilityBadgeRow
            items={detail.requirements ?? []}
            label="requires"
            slug="requirements"
            testIdPrefix={testIdRoot}
          />
          <CapabilityBadgeRow
            items={detail.context_needed ?? []}
            label="context"
            slug="context"
            testIdPrefix={testIdRoot}
          />
          <CapabilityBadgeRow
            items={detail.artifacts_expected ?? []}
            label="artifacts"
            slug="artifacts"
            testIdPrefix={testIdRoot}
          />
          <CapabilityBadgeRow
            items={detail.constraints ?? []}
            label="constraints"
            slug="constraints"
            testIdPrefix={testIdRoot}
          />
          <CapabilityDetailList
            items={detail.execution_outline ?? []}
            label="execution outline"
            slug="execution-outline"
            testIdPrefix={testIdRoot}
          />
          <CapabilityDetailList
            items={detail.examples ?? []}
            label="examples"
            slug="examples"
            testIdPrefix={testIdRoot}
          />
        </div>
      ) : null}
    </li>
  );
}

export function NetworkPeerDetailPanel({ error, isLoading, peer }: NetworkPeerDetailPanelProps) {
  if (isLoading) {
    return (
      <DetailStateFallback testId="network-peer-loading">
        <Loader2
          aria-hidden="true"
          className="size-5 animate-spin text-[color:var(--color-text-tertiary)]"
        />
      </DetailStateFallback>
    );
  }

  if (error) {
    return (
      <DetailStateFallback testId="network-peer-error">
        <Empty
          className="max-w-md"
          icon={AlertCircle}
          title="Unable to load peer"
          description={error.message ?? "Failed to load peer details"}
        />
      </DetailStateFallback>
    );
  }

  if (!peer) {
    return (
      <DetailStateFallback testId="network-peer-empty">
        <Empty
          className="max-w-md"
          icon={Network}
          title="Select a peer"
          description="Inspect capabilities, joined channels, and delivery metrics for any visible peer."
        />
      </DetailStateFallback>
    );
  }

  const displayName = getPeerDisplayName(peer);
  const typeLabel = getPeerTypeLabel({ local: peer.local ?? false });
  const capabilityViews = buildPeerCapabilityViews(
    peer.peer_card?.capabilities,
    peer.capability_catalog
  );
  const hasRichCatalog = capabilityViews.some(hasCapabilityDetail);

  return (
    <section
      className="flex min-h-0 flex-1 flex-col overflow-hidden"
      data-testid="network-peer-detail-panel"
    >
      <header className="border-b border-[color:var(--color-divider)] px-6 py-4">
        <div className="flex flex-wrap items-center gap-3">
          <span
            aria-hidden="true"
            className="inline-flex size-8 items-center justify-center rounded-[var(--radius-md)] border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] text-[color:var(--color-text-secondary)]"
          >
            <Network className="size-4" />
          </span>
          <h2 className="text-[15px] font-semibold tracking-[-0.01em] text-[color:var(--color-text-primary)]">
            {displayName}
          </h2>
          <Pill variant={peer.local ? "accent" : "default"}>{typeLabel}</Pill>
          <MonoBadge>{peer.peer_id}</MonoBadge>
          {peer.session_id ? (
            <Link
              className="ml-auto inline-flex h-8 items-center rounded-[var(--radius-md)] border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] px-3 text-[13px] font-medium text-[color:var(--color-text-primary)] transition-colors hover:bg-[color:var(--color-hover)]"
              params={{ id: peer.session_id }}
              to="/session/$id"
            >
              View Session
            </Link>
          ) : null}
        </div>
        <p className="mt-2 text-[13px] text-[color:var(--color-text-secondary)]">
          {getPeerHeartbeatLabel(peer)}
        </p>
      </header>

      <div className="min-h-0 flex-1 space-y-6 overflow-y-auto px-6 py-5">
        <Section
          label="Capabilities"
          right={
            <div className="flex items-center gap-1.5">
              <MonoBadge tone={hasRichCatalog ? "accent" : "default"}>
                {hasRichCatalog ? "detailed" : "brief"}
              </MonoBadge>
              <MonoBadge>{capabilityViews.length}</MonoBadge>
            </div>
          }
        >
          {capabilityViews.length === 0 ? (
            <Empty
              icon={Network}
              title="No capabilities advertised"
              description="This peer does not advertise any runtime capabilities."
              fill={false}
            />
          ) : (
            <ul className="flex flex-col gap-2" data-testid="network-peer-capabilities">
              {capabilityViews.map(view => (
                <CapabilityRow key={view.id} view={view} />
              ))}
            </ul>
          )}
        </Section>

        <Section label="Channels" right={<MonoBadge>{peer.channel ? 1 : 0}</MonoBadge>}>
          {peer.channel ? (
            <div className="overflow-hidden rounded-[var(--radius-md)] border border-[color:var(--color-divider)] bg-[color:var(--color-surface)]">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="font-mono text-[10px] uppercase tracking-[0.14em] text-[color:var(--color-text-label)]">
                      Channel
                    </TableHead>
                    <TableHead className="font-mono text-[10px] uppercase tracking-[0.14em] text-[color:var(--color-text-label)]">
                      Joined
                    </TableHead>
                    <TableHead className="font-mono text-[10px] uppercase tracking-[0.14em] text-[color:var(--color-text-label)]">
                      Last seen
                    </TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  <TableRow data-testid={`network-peer-channel-${peer.channel}`}>
                    <TableCell className="font-mono text-[13px] text-[color:var(--color-text-primary)]">
                      {peer.channel}
                    </TableCell>
                    <TableCell className="text-[13px] text-[color:var(--color-text-secondary)]">
                      {formatNetworkDateTime(peer.joined_at)}
                    </TableCell>
                    <TableCell className="text-[13px] text-[color:var(--color-text-secondary)]">
                      {formatNetworkDateTime(peer.last_seen)}
                    </TableCell>
                  </TableRow>
                </TableBody>
              </Table>
            </div>
          ) : (
            <Empty
              icon={Network}
              title="No channel membership"
              description="This peer is visible but did not report an active channel membership."
              fill={false}
            />
          )}
        </Section>

        <Section label="Message Statistics">
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
        </Section>

        <p className="font-mono text-[11px] text-[color:var(--color-text-tertiary)]">
          Last updated · {formatNetworkDateTime(peer.last_seen)}
        </p>
      </div>
    </section>
  );
}
