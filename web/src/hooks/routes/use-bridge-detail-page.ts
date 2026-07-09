import { useCallback, useDeferredValue, useMemo, useState } from "react";
import { useNavigate } from "@tanstack/react-router";
import { toast } from "sonner";

import {
  buildBridgeSecretBindingRequest,
  buildBridgeUpdateRequest,
  compactBridgeDeliveryDefaults,
  createBridgeTestDeliveryDraft,
  createBridgeUpdateDraft,
  useBridge,
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
  useTestBridgeDelivery,
  useUpdateBridge,
} from "@/systems/bridges";
import type {
  BridgeListFilter,
  BridgeResolveTargetResponse,
  BridgeTestDeliveryDraft,
  BridgeUpdateDraft,
  TestBridgeDeliveryResponse,
} from "@/systems/bridges";
import { useActiveWorkspace } from "@/systems/workspace";

function bridgeSecretDraftKey(bridgeID: string, bindingName: string) {
  return `${bridgeID}:${bindingName}`;
}

function createOptionalMessage(value: string): string | undefined {
  const normalized = value.trim();
  return normalized === "" ? undefined : normalized;
}

function useBridgeDetailPage(bridgeId: string) {
  const navigate = useNavigate();
  const { activeWorkspace, activeWorkspaceId } = useActiveWorkspace();

  const [isEditDialogOpen, setEditDialogOpen] = useState(false);
  const [isTestDeliveryDialogOpen, setTestDeliveryDialogOpen] = useState(false);
  const [editDraft, setEditDraft] = useState<BridgeUpdateDraft>(() => createBridgeUpdateDraft());
  const [secretInputValues, setSecretInputValues] = useState<Record<string, string>>({});
  const [restartRequiredByID, setRestartRequiredByID] = useState<Record<string, true>>({});
  const [targetSearchQuery, setTargetSearchQuery] = useState("");
  const [targetResolveInput, setTargetResolveInput] = useState("");
  const [targetResolveResult, setTargetResolveResult] =
    useState<BridgeResolveTargetResponse | null>(null);
  const [testDeliveryDraft, setTestDeliveryDraft] = useState<BridgeTestDeliveryDraft>(() =>
    createBridgeTestDeliveryDraft()
  );
  const [testDeliveryResult, setTestDeliveryResult] = useState<TestBridgeDeliveryResponse | null>(
    null
  );

  const deferredTargetSearchQuery = useDeferredValue(targetSearchQuery);

  const bridgeListFilters = useMemo<BridgeListFilter>(() => {
    if (!activeWorkspaceId) {
      return { scope: "global" };
    }
    return { scope: "all", workspace_id: activeWorkspaceId };
  }, [activeWorkspaceId]);

  const bridgesQuery = useBridges(bridgeListFilters, { enabled: Boolean(bridgeId) });
  const providersQuery = useBridgeProviders();
  const updateBridgeMutation = useUpdateBridge();
  const putBridgeSecretBindingMutation = usePutBridgeSecretBinding();
  const deleteBridgeSecretBindingMutation = useDeleteBridgeSecretBinding();
  const enableBridgeMutation = useEnableBridge();
  const disableBridgeMutation = useDisableBridge();
  const restartBridgeMutation = useRestartBridge();
  const resolveBridgeTargetMutation = useResolveBridgeTarget();
  const testDeliveryMutation = useTestBridgeDelivery();

  const bridges = bridgesQuery.data?.bridges ?? [];
  const bridgeHealth = bridgesQuery.data?.bridge_health ?? {};
  const providers = providersQuery.data ?? [];

  const listBridgeSummary = useMemo(
    () => bridges.find(bridge => bridge.id === bridgeId),
    [bridgeId, bridges]
  );

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
  const selectedBridgeProvider = useMemo(
    () =>
      selectedBridge
        ? providers.find(
            provider =>
              provider.extension_name === selectedBridge.extension_name &&
              provider.platform === selectedBridge.platform
          )
        : undefined,
    [providers, selectedBridge]
  );
  const selectedHealth =
    bridgeDetailQuery.data?.health ?? (bridgeId ? bridgeHealth[bridgeId] : undefined);
  const selectedSecretBindings = bridgeSecretBindingsQuery.data ?? [];
  // Strip `${bridgeId}:` prefixes so the panel sees bare binding names.
  const selectedSecretInputMap = useMemo(() => {
    if (!selectedBridge) {
      return {};
    }

    const inputEntries = new Map<string, string>();

    for (const [key, value] of Object.entries(secretInputValues)) {
      const prefix = `${selectedBridge.id}:`;
      if (!key.startsWith(prefix)) {
        continue;
      }

      inputEntries.set(key.slice(prefix.length), value);
    }

    return Object.fromEntries(inputEntries.entries());
  }, [secretInputValues, selectedBridge]);
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

  const openTestDeliveryDialog = () => {
    setTestDeliveryDraft(createBridgeTestDeliveryDraft(selectedBridge));
    setTestDeliveryResult(null);
    setTestDeliveryDialogOpen(true);
  };

  const handleTestDeliveryDialogOpenChange = (open: boolean) => {
    setTestDeliveryDialogOpen(open);
    if (!open) {
      setTestDeliveryResult(null);
    }
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

    try {
      const result = await restartBridgeMutation.mutateAsync({ id: selectedBridge.id });
      clearRestartRequired(result.bridge.id);
      toast.success(`Restarted bridge ${result.bridge.display_name}.`);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to restart bridge");
    }
  };

  const handleTestDelivery = async () => {
    if (!selectedBridge) {
      return;
    }

    try {
      const result = await testDeliveryMutation.mutateAsync({
        id: selectedBridge.id,
        data: {
          message: createOptionalMessage(testDeliveryDraft.message),
          target: {
            bridge_instance_id: selectedBridge.id,
            ...compactBridgeDeliveryDefaults(testDeliveryDraft.target),
          },
        },
      });

      setTestDeliveryResult(result);
      toast.success(`Resolved delivery target for ${selectedBridge.display_name}.`);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to resolve bridge target");
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

  const handleBack = useCallback(() => {
    void navigate({ to: "/bridges" });
  }, [navigate]);

  const detailPanelProps = {
    bridge: selectedBridge,
    emptyMessage: "Bridge not found.",
    error: detailError,
    health: selectedHealth,
    onBack: handleBack,
    state: {
      isLifecyclePending,
      isLoading: detailLoading,
      isRoutesLoading: bridgeRoutesQuery.isLoading && !bridgeRoutesQuery.data,
      isSecretBindingPending,
      isSecretBindingsLoading:
        bridgeSecretBindingsQuery.isLoading && !bridgeSecretBindingsQuery.data,
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
    onOpenTestDelivery: openTestDeliveryDialog,
    onRestartBridge: handleRestartBridge,
    onSaveSecretBinding: handleSaveSecretBinding,
    onSecretDraftChange: handleSecretInputChange,
    provider: selectedBridgeProvider,
    restartRequired,
    routes: bridgeRoutesQuery.data ?? [],
    secretBindings: selectedSecretBindings,
    secretInputValues: selectedSecretInputMap,
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

  const testDeliveryDialogProps = {
    bridgeName: selectedBridge?.display_name,
    draft: testDeliveryDraft,
    isPending: testDeliveryMutation.isPending,
    onDraftChange: setTestDeliveryDraft,
    onOpenChange: handleTestDeliveryDialogOpenChange,
    onSubmit: handleTestDelivery,
    open: isTestDeliveryDialogOpen,
    result: testDeliveryResult,
  };

  return {
    detailPanelProps,
    editDialogProps,
    handleBack,
    selectedBridge,
    testDeliveryDialogProps,
  };
}

export { useBridgeDetailPage };
