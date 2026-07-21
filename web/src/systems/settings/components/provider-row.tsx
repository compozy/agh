import { ChevronRight } from "lucide-react";

import { cn, ListingRow, Pill } from "@agh/ui";

import { providerListingView } from "../lib/provider-listing-view";
import type { SettingsProviderEntry } from "../types";
import { ProviderLogo } from "./provider-logo";

interface ProviderRowProps {
  provider: SettingsProviderEntry;
  onOpen: (entry: SettingsProviderEntry) => void;
}

export function ProviderRow({ provider, onOpen }: ProviderRowProps) {
  const view = providerListingView(provider);
  const testId = `settings-page-providers-row-${provider.name}`;

  return (
    <ListingRow data-state={view.state.label}>
      <ListingRow.Link
        aria-label={`Open ${view.displayName}`}
        data-testid={testId}
        onClick={() => onOpen(provider)}
        render={<button type="button" />}
      >
        <ListingRow.Icon>
          <ProviderLogo className="size-4.5" provider={provider.name} />
        </ListingRow.Icon>
        <ListingRow.Main>
          <ListingRow.Name>
            <ListingRow.Title>{view.displayName}</ListingRow.Title>
            {provider.default ? <Pill tone="accent">Default</Pill> : null}
          </ListingRow.Name>
          <ListingRow.Description className="font-mono text-eyebrow">
            {view.command}
          </ListingRow.Description>
        </ListingRow.Main>
      </ListingRow.Link>
      <ListingRow.Trail>
        <span
          className={cn(
            "inline-flex items-center gap-1.5 text-form-label font-medium",
            view.status.tone === "success"
              ? "text-success"
              : view.status.tone === "danger"
                ? "text-danger"
                : view.status.tone === "info"
                  ? "text-info"
                  : "text-warning"
          )}
          data-testid={`${testId}-status`}
        >
          <Pill.Dot tone={view.status.tone} />
          {view.status.label}
        </span>
        <ListingRow.Stat className="hidden sm:flex">
          <ListingRow.Stat.Value>{view.modelCount || "—"}</ListingRow.Stat.Value>
          <ListingRow.Stat.Label>models</ListingRow.Stat.Label>
        </ListingRow.Stat>
        <ChevronRight aria-hidden="true" className="size-3.5 text-faint" />
      </ListingRow.Trail>
    </ListingRow>
  );
}
