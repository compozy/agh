import { AlertCircle, Loader2 } from "lucide-react";
import { createFileRoute } from "@tanstack/react-router";
import type { Dispatch, SetStateAction } from "react";

import { Input, Pills } from "@agh/ui";
import { useSettingsGeneralPage } from "@/hooks/routes/use-settings-general-page";
import type { SettingsGeneralSection } from "@/systems/settings";
import {
  SettingsFieldRow,
  SettingsPageActions,
  SettingsPageShell,
  SettingsRestartBanner,
  SettingsSaveBar,
  SettingsSectionCard,
  SettingsStatGrid,
  SettingsStatItem,
  SettingsStatusLine,
} from "@/systems/settings/components";

export const Route = createFileRoute("/_app/settings/general")({
  component: GeneralSettingsPage,
});

const PERMISSION_MODES = ["deny-all", "approve-reads", "approve-all"] as const;
type PermissionMode = (typeof PERMISSION_MODES)[number];

type GeneralConfig = SettingsGeneralSection["config"];

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

function GeneralSettingsPage() {
  const page = useSettingsGeneralPage();

  if (page.isLoading) {
    return (
      <div
        className="flex flex-1 items-center justify-center"
        data-testid="settings-page-general-loading"
      >
        <Loader2 className="size-5 animate-spin text-[color:var(--color-text-tertiary)]" />
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
          <AlertCircle className="size-6 text-[color:var(--color-danger)]" />
          <p className="text-sm text-[color:var(--color-text-tertiary)]">
            {page.error?.message ?? "Failed to load general settings"}
          </p>
        </div>
      </div>
    );
  }

  const { envelope, draft, setDraft, restart } = page;
  const runtime = envelope.runtime;
  const configPaths = envelope.config_paths;

  return (
    <SettingsPageShell
      slug="general"
      title="General"
      statusLine={
        <SettingsStatusLine
          data-testid="settings-page-general-status-line"
          daemonAvailable={runtime.available}
          items={[
            <span key="sessions">
              {runtime.active_sessions} active sessions · {runtime.active_agents} agents
            </span>,
            <span key="config" className="font-mono text-[0.7rem]">
              config: {configPaths.global_config}
            </span>,
          ]}
        />
      }
      actions={<SettingsPageActions slug="general" restart={restart} />}
      banner={<SettingsRestartBanner slug="general" restart={restart} />}
      footer={
        <SettingsSaveBar
          slug="general"
          isDirty={page.isDirty}
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
      <DefaultsSection draft={draft} setDraft={setDraft} />
      <PermissionsSection draft={draft} setDraft={setDraft} />
      <SessionSection draft={draft} setDraft={setDraft} />
    </SettingsPageShell>
  );
}

function RuntimeSection({ envelope }: { envelope: SettingsGeneralSection }) {
  const runtime = envelope.runtime;
  return (
    <SettingsSectionCard eyebrow="Runtime" note="read-only">
      <SettingsStatGrid>
        <SettingsStatItem
          label="UDS socket"
          value={runtime.socket ?? envelope.config.daemon.socket}
        />
        <SettingsStatItem
          label="HTTP bind"
          value={
            runtime.http_host && runtime.http_port
              ? `${runtime.http_host}:${runtime.http_port}`
              : `${envelope.config.http.host}:${envelope.config.http.port}`
          }
        />
        <SettingsStatItem
          label="Active sessions"
          value={`${runtime.active_sessions} / ${envelope.config.limits.max_sessions} max`}
        />
        <SettingsStatItem
          label="Concurrent agents"
          value={`${runtime.active_agents} / ${envelope.config.limits.max_concurrent_agents} max`}
        />
      </SettingsStatGrid>
    </SettingsSectionCard>
  );
}

interface DraftSectionProps {
  draft: GeneralConfig;
  setDraft: Dispatch<SetStateAction<GeneralConfig | null>>;
}

function DefaultsSection({ draft, setDraft }: DraftSectionProps) {
  return (
    <SettingsSectionCard eyebrow="Defaults" note="applied to new sessions">
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
              setDraft({
                ...draft,
                defaults: { ...draft.defaults, agent: event.target.value },
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
              setDraft({
                ...draft,
                defaults: { ...draft.defaults, provider: event.target.value },
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid="settings-page-general-default-environment"
        label="Default environment"
        description="Execution profile for new workspaces"
        hint="DEFAULT"
        control={
          <Input
            className="w-56 font-mono"
            data-testid="settings-page-general-default-environment-input"
            value={draft.defaults.environment ?? ""}
            placeholder="local"
            onChange={event =>
              setDraft({
                ...draft,
                defaults: { ...draft.defaults, environment: event.target.value },
              })
            }
          />
        }
      />
    </SettingsSectionCard>
  );
}

function PermissionsSection({ draft, setDraft }: DraftSectionProps) {
  return (
    <SettingsSectionCard eyebrow="Permissions" note="tool approval policy">
      <Pills
        data-testid="settings-page-general-permissions-group"
        aria-label="Tool approval policy"
        value={draft.permissions.mode}
        onChange={mode => setDraft({ ...draft, permissions: { mode } })}
        items={PERMISSION_MODES.map(mode => ({
          value: mode,
          label: mode,
          testId: `settings-page-general-permission-${mode}`,
        }))}
      />
      <p className="text-xs text-[color:var(--color-text-tertiary)]">
        {describePermissionMode(draft.permissions.mode)}
      </p>
    </SettingsSectionCard>
  );
}

function SessionSection({ draft, setDraft }: DraftSectionProps) {
  return (
    <SettingsSectionCard eyebrow="Session" note="runtime limits">
      <SettingsFieldRow
        data-testid="settings-page-general-session-timeout"
        label="Session timeout"
        description="0 disables force-close"
        hint="SECONDS"
        control={
          <div className="flex items-center gap-2">
            <Input
              type="number"
              min={0}
              className="w-28"
              data-testid="settings-page-general-session-timeout-input"
              value={parseSessionTimeoutSeconds(draft.session_timeout)}
              onChange={event =>
                setDraft({
                  ...draft,
                  session_timeout: formatSessionTimeout(Number(event.target.value || 0)),
                })
              }
            />
            <span className="font-mono text-[0.6rem] uppercase tracking-[0.18em] text-[color:var(--color-text-label)]">
              seconds
            </span>
          </div>
        }
      />
    </SettingsSectionCard>
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
