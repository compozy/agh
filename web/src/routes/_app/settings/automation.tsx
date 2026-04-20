import { AlertCircle, ExternalLink, Loader2 } from "lucide-react";
import { createFileRoute, Link } from "@tanstack/react-router";
import { useCallback, useMemo, useState, type Dispatch, type SetStateAction } from "react";

import { Button, Input, Switch } from "@agh/ui";
import { useSettingsAutomationPage } from "@/hooks/routes/use-settings-automation-page";
import type { SettingsAutomationSection } from "@/systems/settings";
import {
  SettingsFieldRow,
  SettingsNumberInput,
  SettingsPageActions,
  SettingsPageShell,
  SettingsRestartBanner,
  SettingsSaveBar,
  SettingsSectionCard,
  SettingsStatGrid,
  SettingsStatItem,
  SettingsStatusLine,
} from "@/systems/settings/components";

export const Route = createFileRoute("/_app/settings/automation")({
  component: AutomationSettingsPage,
});

type AutomationConfig = SettingsAutomationSection["config"];
type AutomationRuntime = SettingsAutomationSection["runtime"];

function AutomationSettingsPage() {
  const page = useSettingsAutomationPage();
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

  if (page.isLoading) {
    return (
      <div
        className="flex flex-1 items-center justify-center"
        data-testid="settings-page-automation-loading"
      >
        <Loader2 className="size-5 animate-spin text-[color:var(--color-text-tertiary)]" />
      </div>
    );
  }

  if (page.error || !page.envelope || !page.draft) {
    return (
      <div
        className="flex flex-1 items-center justify-center"
        data-testid="settings-page-automation-error"
      >
        <div className="flex flex-col items-center gap-2 text-center">
          <AlertCircle className="size-6 text-[color:var(--color-danger)]" />
          <p className="text-sm text-[color:var(--color-text-tertiary)]">
            {page.error?.message ?? "Failed to load automation settings"}
          </p>
          <Button onClick={page.handleRetry} size="sm" type="button" variant="outline">
            Retry
          </Button>
        </div>
      </div>
    );
  }

  const { envelope, draft, setDraft, restart } = page;
  const runtime = envelope.runtime;

  return (
    <SettingsPageShell
      slug="automation"
      title="Automation"
      statusLine={
        <SettingsStatusLine
          data-testid="settings-page-automation-status-line"
          daemonAvailable={runtime.available}
          items={[
            <span key="jobs">
              {runtime.job_enabled}/{runtime.job_total} jobs active
            </span>,
            <span key="triggers">
              {runtime.trigger_enabled}/{runtime.trigger_total} triggers active
            </span>,
          ]}
        />
      }
      actions={<SettingsPageActions slug="automation" restart={restart} />}
      banner={<SettingsRestartBanner slug="automation" restart={restart} />}
      footer={
        <SettingsSaveBar
          slug="automation"
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
      <OperationalLinksRow />
      <ManagerSummarySection runtime={runtime} />
      <EngineSection draft={draft} setDraft={setDraft} />
      <LimitsSection
        draft={draft}
        setDraft={setDraft}
        validationErrors={validationErrors}
        setValidationError={setValidationError}
      />
    </SettingsPageShell>
  );
}

function OperationalLinksRow() {
  return (
    <SettingsSectionCard eyebrow="Operational" note="manage jobs, triggers, and run history">
      <div
        className="flex flex-wrap gap-2"
        data-testid="settings-page-automation-operational-links"
      >
        <Link
          to="/automation"
          className="inline-flex items-center gap-1.5 rounded-md border border-[color:var(--color-divider)] bg-[color:var(--color-surface-elevated)] px-3 py-1.5 text-xs font-medium text-[color:var(--color-text-primary)] hover:bg-[color:var(--color-hover)]"
          data-testid="settings-page-automation-link-automation"
        >
          <ExternalLink className="size-3.5 text-[color:var(--color-text-tertiary)]" />
          Open Automation
        </Link>
      </div>
    </SettingsSectionCard>
  );
}

function ManagerSummarySection({ runtime }: { runtime: AutomationRuntime }) {
  const nextFire = runtime.next_fire ? new Date(runtime.next_fire).toLocaleString() : "—";
  const lastSynced = runtime.last_synced_at
    ? new Date(runtime.last_synced_at).toLocaleString()
    : "—";

  return (
    <SettingsSectionCard eyebrow="Manager" note="read-only">
      <SettingsStatGrid className="xl:grid-cols-3">
        <SettingsStatItem
          label="Engine"
          value={runtime.running ? "running" : "stopped"}
          testId="settings-page-automation-runtime-engine"
        />
        <SettingsStatItem
          label="Scheduler"
          value={runtime.scheduler_running ? "running" : "stopped"}
          testId="settings-page-automation-runtime-scheduler"
        />
        <SettingsStatItem
          label="Jobs (enabled/total)"
          value={`${runtime.job_enabled} / ${runtime.job_total}`}
          testId="settings-page-automation-runtime-jobs"
        />
        <SettingsStatItem
          label="Triggers (enabled/total)"
          value={`${runtime.trigger_enabled} / ${runtime.trigger_total}`}
          testId="settings-page-automation-runtime-triggers"
        />
        <SettingsStatItem
          label="Next fire"
          value={nextFire}
          testId="settings-page-automation-runtime-next-fire"
        />
        <SettingsStatItem
          label="Last synced"
          value={lastSynced}
          testId="settings-page-automation-runtime-last-synced"
        />
      </SettingsStatGrid>
    </SettingsSectionCard>
  );
}

interface DraftSectionProps {
  draft: AutomationConfig;
  setDraft: Dispatch<SetStateAction<AutomationConfig | null>>;
}

function EngineSection({ draft, setDraft }: DraftSectionProps) {
  return (
    <SettingsSectionCard eyebrow="Engine" note="persisted to config.toml">
      <SettingsFieldRow
        data-testid="settings-page-automation-enabled"
        label="Automation engine"
        description="Runs jobs and triggers on the daemon"
        control={
          <Switch
            data-testid="settings-page-automation-enabled-switch"
            checked={draft.enabled}
            onCheckedChange={checked => setDraft({ ...draft, enabled: checked })}
          />
        }
      />
      <SettingsFieldRow
        data-testid="settings-page-automation-timezone"
        label="Timezone"
        description="Used for cron schedule resolution"
        hint="IANA"
        control={
          <Input
            className="w-56 font-mono"
            data-testid="settings-page-automation-timezone-input"
            value={draft.timezone ?? ""}
            placeholder="UTC"
            onChange={event => setDraft({ ...draft, timezone: event.target.value })}
          />
        }
      />
    </SettingsSectionCard>
  );
}

function LimitsSection({
  draft,
  setDraft,
  validationErrors,
  setValidationError,
}: DraftSectionProps & {
  validationErrors: Record<string, string | null>;
  setValidationError: (key: string) => (message: string | null) => void;
}) {
  return (
    <SettingsSectionCard eyebrow="Limits" note="resource caps">
      <SettingsFieldRow
        data-testid="settings-page-automation-max-concurrent"
        label="Max concurrent jobs"
        description="Caps the number of jobs running simultaneously"
        error={validationErrors.maxConcurrentJobs ?? undefined}
        hint="DEFAULT"
        control={
          <SettingsNumberInput
            min={0}
            className="w-24"
            data-testid="settings-page-automation-max-concurrent-input"
            value={draft.max_concurrent_jobs}
            onValidityChange={setValidationError("maxConcurrentJobs")}
            onValueChange={value =>
              setDraft({
                ...draft,
                max_concurrent_jobs: value,
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid="settings-page-automation-fire-limit-max"
        label="Default fire limit"
        description="Maximum invocations per window for new triggers"
        error={validationErrors.defaultFireLimitMax ?? undefined}
        hint="DEFAULT"
        control={
          <div className="flex items-center gap-2">
            <SettingsNumberInput
              min={0}
              className="w-24"
              data-testid="settings-page-automation-fire-limit-max-input"
              value={draft.default_fire_limit.max}
              onValidityChange={setValidationError("defaultFireLimitMax")}
              onValueChange={value =>
                setDraft({
                  ...draft,
                  default_fire_limit: {
                    ...draft.default_fire_limit,
                    max: value,
                  },
                })
              }
            />
            <span className="font-mono text-[0.6rem] uppercase tracking-[0.18em] text-[color:var(--color-text-label)]">
              fires
            </span>
            <span className="text-xs text-[color:var(--color-text-tertiary)]">per</span>
            <Input
              className="w-24 font-mono"
              data-testid="settings-page-automation-fire-limit-window-input"
              value={draft.default_fire_limit.window ?? ""}
              placeholder="1m"
              onChange={event =>
                setDraft({
                  ...draft,
                  default_fire_limit: {
                    ...draft.default_fire_limit,
                    window: event.target.value,
                  },
                })
              }
            />
          </div>
        }
      />
    </SettingsSectionCard>
  );
}
