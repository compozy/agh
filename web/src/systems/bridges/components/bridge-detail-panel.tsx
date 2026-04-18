import { AlertCircle, Loader2, Pencil, Power, RotateCw, SendHorizontal } from "lucide-react";

import { Button, Input, Pill } from "@agh/ui";

import { cn } from "@/lib/utils";

import { pillVariantFromTone } from "@/lib/pill-variant";
import {
  bridgeScopeTone,
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
  const providerConfig = formatBridgeProviderConfig(bridge?.provider_config);
  const effectiveStatus = health?.status ?? bridge?.status;
  const bindingsByName = new Map(secretBindings.map(binding => [binding.binding_name, binding]));

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
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div className="space-y-3">
              <div className="flex flex-wrap items-center gap-3">
                <h2 className="text-xl font-semibold text-[color:var(--color-text-primary)]">
                  {bridge.display_name}
                </h2>
                <Pill
                  variant={pillVariantFromTone(bridgeStatusTone(effectiveStatus ?? bridge.status))}
                >
                  {effectiveStatus ?? bridge.status}
                </Pill>
                <Pill variant={pillVariantFromTone(bridgeScopeTone(bridge.scope))}>
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

            <div className="flex flex-wrap items-center gap-2">
              <Button
                data-testid="edit-bridge-btn"
                disabled={isLifecyclePending}
                onClick={onOpenEdit}
                size="sm"
                type="button"
                variant="outline"
              >
                <Pencil className="size-4" />
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
                <RotateCw className="size-4" />
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
                  <Power className="size-4" />
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
                  <Power className="size-4" />
                  Enable
                </Button>
              )}
            </div>
          </div>
          {restartRequired ? (
            <div
              className="rounded-lg border border-[color:var(--color-warning)] bg-[color:var(--color-warning-tint)] px-4 py-3 text-sm text-[color:var(--color-warning)]"
              data-testid="bridge-restart-required"
            >
              Pending runtime changes require a restart or enable action before the provider picks
              them up.
            </div>
          ) : null}
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
            <DetailFact label="DM policy" value={describeBridgeDmPolicy(bridge.dm_policy)} />
            <DetailFact label="Created" value={formatBridgeDateTime(bridge.created_at)} />
            <DetailFact label="Updated" value={formatBridgeDateTime(bridge.updated_at)} />
          </div>
        </DetailSection>

        <DetailSection title="Provider runtime">
          <div className="grid gap-3 lg:grid-cols-2">
            <DetailFact
              label="Manifest schema"
              value={describeBridgeProviderConfigSchema(provider?.config_schema)}
            />
            <DetailFact
              label="Secret slots"
              value={
                provider?.secret_slots?.length ? provider.secret_slots.length : "None declared"
              }
            />
          </div>

          {provider?.description ? (
            <div className="mt-4 rounded-lg border border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)] px-4 py-3">
              <p className="font-mono text-[0.64rem] uppercase tracking-[0.14em] text-[color:var(--color-text-label)]">
                Provider hint
              </p>
              <p className="mt-2 text-sm leading-relaxed text-[color:var(--color-text-secondary)]">
                {provider.description}
              </p>
            </div>
          ) : null}

          {provider?.secret_slots?.length ? (
            <div className="mt-4 space-y-3" data-testid="bridge-detail-secret-slots">
              {provider.secret_slots.map(slot => {
                const binding = bindingsByName.get(slot.name);
                const inputValue = secretInputValues[slot.name] ?? "";

                return (
                  <article
                    key={slot.name}
                    className="rounded-lg border border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)] px-4 py-3"
                    data-testid={`bridge-secret-binding-${slot.name}`}
                  >
                    <div className="flex flex-wrap items-center gap-2">
                      <p className="font-mono text-[0.68rem] uppercase tracking-[0.14em] text-[color:var(--color-text-primary)]">
                        {slot.name}
                      </p>
                      <Pill
                        variant={pillVariantFromTone(slot.required === false ? "neutral" : "amber")}
                      >
                        {slot.required === false ? "optional" : "required"}
                      </Pill>
                      <Pill variant={pillVariantFromTone(binding ? "green" : "neutral")}>
                        {binding ? "bound" : "unbound"}
                      </Pill>
                    </div>
                    <p className="mt-2 text-sm leading-relaxed text-[color:var(--color-text-secondary)]">
                      {describeBridgeSecretSlot(slot)}
                    </p>
                    <div className="mt-3 grid gap-3 lg:grid-cols-[minmax(0,1fr)_auto] lg:items-end">
                      <div className="space-y-2">
                        <label
                          className="font-mono text-[0.64rem] uppercase tracking-[0.14em] text-[color:var(--color-text-label)]"
                          htmlFor={`bridge-secret-env-${slot.name}`}
                        >
                          Environment variable
                        </label>
                        <Input
                          data-testid={`bridge-secret-env-input-${slot.name}`}
                          id={`bridge-secret-env-${slot.name}`}
                          onChange={event => onSecretDraftChange?.(slot.name, event.target.value)}
                          placeholder="AGH_BRIDGE_TOKEN"
                          value={inputValue}
                        />
                        <p className="text-xs leading-relaxed text-[color:var(--color-text-tertiary)]">
                          The stock daemon resolves bridge secrets from `env:NAME` refs.
                        </p>
                        {binding ? (
                          <p className="text-xs text-[color:var(--color-text-secondary)]">
                            Current ref: <span className="font-mono">{binding.vault_ref}</span>
                          </p>
                        ) : (
                          <p className="text-xs text-[color:var(--color-text-tertiary)]">
                            No secret binding stored.
                          </p>
                        )}
                      </div>
                      <div className="flex flex-wrap items-center gap-2">
                        <Button
                          data-testid={`save-bridge-secret-${slot.name}`}
                          disabled={!inputValue.trim() || isSecretBindingPending}
                          onClick={() => onSaveSecretBinding?.(slot.name)}
                          size="sm"
                          type="button"
                        >
                          Save
                        </Button>
                        <Button
                          data-testid={`delete-bridge-secret-${slot.name}`}
                          disabled={!binding || isSecretBindingPending}
                          onClick={() => onDeleteSecretBinding?.(slot.name)}
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
              })}
            </div>
          ) : isSecretBindingsLoading ? (
            <div className="mt-4 flex items-center gap-2 text-sm text-[color:var(--color-text-tertiary)]">
              <Loader2 className="size-4 animate-spin" />
              <span>Loading secret bindings…</span>
            </div>
          ) : null}

          <div className="mt-4 rounded-lg border border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)] px-4 py-3">
            <p className="font-mono text-[0.64rem] uppercase tracking-[0.14em] text-[color:var(--color-text-label)]">
              Provider config
            </p>
            {providerConfig ? (
              <pre
                className="mt-3 overflow-x-auto rounded-lg border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] p-3 font-mono text-xs leading-5 text-[color:var(--color-text-primary)]"
                data-testid="bridge-detail-provider-config"
              >
                {providerConfig}
              </pre>
            ) : (
              <p className="mt-2 text-sm leading-relaxed text-[color:var(--color-text-secondary)]">
                No provider runtime config stored for this bridge.
              </p>
            )}
          </div>
        </DetailSection>

        <DetailSection action={<Pill>{routes.length}</Pill>} title="Routes">
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
                    <Pill variant={pillVariantFromTone(bridgeScopeTone(route.scope))}>
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
