import { AlertCircle, SearchX } from "lucide-react";

import {
  Button,
  Empty,
  ListingPage,
  ListingToolbar,
  NativeSelect,
  NativeSelectOption,
  RouteState,
} from "@agh/ui";

import type { MarketplaceKind, MarketplaceKindResponse, MarketplaceListing } from "../types";
import { MarketplaceDegradedNotice } from "./marketplace-degraded-notice";
import { MarketplaceGrid, MarketplaceGridSkeleton } from "./marketplace-grid";
import {
  MARKETPLACE_KIND_LABEL,
  type MarketplaceViewSort,
  sortMarketplaceEntries,
} from "./marketplace-ui";

interface MarketplaceKindViewProps {
  data?: MarketplaceKindResponse;
  error?: Error | null;
  isLoading?: boolean;
  isEntryPending?: (entry: MarketplaceListing) => boolean;
  kind: MarketplaceKind;
  query: string;
  sort: MarketplaceViewSort;
  onAction: (entry: MarketplaceListing) => void;
  onClearSearch: () => void;
  onRetry: () => void;
  onSearchChange: (query: string) => void;
  onSortChange: (sort: MarketplaceViewSort) => void;
}

function MarketplaceKindView({
  data,
  error,
  isLoading = false,
  isEntryPending,
  kind,
  query,
  sort,
  onAction,
  onClearSearch,
  onRetry,
  onSearchChange,
  onSortChange,
}: MarketplaceKindViewProps) {
  const label = MARKETPLACE_KIND_LABEL[kind];
  const supportsDownloads = (data?.items ?? []).some(entry => entry.downloads != null);
  const effectiveSort = sort === "downloads" && !supportsDownloads ? "relevance" : sort;
  const entries = sortMarketplaceEntries(data?.items ?? [], effectiveSort);
  const count = data?.total ?? data?.items.length;
  const hasItems = entries.length > 0;
  // Envelope stale/partial OR a query error while a prior envelope is retained (v5 refetch fail).
  const isDegraded = Boolean(data && (data.stale || data.error || error));

  return (
    <ListingPage data-testid={`marketplace-kind-${kind}`}>
      <ListingPage.Head
        count={data ? count : "–"}
        meta={<span>{label} · browse and install</span>}
        title="Marketplace"
      />
      <ListingToolbar>
        <ListingToolbar.Leading>
          <ListingToolbar.Search
            aria-label={`Search ${label.toLowerCase()}`}
            containerClassName="w-full max-w-105"
            onChange={onSearchChange}
            placeholder={`Search ${label.toLowerCase()}`}
            value={query}
          />
        </ListingToolbar.Leading>
        <ListingToolbar.Trailing>
          <NativeSelect
            aria-label={`Sort ${label.toLowerCase()}`}
            onChange={event => onSortChange(event.target.value as MarketplaceViewSort)}
            size="sm"
            value={effectiveSort}
          >
            <NativeSelectOption value="relevance">Sort: Relevance</NativeSelectOption>
            {supportsDownloads ? (
              <NativeSelectOption value="downloads">Sort: Most downloaded</NativeSelectOption>
            ) : null}
            <NativeSelectOption value="name">Sort: Name A–Z</NativeSelectOption>
          </NativeSelect>
        </ListingToolbar.Trailing>
      </ListingToolbar>

      {error && !data ? (
        <RouteState
          action={<Button onClick={onRetry}>Retry</Button>}
          className="border-0 bg-transparent"
          cause={error.message}
          icon={AlertCircle}
          message={`The ${label.toLowerCase()} catalog could not be loaded.`}
          mode="error"
          title={`Unable to load ${label.toLowerCase()}`}
        />
      ) : null}
      {!error && isLoading && !data ? <MarketplaceGridSkeleton count={6} /> : null}
      {isDegraded ? (
        <MarketplaceDegradedNotice
          hasItems={hasItems}
          label={label}
          onRetry={onRetry}
          scope="kind"
        />
      ) : null}
      {!error && !isLoading && !isDegraded && entries.length === 0 ? (
        <Empty
          action={
            query ? (
              <Button onClick={onClearSearch} type="button" variant="ghost">
                Clear search
              </Button>
            ) : undefined
          }
          className="min-h-80"
          description={
            query
              ? `Nothing in ${label.toLowerCase()} matches “${query}”.`
              : `No ${label.toLowerCase()} are available from the configured catalog.`
          }
          fill={false}
          icon={SearchX}
          title={query ? `No ${label.toLowerCase()} match this query` : `No ${label.toLowerCase()}`}
        />
      ) : null}
      {hasItems ? (
        <MarketplaceGrid entries={entries} isEntryPending={isEntryPending} onAction={onAction} />
      ) : null}
      {data?.total !== undefined &&
      data.total !== null &&
      data.total > (data.items?.length ?? 0) ? (
        <p className="font-mono text-mono-id text-muted">
          Showing {data.items.length} of {data.total}
        </p>
      ) : null}
    </ListingPage>
  );
}

export { MarketplaceKindView };
export type { MarketplaceKindViewProps };
