import { AlertCircle, Check, Loader2, Plus, Trash2, X } from "lucide-react";
import { useMemo } from "react";
import { createFileRoute } from "@tanstack/react-router";

import { Button } from "@agh/ui";
import { Pill } from "@/components/design-system";
import {
  useSettingsMCPServersPage,
  type MCPDraft,
  type MCPEditorState,
  type MCPEnvPair,
  type MCPLastAction,
  type MCPScopeSelection,
} from "@/hooks/routes/use-settings-mcp-servers-page";
import { cn } from "@/lib/utils";
import type { SettingsMCPServerEntry, SettingsMCPServerTarget } from "@/systems/settings";
import {
  SettingsCollectionHeader,
  SettingsDeleteDialog,
  SettingsEditorDialog,
  SettingsFieldRow,
  SettingsPageActions,
  SettingsPageShell,
  SettingsRestartBanner,
  SettingsSourceBadge,
  SettingsStatusLine,
} from "@/systems/settings/components";
import type { WorkspacePayload } from "@/systems/workspace";

export const Route = createFileRoute("/_app/settings/mcp-servers")({
  component: MCPServersSettingsPage,
});

function MCPServersSettingsPage() {
  const page = useSettingsMCPServersPage();

  if (page.isLoading) {
    return (
      <div
        className="flex flex-1 items-center justify-center"
        data-testid="settings-page-mcp-servers-loading"
      >
        <Loader2 className="size-5 animate-spin text-[color:var(--color-text-tertiary)]" />
      </div>
    );
  }

  if (page.error || !page.envelope) {
    return (
      <div
        className="flex flex-1 items-center justify-center"
        data-testid="settings-page-mcp-servers-error"
      >
        <div className="flex flex-col items-center gap-2 text-center">
          <AlertCircle className="size-6 text-[color:var(--color-danger)]" />
          <p className="text-sm text-[color:var(--color-text-tertiary)]">
            {page.error?.message ?? "Failed to load MCP servers"}
          </p>
        </div>
      </div>
    );
  }

  const scopeEyebrow = page.selection.scope === "global" ? "Global servers" : "Workspace overrides";
  const scopeSummary =
    page.selection.scope === "global"
      ? `${page.counts.total} defined · injected into every agent`
      : `${page.counts.total} overrides · scoped to ${page.selectedWorkspace?.name ?? page.selection.workspaceId}`;

  return (
    <SettingsPageShell
      slug="mcp-servers"
      title="MCP Servers"
      statusLine={
        <SettingsStatusLine
          data-testid="settings-page-mcp-servers-status-line"
          daemonAvailable
          items={[
            <span key="total" data-testid="settings-page-mcp-servers-total">
              {page.counts.total} servers
            </span>,
            <span key="scope" data-testid="settings-page-mcp-servers-scope-label">
              scope:{" "}
              {page.selection.scope === "global"
                ? "global"
                : (page.selectedWorkspace?.name ?? page.selection.workspaceId)}
            </span>,
            <span key="shadowed" data-testid="settings-page-mcp-servers-shadowed-total">
              {page.counts.shadowed} shadowed sources
            </span>,
          ]}
        />
      }
      actions={<SettingsPageActions slug="mcp-servers" restart={page.restart} />}
      banner={<SettingsRestartBanner slug="mcp-servers" restart={page.restart} />}
    >
      {page.lastAction ? (
        <ActionResultBanner action={page.lastAction} onDismiss={page.dismissLastAction} />
      ) : null}

      <ScopeSelector
        selection={page.selection}
        availableScopes={page.availableScopes}
        workspaces={page.workspaces}
        isLoadingWorkspaces={page.workspacesLoading}
        onSelectGlobal={page.selectGlobal}
        onSelectWorkspace={page.selectWorkspace}
      />

      <SettingsCollectionHeader
        data-testid="settings-page-mcp-servers-header-row"
        eyebrow={scopeEyebrow}
        summary={scopeSummary}
        action={
          <Button
            type="button"
            variant="default"
            size="sm"
            onClick={page.openCreate}
            data-testid="settings-page-mcp-servers-create"
          >
            <Plus className="size-3.5" />
            Add server
          </Button>
        }
      />

      {page.servers.length === 0 ? (
        <div
          className="rounded-md border border-dashed border-[color:var(--color-divider)] px-4 py-8 text-center text-sm text-[color:var(--color-text-tertiary)]"
          data-testid="settings-page-mcp-servers-empty"
        >
          {page.selection.scope === "global"
            ? "No MCP servers defined globally. Use “Add server” to create one in mcp.json or config."
            : "No workspace overrides defined. Add one to shadow the global definition for this workspace."}
        </div>
      ) : (
        <MCPServersTable servers={page.servers} onEdit={page.openEdit} onDelete={page.openDelete} />
      )}

      <MCPServerEditor
        editor={page.editor}
        scope={page.selection.scope}
        isValid={page.editorIsValid}
        isSaving={page.editorIsSaving}
        error={page.editorError}
        warnings={page.editorWarnings}
        existingNames={page.servers.map(entry => entry.name)}
        availableTargets={page.editorAvailableTargets}
        onChange={page.updateDraft}
        onTargetChange={page.setEditorTarget}
        onClose={page.closeEditor}
        onSave={page.saveEditor}
      />

      <MCPServerDeleteDialog
        target={page.deleteTarget.mode === "open" ? page.deleteTarget.entry : null}
        selectedTarget={page.deleteTarget.mode === "open" ? page.deleteTarget.target : "auto"}
        availableTargets={page.deleteAvailableTargets}
        error={page.deleteError}
        isDeleting={page.deleteIsPending}
        onTargetChange={page.setDeleteTargetKind}
        onClose={page.closeDelete}
        onConfirm={page.confirmDelete}
      />
    </SettingsPageShell>
  );
}

interface ScopeSelectorProps {
  selection: MCPScopeSelection;
  availableScopes: readonly ("global" | "workspace")[];
  workspaces: WorkspacePayload[];
  isLoadingWorkspaces: boolean;
  onSelectGlobal: () => void;
  onSelectWorkspace: (workspaceId: string) => void;
}

function ScopeSelector({
  selection,
  availableScopes,
  workspaces,
  isLoadingWorkspaces,
  onSelectGlobal,
  onSelectWorkspace,
}: ScopeSelectorProps) {
  const workspaceScopeAvailable = availableScopes.includes("workspace");
  const selectedWorkspaceId = selection.scope === "workspace" ? selection.workspaceId : null;

  return (
    <div
      className="flex flex-wrap items-center gap-2"
      data-testid="settings-page-mcp-servers-scope-row"
    >
      <ScopeChip
        active={selection.scope === "global"}
        onClick={onSelectGlobal}
        testId="settings-page-mcp-servers-scope-global"
      >
        <span className="font-medium">Global</span>
        <span className="font-mono text-[0.62rem] text-[color:var(--color-text-tertiary)]">
          ~/.agh/mcp.json
        </span>
      </ScopeChip>
      {workspaceScopeAvailable
        ? workspaces.map(workspace => (
            <ScopeChip
              key={workspace.id}
              active={selectedWorkspaceId === workspace.id}
              onClick={() => onSelectWorkspace(workspace.id)}
              testId={`settings-page-mcp-servers-scope-workspace-${workspace.id}`}
            >
              <WorkspaceScopeLabel workspace={workspace} />
            </ScopeChip>
          ))
        : null}
      {workspaceScopeAvailable && workspaces.length === 0 && !isLoadingWorkspaces ? (
        <span
          className="font-mono text-[0.62rem] uppercase tracking-[0.12em] text-[color:var(--color-text-tertiary)]"
          data-testid="settings-page-mcp-servers-scope-workspace-empty"
        >
          no workspaces yet
        </span>
      ) : null}
    </div>
  );
}

function WorkspaceScopeLabel({ workspace }: { workspace: WorkspacePayload }) {
  const initial = workspace.name.charAt(0).toUpperCase() || "W";
  return (
    <span className="flex items-center gap-2">
      <span className="flex size-5 items-center justify-center rounded-sm bg-[color:var(--color-accent-tint)] font-mono text-[0.6rem] uppercase text-[color:var(--color-accent)]">
        {initial}
      </span>
      <span className="font-medium">{workspace.name}</span>
      <span className="font-mono text-[0.62rem] text-[color:var(--color-text-tertiary)]">
        overrides
      </span>
    </span>
  );
}

function ScopeChip({
  active,
  onClick,
  children,
  testId,
}: {
  active: boolean;
  onClick: () => void;
  children: React.ReactNode;
  testId?: string;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      data-testid={testId}
      data-active={active ? "true" : "false"}
      className={cn(
        "flex items-center gap-2 rounded-md border px-3 py-1.5 text-xs transition-colors",
        active
          ? "border-[color:var(--color-accent)] bg-[color:var(--color-accent-tint)] text-[color:var(--color-text-primary)]"
          : "border-[color:var(--color-divider)] bg-[color:var(--color-surface)] text-[color:var(--color-text-secondary)] hover:border-[color:var(--color-divider-strong)]"
      )}
    >
      {children}
    </button>
  );
}

function MCPServersTable({
  servers,
  onEdit,
  onDelete,
}: {
  servers: SettingsMCPServerEntry[];
  onEdit: (entry: SettingsMCPServerEntry) => void;
  onDelete: (entry: SettingsMCPServerEntry) => void;
}) {
  return (
    <div
      className="overflow-hidden rounded-lg border border-[color:var(--color-divider)]"
      data-testid="settings-page-mcp-servers-list"
    >
      <table className="w-full border-collapse text-sm">
        <thead className="bg-[color:var(--color-surface-elevated)] text-[0.6rem] uppercase tracking-[0.18em] text-[color:var(--color-text-label)]">
          <tr>
            <th className="px-4 py-2.5 text-left">Name</th>
            <th className="px-4 py-2.5 text-left">Command</th>
            <th className="px-4 py-2.5 text-left">Source</th>
            <th className="px-4 py-2.5 text-right">Env</th>
            <th className="px-4 py-2.5 text-right">Args</th>
            <th className="px-4 py-2.5 text-right">Actions</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-[color:var(--color-divider)]">
          {servers.map(server => (
            <MCPServerRow
              key={`${server.name}-${server.source_metadata.effective_source.kind}`}
              server={server}
              onEdit={onEdit}
              onDelete={onDelete}
            />
          ))}
        </tbody>
      </table>
    </div>
  );
}

function MCPServerRow({
  server,
  onEdit,
  onDelete,
}: {
  server: SettingsMCPServerEntry;
  onEdit: (entry: SettingsMCPServerEntry) => void;
  onDelete: (entry: SettingsMCPServerEntry) => void;
}) {
  const source = server.source_metadata.effective_source;
  const shadowed = server.source_metadata.shadowed_sources ?? [];
  const envCount = server.env ? Object.keys(server.env).length : 0;
  const argsCount = server.args?.length ?? 0;

  return (
    <tr data-testid={`settings-page-mcp-servers-row-${server.name}`}>
      <td className="px-4 py-3">
        <span className="font-mono text-sm text-[color:var(--color-text-primary)]">
          {server.name}
        </span>
      </td>
      <td
        className="px-4 py-3 font-mono text-xs text-[color:var(--color-text-secondary)]"
        data-testid={`settings-page-mcp-servers-row-${server.name}-command`}
      >
        {server.command}
      </td>
      <td className="px-4 py-3">
        <SettingsSourceBadge
          data-testid={`settings-page-mcp-servers-row-${server.name}-source`}
          source={source}
          shadowed={shadowed}
        />
      </td>
      <td
        className="px-4 py-3 text-right font-mono text-xs text-[color:var(--color-text-secondary)]"
        data-testid={`settings-page-mcp-servers-row-${server.name}-env`}
      >
        {envCount}
      </td>
      <td
        className="px-4 py-3 text-right font-mono text-xs text-[color:var(--color-text-secondary)]"
        data-testid={`settings-page-mcp-servers-row-${server.name}-args`}
      >
        {argsCount}
      </td>
      <td className="px-4 py-3">
        <div className="flex items-center justify-end gap-2">
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={() => onEdit(server)}
            data-testid={`settings-page-mcp-servers-row-${server.name}-edit`}
          >
            Edit
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={() => onDelete(server)}
            data-testid={`settings-page-mcp-servers-row-${server.name}-delete`}
          >
            Delete
          </Button>
        </div>
      </td>
    </tr>
  );
}

interface MCPServerEditorProps {
  editor: MCPEditorState;
  scope: "global" | "workspace";
  isValid: boolean;
  isSaving: boolean;
  error: string | null;
  warnings?: string[];
  existingNames: string[];
  availableTargets: SettingsMCPServerTarget[];
  onChange: (updater: (draft: MCPDraft) => MCPDraft) => void;
  onTargetChange: (target: SettingsMCPServerTarget) => void;
  onClose: () => void;
  onSave: () => void;
}

function MCPServerEditor({
  editor,
  scope,
  isValid,
  isSaving,
  error,
  warnings,
  existingNames,
  availableTargets,
  onChange,
  onTargetChange,
  onClose,
  onSave,
}: MCPServerEditorProps) {
  const open = editor.mode !== "closed";
  if (!open) return null;

  const isCreate = editor.mode === "create";
  const draft = editor.draft;
  const entry = editor.mode === "edit" ? editor.entry : null;
  const target = editor.target;

  const title = isCreate
    ? "Add MCP server"
    : `Edit MCP server · ${editor.mode === "edit" ? editor.name : ""}`;
  const description = isCreate
    ? scope === "workspace"
      ? "Add a workspace-scoped override. Saved entries replace any prior definition for this name in this scope."
      : "Add a new MCP server. Saving writes a full replacement of the named definition in the selected target."
    : "Saving replaces the entire server definition in the selected target (full PUT). Lower-precedence shadowed sources remain untouched.";

  const lowerName = draft.name.trim().toLowerCase();
  const nameConflict =
    isCreate &&
    lowerName.length > 0 &&
    existingNames.some(existing => existing.toLowerCase() === lowerName);

  return (
    <SettingsEditorDialog
      open={open}
      mode={isCreate ? "create" : "edit"}
      title={title}
      slug="mcp-servers"
      description={description}
      metadata={
        entry ? (
          <SettingsSourceBadge
            data-testid="settings-mcp-servers-editor-source"
            source={entry.source_metadata.effective_source}
            shadowed={entry.source_metadata.shadowed_sources ?? []}
          />
        ) : null
      }
      error={error ?? (nameConflict ? `An MCP server named "${draft.name}" already exists.` : null)}
      warnings={warnings}
      canSave={isValid && !nameConflict}
      isSaving={isSaving}
      saveLabel={isCreate ? "Create server" : "Replace definition"}
      onSave={onSave}
      onOpenChange={next => {
        if (!next) onClose();
      }}
    >
      <div className="flex flex-col gap-3">
        <SettingsFieldRow
          data-testid="settings-mcp-servers-editor-name"
          label="Name"
          description={
            isCreate
              ? "Lower-case identifier injected into agents as the MCP server name."
              : "Name is immutable — remove the server and add a new one to rename."
          }
          hint={isCreate ? "REQUIRED" : "LOCKED"}
          control={
            <input
              className="h-8 w-56 rounded-md border border-[color:var(--color-divider)] bg-[color:var(--color-surface-elevated)] px-2 font-mono text-sm text-[color:var(--color-text-primary)] disabled:opacity-60"
              data-testid="settings-mcp-servers-editor-name-input"
              value={draft.name}
              placeholder="e.g. filesystem"
              disabled={!isCreate}
              onChange={event => onChange(current => ({ ...current, name: event.target.value }))}
            />
          }
        />
        <SettingsFieldRow
          data-testid="settings-mcp-servers-editor-command"
          label="Command"
          description="Executable that speaks MCP over stdio (command + args)."
          hint="REQUIRED"
          control={
            <input
              className="h-8 w-72 rounded-md border border-[color:var(--color-divider)] bg-[color:var(--color-surface-elevated)] px-2 font-mono text-sm text-[color:var(--color-text-primary)]"
              data-testid="settings-mcp-servers-editor-command-input"
              value={draft.command}
              placeholder="npx -y @modelcontextprotocol/server-filesystem"
              onChange={event => onChange(current => ({ ...current, command: event.target.value }))}
            />
          }
        />
        <TargetSelector
          target={target}
          availableTargets={availableTargets}
          scope={scope}
          onChange={onTargetChange}
          entry={entry}
          isCreate={isCreate}
        />
        <ArgsEditor
          args={draft.args}
          onChange={nextArgs => onChange(current => ({ ...current, args: nextArgs }))}
        />
        <EnvEditor
          env={draft.env}
          onChange={nextEnv => onChange(current => ({ ...current, env: nextEnv }))}
        />
      </div>
    </SettingsEditorDialog>
  );
}

interface TargetSelectorProps {
  target: SettingsMCPServerTarget;
  availableTargets: SettingsMCPServerTarget[];
  scope: "global" | "workspace";
  entry: SettingsMCPServerEntry | null;
  isCreate: boolean;
  onChange: (target: SettingsMCPServerTarget) => void;
}

function TargetSelector({
  target,
  availableTargets,
  scope,
  entry,
  isCreate,
  onChange,
}: TargetSelectorProps) {
  const description = useMemo(() => {
    if (isCreate) {
      return scope === "workspace"
        ? "Auto writes new entries to the workspace mcp.json. Pick config to write into <workspace>/.agh/config.toml instead."
        : "Auto writes new entries to ~/.agh/mcp.json. Pick config to write into ~/.agh/config.toml instead.";
    }
    if (!entry) return "Where to persist this definition in the selected scope.";
    const effectiveKind = entry.source_metadata.effective_source.kind;
    if (effectiveKind.endsWith("sidecar")) {
      return "Auto replaces the sidecar definition (highest precedence). Choosing config writes a new config override that would shadow the sidecar only if precedence allowed it; in v1 sidecar wins so config entry stays shadowed.";
    }
    if (effectiveKind.endsWith("config")) {
      return "Auto replaces the config definition. Choosing sidecar writes into mcp.json, which would shadow the config entry after save.";
    }
    return "Auto replaces the current highest-precedence definition in the selected scope.";
  }, [entry, isCreate, scope]);

  const selectId = isCreate
    ? "settings-mcp-servers-editor-target-create"
    : "settings-mcp-servers-editor-target-edit";

  return (
    <SettingsFieldRow
      data-testid="settings-mcp-servers-editor-target"
      label="Persistence target"
      description={description}
      hint={scope === "workspace" ? "WORKSPACE" : "GLOBAL"}
      control={
        <div className="flex flex-col gap-1">
          <select
            id={selectId}
            className="h-8 w-56 rounded-md border border-[color:var(--color-divider)] bg-[color:var(--color-surface-elevated)] px-2 font-mono text-sm text-[color:var(--color-text-primary)]"
            data-testid="settings-mcp-servers-editor-target-input"
            value={target}
            onChange={event => onChange(event.target.value as SettingsMCPServerTarget)}
          >
            {availableTargets.map(candidate => (
              <option key={candidate} value={candidate}>
                {targetLabel(candidate)}
              </option>
            ))}
          </select>
          {entry ? (
            <div
              className="flex flex-wrap items-center gap-1 text-[0.62rem] uppercase tracking-[0.12em] text-[color:var(--color-text-label)]"
              data-testid="settings-mcp-servers-editor-available-targets"
            >
              <span>allowed:</span>
              {entry.source_metadata.available_targets.map(available => (
                <Pill key={available} emphasis="muted" kind="state" tone="neutral">
                  {targetWriteLabel(available)}
                </Pill>
              ))}
            </div>
          ) : null}
        </div>
      }
    />
  );
}

function targetLabel(target: SettingsMCPServerTarget): string {
  if (target === "auto") return "auto (highest precedence)";
  if (target === "config") return "config (.agh/config.toml)";
  return "sidecar (mcp.json)";
}

function targetWriteLabel(
  target: "global-config" | "workspace-config" | "global-mcp-sidecar" | "workspace-mcp-sidecar"
): string {
  if (target === "global-config") return "GLOBAL CFG";
  if (target === "workspace-config") return "WS CFG";
  if (target === "global-mcp-sidecar") return "GLOBAL MCP";
  return "WS MCP";
}

function ArgsEditor({ args, onChange }: { args: string[]; onChange: (next: string[]) => void }) {
  return (
    <SettingsFieldRow
      data-testid="settings-mcp-servers-editor-args"
      label="Args"
      description="Passed to the MCP server command in order."
      hint={`${args.length} entries`}
      control={
        <div
          className="flex w-full flex-col gap-1.5"
          data-testid="settings-mcp-servers-editor-args-list"
        >
          {args.map((arg, index) => (
            <div key={index} className="flex items-center gap-2">
              <input
                className="h-8 flex-1 rounded-md border border-[color:var(--color-divider)] bg-[color:var(--color-surface-elevated)] px-2 font-mono text-xs text-[color:var(--color-text-primary)]"
                data-testid={`settings-mcp-servers-editor-args-input-${index}`}
                value={arg}
                placeholder={`arg[${index}]`}
                onChange={event => {
                  const next = [...args];
                  next[index] = event.target.value;
                  onChange(next);
                }}
              />
              <Button
                type="button"
                variant="ghost"
                size="icon-sm"
                onClick={() => onChange(args.filter((_, i) => i !== index))}
                aria-label={`Remove arg ${index}`}
                data-testid={`settings-mcp-servers-editor-args-remove-${index}`}
              >
                <Trash2 className="size-3.5" />
              </Button>
            </div>
          ))}
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={() => onChange([...args, ""])}
            data-testid="settings-mcp-servers-editor-args-add"
          >
            <Plus className="size-3.5" />
            Add arg
          </Button>
        </div>
      }
    />
  );
}

function EnvEditor({
  env,
  onChange,
}: {
  env: MCPEnvPair[];
  onChange: (next: MCPEnvPair[]) => void;
}) {
  return (
    <SettingsFieldRow
      data-testid="settings-mcp-servers-editor-env"
      label="Environment"
      description="Key/value pairs injected when the server launches."
      hint={`${env.length} entries`}
      control={
        <div
          className="flex w-full flex-col gap-1.5"
          data-testid="settings-mcp-servers-editor-env-list"
        >
          {env.map((pair, index) => (
            <div key={index} className="flex items-center gap-2">
              <input
                className="h-8 w-44 rounded-md border border-[color:var(--color-divider)] bg-[color:var(--color-surface-elevated)] px-2 font-mono text-xs text-[color:var(--color-text-primary)]"
                data-testid={`settings-mcp-servers-editor-env-key-${index}`}
                value={pair.key}
                placeholder="KEY"
                onChange={event => {
                  const next = [...env];
                  next[index] = { ...pair, key: event.target.value };
                  onChange(next);
                }}
              />
              <input
                className="h-8 flex-1 rounded-md border border-[color:var(--color-divider)] bg-[color:var(--color-surface-elevated)] px-2 font-mono text-xs text-[color:var(--color-text-primary)]"
                data-testid={`settings-mcp-servers-editor-env-value-${index}`}
                value={pair.value}
                placeholder="value"
                onChange={event => {
                  const next = [...env];
                  next[index] = { ...pair, value: event.target.value };
                  onChange(next);
                }}
              />
              <Button
                type="button"
                variant="ghost"
                size="icon-sm"
                onClick={() => onChange(env.filter((_, i) => i !== index))}
                aria-label={`Remove env ${index}`}
                data-testid={`settings-mcp-servers-editor-env-remove-${index}`}
              >
                <Trash2 className="size-3.5" />
              </Button>
            </div>
          ))}
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={() => onChange([...env, { key: "", value: "" }])}
            data-testid="settings-mcp-servers-editor-env-add"
          >
            <Plus className="size-3.5" />
            Add variable
          </Button>
        </div>
      }
    />
  );
}

interface MCPServerDeleteDialogProps {
  target: SettingsMCPServerEntry | null;
  selectedTarget: SettingsMCPServerTarget;
  availableTargets: SettingsMCPServerTarget[];
  error: string | null;
  isDeleting: boolean;
  onTargetChange: (target: SettingsMCPServerTarget) => void;
  onClose: () => void;
  onConfirm: () => void;
}

function MCPServerDeleteDialog({
  target,
  selectedTarget,
  availableTargets,
  error,
  isDeleting,
  onTargetChange,
  onClose,
  onConfirm,
}: MCPServerDeleteDialogProps) {
  const open = Boolean(target);
  const shadowed = target?.source_metadata.shadowed_sources ?? [];
  const hasShadowed = shadowed.length > 0;
  const effective = target?.source_metadata.effective_source;

  return (
    <SettingsDeleteDialog
      open={open}
      slug="mcp-servers"
      title={target ? `Delete MCP server "${target.name}"?` : "Delete MCP server"}
      description={
        target
          ? selectedTarget === "auto"
            ? "Removes the highest-precedence definition in the selected scope. Lower-precedence definitions may become effective again."
            : `Removes the definition from the selected target (${targetLabel(selectedTarget)}). Other sources for this server remain untouched.`
          : null
      }
      fallbackNote={
        target ? (
          <div className="flex flex-col gap-2">
            {effective ? (
              <div className="flex flex-col gap-1">
                <span className="font-medium">Current effective source</span>
                <SettingsSourceBadge
                  data-testid="settings-mcp-servers-delete-effective"
                  source={effective}
                />
              </div>
            ) : null}
            {hasShadowed ? (
              <div
                className="flex flex-col gap-1"
                data-testid="settings-mcp-servers-delete-shadowed"
              >
                <span className="font-medium">After delete, this becomes effective</span>
                <div className="flex flex-wrap items-center gap-1.5">
                  <SettingsSourceBadge source={shadowed[0]} />
                </div>
                <span>
                  Lower-precedence definitions remain on disk and become the next source the daemon
                  reads at restart.
                </span>
              </div>
            ) : (
              <span data-testid="settings-mcp-servers-delete-no-shadowed">
                No other sources define this server — it will be fully removed after delete.
              </span>
            )}
            <div
              className="flex items-center gap-2"
              data-testid="settings-mcp-servers-delete-target"
            >
              <label
                htmlFor="settings-mcp-servers-delete-target-input"
                className="font-mono text-[0.62rem] uppercase tracking-[0.14em] text-[color:var(--color-text-label)]"
              >
                target
              </label>
              <select
                id="settings-mcp-servers-delete-target-input"
                className="h-7 rounded-md border border-[color:var(--color-divider)] bg-[color:var(--color-surface-elevated)] px-2 font-mono text-xs text-[color:var(--color-text-primary)]"
                data-testid="settings-mcp-servers-delete-target-input"
                value={selectedTarget}
                onChange={event => onTargetChange(event.target.value as SettingsMCPServerTarget)}
              >
                {availableTargets.map(candidate => (
                  <option key={candidate} value={candidate}>
                    {targetLabel(candidate)}
                  </option>
                ))}
              </select>
            </div>
          </div>
        ) : null
      }
      error={error}
      isDeleting={isDeleting}
      confirmLabel="Delete definition"
      onConfirm={onConfirm}
      onOpenChange={next => {
        if (!next) onClose();
      }}
    />
  );
}

function ActionResultBanner({
  action,
  onDismiss,
}: {
  action: MCPLastAction;
  onDismiss: () => void;
}) {
  const isSaved = action.kind === "saved";
  const tone = isSaved ? "success" : "info";
  const toneClasses =
    tone === "success"
      ? "border-[color:var(--color-success)] bg-[color:var(--color-success-tint)] text-[color:var(--color-success)]"
      : "border-[color:var(--color-info)] bg-[color:var(--color-info-tint)] text-[color:var(--color-info)]";
  const restartBadge = action.result.restart_required
    ? "restart required to apply"
    : "applied immediately";
  const writeTargetLabel = action.result.write_target
    ? ` · persisted to ${targetWriteLabel(action.result.write_target)}`
    : "";

  const message = isSaved
    ? `Saved "${action.name}"${writeTargetLabel} · ${restartBadge}.`
    : action.remainingShadowed > 0
      ? `Deleted "${action.name}" · ${action.remainingShadowed} shadowed source${action.remainingShadowed === 1 ? "" : "s"} may become effective on reload · ${restartBadge}.`
      : `Deleted "${action.name}" · no other sources remained · ${restartBadge}.`;

  return (
    <div
      className={`flex items-center justify-between gap-3 rounded-md border px-3 py-2 text-xs ${toneClasses}`}
      data-testid="settings-page-mcp-servers-action-result"
      data-kind={action.kind}
      role="status"
    >
      <span className="flex items-center gap-2">
        <Check className="size-3.5" />
        <span>{message}</span>
      </span>
      <Button
        type="button"
        variant="ghost"
        size="sm"
        onClick={onDismiss}
        data-testid="settings-page-mcp-servers-action-result-dismiss"
      >
        <X className="size-3.5" />
      </Button>
    </div>
  );
}
