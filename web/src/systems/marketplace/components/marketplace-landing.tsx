import { Link } from "@tanstack/react-router";
import { AlertCircle, CircleMinus, Store } from "lucide-react";

import {
  Button,
  Empty,
  ListingPage,
  ListingToolbar,
  PageHead,
  RouteState,
  Section,
  Spinner,
  buttonVariants,
  type ListingViewMode,
} from "@agh/ui";

import {
  marketplaceRouteKindFor,
  type MarketplaceKind,
  type MarketplaceKindResult,
  type MarketplaceListing,
  type MarketplaceSearchResponse,
} from "../types";
import { MarketplaceDegradedNotice } from "./marketplace-degraded-notice";
import { MarketplaceGrid, MarketplaceGridSkeleton } from "./marketplace-grid";
import { MARKETPLACE_KIND_LABEL, MARKETPLACE_KIND_ORDER } from "./marketplace-ui";

interface MarketplaceLandingProps {
  data?: MarketplaceSearchResponse;
  error?: Error | null;
  isFetching?: boolean;
  isEntryPending?: (entry: MarketplaceListing) => boolean;
  isLoading?: boolean;
  query: string;
  view?: ListingViewMode;
  onAction: (entry: MarketplaceListing) => void;
  onClearSearch: () => void;
  onRetry: () => void;
  onSearchChange: (query: string) => void;
  onViewChange?: (view: ListingViewMode) => void;
}

function MarketplaceLanding({
  data,
  error,
  isFetching = false,
  isEntryPending,
  isLoading = false,
  query,
  view = "rows",
  onAction,
  onClearSearch,
  onRetry,
  onSearchChange,
  onViewChange,
}: MarketplaceLandingProps) {
  const results = marketplaceResultsByKind(data);
  const successfulItems = MARKETPLACE_KIND_ORDER.flatMap(kind => results.get(kind)?.items ?? []);
  const allZero = query.trim() !== "" && successfulItems.length === 0 && !hasSourceError(results);

  return (
    <ListingPage data-testid="marketplace-landing">
      <PageHead
        icon={Store}
        meta={<MarketplaceLandingMeta results={results} />}
        title="Marketplace"
      />
      <ListingToolbar>
        <ListingToolbar.Leading>
          <ListingToolbar.Search
            aria-label="Search the marketplace"
            containerClassName="w-full max-w-105"
            onChange={onSearchChange}
            placeholder="Search the marketplace"
            value={query}
          />
        </ListingToolbar.Leading>
        {onViewChange ? (
          <ListingToolbar.Trailing>
            <ListingToolbar.ViewToggle onChange={onViewChange} value={view} />
          </ListingToolbar.Trailing>
        ) : null}
      </ListingToolbar>

      <p aria-live="polite" className="sr-only" data-testid="marketplace-result-announcement">
        {marketplaceAnnouncement(results)}
      </p>

      {error && data ? <MarketplaceDegradedNotice onRetry={onRetry} scope="page" /> : null}
      {error && !data ? (
        <RouteState
          action={<Button onClick={onRetry}>Retry</Button>}
          className="border-0 bg-transparent"
          cause={error.message}
          icon={AlertCircle}
          message="The marketplace catalog could not be loaded."
          mode="error"
          title="Unable to load the marketplace"
        />
      ) : allZero ? (
        <Empty
          action={
            <Button onClick={onClearSearch} size="sm" type="button" variant="neutral">
              Clear search
            </Button>
          }
          description={`Nothing matches “${query}” across skills, extensions, bundles, or MCP servers.`}
          icon={CircleMinus}
          title="No matches in the marketplace"
        />
      ) : (
        <div className="flex flex-col gap-7">
          {MARKETPLACE_KIND_ORDER.map(kind => {
            const result = results.get(kind);
            return (
              <MarketplaceLandingSection
                isFetching={isFetching}
                isEntryPending={isEntryPending}
                isLoading={isLoading && !data}
                key={kind}
                kind={kind}
                onAction={onAction}
                onRetry={onRetry}
                query={query}
                result={result}
                view={view}
              />
            );
          })}
        </div>
      )}
    </ListingPage>
  );
}

interface MarketplaceLandingSectionProps {
  isFetching: boolean;
  isEntryPending?: (entry: MarketplaceListing) => boolean;
  isLoading: boolean;
  kind: MarketplaceKind;
  query: string;
  result?: MarketplaceKindResult;
  view: ListingViewMode;
  onAction: (entry: MarketplaceListing) => void;
  onRetry: () => void;
}

function MarketplaceLandingSection({
  isFetching,
  isEntryPending,
  isLoading,
  kind,
  query,
  result,
  view,
  onAction,
  onRetry,
}: MarketplaceLandingSectionProps) {
  const entries = result?.items.slice(0, 3) ?? [];
  const displayCount = result?.total ?? entries.length;
  const label = MARKETPLACE_KIND_LABEL[kind];
  const hasItems = entries.length > 0;
  const isDegraded = Boolean(result?.error) || Boolean(result?.stale);

  return (
    <Section
      count={isLoading ? undefined : displayCount}
      data-testid={`marketplace-section-${kind}`}
      headClassName="flex-row items-start justify-between"
      label={label}
      right={
        isLoading ? undefined : (
          <Link
            aria-label={`View all ${label.toLowerCase()} results`}
            className={buttonVariants({ size: "sm", variant: "ghost" })}
            search={{ kind: marketplaceRouteKindFor(kind), q: query || undefined }}
            to="/marketplace"
          >
            View all →
          </Link>
        )
      }
      rightClassName="w-auto shrink-0"
    >
      {isLoading ? <MarketplaceGridSkeleton count={3} view={view} /> : null}
      {!isLoading && isDegraded ? (
        <MarketplaceDegradedNotice
          hasItems={hasItems}
          label={label}
          onRetry={onRetry}
          scope="kind"
        />
      ) : null}
      {!isLoading && hasItems ? (
        <div className={isFetching ? "opacity-60" : undefined}>
          <MarketplaceGrid
            entries={entries}
            isEntryPending={isEntryPending}
            onAction={onAction}
            view={view}
          />
        </div>
      ) : null}
      {!isLoading && !isDegraded && !hasItems ? (
        <div className="flex items-center gap-2 text-small-body text-muted">
          <span>{query ? "0 results" : `No ${label.toLowerCase()} are available.`}</span>
          {isFetching ? <Spinner aria-hidden="true" className="size-3" /> : null}
        </div>
      ) : null}
    </Section>
  );
}

function marketplaceResultsByKind(data?: MarketplaceSearchResponse) {
  return new Map(
    (data?.kinds ?? []).map(result => [result.kind as MarketplaceKind, result] as const)
  );
}

function hasSourceError(results: Map<MarketplaceKind, MarketplaceKindResult>): boolean {
  return MARKETPLACE_KIND_ORDER.some(kind => Boolean(results.get(kind)?.error));
}

function MarketplaceLandingMeta({
  results,
}: {
  results: Map<MarketplaceKind, MarketplaceKindResult>;
}) {
  const totals = MARKETPLACE_KIND_ORDER.flatMap(kind => {
    const total = results.get(kind)?.total;
    return total === undefined || total === null
      ? []
      : [{ kind, label: MARKETPLACE_KIND_LABEL[kind].toLowerCase(), total }];
  });
  if (totals.length === 0) return <span>Browse and install runtime capabilities.</span>;
  return totals.map((item, index) => (
    <span className="contents" key={item.kind}>
      {index > 0 ? <PageHead.MetaDot /> : null}
      <span>
        {item.total} {item.label}
      </span>
    </span>
  ));
}

function marketplaceAnnouncement(results: Map<MarketplaceKind, MarketplaceKindResult>): string {
  return MARKETPLACE_KIND_ORDER.map(
    kind => `${MARKETPLACE_KIND_LABEL[kind]}: ${announceKind(results.get(kind))}`
  ).join(". ");
}

function announceKind(result?: MarketplaceKindResult): string {
  const count = result?.items.length ?? 0;
  if (result?.error && count === 0) return "unavailable";
  if (result?.stale || result?.error) return `${count} (may be out of date)`;
  return String(count);
}

export { MarketplaceLanding };
export type { MarketplaceLandingProps };
