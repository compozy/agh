import { AlertCircle, Loader2, Play } from "lucide-react";
import { createFileRoute } from "@tanstack/react-router";
import { useCallback, useMemo, useState, type Dispatch, type SetStateAction } from "react";

import { Button, Input, Switch } from "@agh/ui";
import { useSettingsMemoryPage } from "@/hooks/routes/use-settings-memory-page";
import type { SettingsMemorySection } from "@/systems/settings";
import {
  SettingsFieldRow,
  SettingsNumberInput,
  SettingsPageActions,
  SettingsPageShell,
  SettingsRestartBanner,
  SettingsSaveBar,
  SettingsSectionCard,
  SettingsStatusLine,
} from "@/systems/settings/components";

export const Route = createFileRoute("/_app/settings/memory")({
  component: MemorySettingsPage,
});

type MemoryConfig = SettingsMemorySection["config"];

function MemorySettingsPage() {
  const page = useSettingsMemoryPage();
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
        data-testid="settings-page-memory-loading"
      >
        <Loader2 className="size-5 animate-spin text-[color:var(--color-text-tertiary)]" />
      </div>
    );
  }

  if (page.error || !page.envelope || !page.draft) {
    return (
      <div
        className="flex flex-1 items-center justify-center"
        data-testid="settings-page-memory-error"
      >
        <div className="flex flex-col items-center gap-2 text-center">
          <AlertCircle className="size-6 text-[color:var(--color-danger)]" />
          <p className="text-sm text-[color:var(--color-text-tertiary)]">
            {page.error?.message ?? "Failed to load memory settings"}
          </p>
          <Button onClick={page.handleRetry} size="sm" type="button" variant="outline">
            Retry
          </Button>
        </div>
      </div>
    );
  }

  const { envelope, draft, setDraft, restart } = page;
  const health = envelope.health;

  return (
    <SettingsPageShell
      slug="memory"
      title="Memory"
      statusLine={
        <SettingsStatusLine
          data-testid="settings-page-memory-status-line"
          daemonAvailable={health.available}
          items={[
            <span key="files">{health.file_count} memory files</span>,
            <span key="last" data-testid="settings-page-memory-last-consolidated">
              {health.last_consolidated_at
                ? `last run ${new Date(health.last_consolidated_at).toLocaleString()}`
                : "never consolidated"}
            </span>,
          ]}
        />
      }
      actions={<SettingsPageActions slug="memory" restart={restart} />}
      banner={<SettingsRestartBanner slug="memory" restart={restart} />}
      footer={
        <SettingsSaveBar
          slug="memory"
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
      <MemorySystemSection draft={draft} setDraft={setDraft} />
      <DreamSection
        draft={draft}
        setDraft={setDraft}
        consolidateAvailable={
          envelope.actions.consolidate.available && envelope.health.dream_enabled
        }
        consolidatePending={page.isConsolidating}
        onConsolidate={page.handleConsolidate}
        actionMessage={page.actionMessage}
        validationErrors={validationErrors}
        setValidationError={setValidationError}
      />
    </SettingsPageShell>
  );
}

interface DraftSectionProps {
  draft: MemoryConfig;
  setDraft: Dispatch<SetStateAction<MemoryConfig | null>>;
}

function MemorySystemSection({ draft, setDraft }: DraftSectionProps) {
  return (
    <SettingsSectionCard eyebrow="Memory system">
      <SettingsFieldRow
        data-testid="settings-page-memory-enabled"
        label="Memory persistence"
        description="Save structured recall across sessions"
        control={
          <Switch
            data-testid="settings-page-memory-enabled-switch"
            checked={draft.enabled}
            onCheckedChange={checked => setDraft({ ...draft, enabled: checked })}
          />
        }
      />
      <SettingsFieldRow
        data-testid="settings-page-memory-global-dir"
        label="Global memory directory"
        description="Where user + feedback memories live"
        hint="DEFAULT"
        control={
          <Input
            className="w-72 font-mono"
            data-testid="settings-page-memory-global-dir-input"
            value={draft.global_dir ?? ""}
            placeholder="~/.agh/memory"
            onChange={event => setDraft({ ...draft, global_dir: event.target.value })}
          />
        }
      />
    </SettingsSectionCard>
  );
}

interface DreamSectionProps extends DraftSectionProps {
  consolidateAvailable: boolean;
  consolidatePending: boolean;
  onConsolidate: () => void;
  actionMessage: string | null;
  validationErrors: Record<string, string | null>;
  setValidationError: (key: string) => (message: string | null) => void;
}

function DreamSection({
  draft,
  setDraft,
  consolidateAvailable,
  consolidatePending,
  onConsolidate,
  actionMessage,
  validationErrors,
  setValidationError,
}: DreamSectionProps) {
  const dreamDisabled = !draft.dream.enabled;
  return (
    <SettingsSectionCard
      eyebrow="Dream consolidation"
      note="background compaction"
      headerAction={
        <Button
          type="button"
          variant="outline"
          size="sm"
          data-testid="settings-page-memory-consolidate"
          disabled={!consolidateAvailable || consolidatePending}
          onClick={onConsolidate}
        >
          {consolidatePending ? (
            <Loader2 className="size-3.5 animate-spin" />
          ) : (
            <Play className="size-3.5" />
          )}
          Trigger now
        </Button>
      }
    >
      <SettingsFieldRow
        data-testid="settings-page-memory-dream-enabled"
        label="Automatic consolidation"
        description="Review recent sessions, merge repeats, rewrite user memory"
        control={
          <Switch
            data-testid="settings-page-memory-dream-enabled-switch"
            checked={draft.dream.enabled}
            onCheckedChange={checked =>
              setDraft({ ...draft, dream: { ...draft.dream, enabled: checked } })
            }
          />
        }
      />
      <div
        className="grid gap-4 md:grid-cols-2 xl:grid-cols-4"
        data-disabled={dreamDisabled ? "true" : undefined}
      >
        <DreamField
          label="Consolidation agent"
          testId="settings-page-memory-dream-agent"
          value={draft.dream.agent}
          disabled={dreamDisabled}
          onChange={value => setDraft({ ...draft, dream: { ...draft.dream, agent: value } })}
        />
        <DreamField
          label="Min idle hours"
          testId="settings-page-memory-dream-min-hours"
          type="number"
          value={String(draft.dream.min_hours)}
          errorMessage={validationErrors.minHours ?? undefined}
          suffix="h"
          disabled={dreamDisabled}
          onValidityChange={setValidationError("minHours")}
          onChange={value =>
            setDraft({
              ...draft,
              dream: { ...draft.dream, min_hours: Number(value) || 0 },
            })
          }
        />
        <DreamField
          label="Min sessions"
          testId="settings-page-memory-dream-min-sessions"
          type="number"
          value={String(draft.dream.min_sessions)}
          errorMessage={validationErrors.minSessions ?? undefined}
          disabled={dreamDisabled}
          onValidityChange={setValidationError("minSessions")}
          onChange={value =>
            setDraft({
              ...draft,
              dream: { ...draft.dream, min_sessions: Number(value) || 0 },
            })
          }
        />
        <DreamField
          label="Check interval"
          testId="settings-page-memory-dream-check-interval"
          value={draft.dream.check_interval}
          disabled={dreamDisabled}
          onChange={value =>
            setDraft({ ...draft, dream: { ...draft.dream, check_interval: value } })
          }
        />
      </div>
      {actionMessage ? (
        <p
          className="text-xs text-[color:var(--color-text-tertiary)]"
          data-testid="settings-page-memory-action-message"
        >
          {actionMessage}
        </p>
      ) : null}
    </SettingsSectionCard>
  );
}

interface DreamFieldProps {
  label: string;
  testId: string;
  value: string;
  type?: "text" | "number";
  suffix?: string;
  errorMessage?: string;
  disabled?: boolean;
  onValidityChange?: (message: string | null) => void;
  onChange: (value: string) => void;
}

function DreamField({
  label,
  testId,
  value,
  type = "text",
  suffix,
  errorMessage,
  disabled,
  onValidityChange,
  onChange,
}: DreamFieldProps) {
  return (
    <div className="flex flex-col gap-1">
      <span className="font-mono text-[0.6rem] uppercase tracking-[0.18em] text-[color:var(--color-text-label)]">
        {label}
      </span>
      <div className="flex items-center gap-2">
        {type === "number" ? (
          <SettingsNumberInput
            className="w-full"
            data-testid={testId}
            value={Number.parseInt(value || "0", 10)}
            disabled={disabled}
            min={0}
            onValidityChange={onValidityChange}
            onValueChange={next => onChange(String(next))}
          />
        ) : (
          <Input
            type={type}
            className="w-full"
            data-testid={testId}
            value={value}
            disabled={disabled}
            onChange={event => onChange(event.target.value)}
          />
        )}
        {suffix ? (
          <span className="font-mono text-[0.6rem] uppercase tracking-[0.18em] text-[color:var(--color-text-label)]">
            {suffix}
          </span>
        ) : null}
      </div>
      {errorMessage ? (
        <span className="text-xs text-[color:var(--color-danger)]">{errorMessage}</span>
      ) : null}
    </div>
  );
}
