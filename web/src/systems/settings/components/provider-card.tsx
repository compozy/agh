import { ChevronRight } from "lucide-react";

import { Button, Card, CardContent } from "@agh/ui";

import { getProviderStateView } from "../lib/provider-state";
import type { SettingsProviderEntry } from "../types";
import { ProviderCardHeader } from "./provider-card-header";
import { ProviderCardSummary } from "./provider-card-summary";

interface ProviderCardProps {
  provider: SettingsProviderEntry;
  onOpen: (entry: SettingsProviderEntry) => void;
}

export function ProviderCard({ provider, onOpen }: ProviderCardProps) {
  const state = getProviderStateView(provider);
  const testId = `settings-page-providers-card-${provider.name}`;

  return (
    <Card
      data-testid={testId}
      data-state={state.label}
      className="@container/card flex flex-col gap-3 p-0 transition-colors duration-base ease-out hover:bg-hover"
    >
      <ProviderCardHeader provider={provider} state={state} testId={testId} />
      <CardContent className="flex flex-1 flex-col gap-3 px-4 pb-3">
        {state.hint ? (
          <p className="text-xs text-warning" data-testid={`${testId}-hint`}>
            {state.hint}
          </p>
        ) : null}
        <ProviderCardSummary provider={provider} testId={testId} />
      </CardContent>
      <div className="flex items-center justify-end border-t border-line-soft px-4 py-2">
        <Button
          type="button"
          variant="ghost"
          size="sm"
          onClick={() => onOpen(provider)}
          data-testid={`${testId}-open`}
        >
          Open
          <ChevronRight aria-hidden="true" className="size-3" />
        </Button>
      </div>
    </Card>
  );
}
