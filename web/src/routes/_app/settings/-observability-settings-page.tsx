import { AlertCircle, Download, ExternalLink } from "lucide-react";
import { useState, type Dispatch, type SetStateAction } from "react";

import { useSettingsObservabilityPage } from "@/hooks/routes/use-settings-observability-page";
import {
  restartBannerPropsFor,
  SettingsFieldRow,
  SettingsSaveBar,
  type SettingsObservabilitySection,
} from "@/systems/settings";
import { useSupportBundleDownload } from "@/systems/support";
import {
  Button,
  Eyebrow,
  Metric,
  MetricGrid,
  PageShell,
  Pill,
  RestartBanner,
  Section,
  Spinner,
  StatusLineTopbarSlot,
  Switch,
  useTopbarSlot,
} from "@agh/ui";

type ObservabilityConfig = SettingsObservabilitySection["config"];
type LogTailMeta = SettingsObservabilitySection["log_tail"];
type Runtime = SettingsObservabilitySection["runtime"];

import { ObservabilityNumberField as NumberField, UsageBreakdown } from "./-observability-fields";
import { formatBytes } from "./-observability-format";

function safeLogTailURL(value: string | undefined): string | null {
  if (!value || !URL.canParse(value, "http://localhost")) return null;
  const protocol = new URL(value, "http://localhost").protocol;
  return protocol === "http:" || protocol === "https:" ? value : null;
}

export function ObservabilitySettingsPage() {
  const page = useSettingsObservabilityPage();
  const [validationErrors, setValidationErrors] = useState<Record<string, string | null>>({});
  const setValidationError = (key: string) => (message: string | null) => {
    setValidationErrors(current =>
      current[key] === message ? current : { ...current, [key]: message }
    );
  };
  const isInvalid = Object.values(validationErrors).some(message => message !== null);
  const runtimeForSlot = page.envelope?.runtime;
  const draftForSlot = page.draft;
  const totalStorageForSlot = runtimeForSlot
    ? runtimeForSlot.global_db_size_bytes + runtimeForSlot.session_db_size_bytes
    : 0;
  const capForSlot = draftForSlot?.max_global_bytes ?? 0;
  useTopbarSlot({
    tabs:
      runtimeForSlot && draftForSlot ? (
        <StatusLineTopbarSlot
          data-testid="settings-page-observability-status-line"
          status={runtimeForSlot.available ? "connected" : "error"}
          items={[
            {
              key: "sessions",
              value: `${runtimeForSlot.active_sessions} active sessions`,
              tone: "neutral",
            },
            {
              key: "storage",
              value: (
                <span data-testid="settings-page-observability-storage-summary">
                  storage {formatBytes(totalStorageForSlot)} / {formatBytes(capForSlot)}
                </span>
              ),
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

  const bannerProps = restartBannerPropsFor("observability", restart);

  return (
    <PageShell
      slug="observability"
      banner={bannerProps ? <RestartBanner {...bannerProps} /> : null}
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
          onReset={() => {
            setValidationErrors({});
            page.handleReset();
          }}
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
      <SupportBundleSection />
      <LogTailSection logTail={logTail} runtime={runtime} />
    </PageShell>
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
    <Section divided label="Runtime" note="live capture volume">
      <MetricGrid>
        <Metric
          label="Active sessions"
          value={String(activeSessions)}
          data-testid="settings-page-observability-metric-sessions"
        />
        <Metric
          label="Active agents"
          value={String(activeAgents)}
          data-testid="settings-page-observability-metric-agents"
        />
        <Metric
          label="Storage used"
          value={formatBytes(totalStorage)}
          subtext={`of ${formatBytes(cap)}`}
          data-testid="settings-page-observability-metric-storage"
        />
        <Metric
          label="Capacity"
          value={`${capPercent}%`}
          subtext="of soft cap"
          data-testid="settings-page-observability-metric-capacity"
        />
      </MetricGrid>
    </Section>
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
    <Section
      divided
      label="Capture"
      note="events, transcripts, logs"
      right={
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
      <UsageBreakdown globalBytes={globalBytes} sessionBytes={sessionBytes} cap={cap} />
    </Section>
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
    <Section divided label="Transcripts" note="full replay of agent I/O">
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
    </Section>
  );
}

function LogTailSection({ logTail, runtime }: { logTail: LogTailMeta; runtime: Runtime }) {
  void runtime;
  const streamURL = safeLogTailURL(logTail.stream_url);
  return (
    <Section divided label="Log tail" note="daemon log stream">
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
    </Section>
  );
}

function SupportBundleSection() {
  const supportBundle = useSupportBundleDownload();
  const [approved, setApproved] = useState(false);
  const [consentError, setConsentError] = useState<string | null>(null);
  const operation = supportBundle.operation;
  const operationStatus = operation?.status ?? "idle";
  const errorMessage =
    consentError ??
    (supportBundle.error instanceof Error ? supportBundle.error.message : undefined);

  const handleCreate = async () => {
    if (!approved) {
      setConsentError("Approval is required before creating a support bundle.");
      return;
    }
    setConsentError(null);
    await supportBundle.create({ includeStatus: true, yes: true });
  };

  return (
    <Section divided label="Support bundle" note="redacted daemon archive">
      <div
        className="flex flex-col gap-4 rounded-md border border-line bg-elevated px-4 py-3"
        data-testid="settings-page-observability-support-bundle"
      >
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div className="flex flex-col gap-1">
            <span className="text-sm text-fg">Create support bundle</span>
            <Eyebrow
              className="text-muted"
              data-testid="settings-page-observability-support-bundle-status"
            >
              status: {operationStatus}
            </Eyebrow>
          </div>
          <Button
            type="button"
            size="sm"
            disabled={supportBundle.isPending}
            onClick={() => {
              void handleCreate().catch(() => undefined);
            }}
            data-testid="settings-page-observability-support-bundle-button"
          >
            {supportBundle.isPending ? (
              <Spinner className="size-3.5" />
            ) : (
              <Download className="size-3.5" />
            )}
            {supportBundle.isPending ? "Preparing" : "Create bundle"}
          </Button>
        </div>
        <label className="flex items-start gap-3 text-sm text-subtle">
          <input
            type="checkbox"
            checked={approved}
            onChange={event => {
              setApproved(event.currentTarget.checked);
              if (event.currentTarget.checked) {
                setConsentError(null);
              }
            }}
            className="mt-0.5 size-4 rounded border border-line bg-canvas-soft accent-[var(--accent)] focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent"
            data-testid="settings-page-observability-support-bundle-consent"
          />
          <span>I approve creating a redacted diagnostics archive.</span>
        </label>
        {operation?.size_bytes ? (
          <Eyebrow className="text-muted" data-testid="settings-page-observability-support-size">
            size: {formatBytes(operation.size_bytes)}
          </Eyebrow>
        ) : null}
        {errorMessage ? (
          <p
            role="alert"
            className="text-sm text-danger"
            data-testid="settings-page-observability-support-bundle-error"
          >
            {errorMessage}
          </p>
        ) : null}
      </div>
    </Section>
  );
}
