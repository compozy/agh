import { Loader2, RefreshCw } from "lucide-react";

import { Button, Pill, type PillTone } from "@agh/ui";

import {
  useProviderModelStatus,
  useRefreshProviderModels,
  type ProviderModelSourceStatus,
} from "@/systems/model-catalog";

interface ProviderModelCatalogStatusProps {
  providerId: string;
  testId: string;
}

const REFRESH_STATE_TONE: Record<string, PillTone> = {
  idle: "neutral",
  refreshing: "info",
  succeeded: "success",
  failed: "danger",
};

export function ProviderModelCatalogStatus({
  providerId,
  testId,
}: ProviderModelCatalogStatusProps) {
  const statusQuery = useProviderModelStatus({ providerId });
  const refreshMutation = useRefreshProviderModels();

  if (statusQuery.isLoading) {
    return (
      <div className="flex items-center gap-2 text-xs text-(--color-text-tertiary)">
        <Loader2 className="size-3.5 animate-spin" />
        <span data-testid={`${testId}-loading`}>Loading catalog status…</span>
      </div>
    );
  }

  const sources = statusQuery.data?.sources ?? [];
  const refreshError = errorMessage(refreshMutation.error);
  const queryError = errorMessage(statusQuery.error);

  const handleRefresh = () => {
    refreshMutation.mutate({ providerId, force: true });
  };

  return (
    <div className="flex flex-col gap-2" data-testid={testId}>
      {queryError ? (
        <p className="text-xs text-(--color-danger)" data-testid={`${testId}-error`}>
          {queryError}
        </p>
      ) : null}
      {sources.length === 0 && !queryError ? (
        <p className="text-xs text-(--color-text-tertiary)" data-testid={`${testId}-empty`}>
          No catalog sources reporting yet.
        </p>
      ) : (
        <ul
          className="flex flex-col gap-1 font-mono text-eyebrow text-(--color-text-secondary)"
          data-testid={`${testId}-list`}
        >
          {sources.map(source => (
            <li
              key={source.source_id}
              className="flex flex-wrap items-center gap-1.5"
              data-testid={`${testId}-source-${source.source_id}`}
            >
              <span className="truncate">{source.source_id}</span>
              <Pill mono tone={REFRESH_STATE_TONE[source.refresh_state] ?? "neutral"}>
                {source.refresh_state}
              </Pill>
              {source.stale ? (
                <Pill mono tone="warning">
                  stale
                </Pill>
              ) : null}
              <span
                className="text-(--color-text-tertiary)"
                data-testid={`${testId}-source-${source.source_id}-rows`}
              >
                {formatRowCount(source)}
              </span>
            </li>
          ))}
        </ul>
      )}
      {refreshError ? (
        <p className="text-xs text-(--color-danger)" data-testid={`${testId}-refresh-error`}>
          {refreshError}
        </p>
      ) : null}
      <Button
        type="button"
        variant="ghost"
        size="sm"
        onClick={handleRefresh}
        disabled={refreshMutation.isPending || statusQuery.isFetching}
        data-testid={`${testId}-refresh`}
      >
        <RefreshCw
          aria-hidden="true"
          className={refreshMutation.isPending ? "size-3.5 animate-spin" : "size-3.5"}
        />
        Refresh catalog
      </Button>
    </div>
  );
}

function formatRowCount(source: ProviderModelSourceStatus): string {
  return `${source.row_count} rows`;
}

function errorMessage(error: unknown): string | null {
  if (error instanceof Error && error.message.trim().length > 0) {
    return error.message;
  }
  return null;
}
