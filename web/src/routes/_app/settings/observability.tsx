import { AlertCircle, ExternalLink, Loader2 } from "lucide-react";
import { createFileRoute } from "@tanstack/react-router";
import { useCallback, useMemo, useState, type Dispatch, type SetStateAction } from "react";

import { Button, Pill, Switch } from "@agh/ui";
import { useSettingsObservabilityPage } from "@/hooks/routes/use-settings-observability-page";
import type { SettingsObservabilitySection } from "@/systems/settings";
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

export const Route = createFileRoute("/_app/settings/observability")({
  component: ObservabilitySettingsPage,
});

type ObservabilityConfig = SettingsObservabilitySection["config"];
type LogTailMeta = SettingsObservabilitySection["log_tail"];
type Runtime = SettingsObservabilitySection["runtime"];

const GB = 1024 * 1024 * 1024;
const MB = 1024 * 1024;

function formatBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) return "0 B";
  if (bytes >= GB) return `${(bytes / GB).toFixed(1)} GB`;
  if (bytes >= MB) return `${(bytes / MB).toFixed(0)} MB`;
  if (bytes >= 1024) return `${(bytes / 1024).toFixed(0)} KB`;
  return `${bytes} B`;
}

function ObservabilitySettingsPage() {
  const page = useSettingsObservabilityPage();
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
        data-testid="settings-page-observability-loading"
      >
        <Loader2 className="size-5 animate-spin text-(--color-text-tertiary)" />
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
          <AlertCircle className="size-6 text-(--color-danger)" />
          <p className="text-sm text-(--color-text-tertiary)">
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
    <SettingsPageShell
      slug="observability"
      title="Observability"
      statusLine={
        <SettingsStatusLine
          data-testid="settings-page-observability-status-line"
          daemonAvailable={runtime.available}
          items={[
            <span key="sessions">{runtime.active_sessions} active sessions</span>,
            <span key="storage" data-testid="settings-page-observability-storage-summary">
              storage {formatBytes(totalStorage)} / {formatBytes(cap)}
            </span>,
          ]}
        />
      }
      actions={<SettingsPageActions slug="observability" restart={restart} />}
      banner={<SettingsRestartBanner slug="observability" restart={restart} />}
      footer={
        <SettingsSaveBar
          slug="observability"
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
      <OverviewMetrics
        activeSessions={runtime.active_sessions}
        activeAgents={runtime.active_agents}
        totalStorage={totalStorage}
        cap={cap}
      />
      <CaptureSection
        draft={draft}
        setDraft={setDraft}
        capPercent={capPercent}
        globalBytes={runtime.global_db_size_bytes}
        sessionBytes={runtime.session_db_size_bytes}
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
      <LogTailSection logTail={logTail} runtime={runtime} />
    </SettingsPageShell>
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
    <SettingsSectionCard eyebrow="Runtime" note="live capture volume">
      <SettingsStatGrid>
        <SettingsStatItem
          label="Active sessions"
          value={String(activeSessions)}
          testId="settings-page-observability-metric-sessions"
        />
        <SettingsStatItem
          label="Active agents"
          value={String(activeAgents)}
          testId="settings-page-observability-metric-agents"
        />
        <SettingsStatItem
          label="Storage used"
          value={formatBytes(totalStorage)}
          detail={`of ${formatBytes(cap)}`}
          testId="settings-page-observability-metric-storage"
        />
        <SettingsStatItem
          label="Capacity"
          value={`${capPercent}%`}
          detail="of soft cap"
          testId="settings-page-observability-metric-capacity"
        />
      </SettingsStatGrid>
    </SettingsSectionCard>
  );
}

interface DraftSectionProps {
  draft: ObservabilityConfig;
  setDraft: Dispatch<SetStateAction<ObservabilityConfig | null>>;
}

interface CaptureSectionProps extends DraftSectionProps {
  capPercent: number;
  globalBytes: number;
  sessionBytes: number;
  cap: number;
  validationErrors: Record<string, string | null>;
  setValidationError: (key: string) => (message: string | null) => void;
}

function CaptureSection({
  draft,
  setDraft,
  capPercent,
  globalBytes,
  sessionBytes,
  cap,
  validationErrors,
  setValidationError,
}: CaptureSectionProps) {
  return (
    <SettingsSectionCard
      eyebrow="Capture"
      note="events, transcripts, logs"
      headerAction={
        <Pill
          mono
          tone={capPercent > 85 ? "warning" : "neutral"}
          data-testid="settings-page-observability-cap-percent"
        >
          {capPercent}% of cap
        </Pill>
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
            onCheckedChange={checked => setDraft({ ...draft, enabled: checked })}
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
          onChange={value => setDraft({ ...draft, retention_days: value })}
        />
        <NumberField
          label="Max global bytes"
          testId="settings-page-observability-max-global-bytes"
          value={draft.max_global_bytes}
          errorMessage={validationErrors.maxGlobalBytes ?? undefined}
          suffix="bytes"
          onValidityChange={setValidationError("maxGlobalBytes")}
          onChange={value => setDraft({ ...draft, max_global_bytes: value })}
        />
      </div>
      <UsageBreakdown globalBytes={globalBytes} sessionBytes={sessionBytes} cap={cap} />
    </SettingsSectionCard>
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
    <SettingsSectionCard eyebrow="Transcripts" note="full replay of agent I/O">
      <SettingsFieldRow
        data-testid="settings-page-observability-transcripts-enabled"
        label="Capture transcripts"
        description="Chunked segment-based replay of every prompt + response"
        control={
          <Switch
            data-testid="settings-page-observability-transcripts-enabled-switch"
            checked={draft.transcripts.enabled}
            onCheckedChange={checked =>
              setDraft({
                ...draft,
                transcripts: { ...draft.transcripts, enabled: checked },
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
            setDraft({
              ...draft,
              transcripts: { ...draft.transcripts, segment_bytes: value },
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
            setDraft({
              ...draft,
              transcripts: {
                ...draft.transcripts,
                max_bytes_per_session: value,
              },
            })
          }
        />
      </div>
    </SettingsSectionCard>
  );
}

function LogTailSection({ logTail, runtime }: { logTail: LogTailMeta; runtime: Runtime }) {
  void runtime;
  return (
    <SettingsSectionCard eyebrow="Log tail" note="daemon log stream">
      <div
        className="flex flex-wrap items-center justify-between gap-3 rounded-md border border-(--color-divider) bg-(--color-surface-elevated) px-4 py-3"
        data-testid="settings-page-observability-log-tail"
        data-available={logTail.available ? "true" : "false"}
      >
        <div className="flex flex-col gap-1">
          <span className="text-sm text-(--color-text-primary)">
            {logTail.available ? "Live log tail available" : "Log tail unavailable"}
          </span>
          <span
            className="font-mono text-badge uppercase tracking-mono text-(--color-text-label)"
            data-testid="settings-page-observability-log-tail-transport"
          >
            transport: {logTail.transport ?? "none"}
          </span>
        </div>
        {logTail.available && logTail.stream_url ? (
          <a
            className="inline-flex items-center gap-1.5 text-sm text-accent hover:underline"
            data-testid="settings-page-observability-log-tail-link"
            href={logTail.stream_url}
            rel="noreferrer"
            target="_blank"
          >
            <ExternalLink className="size-3.5" />
            Open stream
          </a>
        ) : null}
      </div>
    </SettingsSectionCard>
  );
}

interface NumberFieldProps {
  label: string;
  testId: string;
  value: number;
  suffix?: string;
  errorMessage?: string;
  onValidityChange: (message: string | null) => void;
  onChange: (value: number) => void;
}

function NumberField({
  label,
  testId,
  value,
  suffix,
  errorMessage,
  onValidityChange,
  onChange,
}: NumberFieldProps) {
  return (
    <div className="flex flex-col gap-1">
      <span className="font-mono text-badge uppercase tracking-mono text-(--color-text-label)">
        {label}
      </span>
      <div className="flex items-center gap-2">
        <SettingsNumberInput
          className="w-full"
          min={0}
          data-testid={testId}
          value={value}
          onValidityChange={onValidityChange}
          onValueChange={onChange}
        />
        {suffix ? (
          <span className="font-mono text-badge uppercase tracking-mono text-(--color-text-label)">
            {suffix}
          </span>
        ) : null}
      </div>
      {errorMessage ? <span className="text-xs text-(--color-danger)">{errorMessage}</span> : null}
    </div>
  );
}

interface UsageBreakdownProps {
  globalBytes: number;
  sessionBytes: number;
  cap: number;
}

function UsageBreakdown({ globalBytes, sessionBytes, cap }: UsageBreakdownProps) {
  const total = Math.max(1, cap);
  const globalPct = Math.min(100, (globalBytes / total) * 100);
  const sessionPct = Math.min(100, (sessionBytes / total) * 100);

  return (
    <div className="flex flex-col gap-2" data-testid="settings-page-observability-usage-breakdown">
      <div className="flex items-center justify-between text-xs text-(--color-text-tertiary)">
        <span className="font-mono uppercase tracking-mono text-(--color-text-label)">
          Usage breakdown
        </span>
      </div>
      <div className="relative h-2 w-full overflow-hidden rounded-full bg-(--color-surface-panel)">
        <div
          className="absolute inset-y-0 left-0 bg-accent"
          style={{ width: `${globalPct}%` }}
          data-testid="settings-page-observability-usage-bar-global"
        />
        <div
          className="absolute inset-y-0 bg-(--color-info)"
          style={{ left: `${globalPct}%`, width: `${sessionPct}%` }}
          data-testid="settings-page-observability-usage-bar-sessions"
        />
      </div>
      <div className="flex flex-wrap gap-4 text-xs text-(--color-text-secondary)">
        <span className="inline-flex items-center gap-1.5">
          <span aria-hidden="true" className="size-2 rounded-full bg-accent" />
          global DB {formatBytes(globalBytes)}
        </span>
        <span className="inline-flex items-center gap-1.5">
          <span aria-hidden="true" className="size-2 rounded-full bg-(--color-info)" />
          session DB {formatBytes(sessionBytes)}
        </span>
      </div>
    </div>
  );
}
