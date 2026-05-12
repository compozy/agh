import { createFileRoute } from "@tanstack/react-router";
import { AlertCircle, Plus, Waypoints } from "lucide-react";

import { useBridgesPage } from "@/hooks/routes/use-bridges-page";
import {
  BridgeCreateDialog,
  BridgeDetailPanel,
  BridgeEditDialog,
  BridgeEmptyState,
  BridgeListPanel,
  BridgeTestDeliveryDialog,
} from "@/systems/bridges";
import type { TopbarRouteContext } from "@/types/topbar";
import { Button, Empty, PillGroup, Spinner, SplitPane, useTopbarSlot } from "@agh/ui";

export const Route = createFileRoute("/_app/bridges")({
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { title: "Bridges", icon: Waypoints },
  }),
  component: BridgesPage,
});

function BridgesPage() {
  const page = useBridgesPage();

  useTopbarSlot({
    count: page.totalBridgeCount,
    tabs: (
      <PillGroup
        aria-label="Bridge scope"
        data-testid="bridge-scope-pills"
        items={[
          { value: "all", label: "All", testId: "bridge-scope-all" },
          { value: "global", label: "Global", testId: "bridge-scope-global" },
          { value: "workspace", label: "Workspace", testId: "bridge-scope-workspace" },
        ]}
        onChange={page.selectScope}
        value={page.activeScope}
      />
    ),
    actions: (
      <Button
        data-testid="create-bridge-btn"
        disabled={!page.canCreateBridge}
        onClick={page.openCreateDialog}
        size="sm"
        type="button"
        variant="outline"
      >
        <Plus className="size-3" />
        Bridge
      </Button>
    ),
  });

  if (page.isInitialLoading) {
    return (
      <div
        className="flex min-h-0 flex-1 items-center justify-center"
        data-testid="bridges-loading"
      >
        <Spinner aria-hidden="true" className="size-5 text-subtle" />
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

  return (
    <>
      <div className="flex min-h-0 flex-1 flex-col overflow-hidden" data-testid="bridges-shell">
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
