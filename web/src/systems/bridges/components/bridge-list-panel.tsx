import { AlertCircle, Waypoints } from "lucide-react";
import { Link } from "@tanstack/react-router";

import {
  Button,
  CatalogCard,
  Empty,
  ListingRow,
  Pill,
  Spinner,
  type ListingViewMode,
} from "@agh/ui";

import {
  bridgeStatusLabel,
  bridgeStatusTone,
  formatBridgeRelativeTime,
} from "../lib/bridge-formatters";
import { effectiveBridgeStatus } from "../lib/bridge-list-filters";
import type { BridgeHealthMap, BridgeSummary } from "../types";

export interface BridgeListPanelProps {
  bridgeHealth: BridgeHealthMap;
  bridges: BridgeSummary[];
  view: ListingViewMode;
  hasActiveFilters?: boolean;
  hasNextPage?: boolean;
  isFetchingNextPage?: boolean;
  onClearFilters: () => void;
  onLoadMore?: () => void;
  isLoading?: boolean;
  errorMessage?: string | null;
}

interface BridgeRowProps {
  bridge: BridgeSummary;
  health?: BridgeHealthMap[string];
}

function BridgeMetaFacts({ bridge, health }: BridgeRowProps) {
  const facts: string[] = [];
  const relative = formatBridgeRelativeTime(health?.last_success_at);
  if (relative) {
    facts.push(relative);
  }
  facts.push(bridge.scope);
  if (health?.route_count !== undefined) {
    facts.push(`${health.route_count} routes`);
  }

  if (facts.length === 0) {
    return null;
  }

  return (
    <ListingRow.Meta>
      {facts.map((fact, index) => (
        <span className="inline-flex items-center gap-1.5" key={`${fact}-${index}`}>
          {index > 0 ? <ListingRow.MetaDot /> : null}
          <span className="font-mono text-badge text-subtle">{fact}</span>
        </span>
      ))}
    </ListingRow.Meta>
  );
}

function BridgeStatusTrail({ bridge, health }: BridgeRowProps) {
  const status = effectiveBridgeStatus(bridge, health);
  return (
    <>
      <Pill mono size="sm" tone="neutral">
        {bridge.platform}
      </Pill>
      <Pill mono size="sm" tone={bridgeStatusTone(status)}>
        {bridgeStatusLabel(status)}
      </Pill>
    </>
  );
}

function BridgeListingRow({ bridge, health }: BridgeRowProps) {
  return (
    <ListingRow data-bridge={bridge.id} data-testid={`bridge-item-${bridge.id}`}>
      <ListingRow.Link
        render={
          <Link
            aria-label={`Open ${bridge.display_name}`}
            params={{ id: bridge.id }}
            to="/bridges/$id"
          />
        }
      >
        <ListingRow.Icon>
          <Waypoints aria-hidden="true" className="size-4" />
        </ListingRow.Icon>
        <ListingRow.Main>
          <ListingRow.Name>
            <ListingRow.Title>{bridge.display_name}</ListingRow.Title>
            <ListingRow.Slug>{bridge.extension_name}</ListingRow.Slug>
          </ListingRow.Name>
          <BridgeMetaFacts bridge={bridge} health={health} />
        </ListingRow.Main>
      </ListingRow.Link>
      <ListingRow.Trail>
        <BridgeStatusTrail bridge={bridge} health={health} />
      </ListingRow.Trail>
    </ListingRow>
  );
}

function BridgeCatalogCard({ bridge, health }: BridgeRowProps) {
  const status = effectiveBridgeStatus(bridge, health);
  const relative = formatBridgeRelativeTime(health?.last_success_at);

  return (
    <CatalogCard actionable data-bridge={bridge.id} data-testid={`bridge-card-${bridge.id}`}>
      <Link
        aria-label={`Open ${bridge.display_name}`}
        className="flex min-w-0 flex-col gap-3"
        params={{ id: bridge.id }}
        to="/bridges/$id"
      >
        <div className="flex items-start gap-3">
          <CatalogCard.Logo>
            <Waypoints className="size-4" />
          </CatalogCard.Logo>
          <div className="flex min-w-0 flex-1 flex-col gap-1">
            <CatalogCard.Title>{bridge.display_name}</CatalogCard.Title>
            <CatalogCard.Meta>
              {relative ? <span>{relative}</span> : null}
              <span>{bridge.scope}</span>
              {health?.route_count !== undefined ? <span>{health.route_count} routes</span> : null}
            </CatalogCard.Meta>
          </div>
        </div>
      </Link>
      <CatalogCard.Actions className="justify-between">
        <Pill mono size="sm" tone="neutral">
          {bridge.platform}
        </Pill>
        <Pill mono size="sm" tone={bridgeStatusTone(status)}>
          {bridgeStatusLabel(status)}
        </Pill>
      </CatalogCard.Actions>
    </CatalogCard>
  );
}

function BridgeListPanel({
  bridgeHealth,
  bridges,
  view,
  hasActiveFilters = false,
  hasNextPage = false,
  isFetchingNextPage = false,
  onClearFilters,
  onLoadMore,
  isLoading = false,
  errorMessage = null,
}: BridgeListPanelProps) {
  const isEmpty = bridges.length === 0;

  if (isLoading && isEmpty) {
    return (
      <div
        className="flex min-h-60 items-center justify-center px-6 py-10"
        data-testid="bridge-list-loading"
      >
        <Spinner aria-hidden="true" className="size-5 text-subtle" />
      </div>
    );
  }

  if (errorMessage && isEmpty) {
    return (
      <div
        className="flex min-h-60 items-center justify-center p-4"
        data-testid="bridge-list-error"
      >
        <Empty
          className="max-w-sm"
          description={errorMessage}
          icon={AlertCircle}
          title="Unable to load bridges"
        />
      </div>
    );
  }

  if (isEmpty) {
    return (
      <div
        className="flex min-h-60 items-center justify-center p-4"
        data-testid="bridge-list-empty"
      >
        <Empty
          action={
            hasActiveFilters ? (
              <Button
                data-testid="bridge-list-clear-filters"
                onClick={onClearFilters}
                size="sm"
                type="button"
                variant="ghost"
              >
                Clear filters
              </Button>
            ) : undefined
          }
          className="max-w-sm"
          description={
            hasActiveFilters ? "Try clearing search or filters." : "No bridges are configured yet."
          }
          icon={Waypoints}
          title={hasActiveFilters ? "No bridges match" : "No bridges yet"}
        />
      </div>
    );
  }

  const catalog =
    view === "cards" ? (
      <div
        className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3"
        data-testid="bridge-list-card-grid"
      >
        {bridges.map(bridge => (
          <BridgeCatalogCard bridge={bridge} health={bridgeHealth[bridge.id]} key={bridge.id} />
        ))}
      </div>
    ) : (
      <div
        className="overflow-hidden rounded-lg border border-line bg-canvas-soft"
        data-testid="bridge-list-rows"
      >
        {bridges.map(bridge => (
          <BridgeListingRow bridge={bridge} health={bridgeHealth[bridge.id]} key={bridge.id} />
        ))}
      </div>
    );

  return (
    <div className="flex flex-col gap-3">
      {catalog}
      {errorMessage ? (
        <p className="text-caption text-danger" data-testid="bridge-list-page-error" role="alert">
          {errorMessage}
        </p>
      ) : null}
      {hasNextPage && onLoadMore ? (
        <div className="flex justify-center">
          <Button
            aria-busy={isFetchingNextPage}
            data-testid="bridge-list-load-more"
            disabled={isFetchingNextPage}
            onClick={onLoadMore}
            size="sm"
            type="button"
            variant="ghost"
          >
            {isFetchingNextPage ? "Loading more bridges…" : "Load more bridges"}
          </Button>
        </div>
      ) : null}
    </div>
  );
}

export { BridgeListPanel };
