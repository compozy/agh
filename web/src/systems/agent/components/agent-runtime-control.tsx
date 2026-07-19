import { Button } from "@agh/ui";

import { RuntimeSelector } from "@/systems/runtime";

import type { AgentPayload } from "../types";
import { useAgentRuntimeEditor } from "../hooks/use-agent-runtime-editor";

export interface AgentRuntimeControlProps {
  agent: AgentPayload;
  workspaceId: string | null;
}

/**
 * Topbar runtime editor for the agent detail route. Owns the immediate
 * Provider · Model · Reasoning mutation and surfaces pending/conflict/error
 * feedback inline beside the selector so the topbar stays single-line.
 */
export function AgentRuntimeControl({ agent, workspaceId }: AgentRuntimeControlProps) {
  const runtime = useAgentRuntimeEditor({ agent, workspaceId });
  const statusMessage = runtime.isPending
    ? { tone: "muted" as const, testId: "agent-detail-runtime-pending", text: "Updating runtime…" }
    : runtime.conflictMessage
      ? {
          tone: "warning" as const,
          testId: "agent-detail-runtime-conflict",
          text: runtime.conflictMessage,
        }
      : runtime.error
        ? { tone: "danger" as const, testId: "agent-detail-runtime-error", text: runtime.error }
        : runtime.providerSourceError
          ? {
              tone: "danger" as const,
              testId: "agent-detail-runtime-providers-error",
              text: runtime.providerSourceError,
            }
          : runtime.modelCatalogError
            ? {
                tone: "warning" as const,
                testId: "agent-detail-runtime-catalog-error",
                text: runtime.modelCatalogError,
              }
            : null;

  return (
    <div className="flex min-w-0 items-center gap-2" data-testid="agent-detail-runtime">
      {statusMessage ? (
        <span
          className={
            statusMessage.tone === "danger"
              ? "max-w-[24ch] truncate text-small-body text-danger"
              : statusMessage.tone === "warning"
                ? "max-w-[24ch] truncate text-small-body text-warning"
                : "max-w-[24ch] truncate text-small-body text-muted"
          }
          data-testid={statusMessage.testId}
          role={statusMessage.tone === "danger" ? "alert" : "status"}
          title={statusMessage.text}
        >
          {statusMessage.text}
        </span>
      ) : null}
      {runtime.providerSourceError ? (
        <Button
          data-testid="agent-detail-runtime-providers-retry"
          onClick={runtime.onRetryProviderSource}
          size="sm"
          type="button"
          variant="ghost"
        >
          Retry providers
        </Button>
      ) : null}
      <RuntimeSelector
        ariaLabelledby="agent-detail-runtime-label"
        catalogLoaded={runtime.modelCatalogLoaded}
        disabled={
          runtime.providersLoading || runtime.providerOptions.length === 0 || runtime.isPending
        }
        loading={runtime.modelCatalogLoading}
        models={runtime.runtimeModels}
        onChange={runtime.onChange}
        onOpenProviderSettings={runtime.onOpenProviderSettings}
        onRefreshCatalog={runtime.onRefreshCatalog}
        providers={runtime.providerOptions}
        refreshing={runtime.modelCatalogRefreshing}
        triggerId="agent-detail-runtime-trigger"
        triggerTestId="agent-detail-runtime-select"
        value={runtime.value}
        variant="default"
      />
      <span className="sr-only" id="agent-detail-runtime-label">
        Agent runtime
      </span>
    </div>
  );
}
