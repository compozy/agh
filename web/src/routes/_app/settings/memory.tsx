import { AlertCircle, Loader2, Play } from "lucide-react";
import { createFileRoute } from "@tanstack/react-router";
import type { Dispatch, SetStateAction } from "react";

import { Button } from "@agh/ui";
import { Switch } from "@/components/ui/switch";
import { useSettingsMemoryPage } from "@/hooks/routes/use-settings-memory-page";
import type { SettingsMemorySection } from "@/systems/settings";
import {
  SettingsFieldRow,
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
      banner={<SettingsRestartBanner slug="memory" restart={restart} />}
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
      />

      <SettingsSaveBar
        slug="memory"
        isDirty={page.isDirty}
        isSaving={page.isSaving}
        error={page.saveError}
        warnings={page.warnings}
        lastAppliedLabel={page.lastAppliedLabel}
        onSave={page.handleSave}
        onReset={page.handleReset}
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
          <input
            className="h-8 w-72 rounded-md border border-[color:var(--color-divider)] bg-[color:var(--color-surface-elevated)] px-2 font-mono text-sm text-[color:var(--color-text-primary)]"
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
}

function DreamSection({
  draft,
  setDraft,
  consolidateAvailable,
  consolidatePending,
  onConsolidate,
  actionMessage,
}: DreamSectionProps) {
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
      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <DreamField
          label="Consolidation agent"
          testId="settings-page-memory-dream-agent"
          value={draft.dream.agent}
          onChange={value => setDraft({ ...draft, dream: { ...draft.dream, agent: value } })}
        />
        <DreamField
          label="Min idle hours"
          testId="settings-page-memory-dream-min-hours"
          type="number"
          value={String(draft.dream.min_hours)}
          suffix="h"
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
  onChange: (value: string) => void;
}

function DreamField({ label, testId, value, type = "text", suffix, onChange }: DreamFieldProps) {
  return (
    <div className="flex flex-col gap-1">
      <span className="font-mono text-[0.6rem] uppercase tracking-[0.18em] text-[color:var(--color-text-label)]">
        {label}
      </span>
      <div className="flex items-center gap-2">
        <input
          type={type}
          className="h-8 w-full rounded-md border border-[color:var(--color-divider)] bg-[color:var(--color-surface-elevated)] px-2 text-sm text-[color:var(--color-text-primary)]"
          data-testid={testId}
          value={value}
          onChange={event => onChange(event.target.value)}
        />
        {suffix ? (
          <span className="font-mono text-[0.6rem] uppercase tracking-[0.18em] text-[color:var(--color-text-label)]">
            {suffix}
          </span>
        ) : null}
      </div>
    </div>
  );
}
