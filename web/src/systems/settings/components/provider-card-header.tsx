import { Eyebrow, Pill } from "@agh/ui";

import type { ProviderStateView } from "../lib/provider-state";
import type { SettingsProviderEntry } from "../types";
import { ProviderLogo } from "./provider-logo";

interface ProviderCardHeaderProps {
  provider: SettingsProviderEntry;
  state: ProviderStateView;
  testId: string;
}

export function ProviderCardHeader({ provider, state, testId }: ProviderCardHeaderProps) {
  return (
    <header className="flex items-start gap-3 px-4 pt-4">
      <span
        className="flex size-10 shrink-0 items-center justify-center rounded-icon-well bg-canvas-soft text-fg"
        data-testid={`${testId}-logo`}
      >
        <ProviderLogo provider={provider.name} className="size-5" />
      </span>
      <div className="flex min-w-0 flex-1 flex-col gap-0.5">
        <div className="flex min-w-0 items-center gap-2">
          <h3
            className="truncate font-mono text-sm font-medium text-fg-strong"
            data-testid={`${testId}-name`}
          >
            {provider.name}
          </h3>
          {provider.default ? (
            <Pill tone="accent" data-testid={`${testId}-default`}>
              DEFAULT
            </Pill>
          ) : null}
        </div>
        {provider.settings.display_name ? (
          <span className="truncate text-xs text-muted" data-testid={`${testId}-display-name`}>
            {provider.settings.display_name}
          </span>
        ) : null}
      </div>
      <span
        className="flex shrink-0 items-center gap-1.5"
        data-testid={`${testId}-status`}
        data-state={state.label}
      >
        <Pill.Dot tone={state.tone} />
        <Eyebrow className="text-subtle">{state.display}</Eyebrow>
      </span>
    </header>
  );
}
