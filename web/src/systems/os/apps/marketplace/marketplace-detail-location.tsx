import { useNavigate } from "@tanstack/react-router";

import { Button, RouteState, useTopbarSlot } from "@agh/ui";
import {
  MarketplaceApiError,
  MarketplaceDetail,
  MarketplaceDetailNotFound,
  MarketplaceDetailSkeleton,
  marketplaceRouteKindFor,
  useMarketplaceActionController,
  useMarketplaceEntry,
  type MarketplaceKind,
} from "@/systems/marketplace";
import { useActiveWorkspace } from "@/systems/workspace";
import type { MarketplaceDetailSearch } from "./marketplace-detail-search";

export function MarketplaceDetailLocation({
  kind,
  entryId,
  search,
}: {
  kind: MarketplaceKind;
  entryId: string;
  search: MarketplaceDetailSearch;
}) {
  return <MarketplaceDetailRouteBody entryId={entryId} kind={kind} search={search} />;
}

function MarketplaceDetailRouteBody({
  entryId,
  kind,
  search,
}: {
  entryId: string;
  kind: MarketplaceKind;
  search: MarketplaceDetailSearch;
}) {
  const navigate = useNavigate();
  const { activeWorkspaceId } = useActiveWorkspace();
  const workspaceId = search.scope === "global" ? null : (search.workspace_id ?? activeWorkspaceId);
  const managementScope = search.scope ?? (workspaceId ? "workspace" : "global");
  const query = useMarketplaceEntry(
    kind === "bundle"
      ? { entryId, kind, workspaceId }
      : { entryId, installedName: search.installed_name, kind, workspaceId }
  );
  const actions = useMarketplaceActionController(workspaceId);
  const entryName = query.data?.entry.name ?? entryId;

  useTopbarSlot({
    crumb: entryName,
  });

  if (query.isLoading) return <MarketplaceDetailSkeleton />;
  if (query.error instanceof MarketplaceApiError && query.error.status === 404) {
    return (
      <MarketplaceDetailNotFound
        onBack={() => void navigate({ to: `/marketplace/${marketplaceRouteKindFor(kind)}` })}
      />
    );
  }
  if (query.error || !query.data) {
    return (
      <div className="flex min-h-0 flex-1 items-center justify-center px-6 py-10">
        <RouteState
          action={<Button onClick={() => void query.refetch()}>Retry</Button>}
          cause={query.error?.message}
          message="The marketplace entry could not be loaded."
          mode="error"
          title="Unable to load this item"
        />
      </div>
    );
  }
  return (
    <>
      <MarketplaceDetail
        data={query.data}
        managementScope={managementScope}
        managementWorkspaceId={workspaceId ?? undefined}
        onAction={actions.handleAction}
        pending={actions.isEntryPending(query.data.entry)}
      />
      {actions.dialogs}
    </>
  );
}
