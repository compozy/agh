import { Link } from "@tanstack/react-router";

import { ListingRow, Pill } from "@agh/ui";

import type { MarketplaceKind, MarketplaceListing } from "../types";
import { MarketplaceEntryAction, MarketplaceEntryStatus } from "./marketplace-entry-actions";
import {
  MARKETPLACE_KIND_SINGULAR,
  formatMarketplaceCount,
  marketplaceKindIcon,
} from "./marketplace-ui";

interface MarketplaceRowProps {
  entry: MarketplaceListing;
  pending?: boolean;
  onAction: (entry: MarketplaceListing) => void;
}

function MarketplaceRow({ entry, pending = false, onAction }: MarketplaceRowProps) {
  const kind = entry.kind as MarketplaceKind;
  const KindGlyph = marketplaceKindIcon(kind);

  return (
    <ListingRow data-testid={`marketplace-row-${entry.entry_id}`}>
      <ListingRow.Link
        render={
          <Link
            aria-disabled={pending || undefined}
            aria-label={`View ${entry.name} details`}
            onClick={event => {
              if (pending) event.preventDefault();
            }}
            params={{ entryId: entry.entry_id, kind }}
            tabIndex={pending ? -1 : undefined}
            to="/marketplace/$kind/$entryId"
          />
        }
      >
        <ListingRow.Icon>
          <KindGlyph aria-hidden="true" className="size-4" />
        </ListingRow.Icon>
        <ListingRow.Main>
          <ListingRow.Name>
            <ListingRow.Title>{entry.name}</ListingRow.Title>
            <Pill mono size="xs" tone="neutral">
              {MARKETPLACE_KIND_SINGULAR[kind]}
            </Pill>
            {entry.version ? <ListingRow.Slug>v{entry.version}</ListingRow.Slug> : null}
          </ListingRow.Name>
          {entry.description ? (
            <ListingRow.Description>{entry.description}</ListingRow.Description>
          ) : null}
          <ListingRow.Meta>
            {entry.author ? <span>{entry.author}</span> : null}
            {entry.author && entry.downloads != null ? <ListingRow.MetaDot /> : null}
            {entry.downloads != null ? (
              <span className="font-mono text-badge tabular-nums text-subtle">
                {formatMarketplaceCount(entry.downloads)} downloads
              </span>
            ) : null}
            {(entry.author || entry.downloads != null) && entry.transport ? (
              <ListingRow.MetaDot />
            ) : null}
            {entry.transport ? (
              <span className="font-mono text-badge text-subtle">{entry.transport}</span>
            ) : null}
            {(entry.author || entry.downloads != null || entry.transport) && entry.tier ? (
              <ListingRow.MetaDot />
            ) : null}
            {entry.tier ? <span>{entry.tier}</span> : null}
          </ListingRow.Meta>
        </ListingRow.Main>
      </ListingRow.Link>
      <ListingRow.Trail className="gap-3">
        <MarketplaceEntryStatus entry={entry} />
        <MarketplaceEntryAction entry={entry} onAction={onAction} pending={pending} />
      </ListingRow.Trail>
    </ListingRow>
  );
}

export { MarketplaceRow };
export type { MarketplaceRowProps };
