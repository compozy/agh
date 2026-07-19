import { useDeferredValue, useState } from "react";
import { toast } from "sonner";

import {
  buildBridgeSecretBindingRequest,
  buildBridgeUpdateRequest,
  bridgeListFilterForScope,
  createBridgeUpdateDraft,
  fingerprintBridgeProviderConfig,
  useBridge,
  useBridgeHealthStream,
  useBridgeProviders,
  useBridgeRoutes,
  useBridgeSecretBindings,
  useBridgeTargets,
  useBridges,
  useDeleteBridgeSecretBinding,
  useDisableBridge,
  useEnableBridge,
  usePutBridgeSecretBinding,
  useRestartBridge,
  useResolveBridgeTarget,
  useUpdateBridge,
} from "@/systems/bridges";
import type { BridgeResolveTargetResponse, BridgeUpdateDraft } from "@/systems/bridges";
import { useActiveWorkspace } from "@/systems/workspace";
import { useBridgeDeliveryTests } from "./use-bridge-delivery-tests";
import { useBridgeSetupFlow } from "./use-bridge-setup-flow";

function bridgeSecretDraftKey(bridgeID: string, bindingName: string) {
  return `${bridgeID}:${bindingName}`;
}

function useBridgeDetailPage(bridgeId: string) {
  const { activeWorkspace, activeWorkspaceId } = useActiveWorkspace();

  const [isEditDialogOpen, setEditDialogOpen] = useState(false);
  const [editDraft, setEditDraft] = useState<BridgeUpdateDraft>(() => createBridgeUpdateDraft());
  const [secretInputValues, setSecretInputValues] = useState<Record<string, string>>({});
  const [restartRequiredByID, setRestartRequiredByID] = useState<Record<string, true>>({});
  const [targetSearchQuery, setTargetSearchQuery] = useState("");
  const [targetResolveInput, setTargetResolveInput] = useState("");
  const [targetResolveResult, setTargetResolveResult] =
    useState<BridgeResolveTargetResponse | null>(null);

  const deferredTargetSearchQuery = useDeferredValue(targetSearchQuery);

  const bridgeListFilters = bridgeListFilterForScope("all", activeWorkspaceId);

  const bridgesQuery = useBridges(bridgeListFilters, { enabled: Boolean(bridgeId) });
  useBridgeHealthStream({
    bridgeIds: bridgeId ? [bridgeId] : [],
    enabled: Boolean(bridgeId),
    filters: bridgeListFilters,
  });
  const providersQuery = useBridgeProviders();
  const updateBridgeMutation = useUpdateBridge();
  const putBridgeSecretBindingMutation = usePutBridgeSecretBinding();
  const deleteBridgeSecretBindingMutation = useDeleteBridgeSecretBinding();
  const enableBridgeMutation = useEnableBridge();
  const disableBridgeMutation = useDisableBridge();
  const restartBridgeMutation = useRestartBridge();
  const resolveBridgeTargetMutation = useResolveBridgeTarget();

  const bridges = bridgesQuery.bridges;
  const bridgeHealth = bridgesQuery.bridgeHealth;
  const providers = providersQuery.data ?? [];

  const listBridgeSummary = bridges.find(bridge => bridge.id === bridgeId);

  const bridgeDetailQuery = useBridge(bridgeId, { enabled: Boolean(bridgeId) });
  const bridgeRoutesQuery = useBridgeRoutes(bridgeId, { enabled: Boolean(bridgeId) });
  const bridgeTargetsQuery = useBridgeTargets(
    bridgeId,
    { limit: 50, q: deferredTargetSearchQuery },
    { enabled: Boolean(bridgeId) }
  );
  const bridgeSecretBindingsQuery = useBridgeSecretBindings(bridgeId, {
    enabled: Boolean(bridgeId),
  });

  // Prefer the detail record; fall back to the list summary while it loads.
  const selectedBridge = bridgeDetailQuery.data?.bridge ?? listBridgeSummary;
  const selectedBridgeProvider = selectedBridge
    ? providers.find(
        provider =>
          provider.extension_name === selectedBridge.extension_name &&
          provider.platform === selectedBridge.platform
      )
    : undefined;
  const selectedHealth =
    bridgeDetailQuery.data?.health ?? (bridgeId ? bridgeHealth[bridgeId] : undefined);
  const selectedSecretBindings = bridgeSecretBindingsQuery.data ?? [];
  const setupFactsReady =
    providersQuery.data !== undefined && bridgeSecretBindingsQuery.data !== undefined;
  const deliveryTests = useBridgeDeliveryTests(selectedBridge);
  const setupFlow = useBridgeSetupFlow({
    bindings: selectedSecretBindings,
    bridge: setupFactsReady ? selectedBridge : undefined,
    health: selectedHealth,
    provider: selectedBridgeProvider,
  });
  // Strip `${bridgeId}:` prefixes so the panel sees bare binding names.
  let selectedSecretInputMap: Record<string, string> = {};
  if (selectedBridge) {
    const inputEntries = new Map<string, string>();
    for (const [key, value] of Object.entries(secretInputValues)) {
      const prefix = `${selectedBridge.id}:`;
      if (!key.startsWith(prefix)) continue;
      inputEntries.set(key.slice(prefix.length), value);
    }
    selectedSecretInputMap = Object.fromEntries(inputEntries.entries());
  }
  const restartRequired =
    selectedBridge != null ? Boolean(restartRequiredByID[selectedBridge.id]) : false;
  const isLifecyclePending =
    enableBridgeMutation.isPending ||
    disableBridgeMutation.isPending ||
    restartBridgeMutation.isPending;
  const isSecretBindingPending =
    putBridgeSecretBindingMutation.isPending || deleteBridgeSecretBindingMutation.isPending;

  // Only the primary detail query is fatal; route/secret failures stay sectional.
  const detailError = bridgeDetailQuery.error ?? null;
  const detailLoading =
    Boolean(bridgeId) &&
    bridgeDetailQuery.isLoading &&
    !bridgeDetailQuery.data &&
    !listBridgeSummary;

  const markRestartRequired = (id: string) => {
    setRestartRequiredByID(current => ({
      ...current,
      [id]: true,
    }));
  };

  const clearRestartRequired = (id: string) => {
    setRestartRequiredByID(current => {
      if (!(id in current)) {
        return current;
      }

      const next = { ...current };
      delete next[id];
      return next;
    });
  };

  const openEditDialog = () => {
    if (!selectedBridge) {
      return;
    }

    setEditDraft(createBridgeUpdateDraft(selectedBridge));
    setEditDialogOpen(true);
  };

  const handleEditDialogOpenChange = (open: boolean) => {
    setEditDialogOpen(open);
  };

  const handleUpdateBridge = async () => {
    if (!selectedBridge) {
      return;
    }

    const requestResult = buildBridgeUpdateRequest(editDraft);
    if (!requestResult.ok) {
      toast.error(requestResult.error);
      return;
    }

    try {
      const result = await updateBridgeMutation.mutateAsync({
        data: requestResult.data,
        id: selectedBridge.id,
      });

      setEditDialogOpen(false);
      const providerConfigChanged =
        fingerprintBridgeProviderConfig(selectedBridge.provider_config) !==
        fingerprintBridgeProviderConfig(result.bridge.provider_config);
      if (providerConfigChanged) {
        setupFlow.clearEvidence();
      } else {
        setupFlow.clearVerification();
      }
      markRestartRequired(selectedBridge.id);
      toast.success(`Updated bridge ${result.bridge.display_name}. Restart to apply changes.`);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to update bridge");
    }
  };

  const handleSecretInputChange = (bindingName: string, value: string) => {
    if (!selectedBridge) {
      return;
    }

    setSecretInputValues(current => ({
      ...current,
      [bridgeSecretDraftKey(selectedBridge.id, bindingName)]: value,
    }));
  };

  const handleSaveSecretBinding = async (bindingName: string) => {
    if (!selectedBridge) {
      return;
    }

    const secretValue = selectedSecretInputMap[bindingName] ?? "";
    const requestResult = buildBridgeSecretBindingRequest(
      selectedBridge.id,
      bindingName,
      secretValue,
      bindingName
    );
    if (!requestResult.ok) {
      toast.error(requestResult.error);
      return;
    }

    try {
      const binding = await putBridgeSecretBindingMutation.mutateAsync({
        bindingName,
        data: requestResult.data,
        id: selectedBridge.id,
      });

      setSecretInputValues(current => ({
        ...current,
        [bridgeSecretDraftKey(selectedBridge.id, binding.binding_name)]: "",
      }));
      setupFlow.clearEvidence();
      markRestartRequired(selectedBridge.id);
      toast.success(`Updated secret binding ${bindingName} for ${selectedBridge.display_name}.`);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to update bridge secret");
    }
  };

  const handleDeleteSecretBinding = async (bindingName: string) => {
    if (!selectedBridge) {
      return;
    }

    try {
      await deleteBridgeSecretBindingMutation.mutateAsync({
        bindingName,
        id: selectedBridge.id,
      });

      setSecretInputValues(current => ({
        ...current,
        [bridgeSecretDraftKey(selectedBridge.id, bindingName)]: "",
      }));
      setupFlow.clearEvidence();
      markRestartRequired(selectedBridge.id);
      toast.success(`Deleted secret binding ${bindingName} for ${selectedBridge.display_name}.`);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to delete bridge secret");
    }
  };

  const handleEnableBridge = async () => {
    if (!selectedBridge) {
      return;
    }

    setupFlow.clearVerification();
    try {
      const result = await enableBridgeMutation.mutateAsync({ id: selectedBridge.id });
      clearRestartRequired(result.bridge.id);
      toast.success(`Enabled bridge ${result.bridge.display_name}.`);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to enable bridge");
    }
  };

  const handleDisableBridge = async () => {
    if (!selectedBridge) {
      return;
    }

    setupFlow.clearVerification();
    try {
      const result = await disableBridgeMutation.mutateAsync({ id: selectedBridge.id });
      toast.success(`Disabled bridge ${result.bridge.display_name}.`);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to disable bridge");
    }
  };

  const handleRestartBridge = async () => {
    if (!selectedBridge) {
      return;
    }

    setupFlow.clearVerification();
    try {
      const result = await restartBridgeMutation.mutateAsync({ id: selectedBridge.id });
      clearRestartRequired(result.bridge.id);
      toast.success(`Restarted bridge ${result.bridge.display_name}.`);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to restart bridge");
    }
  };

  const handleTargetSearchChange = (query: string) => {
    setTargetSearchQuery(query);
  };

  const handleTargetResolveInputChange = (value: string) => {
    setTargetResolveInput(value);
    setTargetResolveResult(null);
  };

  const handleResolveBridgeTarget = async () => {
    if (!selectedBridge) {
      return;
    }

    const name = targetResolveInput.trim();
    if (name === "") {
      toast.error("Enter a bridge target name to resolve.");
      return;
    }

    try {
      const result = await resolveBridgeTargetMutation.mutateAsync({
        id: selectedBridge.id,
        data: { name },
      });
      setTargetResolveResult(result);
      if (result.result.match) {
        toast.success(`Resolved target ${result.result.match.display_name}.`);
        return;
      }
      if (result.result.ambiguous) {
        toast.error(result.diagnostic?.message ?? "Bridge target matched multiple candidates.");
        return;
      }
      toast.error(result.diagnostic?.message ?? "Bridge target was not found.");
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to resolve bridge target");
    }
  };

  const detailPanelProps = {
    bridge: selectedBridge,
    emptyMessage: "Bridge not found.",
    error: detailError,
    health: selectedHealth,
    state: {
      isLifecyclePending,
      isLoading: detailLoading,
      isProviderLoading: providersQuery.isLoading && !providersQuery.data,
      isRoutesLoading: bridgeRoutesQuery.isLoading && !bridgeRoutesQuery.data,
      isSecretBindingPending,
      isSecretBindingsLoading:
        bridgeSecretBindingsQuery.isLoading && !bridgeSecretBindingsQuery.data,
      providerError: providersQuery.error,
      secretBindingsError: bridgeSecretBindingsQuery.error,
    },
    targetDirectory: {
      error: bridgeTargetsQuery.error,
      isLoading: bridgeTargetsQuery.isLoading && !bridgeTargetsQuery.data,
      isResolving: resolveBridgeTargetMutation.isPending,
      onQueryChange: handleTargetSearchChange,
      onResolveInputChange: handleTargetResolveInputChange,
      onResolveSubmit: handleResolveBridgeTarget,
      query: targetSearchQuery,
      resolveInput: targetResolveInput,
      resolveResult: targetResolveResult,
      response: bridgeTargetsQuery.data,
    },
    onDeleteSecretBinding: handleDeleteSecretBinding,
    onDisableBridge: handleDisableBridge,
    onEnableBridge: handleEnableBridge,
    onOpenEdit: openEditDialog,
    onOpenSendTest: deliveryTests.openSendTest,
    onOpenTestDelivery: deliveryTests.openDryRun,
    onRestartBridge: handleRestartBridge,
    onSaveSecretBinding: handleSaveSecretBinding,
    onSecretDraftChange: handleSecretInputChange,
    provider: selectedBridgeProvider,
    restartRequired,
    routes: bridgeRoutesQuery.data ?? [],
    secretBindings: selectedSecretBindings,
    secretInputValues: selectedSecretInputMap,
    setup: {
      isLifecyclePending,
      isRegistering: setupFlow.isRegistering,
      isVerifying: setupFlow.isVerifying,
      onRegisterWebhook: setupFlow.registerWebhook,
      onVerify: setupFlow.verify,
      projection: setupFlow.projection,
    },
    workspaceName:
      selectedBridge?.scope === "workspace" && selectedBridge.workspace_id === activeWorkspaceId
        ? activeWorkspace?.name
        : selectedBridge?.workspace_id,
  };

  const editDialogProps = {
    allowProviderDefaultDmPolicy: selectedBridge?.dm_policy == null,
    bridgeName: selectedBridge?.display_name,
    draft: editDraft,
    isPending: updateBridgeMutation.isPending,
    onDraftChange: setEditDraft,
    onOpenChange: handleEditDialogOpenChange,
    onSubmit: handleUpdateBridge,
    open: isEditDialogOpen,
    provider: selectedBridgeProvider,
  };

  return {
    detailPanelProps,
    editDialogProps,
    selectedBridge,
    sendTestDialogProps: deliveryTests.sendTestDialogProps,
    testDeliveryDialogProps: deliveryTests.dryRunDialogProps,
  };
}

export { useBridgeDetailPage };
