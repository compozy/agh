import { AlertCircle, Bot, ExternalLink } from "lucide-react";
import { createFileRoute, Link } from "@tanstack/react-router";
import { useCallback, useMemo, useState, type Dispatch, type SetStateAction } from "react";

import {
  Button,
  Eyebrow,
  Input,
  Metric,
  MetricGrid,
  PageShell,
  RestartBanner,
  Section,
  Spinner,
  StatusLineTopbarSlot,
  Switch,
  useTopbarSlot,
} from "@agh/ui";
import type { TopbarRouteContext } from "@/types/topbar";
import { useSettingsAutomationPage } from "@/hooks/routes/use-settings-automation-page";
import type { SettingsAutomationSection } from "@/systems/settings";
import {
  SettingsFieldRow,
  SettingsNumberInput,
  SettingsSaveBar,
} from "@/systems/settings/components";
import { restartBannerPropsFor } from "@/systems/settings/lib/restart-banner-mapper";

export const Route = createFileRoute("/_app/settings/automation")({
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { title: "Automation settings", icon: Bot },
  }),
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
  const runtime = page.envelope?.runtime;
  useTopbarSlot({
    tabs: runtime ? (
      <StatusLineTopbarSlot
        data-testid="settings-page-automation-status-line"
        status={runtime.available ? "connected" : "error"}
        items={[
          {
            key: "jobs",
            value: `${runtime.job_enabled}/${runtime.job_total} jobs active`,
            tone: "neutral",
          },
          {
            key: "triggers",
            value: `${runtime.trigger_enabled}/${runtime.trigger_total} triggers active`,
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
        data-testid="settings-page-automation-loading"
      >
        <Spinner className="size-5 text-(--subtle)" />
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
          <AlertCircle className="size-6 text-(--danger)" />
          <p className="text-sm text-(--subtle)">
            {page.error?.message ?? "Failed to load automation settings"}
          </p>
          <Button onClick={page.handleRetry} size="sm" type="button" variant="outline">
            Retry
          </Button>
        </div>
      </div>
    );
  }

  if (!runtime) {
    return null;
  }
  const { draft, setDraft, restart } = page;

  const bannerProps = restartBannerPropsFor("automation", restart);

  return (
    <PageShell
      slug="automation"
      banner={bannerProps ? <RestartBanner {...bannerProps} /> : null}
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
    </PageShell>
  );
}

function OperationalLinksRow() {
  return (
    <Section divided label="Operational" note="manage jobs, triggers, and run history">
      <div
        className="flex flex-wrap gap-2"
        data-testid="settings-page-automation-operational-links"
      >
        <Link
          to="/jobs"
          className="inline-flex items-center gap-1.5 rounded-md border border-(--line) bg-(--elevated) px-3 py-1.5 text-xs font-medium text-(--fg) hover:bg-(--hover)"
          data-testid="settings-page-automation-link-jobs"
        >
          <ExternalLink className="size-3.5 text-(--subtle)" />
          Open Jobs
        </Link>
        <Link
          to="/triggers"
          className="inline-flex items-center gap-1.5 rounded-md border border-(--line) bg-(--elevated) px-3 py-1.5 text-xs font-medium text-(--fg) hover:bg-(--hover)"
          data-testid="settings-page-automation-link-triggers"
        >
          <ExternalLink className="size-3.5 text-(--subtle)" />
          Open Triggers
        </Link>
      </div>
    </Section>
  );
}

function ManagerSummarySection({ runtime }: { runtime: AutomationRuntime }) {
  const nextFire = runtime.next_fire ? new Date(runtime.next_fire).toLocaleString() : "--";
  const lastSynced = runtime.last_synced_at
    ? new Date(runtime.last_synced_at).toLocaleString()
    : "--";

  return (
    <Section divided label="Manager" note="read-only">
      <MetricGrid columns={3}>
        <Metric
          label="Engine"
          value={runtime.running ? "running" : "stopped"}
          data-testid="settings-page-automation-runtime-engine"
        />
        <Metric
          label="Scheduler"
          value={runtime.scheduler_running ? "running" : "stopped"}
          data-testid="settings-page-automation-runtime-scheduler"
        />
        <Metric
          label="Jobs (enabled/total)"
          value={`${runtime.job_enabled} / ${runtime.job_total}`}
          data-testid="settings-page-automation-runtime-jobs"
        />
        <Metric
          label="Triggers (enabled/total)"
          value={`${runtime.trigger_enabled} / ${runtime.trigger_total}`}
          data-testid="settings-page-automation-runtime-triggers"
        />
        <Metric
          label="Next fire"
          value={nextFire}
          data-testid="settings-page-automation-runtime-next-fire"
        />
        <Metric
          label="Last synced"
          value={lastSynced}
          data-testid="settings-page-automation-runtime-last-synced"
        />
      </MetricGrid>
    </Section>
  );
}

interface DraftSectionProps {
  draft: AutomationConfig;
  setDraft: Dispatch<SetStateAction<AutomationConfig | null>>;
}

function EngineSection({ draft, setDraft }: DraftSectionProps) {
  return (
    <Section divided label="Engine" note="persisted to config.toml">
      <SettingsFieldRow
        data-testid="settings-page-automation-enabled"
        label="Automation engine"
        description="Runs jobs and triggers on the daemon"
        control={
          <Switch
            data-testid="settings-page-automation-enabled-switch"
            checked={draft.enabled}
            onCheckedChange={checked =>
              setDraft(prev => {
                const current = prev ?? draft;
                return { ...current, enabled: checked };
              })
            }
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
            onChange={event =>
              setDraft(prev => {
                const current = prev ?? draft;
                return { ...current, timezone: event.target.value };
              })
            }
          />
        }
      />
    </Section>
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
    <Section divided label="Limits" note="resource caps">
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
              setDraft(prev => {
                const current = prev ?? draft;
                return {
                  ...current,
                  max_concurrent_jobs: value,
                };
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
                setDraft(prev => {
                  const current = prev ?? draft;
                  return {
                    ...current,
                    default_fire_limit: {
                      ...current.default_fire_limit,
                      max: value,
                    },
                  };
                })
              }
            />
            <Eyebrow className="text-(--muted)">fires</Eyebrow>
            <span className="text-xs text-(--subtle)">per</span>
            <Input
              className="w-24 font-mono"
              data-testid="settings-page-automation-fire-limit-window-input"
              value={draft.default_fire_limit.window ?? ""}
              placeholder="1m"
              onChange={event =>
                setDraft(prev => {
                  const current = prev ?? draft;
                  return {
                    ...current,
                    default_fire_limit: {
                      ...current.default_fire_limit,
                      window: event.target.value,
                    },
                  };
                })
              }
            />
          </div>
        }
      />
    </Section>
  );
}
