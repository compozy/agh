import {
  AlertCircle,
  Pencil,
  Power,
  RotateCw,
  SendHorizontal,
  Trash2,
  Waypoints,
} from "lucide-react";

import {
  Button,
  CodeBlock,
  ConfirmDialog,
  DataSurface,
  DialogTrigger,
  Empty,
  Eyebrow,
  Field,
  FieldContent,
  FieldDescription,
  FieldTitle,
  Input,
  MetadataList,
  Metric,
  PageHeader,
  Pill,
  type PillTone,
  Section,
  Spinner,
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

const METADATA_TILE_CLASS =
  "rounded-md border border-(--color-divider) bg-(--color-surface) px-4 py-3";
const METADATA_TERM_CLASS = "mb-2 text-(--color-text-label)";
const METADATA_VALUE_CLASS = "text-small-body text-(--color-text-primary)";

function statusToPillTone(status: BridgeStatus): PillTone {
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

  let successRate = "--";
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
      className="rounded-md border border-(--color-divider) bg-(--color-surface) px-4 py-3"
      data-testid={`bridge-secret-binding-${slot.name}`}
    >
      <div className="flex flex-wrap items-center gap-2">
        <Eyebrow tone="accent">{slot.name}</Eyebrow>
        <Pill mono tone={slot.required === false ? "neutral" : "warning"}>
          {slot.required === false ? "OPTIONAL" : "REQUIRED"}
        </Pill>
        <Pill mono tone={binding ? "success" : "neutral"}>
          {binding ? "BOUND" : "UNBOUND"}
        </Pill>
      </div>
      <p className="mt-2 text-xs leading-relaxed text-(--color-text-secondary)">
        {describeBridgeSecretSlot(slot)}
      </p>
      <div className="mt-3 grid gap-3 lg:grid-cols-[minmax(0,1fr)_auto] lg:items-end">
        <Field>
          <FieldContent>
            <FieldTitle>Secret value</FieldTitle>
            <FieldDescription>
              AGH stores bridge secret values in the vault for this bridge.
            </FieldDescription>
          </FieldContent>
          <Input
            data-testid={`bridge-secret-env-input-${slot.name}`}
            id={`bridge-secret-env-${slot.name}`}
            onChange={event => onDraftChange?.(slot.name, event.target.value)}
            placeholder="Paste secret value"
            type="password"
            value={inputValue}
          />
          {binding ? (
            <p className="text-xs text-(--color-text-secondary)">
              Current ref: <span className="font-mono">{binding.secret_ref}</span>
            </p>
          ) : (
            <p className="text-xs text-(--color-text-tertiary)">No secret binding stored.</p>
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
          <ConfirmDialog
            cancelButtonProps={{
              "data-testid": `cancel-delete-bridge-secret-${slot.name}`,
              disabled: isSecretBindingPending,
            }}
            cancelLabel="Cancel"
            confirmButtonProps={{
              "data-testid": `confirm-delete-bridge-secret-${slot.name}`,
            }}
            confirmIcon={Trash2}
            confirmLabel="Delete binding"
            description={`This removes the stored vault binding for ${slot.name}. The provider will not receive this secret until a replacement is saved.`}
            isPending={isSecretBindingPending}
            onConfirm={() => onDelete?.(slot.name)}
            title="Delete secret binding?"
            tone="danger"
          >
            <DialogTrigger
              render={
                <Button
                  data-testid={`delete-bridge-secret-${slot.name}`}
                  disabled={!binding || isSecretBindingPending}
                  size="sm"
                  type="button"
                  variant="outline"
                />
              }
            >
              Delete
            </DialogTrigger>
          </ConfirmDialog>
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
        <DataSurface.Loading
          data-testid="bridge-routes-loading"
          label="Loading bridge routes"
          size="sm"
        />
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
    <Section label="Event stream" right={<Pill mono>{routes.length}</Pill>}>
      <div
        className="overflow-hidden rounded-md border border-(--color-divider) bg-(--color-surface)"
        data-testid="bridge-routes-table"
      >
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>
                <Eyebrow>Status</Eyebrow>
              </TableHead>
              <TableHead>
                <Eyebrow>Agent</Eyebrow>
              </TableHead>
              <TableHead>
                <Eyebrow>Target</Eyebrow>
              </TableHead>
              <TableHead>
                <Eyebrow>Scope</Eyebrow>
              </TableHead>
              <TableHead>
                <Eyebrow>Last activity</Eyebrow>
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
                    <Pill.Dot tone="success" />
                    <Pill mono tone="success">
                      ACTIVE
                    </Pill>
                  </div>
                </TableCell>
                <TableCell>
                  <div className="min-w-0">
                    <div className="text-small-body text-(--color-text-primary)">
                      {route.agent_name}
                    </div>
                    <div className="mt-1 break-all font-mono text-eyebrow text-(--color-text-tertiary)">
                      <Eyebrow className="mr-1" weight="semibold">
                        Session
                      </Eyebrow>
                      <span>{route.session_id}</span>
                    </div>
                  </div>
                </TableCell>
                <TableCell className="font-mono text-xs text-(--color-text-secondary)">
                  {describeBridgeRouteTarget(route)}
                </TableCell>
                <TableCell>
                  <Pill mono tone={route.scope === "workspace" ? "info" : "neutral"}>
                    {route.scope}
                  </Pill>
                </TableCell>
                <TableCell className="font-mono text-xs text-(--color-text-tertiary)">
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
      <MetadataList className="grid gap-3 lg:grid-cols-2">
        <MetadataList.Row
          className={METADATA_TILE_CLASS}
          label="Provider"
          termProps={{ className: METADATA_TERM_CLASS }}
          valueProps={{ className: METADATA_VALUE_CLASS }}
        >
          {bridge.platform} / {bridge.extension_name}
        </MetadataList.Row>
        <MetadataList.Row
          className={METADATA_TILE_CLASS}
          label="Workspace"
          termProps={{ className: METADATA_TERM_CLASS }}
          valueProps={{ className: METADATA_VALUE_CLASS }}
        >
          {bridge.scope === "workspace"
            ? (workspaceName ?? bridge.workspace_id ?? "Unavailable")
            : "Global scope"}
        </MetadataList.Row>
        <MetadataList.Row
          className={METADATA_TILE_CLASS}
          label="Routing policy"
          termProps={{ className: METADATA_TERM_CLASS }}
          valueProps={{ className: METADATA_VALUE_CLASS }}
        >
          {describeBridgeRoutingPolicy(bridge.routing_policy)}
        </MetadataList.Row>
        <MetadataList.Row
          className={METADATA_TILE_CLASS}
          label="Delivery defaults"
          termProps={{ className: METADATA_TERM_CLASS }}
          valueProps={{ className: METADATA_VALUE_CLASS }}
        >
          {describeBridgeDeliveryDefaults(bridge.delivery_defaults)}
        </MetadataList.Row>
        <MetadataList.Row
          className={METADATA_TILE_CLASS}
          label="DM policy"
          termProps={{ className: METADATA_TERM_CLASS }}
          valueProps={{ className: METADATA_VALUE_CLASS }}
        >
          {describeBridgeDmPolicy(bridge.dm_policy)}
        </MetadataList.Row>
        <MetadataList.Row
          className={METADATA_TILE_CLASS}
          label="Created"
          termProps={{ className: METADATA_TERM_CLASS }}
          valueProps={{ className: METADATA_VALUE_CLASS }}
        >
          {formatBridgeDateTime(bridge.created_at)}
        </MetadataList.Row>
        <MetadataList.Row
          className={METADATA_TILE_CLASS}
          label="Updated"
          termProps={{ className: METADATA_TERM_CLASS }}
          valueProps={{ className: METADATA_VALUE_CLASS }}
        >
          {formatBridgeDateTime(bridge.updated_at)}
        </MetadataList.Row>
      </MetadataList>
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
      <MetadataList className="grid gap-3 lg:grid-cols-2">
        <MetadataList.Row
          className={METADATA_TILE_CLASS}
          label="Manifest schema"
          termProps={{ className: METADATA_TERM_CLASS }}
          valueProps={{ className: METADATA_VALUE_CLASS }}
        >
          {describeBridgeProviderConfigSchema(provider?.config_schema)}
        </MetadataList.Row>
        <MetadataList.Row
          className={METADATA_TILE_CLASS}
          label="Secret slots"
          termProps={{ className: METADATA_TERM_CLASS }}
          valueProps={{ className: METADATA_VALUE_CLASS }}
        >
          {provider?.secret_slots?.length ? provider.secret_slots.length : "None declared"}
        </MetadataList.Row>
      </MetadataList>

      {provider?.description ? (
        <p className="mt-3 rounded-md border border-(--color-divider) bg-(--color-surface) px-4 py-3 text-small-body leading-relaxed text-(--color-text-secondary)">
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
        <div className="mt-3 flex items-center gap-2 text-small-body text-(--color-text-tertiary)">
          <Spinner aria-label="Loading secret bindings" className="size-4" />
          <span>Loading secret bindings...</span>
        </div>
      ) : null}

      <div className="mt-3">
        <Eyebrow className="mb-2 block" tone="neutral">
          Provider config
        </Eyebrow>
        {providerConfig ? (
          <CodeBlock
            code={providerConfig}
            copyable={false}
            data-testid="bridge-detail-provider-config"
            showPrompt={false}
          />
        ) : (
          <p className="rounded-md border border-(--color-divider) bg-(--color-surface) px-4 py-3 text-small-body leading-relaxed text-(--color-text-secondary)">
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
  const statusTone = statusToPillTone(effectiveStatus);
  const pulse = effectiveStatus === "starting";

  return (
    <PageHeader
      className="px-6 py-4"
      controls={
        <>
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
        </>
      }
      icon={Waypoints}
      statusRow={
        <>
          <span className="flex items-center gap-2">
            <Pill.Dot pulse={pulse} tone={statusTone} />
            <Pill mono tone={statusTone}>
              {effectiveStatus}
            </Pill>
          </span>
          <Pill mono tone={bridge.scope === "workspace" ? "info" : "neutral"}>
            {bridge.scope}
          </Pill>
        </>
      }
      subtitle={`${bridge.platform} / ${bridge.extension_name}`}
      title={bridge.display_name}
    />
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
  if (isLoading || error || !bridge) {
    const state = isLoading ? "loading" : error ? "error" : "empty";
    return (
      <DataSurface state={state} className="flex min-h-0 flex-1 items-center justify-center p-6">
        <DataSurface.Loading
          data-testid="bridge-detail-loading"
          label="Loading bridge"
          surface="bare"
        />
        <DataSurface.Error
          className="max-w-md"
          data-testid="bridge-detail-error"
          description={error?.message ?? "Failed to load bridge details"}
          icon={AlertCircle}
          title="Unable to load bridge"
        />
        <DataSurface.Empty
          className="max-w-md"
          data-testid="bridge-detail-empty"
          description={emptyMessage}
          icon={Waypoints}
          title="Select a bridge"
        />
      </DataSurface>
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
            className="rounded-md border border-(--color-warning)/40 bg-(--color-warning-tint) px-4 py-3 text-small-body text-(--color-warning)"
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

        <div className="flex items-center justify-between gap-3 rounded-md border border-(--color-divider) bg-(--color-surface) px-5 py-4">
          <div className="space-y-1">
            <Eyebrow className="block" tone="neutral">
              Test delivery
            </Eyebrow>
            <p className="text-small-body text-(--color-text-secondary)">
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
