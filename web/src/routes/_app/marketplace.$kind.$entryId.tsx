import { Link, createFileRoute, useNavigate } from "@tanstack/react-router";
import { ChevronRight } from "lucide-react";

import { Button, RouteState, useTopbarSlot } from "@agh/ui";
import {
  MarketplaceApiError,
  MarketplaceDetail,
  MarketplaceDetailNotFound,
  MarketplaceDetailSkeleton,
  MARKETPLACE_KIND_LABEL,
  isMarketplaceKind,
  marketplaceRouteKindFor,
  useMarketplaceActionController,
  useMarketplaceEntry,
  type MarketplaceKind,
} from "@/systems/marketplace";
import { useActiveWorkspace } from "@/systems/workspace";
import type { TopbarRouteContext } from "@/types/topbar";

export const Route = createFileRoute("/_app/marketplace/$kind/$entryId")({
  beforeLoad: ({ params }): { topbar: TopbarRouteContext } => ({
    topbar: { title: params.entryId },
  }),
  component: MarketplaceDetailRoute,
});

function MarketplaceDetailRoute() {
  const { kind, entryId } = Route.useParams();
  const navigate = useNavigate();
  if (!isMarketplaceKind(kind)) {
    return (
      <MarketplaceDetailNotFound onBack={() => void navigate({ search: {}, to: "/marketplace" })} />
    );
  }
  return <MarketplaceDetailRouteBody entryId={entryId} kind={kind} />;
}

function MarketplaceDetailRouteBody({ entryId, kind }: { entryId: string; kind: MarketplaceKind }) {
  const navigate = useNavigate();
  const { activeWorkspaceId } = useActiveWorkspace();
  const query = useMarketplaceEntry({ entryId, kind, workspaceId: activeWorkspaceId });
  const actions = useMarketplaceActionController(activeWorkspaceId);
  const entryName = query.data?.entry.name ?? entryId;

  useTopbarSlot({
    title: (
      <span
        aria-label={`Marketplace, ${MARKETPLACE_KIND_LABEL[kind]}, ${entryName}`}
        className="flex min-w-0 items-center gap-1.5 text-small-body font-normal text-muted"
      >
        <Link className="shrink-0 hover:text-fg" search={{}} to="/marketplace">
          Marketplace
        </Link>
        <ChevronRight aria-hidden="true" className="size-3 shrink-0 text-faint" />
        <Link
          className="shrink-0 hover:text-fg"
          search={{ kind: marketplaceRouteKindFor(kind) }}
          to="/marketplace"
        >
          {MARKETPLACE_KIND_LABEL[kind]}
        </Link>
        <ChevronRight aria-hidden="true" className="size-3 shrink-0 text-faint" />
        <span className="truncate font-medium text-fg-strong">{entryName}</span>
      </span>
    ),
  });

  if (query.isLoading) return <MarketplaceDetailSkeleton />;
  if (query.error instanceof MarketplaceApiError && query.error.status === 404) {
    return (
      <MarketplaceDetailNotFound
        onBack={() =>
          void navigate({ search: { kind: marketplaceRouteKindFor(kind) }, to: "/marketplace" })
        }
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
        onAction={actions.handleAction}
        pending={actions.isEntryPending(query.data.entry)}
      />
      {actions.dialogs}
    </>
  );
}
