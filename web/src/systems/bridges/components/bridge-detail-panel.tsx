import { AlertCircle, Waypoints } from "lucide-react";

import { cn, DataSurface, PAGE_CONTENT_GUTTER } from "@agh/ui";

import { formatBridgeProviderConfig } from "../lib/bridge-formatters";
import type { BridgeSetupProjection } from "../lib/bridge-setup";
import type {
  BridgeHealth,
  BridgeProvider,
  BridgeRoute,
  BridgeSecretBinding,
  BridgeStatus,
  BridgeSummary,
} from "../types";
import { BridgeDeliveryActions } from "./bridge-delivery-actions";
import { BridgeDetailConfiguration } from "./bridge-detail-configuration";
import { BridgeDetailHeader } from "./bridge-detail-header";
import { BridgeDetailMetrics } from "./bridge-detail-metrics";
import { BridgeEventStreamSection } from "./bridge-event-stream-section";
import { BridgeProviderRuntimeSection } from "./bridge-provider-runtime-section";
import { BridgeSetupChecklist } from "./bridge-setup-checklist";
import {
  BridgeTargetDirectorySection,
  type BridgeTargetDirectoryState,
} from "./bridge-target-directory-section";

interface BridgeDetailPanelProps {
  bridge: BridgeSummary | undefined;
  emptyMessage?: string;
  error: Error | null;
  health: BridgeHealth | undefined;
  onDeleteSecretBinding?: (bindingName: string) => void;
  onDisableBridge?: () => void;
  onEnableBridge?: () => void;
  onOpenEdit?: () => void;
  onOpenSendTest: () => void;
  onOpenTestDelivery: () => void;
  onRestartBridge?: () => void;
  onSaveSecretBinding?: (bindingName: string) => void;
  onSecretDraftChange?: (bindingName: string, value: string) => void;
  provider?: BridgeProvider;
  restartRequired?: boolean;
  routes: BridgeRoute[];
  secretBindings?: BridgeSecretBinding[];
  secretInputValues?: Record<string, string>;
  setup: {
    isLifecyclePending: boolean;
    isRegistering: boolean;
    isVerifying: boolean;
    onRegisterWebhook: () => void;
    onVerify: () => void;
    projection: BridgeSetupProjection | null;
  };
  state: {
    isLifecyclePending?: boolean;
    isLoading: boolean;
    isProviderLoading?: boolean;
    isRoutesLoading: boolean;
    isSecretBindingPending?: boolean;
    isSecretBindingsLoading?: boolean;
    providerError?: Error | null;
    secretBindingsError?: Error | null;
  };
  targetDirectory?: BridgeTargetDirectoryState;
  workspaceName?: string | null;
}

const EMPTY_SECRET_BINDINGS: BridgeSecretBinding[] = [];
const EMPTY_SECRET_INPUT_VALUES: Record<string, string> = {};
const EMPTY_CHECKS_BY_SECRET_SLOT: BridgeSetupProjection["checksBySecretSlot"] = {};

export function BridgeDetailPanel({
  bridge,
  emptyMessage = "Select a bridge to inspect configuration, routes, and delivery health.",
  error,
  health,
  onDeleteSecretBinding,
  onDisableBridge,
  onEnableBridge,
  onOpenEdit,
  onOpenSendTest,
  onOpenTestDelivery,
  onRestartBridge,
  onSaveSecretBinding,
  onSecretDraftChange,
  provider,
  restartRequired = false,
  routes,
  secretBindings = EMPTY_SECRET_BINDINGS,
  secretInputValues = EMPTY_SECRET_INPUT_VALUES,
  setup,
  state,
  targetDirectory,
  workspaceName,
}: BridgeDetailPanelProps) {
  const {
    isLifecyclePending: stateLifecyclePending = false,
    isLoading,
    isProviderLoading = false,
    isRoutesLoading,
    isSecretBindingPending = false,
    isSecretBindingsLoading = false,
    providerError = null,
    secretBindingsError = null,
  } = state;
  const isLifecyclePending =
    setup.isLifecyclePending || stateLifecyclePending || setup.isRegistering || setup.isVerifying;

  if (isLoading || error || !bridge) {
    const surfaceState = isLoading ? "loading" : error ? "error" : "empty";
    return (
      <DataSurface
        className="flex min-h-0 flex-1 items-center justify-center py-10"
        state={surfaceState}
      >
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
  const bridgeDisabled = !bridge.enabled || effectiveStatus === "disabled";

  return (
    <section
      className={cn(PAGE_CONTENT_GUTTER, "flex min-h-0 flex-1 flex-col overflow-hidden")}
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

      <div className="min-h-0 flex-1 space-y-6 overflow-y-auto py-5">
        <BridgeSetupChecklist
          isLifecyclePending={isLifecyclePending}
          isRegistering={setup.isRegistering}
          isVerifying={setup.isVerifying}
          onEnableBridge={onEnableBridge}
          onOpenEdit={onOpenEdit}
          onRegisterWebhook={setup.onRegisterWebhook}
          onRestartBridge={onRestartBridge}
          onVerify={setup.onVerify}
          projection={setup.projection}
        />

        {restartRequired ? (
          <div
            className="rounded-md border border-warning/40 bg-warning-tint px-4 py-3 text-small-body text-warning"
            data-testid="bridge-restart-required"
          >
            Pending runtime changes require a restart or enable action before the provider picks
            them up.
          </div>
        ) : null}

        <BridgeDetailMetrics health={health} routes={routes} />
        <BridgeDetailConfiguration bridge={bridge} workspaceName={workspaceName} />
        <BridgeProviderRuntimeSection
          bindingsByName={bindingsByName}
          checksBySecretSlot={setup.projection?.checksBySecretSlot ?? EMPTY_CHECKS_BY_SECRET_SLOT}
          isProviderLoading={isProviderLoading}
          isSecretBindingPending={isSecretBindingPending}
          isSecretBindingsLoading={isSecretBindingsLoading}
          onDeleteSecretBinding={onDeleteSecretBinding}
          onSaveSecretBinding={onSaveSecretBinding}
          onSecretDraftChange={onSecretDraftChange}
          provider={provider}
          providerError={providerError}
          providerConfig={providerConfig}
          secretBindingsError={secretBindingsError}
          secretInputValues={secretInputValues}
        />
        <BridgeTargetDirectorySection state={targetDirectory} />
        <BridgeEventStreamSection isRoutesLoading={isRoutesLoading} routes={routes} />
        <BridgeDeliveryActions
          bridgeDisabled={bridgeDisabled}
          onOpenDryRun={onOpenTestDelivery}
          onOpenSendTest={onOpenSendTest}
        />
      </div>
    </section>
  );
}
