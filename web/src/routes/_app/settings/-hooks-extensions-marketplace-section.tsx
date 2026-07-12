import { Download, Search, ShieldCheck } from "lucide-react";

import type { SettingsExtensionMarketplaceEntry } from "@/systems/settings";
import {
  Button,
  Empty,
  Eyebrow,
  Input,
  Pill,
  Section,
  Spinner,
  Switch,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@agh/ui";
import { TrustBadge } from "./-hooks-extensions-installed-section";

interface MarketplaceSectionProps {
  entries: SettingsExtensionMarketplaceEntry[];
  query: string;
  setQuery: (value: string) => void;
  allowUnverified: boolean;
  setAllowUnverified: (value: boolean) => void;
  pendingSlug: string | null;
  error: string | null;
  isLoading: boolean;
  canMutate: boolean;
  onSearch: () => void;
  onInstall: (entry: SettingsExtensionMarketplaceEntry) => void;
}

export function MarketplaceSection({
  entries,
  query,
  setQuery,
  allowUnverified,
  setAllowUnverified,
  pendingSlug,
  error,
  isLoading,
  canMutate,
  onSearch,
  onInstall,
}: MarketplaceSectionProps) {
  return (
    <Section
      data-testid="settings-page-hooks-extensions-marketplace-section"
      label="Extension marketplace"
      note="daemon-owned search and install"
      right={
        <label className="flex items-center gap-2 text-xs text-muted">
          <Switch
            aria-label="Allow unverified extension install"
            checked={allowUnverified}
            disabled={!canMutate}
            onCheckedChange={setAllowUnverified}
            data-testid="settings-page-hooks-extensions-marketplace-allow-unverified"
          />
          allow_unverified
        </label>
      }
    >
      <div className="flex flex-col gap-3">
        <div className="flex flex-col gap-2 sm:flex-row">
          <Input
            aria-label="Search extension marketplace"
            className="font-mono"
            data-testid="settings-page-hooks-extensions-marketplace-search-input"
            placeholder="owner/repo or bridge"
            value={query}
            onChange={event => setQuery(event.target.value)}
            onKeyDown={event => {
              if (event.key === "Enter" && !isLoading) {
                onSearch();
              }
            }}
          />
          <Button
            data-testid="settings-page-hooks-extensions-marketplace-search"
            disabled={isLoading}
            onClick={onSearch}
            type="button"
            variant="outline"
          >
            {isLoading ? <Spinner className="size-3.5" /> : <Search className="size-3.5" />}
            Search
          </Button>
        </div>
        {error ? (
          <span
            className="text-xs text-danger"
            data-testid="settings-page-hooks-extensions-marketplace-error"
          >
            {error}
          </span>
        ) : null}
        {isLoading && entries.length === 0 ? (
          <div
            className="flex items-center gap-2 text-xs text-subtle"
            data-testid="settings-page-hooks-extensions-marketplace-loading"
          >
            <Spinner className="size-3" />
            Loading marketplace…
          </div>
        ) : entries.length === 0 ? (
          <Empty
            icon={ShieldCheck}
            title="No marketplace entries"
            description="Search the configured registry for an installable extension slug."
            data-testid="settings-page-hooks-extensions-marketplace-empty"
          />
        ) : (
          <div
            className="overflow-hidden rounded-lg border border-line"
            data-testid="settings-page-hooks-extensions-marketplace-list"
          >
            <Table>
              <TableHeader>
                <TableRow className="bg-elevated">
                  <TableHead className="eyebrow text-muted">Extension</TableHead>
                  <TableHead className="eyebrow text-muted">Source</TableHead>
                  <TableHead className="eyebrow text-muted">Trust</TableHead>
                  <TableHead className="eyebrow w-[1%] text-right text-muted">Install</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {entries.map(entry => (
                  <MarketplaceRow
                    key={`${entry.source}:${entry.slug}`}
                    entry={entry}
                    pending={pendingSlug === entry.slug}
                    canMutate={canMutate}
                    onInstall={onInstall}
                  />
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </div>
    </Section>
  );
}

function MarketplaceRow({
  entry,
  pending,
  canMutate,
  onInstall,
}: {
  entry: SettingsExtensionMarketplaceEntry;
  pending: boolean;
  canMutate: boolean;
  onInstall: (entry: SettingsExtensionMarketplaceEntry) => void;
}) {
  return (
    <TableRow data-testid={`settings-page-hooks-extensions-marketplace-row-${entry.slug}`}>
      <TableCell>
        <div className="flex min-w-0 flex-col gap-0.5">
          <span className="truncate font-mono text-sm text-fg">{entry.name}</span>
          <span className="max-w-md truncate text-xs text-muted">
            {entry.description ?? entry.slug}
          </span>
        </div>
      </TableCell>
      <TableCell>
        <div className="flex flex-col gap-0.5">
          <span className="font-mono text-xs text-fg">{entry.source}</span>
          {entry.version ? <Eyebrow className="text-subtle">{entry.version}</Eyebrow> : null}
        </div>
      </TableCell>
      <TableCell>
        {entry.trust ? <TrustBadge trust={entry.trust} /> : <Pill mono>unknown</Pill>}
      </TableCell>
      <TableCell>
        <div className="flex justify-end">
          <Button
            data-testid={`settings-page-hooks-extensions-marketplace-row-${entry.slug}-install`}
            disabled={pending || !canMutate}
            onClick={() => onInstall(entry)}
            size="sm"
            type="button"
          >
            {pending ? <Spinner className="size-3.5" /> : <Download className="size-3.5" />}
            Install
          </Button>
        </div>
      </TableCell>
    </TableRow>
  );
}
