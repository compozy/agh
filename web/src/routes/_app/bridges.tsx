import { AlertCircle, Loader2, Plus, Waypoints } from "lucide-react";
import { createFileRoute } from "@tanstack/react-router";

import { Button, Empty, PageHeader, PillGroup, SplitPane } from "@agh/ui";
import {
  BridgeCreateDialog,
  BridgeDetailPanel,
  BridgeEditDialog,
  BridgeEmptyState,
  BridgeListPanel,
  BridgeTestDeliveryDialog,
} from "@/systems/bridges";
import { useBridgesPage } from "@/hooks/routes/use-bridges-page";

export const Route = createFileRoute("/_app/bridges")({
  component: BridgesPage,
});

function BridgesPage() {
  const page = useBridgesPage();

  if (page.isInitialLoading) {
    return (
      <div
        className="flex min-h-0 flex-1 items-center justify-center"
        data-testid="bridges-loading"
      >
        <Loader2 aria-hidden="true" className="size-5 animate-spin text-(--color-text-tertiary)" />
      </div>
    );
  }

  if (page.fatalError) {
    return (
      <div
        className="flex min-h-0 flex-1 items-center justify-center px-6 py-10"
        data-testid="bridges-error"
      >
        <Empty
          className="max-w-md"
          description={page.fatalError.message ?? "Failed to load bridges"}
          icon={AlertCircle}
          title="Unable to load bridges"
        />
      </div>
    );
  }

  const primaryAction = (
    <Button
      data-testid="create-bridge-btn"
      disabled={!page.canCreateBridge}
      onClick={page.openCreateDialog}
      size="sm"
      type="button"
      variant="outline"
    >
      <Plus className="size-3.5" />
      Bridge
    </Button>
  );

  const controls = (
    <PillGroup
      aria-label="Bridge scope"
      data-testid="bridge-scope-pills"
      items={[
        { value: "all", label: "ALL", testId: "bridge-scope-all" },
        { value: "global", label: "GLOBAL", testId: "bridge-scope-global" },
        { value: "workspace", label: "WORKSPACE", testId: "bridge-scope-workspace" },
      ]}
      onChange={page.selectScope}
      value={page.activeScope}
    />
  );

  return (
    <>
      <div className="flex min-h-0 flex-1 flex-col overflow-hidden" data-testid="bridges-shell">
        <PageHeader
          count={page.totalBridgeCount}
          controls={controls}
          icon={() => <Waypoints className="size-3.5" data-testid="bridges-shell-icon" />}
          meta={primaryAction}
          title={<span data-testid="bridges-shell-title">Bridges</span>}
        />

        {page.totalBridgeCount === 0 ? (
          <BridgeEmptyState onCreate={page.openCreateDialog} providers={page.providers} />
        ) : (
          <SplitPane
            data-testid="bridges-split-pane"
            detail={<BridgeDetailPanel {...page.detailPanelProps} />}
            list={<BridgeListPanel {...page.listPanelProps} />}
          />
        )}
      </div>

      <BridgeCreateDialog {...page.createDialogProps} />
      <BridgeEditDialog {...page.editDialogProps} />
      <BridgeTestDeliveryDialog {...page.testDeliveryDialogProps} />
    </>
  );
}
