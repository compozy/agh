import { AlertCircle, Loader2, Plus, Waypoints } from "lucide-react";
import { startTransition, useDeferredValue, useMemo, useState } from "react";
import { createFileRoute } from "@tanstack/react-router";
import { toast } from "sonner";

import { PillButton } from "@/components/design-system";
import { Button } from "@/components/ui/button";
import {
  bridgeSecretBindingEnvName,
  buildBridgeCreateRequest,
  buildBridgeSecretBindingRequest,
  buildBridgeUpdateRequest,
  BridgeEditDialog,
  BridgeCreateDialog,
  BridgeDetailPanel,
  BridgeEmptyState,
  BridgeListPanel,
  BridgeTestDeliveryDialog,
  compactBridgeDeliveryDefaults,
  createBridgeCreateDraft,
  createBridgeTestDeliveryDraft,
  createBridgeUpdateDraft,
  findBridgeProviderByKey,
  isBridgeProviderSelectable,
  useBridge,
  useBridgeHealthStream,
  useBridgeProviders,
  useBridgeRoutes,
  useBridgeSecretBindings,
  useBridges,
  useCreateBridge,
  useDeleteBridgeSecretBinding,
  useDisableBridge,
  useEnableBridge,
  usePutBridgeSecretBinding,
  useRestartBridge,
  useTestBridgeDelivery,
  useUpdateBridge,
} from "@/systems/bridges";
import type {
  BridgeCreateDraft,
  BridgeScopeFilter,
  BridgeSummary,
  BridgeTestDeliveryDraft,
  BridgeUpdateDraft,
  TestBridgeDeliveryResponse,
} from "@/systems/bridges";
import { useActiveWorkspace, WorkspacePageShell } from "@/systems/workspace";

export const Route = createFileRoute("/_app/bridges")({
  component: BridgesPage,
});

function matchesBridgeScope(
  bridge: BridgeSummary,
  activeScope: BridgeScopeFilter,
  activeWorkspaceId: string | null
) {
  if (activeScope === "all") {
    return true;
  }

  if (activeScope === "global") {
    return bridge.scope === "global";
  }

  return bridge.scope === "workspace" && bridge.workspace_id === activeWorkspaceId;
}

function matchesBridgeSearch(bridge: BridgeSummary, searchQuery: string) {
  if (!searchQuery) {
    return true;
  }

  const query = searchQuery.toLowerCase();
  return (
    bridge.display_name.toLowerCase().includes(query) ||
    bridge.platform.toLowerCase().includes(query) ||
    bridge.extension_name.toLowerCase().includes(query) ||
    bridge.status.toLowerCase().includes(query)
  );
}

function sortBridges(bridges: BridgeSummary[]) {
  return [...bridges].sort((left, right) => {
    if (left.scope !== right.scope) {
      return left.scope === "global" ? -1 : 1;
    }

    return left.display_name.localeCompare(right.display_name);
  });
}

function bridgeSecretDraftKey(bridgeID: string, bindingName: string) {
  return `${bridgeID}:${bindingName}`;
}

function BridgesPage() {
  const { activeWorkspace, activeWorkspaceId } = useActiveWorkspace();

  const [activeScope, setActiveScope] = useState<BridgeScopeFilter>("all");
  const [searchQuery, setSearchQuery] = useState("");
  const [selectedBridgeId, setSelectedBridgeId] = useState<string | null>(null);
  const [isCreateDialogOpen, setCreateDialogOpen] = useState(false);
  const [isEditDialogOpen, setEditDialogOpen] = useState(false);
  const [isTestDeliveryDialogOpen, setTestDeliveryDialogOpen] = useState(false);
  const [createDraft, setCreateDraft] = useState<BridgeCreateDraft>(() =>
    createBridgeCreateDraft([], activeWorkspaceId)
  );
  const [editDraft, setEditDraft] = useState<BridgeUpdateDraft>(() => createBridgeUpdateDraft());
  const [secretInputValues, setSecretInputValues] = useState<Record<string, string>>({});
  const [restartRequiredByID, setRestartRequiredByID] = useState<Record<string, true>>({});
  const [testDeliveryDraft, setTestDeliveryDraft] = useState<BridgeTestDeliveryDraft>(() =>
    createBridgeTestDeliveryDraft()
  );
  const [testDeliveryResult, setTestDeliveryResult] = useState<TestBridgeDeliveryResponse | null>(
    null
  );

  const deferredSearchQuery = useDeferredValue(searchQuery);
  useBridgeHealthStream();

  const bridgesQuery = useBridges();
  const providersQuery = useBridgeProviders();
  const createBridgeMutation = useCreateBridge();
  const updateBridgeMutation = useUpdateBridge();
  const putBridgeSecretBindingMutation = usePutBridgeSecretBinding();
  const deleteBridgeSecretBindingMutation = useDeleteBridgeSecretBinding();
  const enableBridgeMutation = useEnableBridge();
  const disableBridgeMutation = useDisableBridge();
  const restartBridgeMutation = useRestartBridge();
  const testDeliveryMutation = useTestBridgeDelivery();

  const bridges = bridgesQuery.data?.bridges ?? [];
  const bridgeHealth = bridgesQuery.data?.bridge_health ?? {};
  const providers = providersQuery.data ?? [];
  const totalBridgeCount = bridges.length;
  const canCreateBridge = providers.some(isBridgeProviderSelectable);

  const visibleBridges = useMemo(
    () =>
      sortBridges(
        bridges.filter(
          bridge =>
            matchesBridgeScope(bridge, activeScope, activeWorkspaceId) &&
            matchesBridgeSearch(bridge, deferredSearchQuery)
        )
      ),
    [activeScope, activeWorkspaceId, bridges, deferredSearchQuery]
  );

  const effectiveSelectedBridgeId = useMemo(() => {
    if (selectedBridgeId && visibleBridges.some(bridge => bridge.id === selectedBridgeId)) {
      return selectedBridgeId;
    }

    return visibleBridges[0]?.id ?? null;
  }, [selectedBridgeId, visibleBridges]);

  const selectedBridgeSummary = useMemo(
    () => bridges.find(bridge => bridge.id === effectiveSelectedBridgeId),
    [bridges, effectiveSelectedBridgeId]
  );

  const bridgeDetailQuery = useBridge(effectiveSelectedBridgeId ?? "", {
    enabled: Boolean(effectiveSelectedBridgeId),
  });
  const bridgeRoutesQuery = useBridgeRoutes(effectiveSelectedBridgeId ?? "", {
    enabled: Boolean(effectiveSelectedBridgeId),
  });
  const bridgeSecretBindingsQuery = useBridgeSecretBindings(effectiveSelectedBridgeId ?? "", {
    enabled: Boolean(effectiveSelectedBridgeId),
  });

  const selectedBridge = bridgeDetailQuery.data?.bridge ?? selectedBridgeSummary;
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
    bridgeDetailQuery.data?.health ??
    (effectiveSelectedBridgeId ? bridgeHealth[effectiveSelectedBridgeId] : undefined);
  const selectedSecretBindings = bridgeSecretBindingsQuery.data ?? [];
  const selectedSecretBindingsByName = useMemo(
    () => new Map(selectedSecretBindings.map(binding => [binding.binding_name, binding])),
    [selectedSecretBindings]
  );
  const selectedSecretInputMap = useMemo(() => {
    if (!selectedBridge) {
      return {};
    }

    const inputEntries = new Map<string, string>();
    for (const binding of selectedSecretBindings) {
      inputEntries.set(binding.binding_name, bridgeSecretBindingEnvName(binding));
    }
    for (const [key, value] of Object.entries(secretInputValues)) {
      const prefix = `${selectedBridge.id}:`;
      if (!key.startsWith(prefix)) {
        continue;
      }
      inputEntries.set(key.slice(prefix.length), value);
    }

    return Object.fromEntries(inputEntries.entries());
  }, [secretInputValues, selectedBridge, selectedSecretBindings]);
  const restartRequired =
    selectedBridge != null ? Boolean(restartRequiredByID[selectedBridge.id]) : false;
  const isLifecyclePending =
    enableBridgeMutation.isPending ||
    disableBridgeMutation.isPending ||
    restartBridgeMutation.isPending;
  const isSecretBindingPending =
    putBridgeSecretBindingMutation.isPending || deleteBridgeSecretBindingMutation.isPending;

  const isInitialLoading =
    (bridgesQuery.isLoading && !bridgesQuery.data) ||
    (providersQuery.isLoading && !providersQuery.data);
  const fatalError =
    (!bridgesQuery.data && bridgesQuery.error) || (!providersQuery.data && providersQuery.error);
  const detailError =
    bridgeDetailQuery.error ?? bridgeRoutesQuery.error ?? bridgeSecretBindingsQuery.error ?? null;
  const detailLoading =
    Boolean(effectiveSelectedBridgeId) &&
    bridgeDetailQuery.isLoading &&
    !bridgeDetailQuery.data &&
    !selectedBridgeSummary;

  const listSummary = useMemo(() => {
    if (activeScope === "workspace") {
      if (!activeWorkspace) {
        return "No active workspace selected.";
      }

      return `${visibleBridges.length} bridges in ${activeWorkspace.name}`;
    }

    if (activeScope === "global") {
      return `${visibleBridges.length} global bridges`;
    }

    return `${visibleBridges.length} bridges visible`;
  }, [activeScope, activeWorkspace, visibleBridges.length]);

  const openCreateDialog = () => {
    setCreateDraft(createBridgeCreateDraft(providers, activeWorkspaceId));
    setCreateDialogOpen(true);
  };

  const handleCreateDialogOpenChange = (open: boolean) => {
    setCreateDialogOpen(open);
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

  const markRestartRequired = (bridgeID: string) => {
    setRestartRequiredByID(current => ({
      ...current,
      [bridgeID]: true,
    }));
  };

  const clearRestartRequired = (bridgeID: string) => {
    setRestartRequiredByID(current => {
      if (!(bridgeID in current)) {
        return current;
      }

      const next = { ...current };
      delete next[bridgeID];
      return next;
    });
  };

  const handleCreateBridge = async () => {
    const provider = findBridgeProviderByKey(providers, createDraft.selectedProviderKey);
    if (!provider || !isBridgeProviderSelectable(provider)) {
      toast.error("Select an available bridge provider before creating the bridge.");
      return;
    }
    if (createDraft.scope === "workspace" && !activeWorkspaceId) {
      toast.error("Select an active workspace before creating a workspace-scoped bridge.");
      return;
    }

    const requestResult = buildBridgeCreateRequest(createDraft, provider, activeWorkspaceId);
    if (!requestResult.ok) {
      toast.error(requestResult.error);
      return;
    }

    try {
      const result = await createBridgeMutation.mutateAsync(requestResult.data);

      startTransition(() => {
        setActiveScope(result.bridge.scope);
        setSearchQuery("");
        setSelectedBridgeId(result.bridge.id);
      });
      setCreateDialogOpen(false);
      toast.success(`Created bridge ${result.bridge.display_name}.`);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to create bridge");
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
      markRestartRequired(result.bridge.id);
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

    const envName =
      selectedSecretInputMap[bindingName] ??
      bridgeSecretBindingEnvName(selectedSecretBindingsByName.get(bindingName));
    const requestResult = buildBridgeSecretBindingRequest(envName, bindingName);
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
        [bridgeSecretDraftKey(selectedBridge.id, bindingName)]: bridgeSecretBindingEnvName(binding),
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

  if (isInitialLoading) {
    return (
      <div className="flex flex-1 items-center justify-center" data-testid="bridges-loading">
        <Loader2 className="size-5 animate-spin text-[color:var(--color-text-tertiary)]" />
      </div>
    );
  }

  if (fatalError) {
    return (
      <div className="flex flex-1 items-center justify-center" data-testid="bridges-error">
        <div className="flex flex-col items-center gap-2 text-center">
          <AlertCircle className="size-6 text-[color:var(--color-danger)]" />
          <p className="text-sm text-[color:var(--color-text-tertiary)]">
            {fatalError.message ?? "Failed to load bridges"}
          </p>
        </div>
      </div>
    );
  }

  return (
    <>
      <WorkspacePageShell
        count={totalBridgeCount}
        controls={
          <div className="flex items-center gap-1.5" data-testid="bridge-scope-pills">
            {(["all", "global", "workspace"] as const).map(scope => (
              <PillButton
                key={scope}
                active={activeScope === scope}
                data-testid={`bridge-scope-${scope}`}
                onClick={() =>
                  startTransition(() => {
                    setActiveScope(scope);
                    setSelectedBridgeId(null);
                  })
                }
              >
                {scope.toUpperCase()}
              </PillButton>
            ))}
          </div>
        }
        icon={<Waypoints className="size-4" />}
        meta={
          <Button
            data-testid="create-bridge-btn"
            disabled={!canCreateBridge}
            onClick={openCreateDialog}
            size="lg"
            type="button"
          >
            <Plus className="size-4" />
            Bridge
          </Button>
        }
        title="Bridges"
      >
        {totalBridgeCount === 0 ? (
          <BridgeEmptyState onCreate={openCreateDialog} providers={providers} />
        ) : (
          <>
            <BridgeListPanel
              bridgeHealth={bridgeHealth}
              bridges={visibleBridges}
              onSearchChange={setSearchQuery}
              onSelectBridge={setSelectedBridgeId}
              searchQuery={searchQuery}
              selectedBridgeId={effectiveSelectedBridgeId}
              summary={listSummary}
            />
            <BridgeDetailPanel
              bridge={selectedBridge}
              emptyMessage={
                visibleBridges.length === 0
                  ? "No bridges match the current search or scope filter."
                  : undefined
              }
              error={detailError}
              health={selectedHealth}
              isLifecyclePending={isLifecyclePending}
              isLoading={detailLoading}
              isRoutesLoading={bridgeRoutesQuery.isLoading && !bridgeRoutesQuery.data}
              isSecretBindingPending={isSecretBindingPending}
              isSecretBindingsLoading={
                bridgeSecretBindingsQuery.isLoading && !bridgeSecretBindingsQuery.data
              }
              onDeleteSecretBinding={handleDeleteSecretBinding}
              onDisableBridge={handleDisableBridge}
              onEnableBridge={handleEnableBridge}
              onOpenEdit={openEditDialog}
              onOpenTestDelivery={openTestDeliveryDialog}
              onRestartBridge={handleRestartBridge}
              onSaveSecretBinding={handleSaveSecretBinding}
              onSecretDraftChange={handleSecretInputChange}
              provider={selectedBridgeProvider}
              restartRequired={restartRequired}
              routes={bridgeRoutesQuery.data ?? []}
              secretBindings={selectedSecretBindings}
              secretInputValues={selectedSecretInputMap}
              workspaceName={
                selectedBridge?.scope === "workspace" &&
                selectedBridge.workspace_id === activeWorkspaceId
                  ? activeWorkspace?.name
                  : selectedBridge?.workspace_id
              }
            />
          </>
        )}
      </WorkspacePageShell>

      <BridgeCreateDialog
        activeWorkspaceId={activeWorkspaceId}
        activeWorkspaceName={activeWorkspace?.name}
        draft={createDraft}
        isPending={createBridgeMutation.isPending}
        onDraftChange={setCreateDraft}
        onOpenChange={handleCreateDialogOpenChange}
        onSubmit={handleCreateBridge}
        open={isCreateDialogOpen}
        providers={providers}
      />

      <BridgeEditDialog
        allowProviderDefaultDmPolicy={selectedBridge?.dm_policy == null}
        bridgeName={selectedBridge?.display_name}
        draft={editDraft}
        isPending={updateBridgeMutation.isPending}
        onDraftChange={setEditDraft}
        onOpenChange={handleEditDialogOpenChange}
        onSubmit={handleUpdateBridge}
        open={isEditDialogOpen}
        provider={selectedBridgeProvider}
      />

      <BridgeTestDeliveryDialog
        bridgeName={selectedBridge?.display_name}
        draft={testDeliveryDraft}
        isPending={testDeliveryMutation.isPending}
        onDraftChange={setTestDeliveryDraft}
        onOpenChange={handleTestDeliveryDialogOpenChange}
        onSubmit={handleTestDelivery}
        open={isTestDeliveryDialogOpen}
        result={testDeliveryResult}
      />
    </>
  );
}

function createOptionalMessage(value: string): string | undefined {
  const normalized = value.trim();
  return normalized === "" ? undefined : normalized;
}
