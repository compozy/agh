import { Link } from "@tanstack/react-router";
import { AlertCircle, ExternalLink } from "lucide-react";
import type { Dispatch, SetStateAction } from "react";

import {
  useSettingsSkillsPage,
  type SkillsScopeSelection,
} from "@/hooks/routes/use-settings-skills-page";
import { AgentCommandSelect, type AgentPayload } from "@/systems/agent";
import {
  restartBannerPropsFor,
  SettingsDisabledSkillsSection,
  SettingsFieldRow,
  type SettingsScope,
  type SettingsSkillsSection,
  SettingsPageHead,
} from "@/systems/settings";
import type { WorkspacePayload } from "@/systems/workspace";
import {
  Button,
  Input,
  NativeSelect,
  NativeSelectOption,
  PageShell,
  PillGroup,
  RestartBanner,
  Section,
  Spinner,
  StatusLine,
  Switch,
  type StatusLineItem,
} from "@agh/ui";
import { AllowListField, SaveControls } from "./-skills-controls";

type SkillsConfig = SettingsSkillsSection["config"];
export function SkillsSettingsPage() {
  const page = useSettingsSkillsPage();
  const envelopeForSlot = page.envelope;
  const statusItems: StatusLineItem[] = envelopeForSlot
    ? [
        {
          key: "discovered",
          value: `${envelopeForSlot.discovered_count} discovered`,
          tone: "neutral",
        },
        {
          key: "disabled",
          value: `${envelopeForSlot.disabled_count} disabled`,
          tone: "neutral",
        },
        {
          key: "scope",
          value: (
            <span data-testid="settings-page-skills-scope-label">
              scope:{" "}
              {page.selection.scope === "global"
                ? "global"
                : `agent ${page.selectedAgent?.name ?? page.selection.agentName}`}
            </span>
          ),
          tone: "neutral",
        },
      ]
    : [];
  if (envelopeForSlot && page.selection.scope === "agent" && page.selectedWorkspaceContext) {
    statusItems.push({
      key: "context",
      value: (
        <span data-testid="settings-page-skills-workspace-context-summary">
          context: {page.selectedWorkspaceContext.name}
        </span>
      ),
      tone: "neutral",
    });
  }
  const statusLine = envelopeForSlot ? (
    <StatusLine
      data-testid="settings-page-skills-status-line"
      status={envelopeForSlot.runtime_available ? "connected" : "error"}
      items={statusItems}
    />
  ) : null;

  if (page.isLoading) {
    return (
      <div
        className="flex flex-1 items-center justify-center"
        data-testid="settings-page-skills-loading"
      >
        <Spinner className="size-5 text-subtle" />
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
          <AlertCircle className="size-6 text-danger" />
          <p className="text-sm text-subtle">
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
  const bannerProps = restartBannerPropsFor("skills", restart);

  return (
    <PageShell
      slug="skills"
      banner={bannerProps ? <RestartBanner {...bannerProps} /> : null}
      head={<SettingsPageHead slug="skills" statusLine={statusLine} />}
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
      <OperationalLinksSection />
      <SettingsDisabledSkillsSection
        baselineDisabled={envelope.config.disabled_skills ?? []}
        disabled={draft.disabled_skills ?? []}
        note={
          page.selection.scope === "agent"
            ? `applies immediately · scoped to ${page.selectedAgent?.name ?? page.selection.agentName}${page.selectedWorkspaceContext ? ` via ${page.selectedWorkspaceContext.name}` : ""}`
            : "applies immediately · no restart required"
        }
        emptyTitle={
          page.selection.scope === "agent" ? "No agent-local tombstones" : "No skills installed"
        }
        emptyDescription={
          page.selection.scope === "agent"
            ? "This agent is currently inheriting the effective skill set without disabled logical names."
            : "Manage availability from the Skills operational page; nothing has been disabled yet."
        }
        onToggle={page.toggleDisabled}
        controls={
          <SaveControls
            slug="disabled"
            saveLabel="Apply"
            isDirty={page.isDisabledDirty}
            isSaving={page.isSavingDisabled}
            error={page.saveDisabledError}
            warnings={page.disabledWarnings}
            lastAppliedLabel={page.lastDisabledLabel}
            onSave={page.handleSaveDisabled}
            onReset={page.handleResetDisabled}
          />
        }
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

function OperationalLinksSection() {
  return (
    <Section divided label="Operational" note="manage runtime state outside of settings">
      <div className="flex flex-wrap gap-2" data-testid="settings-page-skills-operational-links">
        <Link
          search={{ tab: "installed" }}
          to="/marketplace/skills"
          className="inline-flex items-center gap-1.5 rounded-md border border-line bg-elevated px-3 py-1.5 text-xs font-medium text-fg hover:bg-hover"
          data-testid="settings-page-skills-link-skills"
        >
          <ExternalLink className="size-3 text-subtle" />
          Open Skills
        </Link>
      </div>
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
      <p className="text-sm text-muted">
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
