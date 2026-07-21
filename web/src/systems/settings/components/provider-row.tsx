import { ChevronRight } from "lucide-react";

import { cn, Pill } from "@agh/ui";

import { getProviderStateView } from "../lib/provider-state";
import type { SettingsProviderEntry } from "../types";
import { providerStatusCopy } from "../lib/provider-copy";
import { ProviderLogo } from "./provider-logo";

interface ProviderRowProps {
  provider: SettingsProviderEntry;
  onOpen: (entry: SettingsProviderEntry) => void;
}

/** Rows view of the provider listing — same vocabulary as the cards, denser. */
export function ProviderRow({ provider, onOpen }: ProviderRowProps) {
  const state = getProviderStateView(provider);
  const curatedCount = (provider.settings.models?.curated ?? []).length;
  const status = providerStatusCopy(state, null);
  const testId = `settings-page-providers-row-${provider.name}`;
  const command = provider.settings.command?.trim() || provider.name;

  return (
    <button
      className={cn(
        "grid w-full grid-cols-[36px_minmax(0,1fr)_auto_14px] items-center gap-3 px-4 py-2.5 text-left",
        "border-t border-line-soft transition-colors duration-base first:border-t-0 hover:bg-row-hover",
        "focus-visible:outline-none focus-visible:shadow-focus-ring",
        "sm:grid-cols-[36px_minmax(0,1fr)_auto_72px_14px]"
      )}
      data-state={state.label}
      data-testid={testId}
      onClick={() => onOpen(provider)}
      type="button"
    >
      <span className="flex size-9 items-center justify-center rounded-md bg-canvas text-fg">
        <ProviderLogo className="size-4.5" provider={provider.name} />
      </span>
      <span className="flex min-w-0 flex-col">
        <span className="flex min-w-0 items-center gap-2">
          <span className="truncate text-ws-name font-semibold text-fg-strong">
            {provider.settings.display_name?.trim() || provider.name}
          </span>
          {provider.default ? <Pill tone="accent">Default</Pill> : null}
        </span>
        <span className="truncate font-mono text-eyebrow text-subtle">{command}</span>
      </span>
      <span
        className={cn(
          "inline-flex items-center gap-1.5 text-form-label font-medium",
          status.tone === "success" ? "text-success" : "text-warning"
        )}
        data-testid={`${testId}-status`}
      >
        <Pill.Dot tone={status.tone} />
        {status.label}
      </span>
      <span className="hidden text-right text-form-label tabular-nums text-muted sm:inline">
        {curatedCount > 0 ? `${curatedCount} models` : "—"}
      </span>
      <ChevronRight aria-hidden="true" className="size-3.5 text-faint" />
    </button>
  );
}
