import { createFileRoute } from "@tanstack/react-router";
import { AlertCircle, ExternalLink, Settings as SettingsIcon } from "lucide-react";
import { useCallback, useMemo, useState, type Dispatch, type SetStateAction } from "react";

import { useSettingsGeneralPage } from "@/hooks/routes/use-settings-general-page";
import type { SettingsGeneralSection, SettingsUpdateStatus } from "@/systems/settings";
import {
  SettingsApplyRecordsPanel,
  SettingsFieldRow,
  SettingsNumberInput,
  SettingsSaveBar,
} from "@/systems/settings/components";
import { restartBannerPropsFor } from "@/systems/settings/lib/restart-banner-mapper";
import type { TopbarRouteContext } from "@/types/topbar";
import {
  Button,
  Eyebrow,
  Input,
  Metric,
  MetricGrid,
  PageShell,
  PillGroup,
  RestartBanner,
  Section,
  Spinner,
  StatusLineTopbarSlot,
  useTopbarSlot,
} from "@agh/ui";

export const Route = createFileRoute("/_app/settings/general")({
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { title: "General settings", icon: SettingsIcon },
  }),
  component: GeneralSettingsPage,
});

const PERMISSION_MODES = ["deny-all", "approve-reads", "approve-all"] as const;
type PermissionMode = (typeof PERMISSION_MODES)[number];

type GeneralConfig = SettingsGeneralSection["config"];
type UpdateQuery = ReturnType<typeof useSettingsGeneralPage>["update"];

function parseSessionTimeoutSeconds(raw: string): number {
  if (!raw) return 0;
  const match = /^(\d+)(s|m|h)?$/i.exec(raw.trim());
  if (!match) return 0;
  const value = Number.parseInt(match[1] ?? "0", 10);
  const unit = (match[2] ?? "s").toLowerCase();
  if (unit === "h") return value * 3600;
  if (unit === "m") return value * 60;
  return value;
}

function formatSessionTimeout(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds <= 0) return "0s";
  return `${Math.floor(seconds)}s`;
}

function formatUpdateTimestamp(value?: string | null): string {
  if (!value) return "--";
  const parsed = new Date(value);
  return Number.isNaN(parsed.getTime()) ? "--" : parsed.toLocaleString();
}

function formatUpdateStatus(status?: SettingsUpdateStatus["status"]): string {
  if (!status) return "unknown";
  return status.replace(/-/g, " ");
}

function GeneralSettingsPage() {
  const page = useSettingsGeneralPage();
  const [validationErrors, setValidationErrors] = useState<Record<string, string | null>>({});
  const setValidationError = useCallback(
    (key: string) => (message: string | null) => {
      setValidationErrors(current =>
        current[key] === message ? current : { ...current, [key]: message }
      );
    },
    []
  );
  const isInvalid = useMemo(
    () => Object.values(validationErrors).some(message => message !== null),
    [validationErrors]
  );
  const runtime = page.envelope?.runtime;
  const configPaths = page.envelope?.config_paths;
  useTopbarSlot({
    tabs:
      runtime && configPaths ? (
        <StatusLineTopbarSlot
          data-testid="settings-page-general-status-line"
          status={runtime.available ? "connected" : "error"}
          items={[
            {
              key: "sessions",
              value: `${runtime.active_sessions} active sessions · ${runtime.active_agents} agents`,
              tone: "neutral",
            },
            {
              key: "config",
              value: <span className="font-mono">config: {configPaths.global_config}</span>,
              tone: "neutral",
            },
          ]}
        />
      ) : undefined,
  });

  if (page.isLoading) {
    return (
      <div
        className="flex flex-1 items-center justify-center"
        data-testid="settings-page-general-loading"
      >
        <Spinner className="size-5 text-subtle" />
      </div>
    );
  }

  if (page.error || !page.envelope || !page.draft) {
    return (
      <div
        className="flex flex-1 items-center justify-center"
        data-testid="settings-page-general-error"
      >
        <div className="flex flex-col items-center gap-2 text-center">
          <AlertCircle className="size-6 text-danger" />
          <p className="text-sm text-subtle">
            {page.error?.message ?? "Failed to load general settings"}
          </p>
          <Button onClick={page.handleRetry} size="sm" type="button" variant="outline">
            Retry
          </Button>
        </div>
      </div>
    );
  }

  const { envelope, draft, setDraft, restart, update } = page;

  const bannerProps = restartBannerPropsFor("general", restart);

  return (
    <PageShell
      slug="general"
      banner={bannerProps ? <RestartBanner {...bannerProps} /> : null}
      footer={
        <SettingsSaveBar
          slug="general"
          isDirty={page.isDirty}
          isInvalid={isInvalid}
          isSaving={page.isSaving}
          error={page.saveError}
          warnings={page.warnings}
          lastAppliedLabel={page.lastAppliedLabel}
          onSave={page.handleSave}
          onReset={page.handleReset}
        />
      }
    >
      <RuntimeSection envelope={envelope} />
      <SettingsApplyRecordsPanel
        records={page.applyRecords.data?.entries ?? []}
        isLoading={page.applyRecords.isLoading}
        isFetching={page.applyRecords.isFetching}
        error={page.applyRecords.error instanceof Error ? page.applyRecords.error : null}
        reloadError={page.reloadError}
        reloadResult={page.reloadResult}
        isReloading={page.isReloading}
        onRefresh={() => void page.applyRecords.refetch()}
        onReload={page.handleReload}
      />
      <SoftwareUpdateSection update={update} />
      <DefaultsSection draft={draft} setDraft={setDraft} />
      <PermissionsSection draft={draft} setDraft={setDraft} />
      <SessionSection
        draft={draft}
        setDraft={setDraft}
        timeoutError={validationErrors.sessionTimeout ?? undefined}
        onTimeoutValidityChange={setValidationError("sessionTimeout")}
      />
    </PageShell>
  );
}

function RuntimeSection({ envelope }: { envelope: SettingsGeneralSection }) {
  const runtime = envelope.runtime;
  return (
    <Section divided label="Runtime" note="read-only">
      <MetricGrid>
        <Metric label="UDS socket" value={runtime.socket ?? envelope.config.daemon.socket} />
        <Metric
          label="HTTP bind"
          value={
            runtime.http_host && runtime.http_port
              ? `${runtime.http_host}:${runtime.http_port}`
              : `${envelope.config.http.host}:${envelope.config.http.port}`
          }
        />
        <Metric label="Active sessions" value={String(runtime.active_sessions)} />
        <Metric
          label="Concurrent agents"
          value={`${runtime.active_agents} / ${envelope.config.limits.max_concurrent_agents} max`}
        />
      </MetricGrid>
    </Section>
  );
}

function SoftwareUpdateSection({ update }: { update: UpdateQuery }) {
  const snapshot = update.data ?? null;
  const transportError =
    update.error instanceof Error ? update.error.message : "Failed to load update status";
  const releaseLink = snapshot?.release_url ? (
    <Button
      variant="outline"
      size="sm"
      render={
        <a
          href={snapshot.release_url}
          rel="noreferrer"
          target="_blank"
          data-testid="settings-page-general-update-release-link"
        />
      }
    >
      <ExternalLink className="size-3 text-subtle" />
      Release notes
    </Button>
  ) : null;
  const refreshIndicator = update.isFetching ? (
    <span className="inline-flex items-center gap-1.5 text-xs text-muted">
      <Spinner className="size-3 text-subtle" />
      Checking
    </span>
  ) : null;
  const retryButton = update.error ? (
    <Button
      type="button"
      variant="outline"
      size="sm"
      onClick={() => void update.refetch()}
      data-testid="settings-page-general-update-retry"
    >
      Retry
    </Button>
  ) : null;
  const statusValue = snapshot
    ? formatUpdateStatus(snapshot.status)
    : update.isLoading || update.isFetching
      ? "checking"
      : "unavailable";
  const lastError = snapshot?.last_error ?? (snapshot ? null : transportError);

  return (
    <Section
      divided
      label="Software update"
      note="Read-only. AGH self-updates direct-binary installs on macOS and Linux; managed installs return exact upgrade guidance."
      right={
        releaseLink || refreshIndicator || retryButton ? (
          <div className="flex flex-wrap items-center gap-2">
            {releaseLink}
            {refreshIndicator}
            {retryButton}
          </div>
        ) : undefined
      }
    >
      <MetricGrid>
        <Metric
          label="Status"
          value={statusValue}
          subtext={snapshot?.supported ? undefined : "manual update path"}
          data-testid="settings-page-general-update-status"
        />
        <Metric
          label="Current version"
          value={snapshot?.current_version ?? "--"}
          data-testid="settings-page-general-update-current-version"
        />
        <Metric
          label="Latest stable"
          value={snapshot?.latest_version ?? "--"}
          data-testid="settings-page-general-update-latest-version"
        />
        <Metric
          label="Install method"
          value={snapshot?.install_method ?? "--"}
          data-testid="settings-page-general-update-install-method"
        />
        <Metric
          label="Managed"
          value={snapshot ? (snapshot.managed ? "yes" : "no") : "--"}
          data-testid="settings-page-general-update-managed"
        />
        <Metric
          label="Last checked"
          value={formatUpdateTimestamp(snapshot?.checked_at)}
          data-testid="settings-page-general-update-checked-at"
        />
      </MetricGrid>
      {snapshot?.recommendation ? (
        <SettingsFieldRow
          data-testid="settings-page-general-update-recommendation"
          label="Next action"
          description="Exact command or package-manager path for this install"
          control={
            <span className="max-w-136 font-mono text-xs text-fg">{snapshot.recommendation}</span>
          }
        />
      ) : null}
      {lastError ? (
        <SettingsFieldRow
          data-testid="settings-page-general-update-last-error"
          label="Last error"
          description="The last update refresh that failed"
          control={<span className="max-w-136 font-mono text-xs text-danger">{lastError}</span>}
        />
      ) : null}
    </Section>
  );
}

interface DraftSectionProps {
  draft: GeneralConfig;
  setDraft: Dispatch<SetStateAction<GeneralConfig | null>>;
}

function DefaultsSection({ draft, setDraft }: DraftSectionProps) {
  return (
    <Section divided label="Defaults" note="applied to new sessions">
      <SettingsFieldRow
        data-testid="settings-page-general-default-agent"
        label="Default agent"
        description="Used when a new session doesn't specify one"
        hint="CONFIG.TOML"
        control={
          <Input
            className="w-56"
            data-testid="settings-page-general-default-agent-input"
            value={draft.defaults.agent}
            onChange={event =>
              setDraft(prev => {
                const current = prev ?? draft;
                return {
                  ...current,
                  defaults: { ...current.defaults, agent: event.target.value },
                };
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid="settings-page-general-default-provider"
        label="Default provider"
        description="LLM backend agents spawn against"
        hint="OPTIONAL"
        control={
          <Input
            className="w-56"
            data-testid="settings-page-general-default-provider-input"
            value={draft.defaults.provider ?? ""}
            placeholder="auto"
            onChange={event =>
              setDraft(prev => {
                const current = prev ?? draft;
                return {
                  ...current,
                  defaults: { ...current.defaults, provider: event.target.value },
                };
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid="settings-page-general-default-sandbox"
        label="Default sandbox"
        description="Execution profile for new workspaces"
        hint="DEFAULT"
        control={
          <Input
            className="w-56 font-mono"
            data-testid="settings-page-general-default-sandbox-input"
            value={draft.defaults.sandbox ?? ""}
            placeholder="local"
            onChange={event =>
              setDraft(prev => {
                const current = prev ?? draft;
                return {
                  ...current,
                  defaults: { ...current.defaults, sandbox: event.target.value },
                };
              })
            }
          />
        }
      />
    </Section>
  );
}

function PermissionsSection({ draft, setDraft }: DraftSectionProps) {
  return (
    <Section divided label="Permissions" note="tool approval policy">
      <PillGroup
        className="max-w-full flex-wrap"
        data-testid="settings-page-general-permissions-group"
        aria-label="Tool approval policy"
        value={draft.permissions.mode}
        onChange={mode =>
          setDraft(prev => {
            const current = prev ?? draft;
            return { ...current, permissions: { mode } };
          })
        }
        items={PERMISSION_MODES.map(mode => ({
          value: mode,
          label: mode,
          testId: `settings-page-general-permission-${mode}`,
        }))}
      />
      <p className="text-xs text-subtle">{describePermissionMode(draft.permissions.mode)}</p>
    </Section>
  );
}

function SessionSection({
  draft,
  setDraft,
  timeoutError,
  onTimeoutValidityChange,
}: DraftSectionProps & {
  timeoutError?: string;
  onTimeoutValidityChange: (message: string | null) => void;
}) {
  return (
    <Section divided label="Session" note="runtime limits">
      <SettingsFieldRow
        data-testid="settings-page-general-session-timeout"
        label="Session timeout"
        description="0 disables force-close"
        error={timeoutError}
        hint="SECONDS"
        control={
          <div className="flex max-w-full flex-wrap items-center gap-2">
            <SettingsNumberInput
              min={0}
              className="w-28"
              data-testid="settings-page-general-session-timeout-input"
              value={parseSessionTimeoutSeconds(draft.session_timeout)}
              onValidityChange={onTimeoutValidityChange}
              onValueChange={value =>
                setDraft(prev => {
                  const current = prev ?? draft;
                  return {
                    ...current,
                    session_timeout: formatSessionTimeout(value),
                  };
                })
              }
            />
            <Eyebrow className="text-muted">seconds</Eyebrow>
          </div>
        }
      />
    </Section>
  );
}

function describePermissionMode(mode: PermissionMode): string {
  switch (mode) {
    case "deny-all":
      return "All tool calls denied unless explicitly allowed by agent frontmatter.";
    case "approve-reads":
      return "Read-only tool calls auto-approved. Writes require confirmation.";
    case "approve-all":
      return "All tool calls auto-approved. Agents can lower this individually via their permissions: frontmatter.";
  }
}
