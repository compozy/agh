import { RefreshCw } from "lucide-react";

import {
  Button,
  Item,
  ItemActions,
  ItemContent,
  ItemGroup,
  ItemTitle,
  Pill,
  Spinner,
} from "@agh/ui";

import {
  modelRefreshStateTone,
  useProviderModelStatus,
  useRefreshProviderModels,
  type ProviderModelSourceStatus,
} from "@/systems/model-catalog";

interface ProviderModelCatalogStatusProps {
  providerId: string;
  testId: string;
}

export function ProviderModelCatalogStatus({
  providerId,
  testId,
}: ProviderModelCatalogStatusProps) {
  const statusQuery = useProviderModelStatus({ providerId });
  const refreshMutation = useRefreshProviderModels();

  if (statusQuery.isLoading) {
    return (
      <div className="flex items-center gap-2 text-xs text-subtle">
        <Spinner className="size-3" />
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
        <p className="text-xs text-danger" data-testid={`${testId}-error`}>
          {queryError}
        </p>
      ) : null}
      {sources.length === 0 && !queryError ? (
        <p className="text-xs text-subtle" data-testid={`${testId}-empty`}>
          No catalog sources reporting yet.
        </p>
      ) : (
        <ItemGroup
          className="flex flex-col gap-1 font-mono text-eyebrow text-muted"
          data-testid={`${testId}-list`}
        >
          {sources.map(source => (
            <Item
              key={source.source_id}
              className="gap-1.5 rounded-none border-0 p-0"
              size="xs"
              data-testid={`${testId}-source-${source.source_id}`}
            >
              <ItemContent className="min-w-0 flex-none">
                <ItemTitle className="text-eyebrow">
                  <span className="truncate">{source.source_id}</span>
                </ItemTitle>
              </ItemContent>
              <ItemActions className="flex-wrap gap-1.5">
                <Pill mono tone={modelRefreshStateTone(source.refresh_state)}>
                  {source.refresh_state}
                </Pill>
                {source.stale ? (
                  <Pill mono tone="warning">
                    stale
                  </Pill>
                ) : null}
                <span
                  className="text-subtle"
                  data-testid={`${testId}-source-${source.source_id}-rows`}
                >
                  {formatRowCount(source)}
                </span>
              </ItemActions>
            </Item>
          ))}
        </ItemGroup>
      )}
      {refreshError ? (
        <p className="text-xs text-danger" data-testid={`${testId}-refresh-error`}>
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
          className={refreshMutation.isPending ? "size-3 animate-spin" : "size-3"}
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
