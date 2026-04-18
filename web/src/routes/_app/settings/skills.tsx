import { AlertCircle, ExternalLink, Loader2, Wrench } from "lucide-react";
import { createFileRoute, Link } from "@tanstack/react-router";
import type { Dispatch, SetStateAction } from "react";

import { Button, Switch } from "@agh/ui";
import { useSettingsSkillsPage } from "@/hooks/routes/use-settings-skills-page";
import type { SettingsSkillsSection } from "@/systems/settings";
import {
  SettingsFieldRow,
  SettingsPageActions,
  SettingsPageShell,
  SettingsRestartBanner,
  SettingsSectionCard,
  SettingsStatusLine,
} from "@/systems/settings/components";

export const Route = createFileRoute("/_app/settings/skills")({
  component: SkillsSettingsPage,
});

type SkillsConfig = SettingsSkillsSection["config"];

function SkillsSettingsPage() {
  const page = useSettingsSkillsPage();

  if (page.isLoading) {
    return (
      <div
        className="flex flex-1 items-center justify-center"
        data-testid="settings-page-skills-loading"
      >
        <Loader2 className="size-5 animate-spin text-[color:var(--color-text-tertiary)]" />
      </div>
    );
  }

  if (page.error || !page.envelope || !page.draft) {
    return (
      <div
        className="flex flex-1 items-center justify-center"
        data-testid="settings-page-skills-error"
      >
        <div className="flex flex-col items-center gap-2 text-center">
          <AlertCircle className="size-6 text-[color:var(--color-danger)]" />
          <p className="text-sm text-[color:var(--color-text-tertiary)]">
            {page.error?.message ?? "Failed to load skills settings"}
          </p>
        </div>
      </div>
    );
  }

  const { envelope, draft, setDraft, restart } = page;

  return (
    <SettingsPageShell
      slug="skills"
      title="Skills"
      statusLine={
        <SettingsStatusLine
          data-testid="settings-page-skills-status-line"
          daemonAvailable={envelope.runtime_available}
          items={[
            <span key="discovered">{envelope.discovered_count} discovered</span>,
            <span key="disabled">{envelope.disabled_count} disabled</span>,
          ]}
        />
      }
      actions={<SettingsPageActions slug="skills" restart={restart} />}
      banner={<SettingsRestartBanner slug="skills" restart={restart} />}
    >
      <OperationalLinksRow />
      <DisabledSkillsSection
        envelope={envelope}
        draft={draft}
        onToggle={page.toggleDisabled}
        isDirty={page.isDisabledDirty}
        isSaving={page.isSavingDisabled}
        error={page.saveDisabledError}
        warnings={page.disabledWarnings}
        lastAppliedLabel={page.lastDisabledLabel}
        onSave={page.handleSaveDisabled}
        onReset={page.handleResetDisabled}
      />
      <PolicySection
        draft={draft}
        setDraft={setDraft}
        isDirty={page.isPolicyDirty}
        isSaving={page.isSavingPolicy}
        error={page.savePolicyError}
        warnings={page.policyWarnings}
        lastAppliedLabel={page.lastPolicyLabel}
        onSave={page.handleSavePolicy}
        onReset={page.handleResetPolicy}
      />
    </SettingsPageShell>
  );
}

function OperationalLinksRow() {
  return (
    <SettingsSectionCard eyebrow="Operational" note="manage runtime state outside of settings">
      <div className="flex flex-wrap gap-2" data-testid="settings-page-skills-operational-links">
        <Link
          to="/skills"
          className="inline-flex items-center gap-1.5 rounded-md border border-[color:var(--color-divider)] bg-[color:var(--color-surface-elevated)] px-3 py-1.5 text-xs font-medium text-[color:var(--color-text-primary)] hover:bg-[color:var(--color-hover)]"
          data-testid="settings-page-skills-link-skills"
        >
          <ExternalLink className="size-3.5 text-[color:var(--color-text-tertiary)]" />
          Open Skills
        </Link>
      </div>
    </SettingsSectionCard>
  );
}

interface DisabledSkillsSectionProps {
  envelope: SettingsSkillsSection;
  draft: SkillsConfig;
  onToggle: (name: string) => void;
  isDirty: boolean;
  isSaving: boolean;
  error: string | null;
  warnings?: string[];
  lastAppliedLabel: string | null;
  onSave: () => void;
  onReset: () => void;
}

function DisabledSkillsSection({
  envelope,
  draft,
  onToggle,
  isDirty,
  isSaving,
  error,
  warnings,
  lastAppliedLabel,
  onSave,
  onReset,
}: DisabledSkillsSectionProps) {
  const disabled = draft.disabled_skills ?? [];
  const baseline = envelope.config.disabled_skills ?? [];
  const candidates = Array.from(new Set([...baseline, ...disabled])).sort();

  return (
    <SettingsSectionCard
      eyebrow="Disabled skills"
      note="applies immediately · no restart required"
      headerAction={
        <SaveControls
          slug="disabled"
          saveLabel="Apply"
          isDirty={isDirty}
          isSaving={isSaving}
          error={error}
          warnings={warnings}
          lastAppliedLabel={lastAppliedLabel}
          onSave={onSave}
          onReset={onReset}
        />
      }
    >
      {candidates.length === 0 ? (
        <p
          className="text-xs text-[color:var(--color-text-tertiary)]"
          data-testid="settings-page-skills-disabled-empty"
        >
          No skills have been disabled. Manage availability from the Skills operational page.
        </p>
      ) : (
        <ul className="flex flex-col gap-2" data-testid="settings-page-skills-disabled-list">
          {candidates.map(name => {
            const isDisabled = disabled.includes(name);
            return (
              <li
                key={name}
                className="flex items-center justify-between gap-3 rounded-md border border-[color:var(--color-divider)] bg-[color:var(--color-surface-elevated)] px-3 py-2"
                data-testid={`settings-page-skills-disabled-item-${name}`}
              >
                <div className="flex min-w-0 items-center gap-2">
                  <Wrench className="size-3.5 text-[color:var(--color-text-tertiary)]" />
                  <span className="truncate text-sm text-[color:var(--color-text-primary)]">
                    {name}
                  </span>
                </div>
                <Switch
                  data-testid={`settings-page-skills-disabled-toggle-${name}`}
                  checked={isDisabled}
                  onCheckedChange={() => onToggle(name)}
                  aria-label={`Toggle ${name}`}
                />
              </li>
            );
          })}
        </ul>
      )}
    </SettingsSectionCard>
  );
}

interface PolicySectionProps {
  draft: SkillsConfig;
  setDraft: Dispatch<SetStateAction<SkillsConfig | null>>;
  isDirty: boolean;
  isSaving: boolean;
  error: string | null;
  warnings?: string[];
  lastAppliedLabel: string | null;
  onSave: () => void;
  onReset: () => void;
}

function PolicySection({
  draft,
  setDraft,
  isDirty,
  isSaving,
  error,
  warnings,
  lastAppliedLabel,
  onSave,
  onReset,
}: PolicySectionProps) {
  return (
    <SettingsSectionCard
      eyebrow="Marketplace & policy"
      note="restart required to apply"
      headerAction={
        <SaveControls
          slug="policy"
          saveLabel="Save"
          isDirty={isDirty}
          isSaving={isSaving}
          error={error}
          warnings={warnings}
          lastAppliedLabel={lastAppliedLabel}
          onSave={onSave}
          onReset={onReset}
        />
      }
    >
      <SettingsFieldRow
        data-testid="settings-page-skills-enabled"
        label="Skill registry"
        description="Enable discovery and task resolution"
        hint="CONFIG.TOML"
        control={
          <Switch
            data-testid="settings-page-skills-enabled-switch"
            checked={draft.enabled}
            onCheckedChange={checked => setDraft({ ...draft, enabled: checked })}
          />
        }
      />
      <SettingsFieldRow
        data-testid="settings-page-skills-poll-interval"
        label="Poll interval"
        description="How often the registry re-scans sources"
        hint="DEFAULT"
        control={
          <input
            className="h-8 w-32 rounded-md border border-[color:var(--color-divider)] bg-[color:var(--color-surface-elevated)] px-2 font-mono text-sm text-[color:var(--color-text-primary)]"
            data-testid="settings-page-skills-poll-interval-input"
            value={draft.poll_interval ?? ""}
            placeholder="5m"
            onChange={event => setDraft({ ...draft, poll_interval: event.target.value })}
          />
        }
      />
      <SettingsFieldRow
        data-testid="settings-page-skills-marketplace-registry"
        label="Marketplace registry"
        description="Identifier of the marketplace publisher"
        hint="CONFIG.TOML"
        control={
          <input
            className="h-8 w-56 rounded-md border border-[color:var(--color-divider)] bg-[color:var(--color-surface-elevated)] px-2 text-sm text-[color:var(--color-text-primary)]"
            data-testid="settings-page-skills-marketplace-registry-input"
            value={draft.marketplace.registry ?? ""}
            onChange={event =>
              setDraft({
                ...draft,
                marketplace: { ...draft.marketplace, registry: event.target.value },
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid="settings-page-skills-marketplace-base-url"
        label="Marketplace base URL"
        description="Override the registry's default endpoint"
        hint="OPTIONAL"
        control={
          <input
            className="h-8 w-72 rounded-md border border-[color:var(--color-divider)] bg-[color:var(--color-surface-elevated)] px-2 font-mono text-sm text-[color:var(--color-text-primary)]"
            data-testid="settings-page-skills-marketplace-base-url-input"
            value={draft.marketplace.base_url ?? ""}
            placeholder="https://"
            onChange={event =>
              setDraft({
                ...draft,
                marketplace: { ...draft.marketplace, base_url: event.target.value },
              })
            }
          />
        }
      />
      <AllowListField
        label="Allowed MCP installs"
        description="Comma-separated list of marketplace MCP packages that may be installed"
        testId="settings-page-skills-allowed-mcp"
        value={draft.allowed_marketplace_mcp ?? []}
        onChange={value => setDraft({ ...draft, allowed_marketplace_mcp: value })}
      />
      <AllowListField
        label="Allowed hook installs"
        description="Comma-separated list of marketplace hook packages that may be installed"
        testId="settings-page-skills-allowed-hooks"
        value={draft.allowed_marketplace_hooks ?? []}
        onChange={value => setDraft({ ...draft, allowed_marketplace_hooks: value })}
      />
    </SettingsSectionCard>
  );
}

interface AllowListFieldProps {
  label: string;
  description: string;
  testId: string;
  value: string[];
  onChange: (value: string[]) => void;
}

function AllowListField({ label, description, testId, value, onChange }: AllowListFieldProps) {
  return (
    <SettingsFieldRow
      data-testid={testId}
      label={label}
      description={description}
      hint="LIST"
      control={
        <input
          className="h-8 w-72 rounded-md border border-[color:var(--color-divider)] bg-[color:var(--color-surface-elevated)] px-2 font-mono text-sm text-[color:var(--color-text-primary)]"
          data-testid={`${testId}-input`}
          value={value.join(", ")}
          placeholder="none"
          onChange={event =>
            onChange(
              event.target.value
                .split(",")
                .map(entry => entry.trim())
                .filter(entry => entry.length > 0)
            )
          }
        />
      }
    />
  );
}

interface SaveControlsProps {
  slug: string;
  saveLabel: string;
  isDirty: boolean;
  isSaving: boolean;
  error: string | null;
  warnings?: string[];
  lastAppliedLabel: string | null;
  onSave: () => void;
  onReset: () => void;
}

function SaveControls({
  slug,
  saveLabel,
  isDirty,
  isSaving,
  error,
  warnings,
  lastAppliedLabel,
  onSave,
  onReset,
}: SaveControlsProps) {
  const disabled = !isDirty || isSaving;
  return (
    <div
      className="flex items-center gap-2"
      data-testid={`settings-page-skills-${slug}-controls`}
      data-dirty={isDirty ? "true" : "false"}
    >
      {error ? (
        <span
          className="text-xs text-[color:var(--color-danger)]"
          data-testid={`settings-page-skills-${slug}-error`}
        >
          {error}
        </span>
      ) : warnings && warnings.length > 0 ? (
        <span
          className="text-xs text-[color:var(--color-warning)]"
          data-testid={`settings-page-skills-${slug}-warning`}
        >
          {warnings.join(" · ")}
        </span>
      ) : lastAppliedLabel ? (
        <span
          className="text-xs text-[color:var(--color-text-tertiary)]"
          data-testid={`settings-page-skills-${slug}-applied`}
        >
          {lastAppliedLabel}
        </span>
      ) : null}
      <Button
        type="button"
        variant="ghost"
        size="sm"
        onClick={onReset}
        disabled={!isDirty || isSaving}
        data-testid={`settings-page-skills-${slug}-reset`}
      >
        Discard
      </Button>
      <Button
        type="button"
        variant="default"
        size="sm"
        onClick={onSave}
        disabled={disabled}
        data-testid={`settings-page-skills-${slug}-save`}
      >
        {isSaving ? <Loader2 className="size-3.5 animate-spin" /> : null}
        {isSaving ? "Saving…" : saveLabel}
      </Button>
    </div>
  );
}
