import { AlertCircle, ExternalLink } from "lucide-react";
import { useState, type Dispatch, type SetStateAction } from "react";

import { useSettingsObservabilityPage } from "@/systems/settings/hooks/use-settings-observability-page";
import {
  SettingsFieldRow,
  SettingsGroup,
  SettingsPageFrame,
  SettingsSaveBar,
  SettingsRuntimeUnavailable,
  useSettingsSaveBarState,
  useSettingsTopbar,
  SettingsTile,
  SettingsTiles,
  type SettingsObservabilitySection,
} from "@/systems/settings";
import { Button, Eyebrow, Pill, Spinner, Switch } from "@agh/ui";

type ObservabilityConfig = SettingsObservabilitySection["config"];
type LogTailMeta = SettingsObservabilitySection["log_tail"];
type Runtime = SettingsObservabilitySection["runtime"];

import { ObservabilityNumberField as NumberField, UsageBreakdown } from "./-observability-fields";
import { formatBytes } from "./-observability-format";
import { ObservabilitySupportBundleSection } from "./-observability-support-bundle-section";

function safeLogTailURL(value: string | undefined): string | null {
  if (!value || !URL.canParse(value, "http://localhost")) return null;
  const protocol = new URL(value, "http://localhost").protocol;
  return protocol === "http:" || protocol === "https:" ? value : null;
}

export function ObservabilitySettingsPage() {
  const page = useSettingsObservabilityPage();
  useSettingsTopbar("observability");
  const [validationErrors, setValidationErrors] = useState<Record<string, string | null>>({});
  const setValidationError = (key: string) => (message: string | null) => {
    setValidationErrors(current =>
      current[key] === message ? current : { ...current, [key]: message }
    );
  };
  const isInvalid = Object.values(validationErrors).some(message => message !== null);
  const saveBarState = useSettingsSaveBarState({
    isDirty: page.isDirty,
    isInvalid,
    isSaving: page.isSaving,
    error: page.saveError,
    warnings: page.warnings,
    lastAppliedLabel: page.lastAppliedLabel,
  });

  if (page.isLoading) {
    return (
      <div
        className="flex flex-1 items-center justify-center"
        data-testid="settings-page-observability-loading"
      >
        <Spinner className="size-5 text-subtle" />
      </div>
    );
  }

  if (page.error || !page.envelope || !page.draft) {
    return (
      <div
        className="flex flex-1 items-center justify-center"
        data-testid="settings-page-observability-error"
      >
        <div className="flex flex-col items-center gap-2 text-center">
          <AlertCircle className="size-6 text-danger" />
          <p className="text-sm text-subtle">
            {page.error?.message ?? "Failed to load observability settings"}
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
  const logTail = envelope.log_tail;
  const totalStorage = runtime.global_db_size_bytes + runtime.session_db_size_bytes;
  const cap = draft.max_global_bytes;
  const capPercent = cap > 0 ? Math.min(100, Math.round((totalStorage / cap) * 100)) : 0;

  return (
    <SettingsPageFrame
      description="What this daemon records about sessions, and how much disk it may use."
      meta={
        runtime.available
          ? [
              {
                key: "sessions",
                content: (
                  <span>
                    <span className="font-medium text-muted">{runtime.active_sessions}</span> active
                    sessions
                  </span>
                ),
              },
              {
                key: "storage",
                content: (
                  <span data-testid="settings-page-observability-storage-summary">
                    storage {formatBytes(totalStorage)} of {formatBytes(cap)}
                  </span>
                ),
              },
            ]
          : [{ key: "runtime", content: <span>runtime unavailable</span> }]
      }
      restart={restart}
      saveBar={
        <SettingsSaveBar
          slug="observability"
          state={saveBarState}
          onSave={page.handleSave}
          onReset={() => {
            setValidationErrors({});
            page.handleReset();
          }}
        />
      }
      slug="observability"
    >
      {runtime.available ? (
        <OverviewMetrics
          activeSessions={runtime.active_sessions}
          activeAgents={runtime.active_agents}
          totalStorage={totalStorage}
          cap={cap}
        />
      ) : (
        <SettingsRuntimeUnavailable
          slug="observability"
          description="Session, agent, and storage measurements could not be read."
        />
      )}
      <CaptureSection
        draft={draft}
        setDraft={setDraft}
        usage={
          runtime.available
            ? {
                capPercent,
                globalBytes: runtime.global_db_size_bytes,
                sessionBytes: runtime.session_db_size_bytes,
              }
            : null
        }
        cap={cap}
        validationErrors={validationErrors}
        setValidationError={setValidationError}
      />
      <TranscriptsSection
        draft={draft}
        setDraft={setDraft}
        validationErrors={validationErrors}
        setValidationError={setValidationError}
      />
      <ObservabilitySupportBundleSection />
      <LogTailSection logTail={logTail} runtime={runtime} />
    </SettingsPageFrame>
  );
}

interface OverviewMetricsProps {
  activeSessions: number;
  activeAgents: number;
  totalStorage: number;
  cap: number;
}

function OverviewMetrics({
  activeSessions,
  activeAgents,
  totalStorage,
  cap,
}: OverviewMetricsProps) {
  const capPercent = cap > 0 ? Math.min(100, Math.round((totalStorage / cap) * 100)) : 0;
  return (
    <SettingsGroup bare description="Live capture volume. Read-only." title="Runtime">
      <SettingsTiles>
        <SettingsTile
          data-testid="settings-page-observability-metric-sessions"
          dotTone={activeSessions > 0 ? "success" : "neutral"}
          label="Active sessions"
          value={String(activeSessions)}
        />
        <SettingsTile
          data-testid="settings-page-observability-metric-agents"
          label="Active agents"
          value={String(activeAgents)}
        />
        <SettingsTile
          data-testid="settings-page-observability-metric-storage"
          detail={`of ${formatBytes(cap)}`}
          label="Storage used"
          value={formatBytes(totalStorage)}
        />
        <SettingsTile
          data-testid="settings-page-observability-metric-capacity"
          detail="of soft cap"
          label="Capacity"
          value={`${capPercent}%`}
        />
      </SettingsTiles>
    </SettingsGroup>
  );
}

interface DraftSectionProps {
  draft: ObservabilityConfig;
  setDraft: Dispatch<SetStateAction<ObservabilityConfig | null>>;
}

interface CaptureSectionProps extends DraftSectionProps {
  usage: { capPercent: number; globalBytes: number; sessionBytes: number } | null;
  cap: number;
  validationErrors: Record<string, string | null>;
  setValidationError: (key: string) => (message: string | null) => void;
}

function CaptureSection({
  draft,
  setDraft,
  usage,
  cap,
  validationErrors,
  setValidationError,
}: CaptureSectionProps) {
  return (
    <SettingsGroup
      title="Capture"
      description="events, transcripts, logs"
      action={
        usage ? (
          <Pill
            mono
            tone={usage.capPercent > 85 ? "warning" : "neutral"}
            data-testid="settings-page-observability-cap-percent"
          >
            {usage.capPercent}% of cap
          </Pill>
        ) : null
      }
    >
      <SettingsFieldRow
        data-testid="settings-page-observability-enabled"
        label="Event capture"
        description="Persist every session event to SQLite for replay"
        control={
          <Switch
            data-testid="settings-page-observability-enabled-switch"
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
      <div className="grid gap-4 md:grid-cols-2">
        <NumberField
          label="Retention"
          testId="settings-page-observability-retention-days"
          value={draft.retention_days}
          errorMessage={validationErrors.retentionDays ?? undefined}
          suffix="days"
          onValidityChange={setValidationError("retentionDays")}
          onChange={value =>
            setDraft(prev => {
              const current = prev ?? draft;
              return { ...current, retention_days: value };
            })
          }
        />
        <NumberField
          label="Max global bytes"
          testId="settings-page-observability-max-global-bytes"
          value={draft.max_global_bytes}
          errorMessage={validationErrors.maxGlobalBytes ?? undefined}
          suffix="bytes"
          onValidityChange={setValidationError("maxGlobalBytes")}
          onChange={value =>
            setDraft(prev => {
              const current = prev ?? draft;
              return { ...current, max_global_bytes: value };
            })
          }
        />
      </div>
      {usage ? (
        <UsageBreakdown
          globalBytes={usage.globalBytes}
          sessionBytes={usage.sessionBytes}
          cap={cap}
        />
      ) : null}
    </SettingsGroup>
  );
}

function TranscriptsSection({
  draft,
  setDraft,
  validationErrors,
  setValidationError,
}: DraftSectionProps & {
  validationErrors: Record<string, string | null>;
  setValidationError: (key: string) => (message: string | null) => void;
}) {
  return (
    <SettingsGroup title="Transcripts" description="full replay of agent I/O">
      <SettingsFieldRow
        data-testid="settings-page-observability-transcripts-enabled"
        label="Capture transcripts"
        description="Chunked segment-based replay of every prompt + response"
        control={
          <Switch
            data-testid="settings-page-observability-transcripts-enabled-switch"
            checked={draft.transcripts.enabled}
            onCheckedChange={checked =>
              setDraft(prev => {
                const current = prev ?? draft;
                return {
                  ...current,
                  transcripts: { ...current.transcripts, enabled: checked },
                };
              })
            }
          />
        }
      />
      <div className="grid gap-4 md:grid-cols-2">
        <NumberField
          label="Segment size"
          testId="settings-page-observability-segment-bytes"
          value={draft.transcripts.segment_bytes}
          errorMessage={validationErrors.segmentBytes ?? undefined}
          suffix="bytes"
          onValidityChange={setValidationError("segmentBytes")}
          onChange={value =>
            setDraft(prev => {
              const current = prev ?? draft;
              return {
                ...current,
                transcripts: { ...current.transcripts, segment_bytes: value },
              };
            })
          }
        />
        <NumberField
          label="Max per session"
          testId="settings-page-observability-transcripts-max-bytes"
          value={draft.transcripts.max_bytes_per_session}
          errorMessage={validationErrors.maxBytesPerSession ?? undefined}
          suffix="bytes"
          onValidityChange={setValidationError("maxBytesPerSession")}
          onChange={value =>
            setDraft(prev => {
              const current = prev ?? draft;
              return {
                ...current,
                transcripts: {
                  ...current.transcripts,
                  max_bytes_per_session: value,
                },
              };
            })
          }
        />
      </div>
    </SettingsGroup>
  );
}

function LogTailSection({ logTail, runtime }: { logTail: LogTailMeta; runtime: Runtime }) {
  void runtime;
  const streamURL = safeLogTailURL(logTail.stream_url);
  return (
    <SettingsGroup title="Log tail" description="daemon log stream">
      <div
        className="flex flex-wrap items-center justify-between gap-3 rounded-md border border-line bg-elevated px-4 py-3"
        data-testid="settings-page-observability-log-tail"
        data-available={logTail.available ? "true" : "false"}
      >
        <div className="flex flex-col gap-1">
          <span className="text-sm text-fg">
            {logTail.available ? "Live log tail available" : "Log tail unavailable"}
          </span>
          <Eyebrow
            className="text-muted"
            data-testid="settings-page-observability-log-tail-transport"
          >
            transport: {logTail.transport ?? "none"}
          </Eyebrow>
        </div>
        {logTail.available && streamURL ? (
          <a
            className="inline-flex items-center gap-1.5 text-sm text-accent hover:underline"
            data-testid="settings-page-observability-log-tail-link"
            href={streamURL}
            rel="noreferrer"
            target="_blank"
          >
            <ExternalLink className="size-3" />
            Open stream
          </a>
        ) : null}
      </div>
    </SettingsGroup>
  );
}
