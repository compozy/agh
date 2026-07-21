import { cn, Pill } from "@agh/ui";

import { providerAuthSummary, providerStatusCopy } from "../lib/provider-copy";
import { getProviderStateView } from "../lib/provider-state";
import type { SettingsProviderEntry } from "../types";
import { ProviderLogo } from "./provider-logo";

interface ProviderCardProps {
  provider: SettingsProviderEntry;
  onOpen: (entry: SettingsProviderEntry) => void;
}

/**
 * Provider card (settings-providers prototype): logo well, name + Default
 * pill, command in mono, one humanized status line, and a plain-language
 * sign-in summary. The whole card opens the inspector sheet.
 */
export function ProviderCard({ provider, onOpen }: ProviderCardProps) {
  const state = getProviderStateView(provider);
  const curatedCount = (provider.settings.models?.curated ?? []).length;
  const status = providerStatusCopy(state, curatedCount);
  const testId = `settings-page-providers-card-${provider.name}`;
  const command = provider.settings.command?.trim() || provider.name;

  return (
    <button
      className={cn(
        "flex flex-col gap-3 rounded-lg border border-line bg-canvas-soft px-4 py-3.5 text-left",
        "transition-colors duration-base hover:border-line-strong hover:bg-elevated",
        "focus-visible:outline-none focus-visible:shadow-focus-ring"
      )}
      data-state={state.label}
      data-testid={testId}
      onClick={() => onOpen(provider)}
      type="button"
    >
      <div className="flex min-w-0 items-center gap-3">
        <span
          className="flex size-9 shrink-0 items-center justify-center rounded-md bg-canvas text-fg"
          data-testid={`${testId}-logo`}
        >
          <ProviderLogo className="size-4.5" provider={provider.name} />
        </span>
        <span className="flex min-w-0 flex-col">
          <span className="flex min-w-0 items-center gap-2">
            <span
              className="truncate text-ws-name font-semibold text-fg-strong"
              data-testid={`${testId}-name`}
            >
              {provider.settings.display_name?.trim() || provider.name}
            </span>
            {provider.default ? (
              <Pill data-testid={`${testId}-default`} tone="accent">
                Default
              </Pill>
            ) : null}
          </span>
          <span className="truncate font-mono text-eyebrow text-subtle">{command}</span>
        </span>
      </div>
      <span
        className={cn(
          "inline-flex items-center gap-1.5 text-form-label font-medium",
          status.tone === "success" ? "text-success" : "text-warning"
        )}
        data-state={state.label}
        data-testid={`${testId}-status`}
      >
        <Pill.Dot tone={status.tone} />
        {status.label}
      </span>
      <span className="flex items-center justify-between gap-2 border-t border-line-soft pt-2.5 text-form-label text-muted">
        {providerAuthSummary(provider)}
        <span className="text-form-label font-medium text-fg">
          {state.label === "unconfigured" ? "Set up" : "Edit"}
        </span>
      </span>
    </button>
  );
}
