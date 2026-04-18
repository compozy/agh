import { AlertCircle, Loader2, Plus, Waypoints } from "lucide-react";
import { createFileRoute } from "@tanstack/react-router";

import { Button, Pills } from "@agh/ui";
import {
  BridgeCreateDialog,
  BridgeDetailPanel,
  BridgeEditDialog,
  BridgeEmptyState,
  BridgeListPanel,
  BridgeTestDeliveryDialog,
} from "@/systems/bridges";
import { useBridgesPage } from "@/hooks/routes/use-bridges-page";
import { WorkspacePageShell } from "@/systems/workspace";

export const Route = createFileRoute("/_app/bridges")({
  component: BridgesPage,
});

function BridgesPage() {
  const page = useBridgesPage();

  if (page.isInitialLoading) {
    return (
      <div className="flex flex-1 items-center justify-center" data-testid="bridges-loading">
        <Loader2 className="size-5 animate-spin text-[color:var(--color-text-tertiary)]" />
      </div>
    );
  }

  if (page.fatalError) {
    return (
      <div className="flex flex-1 items-center justify-center" data-testid="bridges-error">
        <div className="flex flex-col items-center gap-2 text-center">
          <AlertCircle className="size-6 text-[color:var(--color-danger)]" />
          <p className="text-sm text-[color:var(--color-text-tertiary)]">
            {page.fatalError.message ?? "Failed to load bridges"}
          </p>
        </div>
      </div>
    );
  }

  return (
    <>
      <WorkspacePageShell
        title="Bridges"
        icon={<Waypoints className="size-4" />}
        count={page.totalBridgeCount}
        controls={
          <Pills
            data-testid="bridge-scope-pills"
            value={page.activeScope}
            onChange={page.selectScope}
            items={[
              { value: "all", label: "ALL", testId: "bridge-scope-all" },
              { value: "global", label: "GLOBAL", testId: "bridge-scope-global" },
              { value: "workspace", label: "WORKSPACE", testId: "bridge-scope-workspace" },
            ]}
          />
        }
        meta={
          <Button
            data-testid="create-bridge-btn"
            disabled={!page.canCreateBridge}
            onClick={page.openCreateDialog}
            size="lg"
            type="button"
          >
            <Plus className="size-4" />
            Bridge
          </Button>
        }
      >
        {page.totalBridgeCount === 0 ? (
          <BridgeEmptyState onCreate={page.openCreateDialog} providers={page.providers} />
        ) : (
          <>
            <BridgeListPanel {...page.listPanelProps} />
            <BridgeDetailPanel {...page.detailPanelProps} />
          </>
        )}
      </WorkspacePageShell>

      <BridgeCreateDialog {...page.createDialogProps} />
      <BridgeEditDialog {...page.editDialogProps} />
      <BridgeTestDeliveryDialog {...page.testDeliveryDialogProps} />
    </>
  );
}
