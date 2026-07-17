import { Box, Plug, Puzzle, Wrench } from "lucide-react";
import { Link } from "@tanstack/react-router";

import { Button, Pill, Spinner, buttonVariants } from "@agh/ui";

import type { MarketplaceEntryResponse, MarketplaceKind, MarketplaceListing } from "../types";
import { formatMarketplaceCount } from "./marketplace-ui";

interface MarketplaceDetailHeroProps {
  entry: MarketplaceEntryResponse["entry"];
  pending: boolean;
  onAction: (entry: MarketplaceListing) => void;
}

const KIND_ICON: Record<MarketplaceKind, typeof Wrench> = {
  skill: Wrench,
  extension: Puzzle,
  bundle: Box,
  mcp: Plug,
};

function MarketplaceDetailHero({ entry, pending, onAction }: MarketplaceDetailHeroProps) {
  const kind = entry.kind as MarketplaceKind;
  const Icon = KIND_ICON[kind] ?? Box;
  const blocked = kind === "extension" && entry.trust?.decision === "blocked";

  return (
    <header className="flex flex-col gap-3 border-b border-line py-5 md:flex-row md:items-start md:gap-3.5">
      <span className="grid size-(--size-provider-logo-well) shrink-0 place-items-center rounded-lg bg-elevated text-muted">
        <Icon aria-hidden="true" className="size-5" />
      </span>
      <div className="flex min-w-0 flex-1 flex-col gap-2">
        <div className="flex min-w-0 flex-wrap items-center gap-2">
          <h1 className="truncate text-detail-h1 font-semibold tracking-detail-h1 text-fg-strong">
            {entry.name}
          </h1>
          <Pill mono>{kind}</Pill>
          <MarketplaceTrustPill entry={entry} />
          {entry.version ? <Pill mono>v{entry.version}</Pill> : null}
        </div>
        <div className="flex flex-wrap items-center gap-2 text-small-body text-muted">
          {entry.author ? <span>{entry.author}</span> : null}
          <span>{entry.source}</span>
          {entry.downloads !== undefined && entry.downloads !== null ? (
            <span>{formatMarketplaceCount(entry.downloads)} downloads</span>
          ) : null}
        </div>
      </div>
      <div className="flex w-full flex-col items-start gap-1.5 md:w-auto md:items-end">
        <div className="flex flex-wrap items-center gap-2">
          {entry.installed && entry.manage_path ? (
            <a
              aria-label={`Manage ${entry.name}`}
              className={buttonVariants({ size: "sm", variant: "ghost" })}
              href={entry.manage_path}
            >
              Manage →
            </a>
          ) : null}
          {shouldShowDetailAction(entry) ? (
            <Button
              aria-disabled={blocked || undefined}
              aria-label={`${detailAction(entry)} ${entry.name}`}
              className={blocked ? "pointer-events-none opacity-50" : undefined}
              data-testid="marketplace-detail-action"
              disabled={pending}
              onClick={() => {
                if (!blocked) onAction(entry);
              }}
              size="sm"
              type="button"
              variant={blocked ? "neutral" : "default"}
            >
              {pending ? <Spinner aria-hidden="true" className="size-3" /> : null}
              {pending ? detailPendingAction(entry) : detailAction(entry)}
            </Button>
          ) : null}
        </div>
        {blocked ? (
          <p className="max-w-64 text-left text-form-hint text-danger md:text-right">
            Blocked by extensions policy.
            <Link className="ml-1 underline underline-offset-3" to="/settings/extensions">
              Settings › Extensions
            </Link>
          </p>
        ) : null}
      </div>
    </header>
  );
}

function MarketplaceTrustPill({ entry }: { entry: MarketplaceEntryResponse["entry"] }) {
  if (entry.kind === "mcp")
    return (
      <Pill mono tone="info">
        curated
      </Pill>
    );
  const trust = entry.trust;
  if (!trust) return null;
  if (trust.decision === "verified")
    return (
      <Pill mono tone="success">
        verified
      </Pill>
    );
  if (trust.decision === "blocked")
    return (
      <Pill mono tone="danger">
        blocked
      </Pill>
    );
  return (
    <Pill mono tone="warning">
      unverified · {trust.warnings?.length ?? 0}
    </Pill>
  );
}

function shouldShowDetailAction(entry: MarketplaceEntryResponse["entry"]): boolean {
  return !entry.installed || entry.update_available;
}

function detailAction(entry: MarketplaceEntryResponse["entry"]): string {
  if (entry.update_available) return "Update";
  if (entry.kind === "bundle") return "Activate";
  return "Install";
}

function detailPendingAction(entry: MarketplaceEntryResponse["entry"]): string {
  if (entry.update_available) return "Updating…";
  if (entry.kind === "bundle") return "Activating…";
  return "Installing…";
}

export { MarketplaceDetailHero };
