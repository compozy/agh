import { AlertCircle, Loader2, Plus, Waypoints } from "lucide-react";
import { startTransition, useDeferredValue, useMemo, useState } from "react";
import { createFileRoute } from "@tanstack/react-router";
import { toast } from "sonner";

import { PillButton } from "@/components/design-system";
import { Button } from "@/components/ui/button";
import {
  BridgeCreateDialog,
  BridgeDetailPanel,
  BridgeEmptyState,
  BridgeListPanel,
  BridgeTestDeliveryDialog,
  compactBridgeDeliveryDefaults,
  createBridgeCreateDraft,
  createBridgeTestDeliveryDraft,
  findBridgeProviderByKey,
  isBridgeProviderSelectable,
  useBridge,
  useBridgeProviders,
  useBridgeRoutes,
  useBridges,
  useCreateBridge,
  useTestBridgeDelivery,
} from "@/systems/bridges";
import type {
  BridgeCreateDraft,
  BridgeScopeFilter,
  BridgeSummary,
  BridgeTestDeliveryDraft,
  TestBridgeDeliveryResponse,
} from "@/systems/bridges";
import { useActiveWorkspace } from "@/systems/workspace";
import { WorkspacePageShell } from "@/systems/workspace/components/workspace-page-shell";

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

function BridgesPage() {
  const { activeWorkspace, activeWorkspaceId } = useActiveWorkspace();

  const [activeScope, setActiveScope] = useState<BridgeScopeFilter>("all");
  const [searchQuery, setSearchQuery] = useState("");
  const [selectedBridgeId, setSelectedBridgeId] = useState<string | null>(null);
  const [isCreateDialogOpen, setCreateDialogOpen] = useState(false);
  const [isTestDeliveryDialogOpen, setTestDeliveryDialogOpen] = useState(false);
  const [createDraft, setCreateDraft] = useState<BridgeCreateDraft>(() =>
    createBridgeCreateDraft([], activeWorkspaceId)
  );
  const [testDeliveryDraft, setTestDeliveryDraft] = useState<BridgeTestDeliveryDraft>(() =>
    createBridgeTestDeliveryDraft()
  );
  const [testDeliveryResult, setTestDeliveryResult] = useState<TestBridgeDeliveryResponse | null>(
    null
  );

  const deferredSearchQuery = useDeferredValue(searchQuery);

  const bridgesQuery = useBridges();
  const providersQuery = useBridgeProviders();
  const createBridgeMutation = useCreateBridge();
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

  const selectedBridge = bridgeDetailQuery.data?.bridge ?? selectedBridgeSummary;
  const selectedHealth =
    bridgeDetailQuery.data?.health ??
    (effectiveSelectedBridgeId ? bridgeHealth[effectiveSelectedBridgeId] : undefined);

  const isInitialLoading =
    (bridgesQuery.isLoading && !bridgesQuery.data) ||
    (providersQuery.isLoading && !providersQuery.data);
  const fatalError =
    (!bridgesQuery.data && bridgesQuery.error) || (!providersQuery.data && providersQuery.error);
  const detailError = bridgeDetailQuery.error ?? bridgeRoutesQuery.error ?? null;
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

  const handleCreateBridge = async () => {
    const provider = findBridgeProviderByKey(providers, createDraft.selectedProviderKey);
    if (!provider || !isBridgeProviderSelectable(provider)) {
      toast.error("Select an available bridge provider before creating the bridge.");
      return;
    }

    const scope =
      createDraft.scope === "workspace" && !activeWorkspaceId ? "global" : createDraft.scope;

    try {
      const result = await createBridgeMutation.mutateAsync({
        delivery_defaults: compactBridgeDeliveryDefaults(createDraft.deliveryDefaults),
        display_name: createDraft.displayName.trim(),
        enabled: true,
        extension_name: provider.extension_name,
        platform: provider.platform,
        routing_policy: createDraft.routingPolicy,
        scope,
        status: "starting",
        workspace_id: scope === "workspace" ? (activeWorkspaceId ?? undefined) : undefined,
      });

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
              isLoading={detailLoading}
              isRoutesLoading={bridgeRoutesQuery.isLoading && !bridgeRoutesQuery.data}
              onOpenTestDelivery={openTestDeliveryDialog}
              routes={bridgeRoutesQuery.data ?? []}
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
