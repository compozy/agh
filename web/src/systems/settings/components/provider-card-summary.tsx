import { Pill, type PillTone } from "@agh/ui";

import { useProviderModelStatus } from "@/systems/model-catalog";

import type { SettingsProviderEntry } from "../types";
import { SettingsSourceBadge } from "./settings-source-badge";

interface ProviderCardSummaryProps {
  provider: SettingsProviderEntry;
  testId: string;
}

export function ProviderCardSummary({ provider, testId }: ProviderCardSummaryProps) {
  const curated = (provider.settings.models?.curated ?? []).flatMap(model =>
    model.id ? [model.id] : []
  );
  const defaultModel = provider.settings.models?.default?.trim() ?? "";
  const authState = provider.auth_status?.state?.trim() ?? "";
  const authTone = authStateTone(authState);
  const showModelRow = defaultModel.length > 0 || curated.length > 0;
  const showAuthRow = authState.length > 0;
  const showCatalogRow = provider.command_available;

  return (
    <dl className="grid grid-cols-[5rem_minmax(0,1fr)] gap-x-3 gap-y-2 @[20rem]/card:grid-cols-[6rem_minmax(0,1fr)]">
      {showModelRow ? (
        <Row label="Model" testId={`${testId}-model-row`}>
          <span className="flex min-w-0 flex-wrap items-baseline gap-x-2">
            {defaultModel ? (
              <code className="truncate font-mono text-xs text-fg" data-testid={`${testId}-model`}>
                {defaultModel}
              </code>
            ) : (
              <span className="text-xs text-subtle">No default</span>
            )}
            {curated.length > 0 ? (
              <span className="text-xs text-subtle tabular-nums" data-testid={`${testId}-curated`}>
                +{curated.length} curated
              </span>
            ) : null}
          </span>
        </Row>
      ) : null}

      {showAuthRow ? (
        <Row label="Auth" testId={`${testId}-auth-row`}>
          {authTone === "warning" || authTone === "danger" ? (
            <Pill tone={authTone} mono data-testid={`${testId}-auth-state`}>
              {authState}
            </Pill>
          ) : (
            <code className="font-mono text-xs text-muted" data-testid={`${testId}-auth-state`}>
              {authState}
            </code>
          )}
        </Row>
      ) : null}

      {showCatalogRow ? (
        <Row label="Catalog" testId={`${testId}-catalog-row`}>
          <CatalogInline providerId={provider.name} testId={`${testId}-catalog`} />
        </Row>
      ) : null}

      <Row label="Source" testId={`${testId}-source-row`}>
        <SettingsSourceBadge
          data-testid={`${testId}-source`}
          source={provider.source_metadata.effective_source}
        />
      </Row>
    </dl>
  );
}

function Row({
  label,
  testId,
  children,
}: {
  label: string;
  testId: string;
  children: React.ReactNode;
}) {
  return (
    <div className="col-span-2 grid grid-cols-subgrid items-center" data-testid={testId}>
      <dt className="text-xs text-subtle">{label}</dt>
      <dd className="min-w-0">{children}</dd>
    </div>
  );
}

function CatalogInline({ providerId, testId }: { providerId: string; testId: string }) {
  const statusQuery = useProviderModelStatus({ providerId });
  const summary = deriveCatalogSummary({
    isLoading: statusQuery.isLoading,
    error: statusQuery.error,
    sources: statusQuery.data?.sources ?? [],
  });
  return (
    <span className="flex items-center gap-1.5" data-testid={testId}>
      <Pill.Dot tone={summary.tone} />
      <span className="text-xs text-muted tabular-nums">{summary.label}</span>
    </span>
  );
}

function deriveCatalogSummary({
  isLoading,
  error,
  sources,
}: {
  isLoading: boolean;
  error: unknown;
  sources: { refresh_state?: string; stale?: boolean }[];
}): { tone: PillTone; label: string } {
  if (isLoading) return { tone: "neutral", label: "loading…" };
  if (error) return { tone: "danger", label: "unavailable" };
  if (sources.length === 0) return { tone: "neutral", label: "no sources" };
  const total = sources.length;
  const failed = sources.filter(source => source.refresh_state === "failed").length;
  const stale = sources.filter(source => source.stale === true).length;
  if (failed > 0) return { tone: "danger", label: `${failed}/${total} failed` };
  if (stale > 0) return { tone: "warning", label: `${stale}/${total} stale` };
  return { tone: "success", label: `${total} fresh` };
}

function authStateTone(state: string): PillTone {
  switch (state) {
    case "missing_required":
    case "needs_login":
      return "warning";
    default:
      return "neutral";
  }
}
