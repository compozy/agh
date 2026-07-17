import { Link } from "@tanstack/react-router";
import { Box, Plug, Puzzle, Wrench } from "lucide-react";

import {
  Button,
  CatalogCard,
  Pill,
  Spinner,
  Tooltip,
  TooltipContent,
  TooltipTrigger,
  buttonVariants,
} from "@agh/ui";

import type { MarketplaceKind, MarketplaceListing } from "../types";
import { formatMarketplaceCount } from "./marketplace-ui";

interface MarketplaceCardProps {
  entry: MarketplaceListing;
  pending?: boolean;
  onAction: (entry: MarketplaceListing) => void;
}

const KIND_ICON: Record<MarketplaceKind, typeof Wrench> = {
  skill: Wrench,
  extension: Puzzle,
  bundle: Box,
  mcp: Plug,
};

function MarketplaceCard({ entry, pending = false, onAction }: MarketplaceCardProps) {
  const kind = entry.kind as MarketplaceKind;
  const Icon = KIND_ICON[kind] ?? Box;
  const blocked = kind === "extension" && entry.trust?.decision === "blocked";
  const action = marketplaceCardAction(entry);

  return (
    <CatalogCard data-testid={`marketplace-card-${entry.entry_id}`} actionable={!pending}>
      <Link
        aria-disabled={pending || undefined}
        aria-label={`View ${entry.name} details`}
        className="flex min-w-0 flex-col gap-3 rounded focus-visible:outline-none focus-visible:shadow-focus-ring"
        onClick={event => {
          if (pending) event.preventDefault();
        }}
        params={{ entryId: entry.entry_id, kind }}
        tabIndex={pending ? -1 : undefined}
        to="/marketplace/$kind/$entryId"
      >
        <div className="flex min-w-0 items-start gap-3">
          <CatalogCard.Logo tone={kind === "extension" ? "neutral" : "accent"}>
            <Icon aria-hidden="true" className="size-3.5" />
          </CatalogCard.Logo>
          <div className="min-w-0 flex-1">
            <CatalogCard.Title>{entry.name}</CatalogCard.Title>
            <CatalogCard.Meta>
              {entry.author ? <span>{entry.author}</span> : null}
              {entry.version ? (
                <span className="normal-case font-mono tracking-normal">{`v${entry.version}`}</span>
              ) : null}
              {entry.downloads !== undefined && entry.downloads !== null ? (
                <span className="normal-case tabular-nums">
                  {formatMarketplaceCount(entry.downloads)}
                </span>
              ) : null}
              {entry.transport ? (
                <span className="normal-case font-mono tracking-normal">{entry.transport}</span>
              ) : null}
              {entry.tier ? <span>{entry.tier}</span> : null}
            </CatalogCard.Meta>
          </div>
        </div>
        <CatalogCard.Description className="line-clamp-2 min-h-12">
          {entry.description}
        </CatalogCard.Description>
      </Link>

      <CatalogCard.Actions>
        <MarketplaceCardStatus entry={entry} />
        <div className="ml-auto">
          {entry.installed && !entry.update_available && entry.manage_path ? (
            <a
              aria-label={`Manage ${entry.name}`}
              className={buttonVariants({ size: "sm", variant: "ghost" })}
              href={entry.manage_path}
            >
              Manage
            </a>
          ) : blocked ? (
            <Tooltip>
              <TooltipTrigger
                render={
                  <Button
                    aria-disabled="true"
                    aria-label={`Install ${entry.name}, blocked by extensions policy`}
                    data-testid={`marketplace-action-${entry.entry_id}`}
                    onClick={event => event.preventDefault()}
                    size="sm"
                    type="button"
                    variant="neutral"
                  />
                }
              >
                Install
              </TooltipTrigger>
              <TooltipContent>Blocked by extensions policy</TooltipContent>
            </Tooltip>
          ) : (
            <Button
              aria-label={`${action} ${entry.name}`}
              data-testid={`marketplace-action-${entry.entry_id}`}
              disabled={pending}
              onClick={() => onAction(entry)}
              size="sm"
              type="button"
              variant="neutral"
            >
              {pending ? <Spinner aria-hidden="true" className="size-3" /> : null}
              {pending ? marketplacePendingLabel(entry) : action}
            </Button>
          )}
        </div>
      </CatalogCard.Actions>
    </CatalogCard>
  );
}

function MarketplaceCardStatus({ entry }: { entry: MarketplaceListing }) {
  if (entry.update_available) {
    return (
      <Pill mono tone="warning">
        {entry.version ? `v${entry.version} available` : "update available"}
      </Pill>
    );
  }
  if (entry.installed) {
    return (
      <Pill mono tone="success">
        installed
      </Pill>
    );
  }
  if (entry.kind === "extension" && entry.trust) {
    const warningCount = entry.trust.warnings?.length ?? 0;
    if (entry.trust.decision === "verified") {
      return (
        <Pill mono tone="success">
          verified
        </Pill>
      );
    }
    return (
      <Pill mono tone={entry.trust.decision === "blocked" ? "danger" : "warning"}>
        {entry.trust.decision === "blocked" ? "blocked" : "unverified"}
        {warningCount > 0 ? ` · ${warningCount}` : ""}
      </Pill>
    );
  }
  if (entry.kind === "mcp") {
    return (
      <Pill mono tone="info">
        curated
      </Pill>
    );
  }
  return null;
}

function marketplaceCardAction(entry: MarketplaceListing): string {
  if (entry.update_available) return "Update";
  if (entry.kind === "bundle") return "Activate";
  return "Install";
}

function marketplacePendingLabel(entry: MarketplaceListing): string {
  if (entry.update_available) return "Updating…";
  if (entry.kind === "bundle") return "Activating…";
  return "Installing…";
}

export { MarketplaceCard };
export type { MarketplaceCardProps };
