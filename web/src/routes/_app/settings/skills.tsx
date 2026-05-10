import { AlertCircle, ExternalLink, Loader2, Wrench } from "lucide-react";
import { createFileRoute, Link } from "@tanstack/react-router";
import type { Dispatch, SetStateAction } from "react";

import {
  Button,
  Empty,
  Input,
  NativeSelect,
  NativeSelectOption,
  PageShell,
  PillGroup,
  Section,
  Switch,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  useTopbarSlot,
} from "@agh/ui";
import type { TopbarRouteContext } from "@/types/topbar";
import {
  useSettingsSkillsPage,
  type SkillsScopeSelection,
} from "@/hooks/routes/use-settings-skills-page";
import { AgentCommandSelect, type AgentPayload } from "@/systems/agent";
import type { SettingsScope, SettingsSkillsSection } from "@/systems/settings";
import {
  SettingsFieldRow,
  SettingsPageActions,
  SettingsRestartBanner,
  SettingsStatusLine,
} from "@/systems/settings/components";
import type { WorkspacePayload } from "@/systems/workspace";

export const Route = createFileRoute("/_app/settings/skills")({
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { title: "Skills settings", icon: Wrench },
  }),
  component: SkillsSettingsPage,
});

type SkillsConfig = SettingsSkillsSection["config"];

function SkillsSettingsPage() {
  const page = useSettingsSkillsPage();
  const envelopeForSlot = page.envelope;
  useTopbarSlot({
    tabs: envelopeForSlot ? (
      <SettingsStatusLine
        data-testid="settings-page-skills-status-line"
        status={envelopeForSlot.runtime_available ? "connected" : "error"}
        items={[
          <span key="discovered">{envelopeForSlot.discovered_count} discovered</span>,
          <span key="disabled">{envelopeForSlot.disabled_count} disabled</span>,
          <span key="scope" data-testid="settings-page-skills-scope-label">
            scope:{" "}
            {page.selection.scope === "global"
              ? "global"
              : `agent ${page.selectedAgent?.name ?? page.selection.agentName}`}
          </span>,
          page.selection.scope === "agent" && page.selectedWorkspaceContext ? (
            <span key="context" data-testid="settings-page-skills-workspace-context-summary">
              context: {page.selectedWorkspaceContext.name}
            </span>
          ) : null,
        ]}
      />
    ) : undefined,
    actions: envelopeForSlot ? (
      <SettingsPageActions slug="skills" restart={page.restart} />
    ) : undefined,
  });

  if (page.isLoading) {
    return (
      <div
        className="flex flex-1 items-center justify-center"
        data-testid="settings-page-skills-loading"
      >
        <Loader2 className="size-5 animate-spin text-(--subtle)" />
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
          <AlertCircle className="size-6 text-(--danger)" />
          <p className="text-sm text-(--subtle)">
            {page.error?.message ?? "Failed to load skills settings"}
          </p>
          <Button onClick={page.handleRetry} size="sm" type="button" variant="outline">
            Retry
          </Button>
        </div>
      </div>
    );
  }

  const { envelope, draft, setDraft, restart } = page;

  return (
    <PageShell
      slug="skills"
      banner={<SettingsRestartBanner slug="skills" restart={restart} />}
    >
      <ScopeSelector
        selection={page.selection}
        availableScopes={page.availableScopes}
        agents={page.agents}
        workspaces={page.workspaces}
        onSelectGlobal={page.selectGlobal}
        onSelectAgentScope={page.selectAgentScope}
        onSelectAgent={page.selectAgent}
        onSelectWorkspaceContext={page.selectWorkspaceContext}
      />
      <OperationalLinksRow />
      <DisabledSkillsSection
        envelope={envelope}
        selection={page.selection}
        selectedAgent={page.selectedAgent}
        selectedWorkspaceContext={page.selectedWorkspaceContext}
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
      {page.selection.scope === "global" ? (
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
      ) : (
        <AgentScopePolicyNotice />
      )}
    </PageShell>
  );
}

type SkillsScopeValue = "global" | "agent";

interface ScopeSelectorProps {
  selection: SkillsScopeSelection;
  availableScopes: readonly SettingsScope[];
  agents: AgentPayload[];
  workspaces: WorkspacePayload[];
  onSelectGlobal: () => void;
  onSelectAgentScope: () => void;
  onSelectAgent: (agentName: string) => void;
  onSelectWorkspaceContext: (workspaceId: string) => void;
}

function ScopeSelector({
  selection,
  availableScopes,
  agents,
  workspaces,
  onSelectGlobal,
  onSelectAgentScope,
  onSelectAgent,
  onSelectWorkspaceContext,
}: ScopeSelectorProps) {
  const agentScopeAvailable = availableScopes.includes("agent");
  const items: Array<{ value: SkillsScopeValue; label: string; testId: string }> = [
    {
      value: "global",
      label: "Global",
      testId: "settings-page-skills-scope-global",
    },
  ];
  if (agentScopeAvailable) {
    items.push({
      value: "agent",
      label: "Agent",
      testId: "settings-page-skills-scope-agent",
    });
  }

  return (
    <Section
      divided
      label="Scope"
      note="agent scope only changes logical disabled skills for one effective agent"
    >
      <div
        className="flex flex-wrap items-center gap-2"
        data-testid="settings-page-skills-scope-row"
      >
        <PillGroup<SkillsScopeValue>
          items={items}
          value={selection.scope}
          size="sm"
          aria-label="Skills scope"
          onChange={next => {
            if (next === "global") {
              onSelectGlobal();
              return;
            }
            onSelectAgentScope();
          }}
        />
      </div>

      {selection.scope === "agent" ? (
        <div className="mt-4 grid gap-4 md:grid-cols-2">
          <SettingsFieldRow
            data-testid="settings-page-skills-agent-select"
            label="Agent"
            description="Select the logical agent that receives the tombstone list"
            hint="AGENT.MD"
            control={
              <AgentCommandSelect
                agents={agents}
                value={selection.agentName || null}
                onChange={next => onSelectAgent(next ?? "")}
                triggerTestId="settings-agent-select"
                className="w-56"
                placeholder="Select an agent"
              />
            }
          />
          <SettingsFieldRow
            data-testid="settings-page-skills-workspace-context"
            label="Workspace context"
            description="Optional workspace resolver context for the selected agent"
            hint="OPTIONAL"
            control={
              <NativeSelect
                className="w-56"
                data-testid="settings-page-skills-workspace-context-input"
                value={selection.workspaceId ?? ""}
                onChange={event => onSelectWorkspaceContext(event.target.value)}
              >
                <NativeSelectOption value="">Global resolution</NativeSelectOption>
                {workspaces.map(workspace => (
                  <NativeSelectOption key={workspace.id} value={workspace.id}>
                    {workspace.name}
                  </NativeSelectOption>
                ))}
              </NativeSelect>
            }
          />
        </div>
      ) : null}
    </Section>
  );
}

function OperationalLinksRow() {
  return (
    <Section divided label="Operational" note="manage runtime state outside of settings">
      <div className="flex flex-wrap gap-2" data-testid="settings-page-skills-operational-links">
        <Link
          to="/skills"
          className="inline-flex items-center gap-1.5 rounded-md border border-(--line) bg-(--elevated) px-3 py-1.5 text-xs font-medium text-(--fg) hover:bg-(--hover)"
          data-testid="settings-page-skills-link-skills"
        >
          <ExternalLink className="size-3.5 text-(--subtle)" />
          Open Skills
        </Link>
      </div>
    </Section>
  );
}

interface DisabledSkillsSectionProps {
  envelope: SettingsSkillsSection;
  selection: SkillsScopeSelection;
  selectedAgent: AgentPayload | null;
  selectedWorkspaceContext: WorkspacePayload | null;
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
  selection,
  selectedAgent,
  selectedWorkspaceContext,
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
    <Section
      divided
      label="Disabled skills"
      note={
        selection.scope === "agent"
          ? `applies immediately · scoped to ${selectedAgent?.name ?? selection.agentName}${selectedWorkspaceContext ? ` via ${selectedWorkspaceContext.name}` : ""}`
          : "applies immediately · no restart required"
      }
      right={
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
        <Empty
          icon={Wrench}
          title={selection.scope === "agent" ? "No agent-local tombstones" : "No skills installed"}
          description={
            selection.scope === "agent"
              ? "This agent is currently inheriting the effective skill set without disabled logical names."
              : "Manage availability from the Skills operational page; nothing has been disabled yet."
          }
          data-testid="settings-page-skills-disabled-empty"
        />
      ) : (
        <div
          className="overflow-hidden rounded-lg border border-(--line)"
          data-testid="settings-page-skills-disabled-list"
        >
          <Table>
            <TableHeader>
              <TableRow className="bg-(--elevated)">
                <TableHead className="text-badge uppercase tracking-mono text-(--muted)">
                  Skill
                </TableHead>
                <TableHead className="text-badge uppercase tracking-mono text-(--muted)">
                  Identifier
                </TableHead>
                <TableHead className="w-[1%] text-right text-badge uppercase tracking-mono text-(--muted)">
                  Disabled
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {candidates.map(name => (
                <TableRow key={name} data-testid={`settings-page-skills-disabled-item-${name}`}>
                  <TableCell>
                    <div className="flex min-w-0 items-center gap-2">
                      <Wrench className="size-3.5 text-(--subtle)" />
                      <span className="truncate text-sm text-(--fg)">{name}</span>
                    </div>
                  </TableCell>
                  <TableCell className="font-mono text-xs text-(--muted)">{name}</TableCell>
                  <TableCell className="text-right">
                    <Switch
                      data-testid={`settings-page-skills-disabled-toggle-${name}`}
                      checked={disabled.includes(name)}
                      onCheckedChange={() => onToggle(name)}
                      aria-label={`Toggle ${name}`}
                    />
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}
    </Section>
  );
}

function AgentScopePolicyNotice() {
  return (
    <Section
      divided
      label="Marketplace & policy"
      note="read-only in agent scope"
      data-testid="settings-page-skills-agent-policy-note"
    >
      <p className="text-sm text-(--muted)">
        Agent scope only supports logical `skills.disabled_skills` tombstones. Registry enablement,
        poll interval, and marketplace allowlists remain global settings.
      </p>
    </Section>
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
    <Section
      divided
      label="Marketplace & policy"
      note="restart required to apply"
      right={
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
        data-testid="settings-page-skills-poll-interval"
        label="Poll interval"
        description="How often the registry re-scans sources"
        hint="DEFAULT"
        control={
          <Input
            className="w-32 font-mono"
            data-testid="settings-page-skills-poll-interval-input"
            value={draft.poll_interval ?? ""}
            placeholder="5m"
            onChange={event =>
              setDraft(prev => {
                const current = prev ?? draft;
                return { ...current, poll_interval: event.target.value };
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid="settings-page-skills-marketplace-registry"
        label="Marketplace registry"
        description="Identifier of the marketplace publisher"
        hint="CONFIG.TOML"
        control={
          <Input
            className="w-56"
            data-testid="settings-page-skills-marketplace-registry-input"
            value={draft.marketplace.registry ?? ""}
            onChange={event =>
              setDraft(prev => {
                const current = prev ?? draft;
                return {
                  ...current,
                  marketplace: { ...current.marketplace, registry: event.target.value },
                };
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
          <Input
            className="w-72 font-mono"
            data-testid="settings-page-skills-marketplace-base-url-input"
            value={draft.marketplace.base_url ?? ""}
            placeholder="https://"
            onChange={event =>
              setDraft(prev => {
                const current = prev ?? draft;
                return {
                  ...current,
                  marketplace: { ...current.marketplace, base_url: event.target.value },
                };
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
        onChange={value =>
          setDraft(prev => {
            const current = prev ?? draft;
            return { ...current, allowed_marketplace_mcp: value };
          })
        }
      />
      <AllowListField
        label="Allowed hook installs"
        description="Comma-separated list of marketplace hook packages that may be installed"
        testId="settings-page-skills-allowed-hooks"
        value={draft.allowed_marketplace_hooks ?? []}
        onChange={value =>
          setDraft(prev => {
            const current = prev ?? draft;
            return { ...current, allowed_marketplace_hooks: value };
          })
        }
      />
    </Section>
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
        <Input
          className="w-72 font-mono"
          data-testid={`${testId}-input`}
          value={value.join(", ")}
          placeholder="none"
          onChange={event =>
            onChange(
              event.target.value.split(",").reduce<string[]>((entries, entry) => {
                const trimmed = entry.trim();
                if (trimmed.length > 0) {
                  entries.push(trimmed);
                }
                return entries;
              }, [])
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
  const liveRegion = error ? "assertive" : "polite";

  return (
    <div
      className="flex items-center gap-2"
      data-testid={`settings-page-skills-${slug}-controls`}
      data-dirty={isDirty ? "true" : "false"}
    >
      <div className="min-w-0" role="status" aria-live={liveRegion}>
        {error ? (
          <span
            className="text-xs text-(--danger)"
            data-testid={`settings-page-skills-${slug}-error`}
          >
            {error}
          </span>
        ) : warnings && warnings.length > 0 ? (
          <span
            className="text-xs text-(--warning)"
            data-testid={`settings-page-skills-${slug}-warning`}
          >
            {warnings.join(" · ")}
          </span>
        ) : isDirty ? (
          <span
            className="text-xs text-(--subtle)"
            data-testid={`settings-page-skills-${slug}-dirty`}
          >
            Unsaved changes
          </span>
        ) : lastAppliedLabel ? (
          <span
            className="text-xs text-(--subtle)"
            data-testid={`settings-page-skills-${slug}-applied`}
          >
            {lastAppliedLabel}
          </span>
        ) : null}
      </div>
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
        {isSaving ? "Saving..." : saveLabel}
      </Button>
    </div>
  );
}
