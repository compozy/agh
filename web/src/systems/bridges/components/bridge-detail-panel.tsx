import {
  AlertCircle,
  Loader2,
  Pencil,
  Power,
  RotateCw,
  SendHorizontal,
  Waypoints,
} from "lucide-react";

import {
  Button,
  CodeBlock,
  Empty,
  Field,
  FieldContent,
  FieldDescription,
  FieldTitle,
  Input,
  Metric,
  MonoBadge,
  type MonoBadgeTone,
  Section,
  StatusDot,
  type StatusDotTone,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@agh/ui";

import {
  bridgeStatusTone,
  describeBridgeDmPolicy,
  describeBridgeDeliveryDefaults,
  describeBridgeProviderConfigSchema,
  describeBridgeRouteTarget,
  describeBridgeRoutingPolicy,
  describeBridgeSecretSlot,
  formatBridgeProviderConfig,
  formatBridgeDateTime,
  formatBridgeRelativeTime,
} from "../lib/bridge-formatters";
import type {
  BridgeHealth,
  BridgeProvider,
  BridgeRoute,
  BridgeSecretBinding,
  BridgeStatus,
  BridgeSummary,
} from "../types";

interface BridgeDetailPanelProps {
  bridge: BridgeSummary | undefined;
  emptyMessage?: string;
  error: Error | null;
  health: BridgeHealth | undefined;
  isLifecyclePending?: boolean;
  isLoading: boolean;
  isRoutesLoading: boolean;
  isSecretBindingPending?: boolean;
  isSecretBindingsLoading?: boolean;
  onDeleteSecretBinding?: (bindingName: string) => void;
  onDisableBridge?: () => void;
  onEnableBridge?: () => void;
  onOpenEdit?: () => void;
  onOpenTestDelivery: () => void;
  onRestartBridge?: () => void;
  onSaveSecretBinding?: (bindingName: string) => void;
  onSecretDraftChange?: (bindingName: string, value: string) => void;
  provider?: BridgeProvider;
  restartRequired?: boolean;
  routes: BridgeRoute[];
  secretBindings?: BridgeSecretBinding[];
  secretInputValues?: Record<string, string>;
  workspaceName?: string | null;
}

interface BridgeMetrics {
  activeRoutes: string;
  eventsTotal: string;
  lastDelivery: string;
  successRate: string;
  successTone: "default" | "accent" | "success" | "warning" | "danger";
}

function statusToStatusDotTone(status: BridgeStatus): StatusDotTone {
  if (status === "disabled") return "danger";
  switch (bridgeStatusTone(status)) {
    case "green":
      return "success";
    case "amber":
      return "warning";
    case "danger":
      return "danger";
    case "violet":
      return "info";
    default:
      return "neutral";
  }
}

function statusToMonoBadgeTone(status: BridgeStatus): MonoBadgeTone {
  if (status === "disabled") return "danger";
  switch (bridgeStatusTone(status)) {
    case "green":
      return "success";
    case "amber":
      return "warning";
    case "danger":
      return "danger";
    case "violet":
      return "info";
    default:
      return "neutral";
  }
}

function computeBridgeMetrics(
  health: BridgeHealth | undefined,
  routes: BridgeRoute[]
): BridgeMetrics {
  const backlog = health?.delivery_backlog ?? 0;
  const failures = health?.delivery_failures_total ?? 0;
  const dropped = health?.delivery_dropped_total ?? 0;
  const active = health?.route_count ?? routes.length;
  const total = backlog + failures + dropped + active;
  const successLike = active;

  let successRate = "—";
  let successTone: BridgeMetrics["successTone"] = "default";
  if (total > 0) {
    const pct = (successLike / total) * 100;
    successRate = `${Math.round(pct)}%`;
    successTone = pct >= 90 ? "success" : pct >= 70 ? "default" : "warning";
  }

  return {
    activeRoutes: String(active),
    eventsTotal: String(total),
    lastDelivery: formatBridgeRelativeTime(health?.last_success_at),
    successRate,
    successTone,
  };
}

function BridgeStateFallback({ children, testId }: { children: React.ReactNode; testId: string }) {
  return (
    <div className="flex min-h-0 flex-1 items-center justify-center p-6" data-testid={testId}>
      {children}
    </div>
  );
}

function BridgeDetailFact({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="rounded-[var(--radius-md)] border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] px-4 py-3">
      <p className="font-mono text-[10px] uppercase tracking-[0.12em] text-[color:var(--color-text-label)]">
        {label}
      </p>
      <div className="mt-2 text-[13px] text-[color:var(--color-text-primary)]">{value}</div>
    </div>
  );
}

interface SecretSlotCardProps {
  binding?: BridgeSecretBinding;
  inputValue: string;
  isSecretBindingPending: boolean;
  onDelete?: (bindingName: string) => void;
  onDraftChange?: (bindingName: string, value: string) => void;
  onSave?: (bindingName: string) => void;
  slot: NonNullable<BridgeProvider["secret_slots"]>[number];
}

function SecretSlotCard({
  binding,
  inputValue,
  isSecretBindingPending,
  onDelete,
  onDraftChange,
  onSave,
  slot,
}: SecretSlotCardProps) {
  return (
    <article
      className="rounded-[var(--radius-md)] border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] px-4 py-3"
      data-testid={`bridge-secret-binding-${slot.name}`}
    >
      <div className="flex flex-wrap items-center gap-2">
        <span className="font-mono text-[11px] uppercase tracking-[0.12em] text-[color:var(--color-text-primary)]">
          {slot.name}
        </span>
        <MonoBadge tone={slot.required === false ? "neutral" : "warning"}>
          {slot.required === false ? "OPTIONAL" : "REQUIRED"}
        </MonoBadge>
        <MonoBadge tone={binding ? "success" : "neutral"}>
          {binding ? "BOUND" : "UNBOUND"}
        </MonoBadge>
      </div>
      <p className="mt-2 text-[12px] leading-relaxed text-[color:var(--color-text-secondary)]">
        {describeBridgeSecretSlot(slot)}
      </p>
      <div className="mt-3 grid gap-3 lg:grid-cols-[minmax(0,1fr)_auto] lg:items-end">
        <Field>
          <FieldContent>
            <FieldTitle>Environment variable</FieldTitle>
            <FieldDescription>
              The stock daemon resolves bridge secrets from `env:NAME` refs.
            </FieldDescription>
          </FieldContent>
          <Input
            data-testid={`bridge-secret-env-input-${slot.name}`}
            id={`bridge-secret-env-${slot.name}`}
            onChange={event => onDraftChange?.(slot.name, event.target.value)}
            placeholder="AGH_BRIDGE_TOKEN"
            value={inputValue}
          />
          {binding ? (
            <p className="text-[12px] text-[color:var(--color-text-secondary)]">
              Current ref: <span className="font-mono">{binding.vault_ref}</span>
            </p>
          ) : (
            <p className="text-[12px] text-[color:var(--color-text-tertiary)]">
              No secret binding stored.
            </p>
          )}
        </Field>
        <div className="flex flex-wrap items-center gap-2">
          <Button
            data-testid={`save-bridge-secret-${slot.name}`}
            disabled={!inputValue.trim() || isSecretBindingPending}
            onClick={() => onSave?.(slot.name)}
            size="sm"
            type="button"
          >
            Save
          </Button>
          <Button
            data-testid={`delete-bridge-secret-${slot.name}`}
            disabled={!binding || isSecretBindingPending}
            onClick={() => onDelete?.(slot.name)}
            size="sm"
            type="button"
            variant="outline"
          >
            Delete
          </Button>
        </div>
      </div>
    </article>
  );
}

function BridgeMetricsSection({
  health,
  routes,
}: {
  health: BridgeHealth | undefined;
  routes: BridgeRoute[];
}) {
  const metrics = computeBridgeMetrics(health, routes);

  return (
    <Section label="Delivery metrics">
      <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
        <Metric
          data-testid="bridge-metric-events-24h"
          label="Events (24h)"
          subtext="backlog + failures + active"
          value={metrics.eventsTotal}
        />
        <Metric
          data-testid="bridge-metric-success-rate"
          label="Success rate"
          subtext="active vs. backlog"
          tone={metrics.successTone}
          value={metrics.successRate}
        />
        <Metric
          data-testid="bridge-metric-last-delivery"
          label="Last delivery"
          subtext="most recent success"
          value={metrics.lastDelivery}
        />
        <Metric
          data-testid="bridge-metric-active-routes"
          label="Active routes"
          subtext="sessions mapped"
          tone="accent"
          value={metrics.activeRoutes}
        />
      </div>
    </Section>
  );
}

function BridgeEventStreamSection({
  isRoutesLoading,
  routes,
}: {
  isRoutesLoading: boolean;
  routes: BridgeRoute[];
}) {
  if (isRoutesLoading) {
    return (
      <Section label="Event stream">
        <div
          className="flex min-h-28 items-center justify-center rounded-[var(--radius-md)] border border-[color:var(--color-divider)] bg-[color:var(--color-surface)]"
          data-testid="bridge-routes-loading"
        >
          <Loader2
            aria-hidden="true"
            className="size-4 animate-spin text-[color:var(--color-text-tertiary)]"
          />
        </div>
      </Section>
    );
  }

  if (routes.length === 0) {
    return (
      <Section label="Event stream">
        <div data-testid="bridge-routes-empty">
          <Empty
            description="This bridge has not mapped any sessions yet. Use test delivery to resolve a target or wait for the first inbound route to be claimed."
            icon={Waypoints}
            title="No routes"
          />
        </div>
      </Section>
    );
  }

  return (
    <Section label="Event stream" right={<MonoBadge>{routes.length}</MonoBadge>}>
      <div
        className="overflow-hidden rounded-[var(--radius-md)] border border-[color:var(--color-divider)] bg-[color:var(--color-surface)]"
        data-testid="bridge-routes-table"
      >
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="font-mono text-[10px] uppercase tracking-[0.12em] text-[color:var(--color-text-label)]">
                Status
              </TableHead>
              <TableHead className="font-mono text-[10px] uppercase tracking-[0.12em] text-[color:var(--color-text-label)]">
                Agent
              </TableHead>
              <TableHead className="font-mono text-[10px] uppercase tracking-[0.12em] text-[color:var(--color-text-label)]">
                Target
              </TableHead>
              <TableHead className="font-mono text-[10px] uppercase tracking-[0.12em] text-[color:var(--color-text-label)]">
                Scope
              </TableHead>
              <TableHead className="font-mono text-[10px] uppercase tracking-[0.12em] text-[color:var(--color-text-label)]">
                Last activity
              </TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {routes.map(route => (
              <TableRow
                data-testid={`bridge-route-${route.session_id}`}
                key={`${route.session_id}:${route.routing_key_hash}`}
              >
                <TableCell>
                  <div className="flex items-center gap-2">
                    <StatusDot tone="success" />
                    <MonoBadge tone="success">ACTIVE</MonoBadge>
                  </div>
                </TableCell>
                <TableCell>
                  <div className="min-w-0">
                    <div className="text-[13px] text-[color:var(--color-text-primary)]">
                      {route.agent_name}
                    </div>
                    <div className="mt-1 break-all font-mono text-[11px] text-[color:var(--color-text-tertiary)]">
                      session {route.session_id}
                    </div>
                  </div>
                </TableCell>
                <TableCell className="font-mono text-[12px] text-[color:var(--color-text-secondary)]">
                  {describeBridgeRouteTarget(route)}
                </TableCell>
                <TableCell>
                  <MonoBadge tone={route.scope === "workspace" ? "info" : "neutral"}>
                    {route.scope}
                  </MonoBadge>
                </TableCell>
                <TableCell className="font-mono text-[12px] text-[color:var(--color-text-tertiary)]">
                  {formatBridgeRelativeTime(route.last_activity_at)}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    </Section>
  );
}

function BridgeConfigurationSection({
  bridge,
  workspaceName,
}: {
  bridge: BridgeSummary;
  workspaceName?: string | null;
}) {
  return (
    <Section label="Configuration">
      <div className="grid gap-3 lg:grid-cols-2">
        <BridgeDetailFact
          label="Provider"
          value={`${bridge.platform} / ${bridge.extension_name}`}
        />
        <BridgeDetailFact
          label="Workspace"
          value={
            bridge.scope === "workspace"
              ? (workspaceName ?? bridge.workspace_id ?? "Unavailable")
              : "Global scope"
          }
        />
        <BridgeDetailFact
          label="Routing policy"
          value={describeBridgeRoutingPolicy(bridge.routing_policy)}
        />
        <BridgeDetailFact
          label="Delivery defaults"
          value={describeBridgeDeliveryDefaults(bridge.delivery_defaults)}
        />
        <BridgeDetailFact label="DM policy" value={describeBridgeDmPolicy(bridge.dm_policy)} />
        <BridgeDetailFact label="Created" value={formatBridgeDateTime(bridge.created_at)} />
        <BridgeDetailFact label="Updated" value={formatBridgeDateTime(bridge.updated_at)} />
      </div>
    </Section>
  );
}

interface BridgeProviderRuntimeSectionProps {
  bindingsByName: Map<string, BridgeSecretBinding>;
  isSecretBindingPending: boolean;
  isSecretBindingsLoading: boolean;
  onDeleteSecretBinding?: (bindingName: string) => void;
  onSaveSecretBinding?: (bindingName: string) => void;
  onSecretDraftChange?: (bindingName: string, value: string) => void;
  provider?: BridgeProvider;
  providerConfig: string;
  secretInputValues: Record<string, string>;
}

function BridgeProviderRuntimeSection({
  bindingsByName,
  isSecretBindingPending,
  isSecretBindingsLoading,
  onDeleteSecretBinding,
  onSaveSecretBinding,
  onSecretDraftChange,
  provider,
  providerConfig,
  secretInputValues,
}: BridgeProviderRuntimeSectionProps) {
  return (
    <Section label="Provider runtime">
      <div className="grid gap-3 lg:grid-cols-2">
        <BridgeDetailFact
          label="Manifest schema"
          value={describeBridgeProviderConfigSchema(provider?.config_schema)}
        />
        <BridgeDetailFact
          label="Secret slots"
          value={provider?.secret_slots?.length ? provider.secret_slots.length : "None declared"}
        />
      </div>

      {provider?.description ? (
        <p className="mt-3 rounded-[var(--radius-md)] border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] px-4 py-3 text-[13px] leading-relaxed text-[color:var(--color-text-secondary)]">
          {provider.description}
        </p>
      ) : null}

      {provider?.secret_slots?.length ? (
        <div className="mt-3 space-y-3" data-testid="bridge-detail-secret-slots">
          {provider.secret_slots.map(slot => (
            <SecretSlotCard
              binding={bindingsByName.get(slot.name)}
              inputValue={secretInputValues[slot.name] ?? ""}
              isSecretBindingPending={isSecretBindingPending}
              key={slot.name}
              onDelete={onDeleteSecretBinding}
              onDraftChange={onSecretDraftChange}
              onSave={onSaveSecretBinding}
              slot={slot}
            />
          ))}
        </div>
      ) : isSecretBindingsLoading ? (
        <div className="mt-3 flex items-center gap-2 text-[13px] text-[color:var(--color-text-tertiary)]">
          <Loader2 aria-hidden="true" className="size-4 animate-spin" />
          <span>Loading secret bindings…</span>
        </div>
      ) : null}

      <div className="mt-3">
        <p className="mb-2 font-mono text-[10px] uppercase tracking-[0.12em] text-[color:var(--color-text-label)]">
          Provider config
        </p>
        {providerConfig ? (
          <CodeBlock
            code={providerConfig}
            copyable={false}
            data-testid="bridge-detail-provider-config"
            showPrompt={false}
          />
        ) : (
          <p className="rounded-[var(--radius-md)] border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] px-4 py-3 text-[13px] leading-relaxed text-[color:var(--color-text-secondary)]">
            No provider runtime config stored for this bridge.
          </p>
        )}
      </div>
    </Section>
  );
}

interface BridgeDetailHeaderProps {
  bridge: BridgeSummary;
  effectiveStatus: BridgeStatus;
  isLifecyclePending: boolean;
  onDisableBridge?: () => void;
  onEnableBridge?: () => void;
  onOpenEdit?: () => void;
  onRestartBridge?: () => void;
}

function BridgeDetailHeader({
  bridge,
  effectiveStatus,
  isLifecyclePending,
  onDisableBridge,
  onEnableBridge,
  onOpenEdit,
  onRestartBridge,
}: BridgeDetailHeaderProps) {
  const statusDotTone = statusToStatusDotTone(effectiveStatus);
  const statusBadgeTone = statusToMonoBadgeTone(effectiveStatus);
  const pulse = effectiveStatus === "starting";

  return (
    <header className="border-b border-[color:var(--color-divider)] px-6 py-4">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0 flex-1 space-y-2">
          <div className="flex flex-wrap items-center gap-2">
            <span
              aria-hidden="true"
              className="inline-flex size-8 items-center justify-center rounded-[var(--radius-md)] border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] text-[color:var(--color-accent)]"
            >
              <Waypoints className="size-4" />
            </span>
            <StatusDot pulse={pulse} tone={statusDotTone} />
            <h2 className="text-[15px] font-semibold tracking-[-0.01em] text-[color:var(--color-text-primary)]">
              {bridge.display_name}
            </h2>
            <MonoBadge tone={statusBadgeTone}>{effectiveStatus}</MonoBadge>
            <MonoBadge tone={bridge.scope === "workspace" ? "info" : "neutral"}>
              {bridge.scope}
            </MonoBadge>
          </div>
          <p className="text-[12px] text-[color:var(--color-text-secondary)]">
            {bridge.platform} · {bridge.extension_name}
          </p>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <Button
            data-testid="edit-bridge-btn"
            disabled={isLifecyclePending}
            onClick={onOpenEdit}
            size="sm"
            type="button"
            variant="outline"
          >
            <Pencil className="size-3.5" />
            Edit
          </Button>
          <Button
            data-testid="restart-bridge-btn"
            disabled={isLifecyclePending}
            onClick={onRestartBridge}
            size="sm"
            type="button"
            variant="outline"
          >
            <RotateCw className="size-3.5" />
            Restart
          </Button>
          {bridge.enabled ? (
            <Button
              data-testid="disable-bridge-btn"
              disabled={isLifecyclePending}
              onClick={onDisableBridge}
              size="sm"
              type="button"
              variant="outline"
            >
              <Power className="size-3.5" />
              Disable
            </Button>
          ) : (
            <Button
              data-testid="enable-bridge-btn"
              disabled={isLifecyclePending}
              onClick={onEnableBridge}
              size="sm"
              type="button"
            >
              <Power className="size-3.5" />
              Enable
            </Button>
          )}
        </div>
      </div>
    </header>
  );
}

export function BridgeDetailPanel({
  bridge,
  emptyMessage = "Select a bridge to inspect configuration, routes, and delivery health.",
  error,
  health,
  isLifecyclePending = false,
  isLoading,
  isRoutesLoading,
  isSecretBindingPending = false,
  isSecretBindingsLoading = false,
  onDeleteSecretBinding,
  onDisableBridge,
  onEnableBridge,
  onOpenEdit,
  onOpenTestDelivery,
  onRestartBridge,
  onSaveSecretBinding,
  onSecretDraftChange,
  provider,
  restartRequired = false,
  routes,
  secretBindings = [],
  secretInputValues = {},
  workspaceName,
}: BridgeDetailPanelProps) {
  if (isLoading) {
    return (
      <BridgeStateFallback testId="bridge-detail-loading">
        <Loader2
          aria-hidden="true"
          className="size-5 animate-spin text-[color:var(--color-text-tertiary)]"
        />
      </BridgeStateFallback>
    );
  }

  if (error) {
    return (
      <BridgeStateFallback testId="bridge-detail-error">
        <Empty
          className="max-w-md"
          description={error.message ?? "Failed to load bridge details"}
          icon={AlertCircle}
          title="Unable to load bridge"
        />
      </BridgeStateFallback>
    );
  }

  if (!bridge) {
    return (
      <BridgeStateFallback testId="bridge-detail-empty">
        <Empty
          className="max-w-md"
          description={emptyMessage}
          icon={Waypoints}
          title="Select a bridge"
        />
      </BridgeStateFallback>
    );
  }

  const providerConfig = formatBridgeProviderConfig(bridge.provider_config);
  const effectiveStatus = (health?.status ?? bridge.status) as BridgeStatus;
  const bindingsByName = new Map(secretBindings.map(binding => [binding.binding_name, binding]));
  const disabledBridge = effectiveStatus === "disabled";

  return (
    <section
      className="flex min-h-0 flex-1 flex-col overflow-hidden"
      data-testid="bridge-detail-panel"
    >
      <BridgeDetailHeader
        bridge={bridge}
        effectiveStatus={effectiveStatus}
        isLifecyclePending={isLifecyclePending}
        onDisableBridge={onDisableBridge}
        onEnableBridge={onEnableBridge}
        onOpenEdit={onOpenEdit}
        onRestartBridge={onRestartBridge}
      />

      <div className="min-h-0 flex-1 space-y-6 overflow-y-auto px-6 py-5">
        {restartRequired ? (
          <div
            className="rounded-[var(--radius-md)] border border-[color:var(--color-warning)]/40 bg-[color:var(--color-warning-tint)] px-4 py-3 text-[13px] text-[color:var(--color-warning)]"
            data-testid="bridge-restart-required"
          >
            Pending runtime changes require a restart or enable action before the provider picks
            them up.
          </div>
        ) : null}

        <BridgeMetricsSection health={health} routes={routes} />

        <BridgeConfigurationSection bridge={bridge} workspaceName={workspaceName} />

        <BridgeProviderRuntimeSection
          bindingsByName={bindingsByName}
          isSecretBindingPending={isSecretBindingPending}
          isSecretBindingsLoading={isSecretBindingsLoading}
          onDeleteSecretBinding={onDeleteSecretBinding}
          onSaveSecretBinding={onSaveSecretBinding}
          onSecretDraftChange={onSecretDraftChange}
          provider={provider}
          providerConfig={providerConfig}
          secretInputValues={secretInputValues}
        />

        <BridgeEventStreamSection isRoutesLoading={isRoutesLoading} routes={routes} />

        <div className="flex items-center justify-between gap-3 rounded-[var(--radius-md)] border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] px-5 py-4">
          <div className="space-y-1">
            <p className="font-mono text-[10px] uppercase tracking-[0.12em] text-[color:var(--color-text-label)]">
              Test delivery
            </p>
            <p className="text-[13px] text-[color:var(--color-text-secondary)]">
              Resolve the outbound target using bridge defaults plus any explicit target override.
            </p>
          </div>
          <Button
            data-testid="open-test-delivery-btn"
            disabled={disabledBridge}
            onClick={onOpenTestDelivery}
            size="sm"
            type="button"
            variant="outline"
          >
            <SendHorizontal className="size-3.5" />
            Send Test
          </Button>
        </div>
      </div>
    </section>
  );
}
