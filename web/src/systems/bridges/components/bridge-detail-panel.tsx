import { AlertCircle, Loader2, SendHorizontal } from "lucide-react";

import { Pill } from "@/components/design-system";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

import {
  bridgeScopeTone,
  bridgeStatusTone,
  describeBridgeDeliveryDefaults,
  describeBridgeRouteTarget,
  describeBridgeRoutingPolicy,
  formatBridgeDateTime,
  formatBridgeRelativeTime,
} from "../lib/bridge-formatters";
import type { BridgeHealth, BridgeRoute, BridgeSummary } from "../types";

interface BridgeDetailPanelProps {
  bridge: BridgeSummary | undefined;
  emptyMessage?: string;
  error: Error | null;
  health: BridgeHealth | undefined;
  isLoading: boolean;
  isRoutesLoading: boolean;
  onOpenTestDelivery: () => void;
  routes: BridgeRoute[];
  workspaceName?: string | null;
}

function DetailSection({
  children,
  title,
  action,
}: {
  action?: React.ReactNode;
  children: React.ReactNode;
  title: string;
}) {
  return (
    <section className="rounded-xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] p-5">
      <div className="mb-4 flex items-center justify-between gap-3">
        <div>
          <p className="font-mono text-[0.68rem] uppercase tracking-[0.16em] text-[color:var(--color-text-label)]">
            {title}
          </p>
        </div>
        {action}
      </div>
      {children}
    </section>
  );
}

function DetailFact({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="rounded-lg border border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)] px-4 py-3">
      <p className="font-mono text-[0.64rem] uppercase tracking-[0.14em] text-[color:var(--color-text-label)]">
        {label}
      </p>
      <div className="mt-2 text-sm text-[color:var(--color-text-primary)]">{value}</div>
    </div>
  );
}

function MetricCard({
  label,
  toneClassName,
  value,
}: {
  label: string;
  toneClassName?: string;
  value: React.ReactNode;
}) {
  return (
    <div className="rounded-xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)] p-4">
      <p className="font-mono text-[0.64rem] uppercase tracking-[0.16em] text-[color:var(--color-text-label)]">
        {label}
      </p>
      <p
        className={cn(
          "mt-3 text-2xl font-semibold text-[color:var(--color-text-primary)]",
          toneClassName
        )}
      >
        {value}
      </p>
    </div>
  );
}

export function BridgeDetailPanel({
  bridge,
  emptyMessage = "Select a bridge to inspect configuration, routes, and delivery health.",
  error,
  health,
  isLoading,
  isRoutesLoading,
  onOpenTestDelivery,
  routes,
  workspaceName,
}: BridgeDetailPanelProps) {
  if (isLoading) {
    return (
      <div className="flex flex-1 items-center justify-center" data-testid="bridge-detail-loading">
        <Loader2 className="size-5 animate-spin text-[color:var(--color-text-tertiary)]" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex flex-1 items-center justify-center" data-testid="bridge-detail-error">
        <div className="flex flex-col items-center gap-2 text-center">
          <AlertCircle className="size-6 text-[color:var(--color-danger)]" />
          <p className="text-sm text-[color:var(--color-text-tertiary)]">
            {error.message ?? "Failed to load bridge details"}
          </p>
        </div>
      </div>
    );
  }

  if (!bridge) {
    return (
      <div className="flex flex-1 items-center justify-center" data-testid="bridge-detail-empty">
        <p className="max-w-md text-center text-sm leading-relaxed text-[color:var(--color-text-tertiary)]">
          {emptyMessage}
        </p>
      </div>
    );
  }

  return (
    <div className="flex flex-1 overflow-y-auto" data-testid="bridge-detail-panel">
      <div className="flex w-full flex-col gap-4 p-6">
        <div className="space-y-3">
          <div className="flex flex-wrap items-center gap-3">
            <h2 className="text-xl font-semibold text-[color:var(--color-text-primary)]">
              {bridge.display_name}
            </h2>
            <Pill emphasis="strong" kind="state" tone={bridgeStatusTone(bridge.status)}>
              {bridge.status}
            </Pill>
            <Pill kind="tag" tone={bridgeScopeTone(bridge.scope)}>
              {bridge.scope}
            </Pill>
          </div>
          <div className="flex flex-wrap items-center gap-3 text-sm text-[color:var(--color-text-secondary)]">
            <span>{bridge.platform}</span>
            <span className="text-[color:var(--color-text-tertiary)]">/</span>
            <span>{bridge.extension_name}</span>
            <span className="text-[color:var(--color-text-tertiary)]">/</span>
            <span>Last success {formatBridgeRelativeTime(health?.last_success_at)}</span>
          </div>
        </div>

        <DetailSection title="Configuration">
          <div className="grid gap-3 lg:grid-cols-2">
            <DetailFact label="Provider" value={`${bridge.platform} / ${bridge.extension_name}`} />
            <DetailFact
              label="Workspace"
              value={
                bridge.scope === "workspace"
                  ? (workspaceName ?? bridge.workspace_id ?? "Unavailable")
                  : "Global scope"
              }
            />
            <DetailFact
              label="Routing policy"
              value={describeBridgeRoutingPolicy(bridge.routing_policy)}
            />
            <DetailFact
              label="Delivery defaults"
              value={describeBridgeDeliveryDefaults(bridge.delivery_defaults)}
            />
            <DetailFact label="Created" value={formatBridgeDateTime(bridge.created_at)} />
            <DetailFact label="Updated" value={formatBridgeDateTime(bridge.updated_at)} />
          </div>
        </DetailSection>

        <DetailSection
          action={
            <Pill emphasis="strong" kind="state" tone="neutral">
              {routes.length}
            </Pill>
          }
          title="Routes"
        >
          {isRoutesLoading ? (
            <div className="flex items-center gap-2 text-sm text-[color:var(--color-text-tertiary)]">
              <Loader2 className="size-4 animate-spin" />
              <span>Loading routes…</span>
            </div>
          ) : routes.length === 0 ? (
            <div
              className="rounded-xl border border-dashed border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)] px-5 py-8 text-center"
              data-testid="bridge-routes-empty"
            >
              <p className="text-sm font-medium text-[color:var(--color-text-primary)]">
                No routes
              </p>
              <p className="mt-2 text-sm leading-relaxed text-[color:var(--color-text-secondary)]">
                This bridge has not mapped any sessions yet. Use test delivery to resolve a target
                or wait for the first inbound route to be claimed.
              </p>
            </div>
          ) : (
            <div className="space-y-3">
              {routes.map(route => (
                <article
                  key={`${route.session_id}:${route.routing_key_hash}`}
                  className="rounded-lg border border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)] px-4 py-3"
                  data-testid={`bridge-route-${route.session_id}`}
                >
                  <div className="flex flex-wrap items-start justify-between gap-3">
                    <div className="space-y-1">
                      <p className="text-sm font-medium text-[color:var(--color-text-primary)]">
                        {route.agent_name}
                      </p>
                      <p className="text-xs text-[color:var(--color-text-secondary)]">
                        {describeBridgeRouteTarget(route)}
                      </p>
                    </div>
                    <Pill kind="tag" tone={bridgeScopeTone(route.scope)}>
                      {route.scope}
                    </Pill>
                  </div>
                  <div className="mt-3 flex flex-wrap items-center gap-3 text-xs text-[color:var(--color-text-tertiary)]">
                    <span>Session {route.session_id}</span>
                    <span>Last activity {formatBridgeRelativeTime(route.last_activity_at)}</span>
                  </div>
                </article>
              ))}
            </div>
          )}
        </DetailSection>

        <DetailSection title="Delivery health">
          <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
            <MetricCard label="Backlog" value={health?.delivery_backlog ?? 0} />
            <MetricCard label="Dropped" value={health?.delivery_dropped_total ?? 0} />
            <MetricCard
              label="Failed"
              toneClassName="text-[color:var(--color-danger)]"
              value={health?.delivery_failures_total ?? 0}
            />
            <MetricCard
              label="Last success"
              value={formatBridgeRelativeTime(health?.last_success_at)}
            />
          </div>

          {health?.last_error ? (
            <div className="mt-4 rounded-lg border border-[color:var(--color-danger)] bg-[color:var(--color-danger-tint)] px-4 py-3 text-sm text-[color:var(--color-danger)]">
              {health.last_error}
            </div>
          ) : null}
        </DetailSection>

        <div className="flex items-center justify-between gap-3 rounded-xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] px-5 py-4">
          <div className="space-y-1">
            <p className="font-mono text-[0.68rem] uppercase tracking-[0.16em] text-[color:var(--color-text-label)]">
              Test delivery
            </p>
            <p className="text-sm text-[color:var(--color-text-secondary)]">
              Resolve the outbound target using bridge defaults plus any explicit target override.
            </p>
          </div>
          <Button
            className="border-[color:var(--color-accent)] bg-transparent text-[color:var(--color-accent)] hover:bg-[color:var(--color-accent-tint)] hover:text-[color:var(--color-accent)]"
            data-testid="open-test-delivery-btn"
            onClick={onOpenTestDelivery}
            size="lg"
            type="button"
            variant="outline"
          >
            <SendHorizontal className="size-4" />
            Test Delivery
          </Button>
        </div>
      </div>
    </div>
  );
}
