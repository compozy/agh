import { DetailHeader, KindIcon, Pill, providerKindIconRegistry } from "@agh/ui";

import { RuntimeSelector } from "@/systems/runtime";

import { formatAgentOriginLabel, formatCategoryMetaSegment } from "../lib/agent-fleet-projection";
import type { AgentPayload } from "../types";
import { useAgentRuntimeEditor } from "../hooks/use-agent-runtime-editor";
import { AgentPageStatusPill } from "./agent-page-header";

export interface AgentDetailHeaderProps {
  agent: AgentPayload;
  activeCount: number;
  workspaceId: string | null;
  /** Hide Active/Idle status when catalog metrics are loading or unavailable. */
  metricsUnavailable?: boolean;
}

export function AgentDetailHeader({
  agent,
  activeCount,
  workspaceId,
  metricsUnavailable = false,
}: AgentDetailHeaderProps) {
  const runtime = useAgentRuntimeEditor({ agent, workspaceId });
  const category = formatCategoryMetaSegment(agent.category_path);
  const origin = formatAgentOriginLabel(agent.origin);
  const hasDiagnostics = Array.isArray(agent.diagnostics) && agent.diagnostics.length > 0;

  return (
    <div
      className="flex flex-col gap-3 border-b border-line px-6 pb-5 pt-5"
      data-testid="agent-detail-header"
    >
      <DetailHeader
        className="border-0 p-0"
        title={
          <span className="inline-flex min-w-0 items-center gap-3.5">
            <span
              aria-hidden="true"
              className="grid size-[42px] shrink-0 place-items-center rounded-lg bg-elevated text-muted shadow-highlight"
              data-testid="agent-detail-header-icon"
            >
              <KindIcon
                className="size-5"
                kind={agent.provider}
                registry={providerKindIconRegistry}
                size="sm"
                tone="default"
              />
            </span>
            <span className="truncate" data-testid="agent-detail-header-name">
              {agent.name}
            </span>
          </span>
        }
        pills={
          <div className="flex flex-wrap items-center gap-1.5">
            {!metricsUnavailable ? <AgentPageStatusPill activeCount={activeCount} /> : null}
            {hasDiagnostics ? (
              <Pill tone="warning" size="sm" data-testid="agent-detail-header-invalid">
                Invalid
              </Pill>
            ) : null}
          </div>
        }
        meta={
          <div
            className="flex flex-wrap items-center gap-2 text-form-label text-muted"
            data-testid="agent-detail-header-meta"
          >
            {category ? <span className="truncate">{category}</span> : null}
            {category && origin ? (
              <span aria-hidden="true" className="size-0.5 rounded-full bg-faint" />
            ) : null}
            {origin ? (
              <span className="font-mono text-badge tracking-mono text-muted">{origin}</span>
            ) : null}
          </div>
        }
        actions={
          <div className="flex min-w-0 flex-col items-end gap-1.5">
            <RuntimeSelector
              value={runtime.value}
              onChange={runtime.onChange}
              providers={runtime.providerOptions}
              models={runtime.runtimeModels}
              variant="default"
              loading={runtime.modelCatalogLoading}
              catalogLoaded={runtime.modelCatalogLoaded}
              refreshing={runtime.modelCatalogRefreshing}
              onRefreshCatalog={runtime.onRefreshCatalog}
              onOpenProviderSettings={runtime.onOpenProviderSettings}
              disabled={
                runtime.providersLoading ||
                runtime.providerOptions.length === 0 ||
                runtime.isPending
              }
              ariaLabelledby="agent-detail-runtime-label"
              triggerId="agent-detail-runtime-trigger"
              triggerTestId="agent-detail-runtime-select"
            />
            <span id="agent-detail-runtime-label" className="sr-only">
              Agent runtime
            </span>
            {runtime.isPending ? (
              <p className="text-small-body text-muted" data-testid="agent-detail-runtime-pending">
                Updating runtime…
              </p>
            ) : null}
            {runtime.conflictMessage ? (
              <p
                className="max-w-sm text-right text-small-body text-warning"
                role="status"
                data-testid="agent-detail-runtime-conflict"
              >
                {runtime.conflictMessage}
              </p>
            ) : null}
            {runtime.error ? (
              <p
                className="max-w-sm text-right text-small-body text-danger"
                role="alert"
                data-testid="agent-detail-runtime-error"
              >
                {runtime.error}
              </p>
            ) : null}
            {runtime.modelCatalogError ? (
              <p className="max-w-sm text-right text-small-body text-warning" role="status">
                {runtime.modelCatalogError}
              </p>
            ) : null}
          </div>
        }
      />
    </div>
  );
}
