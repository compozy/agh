import { AlertCircle, Check, Loader2, Plus, Server, Trash2, X } from "lucide-react";
import { useMemo } from "react";
import { createFileRoute } from "@tanstack/react-router";

import {
  Alert,
  AlertAction,
  AlertDescription,
  Button,
  Empty,
  Input,
  Pill,
  NativeSelect,
  NativeSelectOption,
  PillGroup,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@agh/ui";
import {
  useSettingsMCPServersPage,
  type MCPDraft,
  type MCPEditorState,
  type MCPEnvPair,
  type MCPLastAction,
  type MCPScopeSelection,
} from "@/hooks/routes/use-settings-mcp-servers-page";
import type {
  SettingsMCPServerEntry,
  SettingsMCPServerTarget,
  SettingsScope,
  SettingsWriteTarget,
} from "@/systems/settings";
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
        <Loader2 className="size-5 animate-spin text-(--color-text-tertiary)" />
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
          <AlertCircle className="size-6 text-(--color-danger)" />
          <p className="text-sm text-(--color-text-tertiary)">
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
        <Empty
          icon={Server}
          title="No MCP servers configured"
          description={
            page.selection.scope === "global"
              ? 'Use "Add server" to create an entry in mcp.json or config.'
              : "No workspace overrides defined. Add one to shadow the global definition for this workspace."
          }
          data-testid="settings-page-mcp-servers-empty"
        />
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

type ScopeValue = "global" | `ws:${string}`;

interface ScopeSelectorProps {
  selection: MCPScopeSelection;
  availableScopes: readonly SettingsScope[];
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
  const currentValue: ScopeValue =
    selection.scope === "workspace" ? (`ws:${selection.workspaceId}` as ScopeValue) : "global";

  const items: Array<{ value: ScopeValue; label: React.ReactNode; testId: string }> = [
    {
      value: "global",
      label: <ScopeLabel primary="Global" mono="~/.agh/mcp.json" />,
      testId: "settings-page-mcp-servers-scope-global",
    },
  ];
  if (workspaceScopeAvailable) {
    for (const workspace of workspaces) {
      items.push({
        value: `ws:${workspace.id}` as ScopeValue,
        label: <ScopeLabel primary={workspace.name} mono="overrides" />,
        testId: `settings-page-mcp-servers-scope-workspace-${workspace.id}`,
      });
    }
  }

  return (
    <div
      className="flex flex-wrap items-center gap-2"
      data-testid="settings-page-mcp-servers-scope-row"
    >
      <PillGroup<ScopeValue>
        items={items}
        value={currentValue}
        size="sm"
        aria-label="Catalog scope"
        onChange={next => {
          if (next === "global") {
            onSelectGlobal();
            return;
          }
          onSelectWorkspace(next.slice(3));
        }}
      />
      {workspaceScopeAvailable && workspaces.length === 0 && !isLoadingWorkspaces ? (
        <span
          className="font-mono text-badge uppercase tracking-mono text-(--color-text-tertiary)"
          data-testid="settings-page-mcp-servers-scope-workspace-empty"
        >
          no workspaces yet
        </span>
      ) : null}
    </div>
  );
}

function ScopeLabel({ primary, mono }: { primary: string; mono: string }) {
  return (
    <span className="inline-flex items-center gap-2">
      <span className="font-medium">{primary}</span>
      <span className="font-mono text-badge normal-case tracking-mono text-(--color-text-tertiary)">
        {mono}
      </span>
    </span>
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
      className="overflow-hidden rounded-lg border border-(--color-divider)"
      data-testid="settings-page-mcp-servers-list"
    >
      <Table>
        <TableHeader>
          <TableRow className="bg-(--color-surface-elevated)">
            <TableHead className="text-badge uppercase tracking-mono text-(--color-text-label)">
              Name
            </TableHead>
            <TableHead className="text-badge uppercase tracking-mono text-(--color-text-label)">
              Endpoint
            </TableHead>
            <TableHead className="text-badge uppercase tracking-mono text-(--color-text-label)">
              Source
            </TableHead>
            <TableHead className="text-right text-badge uppercase tracking-mono text-(--color-text-label)">
              Env
            </TableHead>
            <TableHead className="text-right text-badge uppercase tracking-mono text-(--color-text-label)">
              Args
            </TableHead>
            <TableHead className="w-[1%] text-right text-badge uppercase tracking-mono text-(--color-text-label)">
              Actions
            </TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {servers.map(server => (
            <MCPServerRow
              key={`${server.name}-${server.source_metadata.effective_source.kind}`}
              server={server}
              onEdit={onEdit}
              onDelete={onDelete}
            />
          ))}
        </TableBody>
      </Table>
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
  const endpoint = server.transport === "stdio" ? server.command : server.url;
  const canEdit = server.transport === "stdio";

  return (
    <TableRow data-testid={`settings-page-mcp-servers-row-${server.name}`}>
      <TableCell>
        <div className="flex min-w-0 items-center gap-2.5">
          <Pill.Dot
            tone="success"
            size="md"
            data-testid={`settings-page-mcp-servers-row-${server.name}-status`}
            data-tone="configured"
          />
          <span className="font-mono text-sm text-(--color-text-primary)">{server.name}</span>
        </div>
      </TableCell>
      <TableCell
        className="font-mono text-xs text-(--color-text-secondary)"
        data-testid={`settings-page-mcp-servers-row-${server.name}-command`}
      >
        {endpoint ?? "-"}
      </TableCell>
      <TableCell>
        <SettingsSourceBadge
          data-testid={`settings-page-mcp-servers-row-${server.name}-source`}
          source={source}
          shadowed={shadowed}
        />
      </TableCell>
      <TableCell
        className="text-right font-mono text-xs text-(--color-text-secondary)"
        data-testid={`settings-page-mcp-servers-row-${server.name}-env`}
      >
        {envCount}
      </TableCell>
      <TableCell
        className="text-right font-mono text-xs text-(--color-text-secondary)"
        data-testid={`settings-page-mcp-servers-row-${server.name}-args`}
      >
        {argsCount}
      </TableCell>
      <TableCell>
        <div className="flex items-center justify-end gap-2">
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={() => onEdit(server)}
            disabled={!canEdit}
            title={canEdit ? undefined : "Remote MCP servers are managed from config and CLI auth."}
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
      </TableCell>
    </TableRow>
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
            <Input
              className="w-56 font-mono disabled:opacity-60"
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
            <Input
              className="w-72 font-mono"
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

  return (
    <SettingsFieldRow
      data-testid="settings-mcp-servers-editor-target"
      label="Persistence target"
      description={description}
      hint={scope === "workspace" ? "WORKSPACE" : "GLOBAL"}
      control={
        <div className="flex flex-col gap-1">
          <NativeSelect
            className="w-56 font-mono"
            data-testid="settings-mcp-servers-editor-target-input"
            value={target}
            onChange={event => onChange(event.target.value as SettingsMCPServerTarget)}
          >
            {availableTargets.map(candidate => (
              <NativeSelectOption key={candidate} value={candidate}>
                {targetLabel(candidate)}
              </NativeSelectOption>
            ))}
          </NativeSelect>
          {entry ? (
            <div
              className="flex flex-wrap items-center gap-1 text-badge uppercase tracking-mono text-(--color-text-label)"
              data-testid="settings-mcp-servers-editor-available-targets"
            >
              <span>allowed:</span>
              {entry.source_metadata.available_targets.map(available => (
                <Pill mono key={available} tone="neutral">
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

function targetWriteLabel(target: SettingsWriteTarget): string {
  if (target === "global-config") return "GLOBAL CFG";
  if (target === "workspace-config") return "WS CFG";
  if (target === "global-mcp-sidecar") return "GLOBAL MCP";
  if (target === "workspace-mcp-sidecar") return "WS MCP";
  if (target === "global-agent-file") return "GLOBAL AGENT";
  return "WS AGENT";
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
            // biome-ignore lint/suspicious/noArrayIndexKey: args list is ordered and stable per edit
            <div key={index} className="flex items-center gap-2">
              <Input
                className="flex-1 font-mono"
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
            // biome-ignore lint/suspicious/noArrayIndexKey: env list is ordered and stable per edit
            <div key={index} className="flex items-center gap-2">
              <Input
                className="w-44 font-mono"
                data-testid={`settings-mcp-servers-editor-env-key-${index}`}
                value={pair.key}
                placeholder="KEY"
                onChange={event => {
                  const next = [...env];
                  next[index] = { ...pair, key: event.target.value };
                  onChange(next);
                }}
              />
              <Input
                className="flex-1 font-mono"
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
                className="font-mono text-badge uppercase tracking-mono text-(--color-text-label)"
              >
                target
              </label>
              <NativeSelect
                id="settings-mcp-servers-delete-target-input"
                className="w-56 font-mono"
                data-testid="settings-mcp-servers-delete-target-input"
                value={selectedTarget}
                onChange={event => onTargetChange(event.target.value as SettingsMCPServerTarget)}
              >
                {availableTargets.map(candidate => (
                  <NativeSelectOption key={candidate} value={candidate}>
                    {targetLabel(candidate)}
                  </NativeSelectOption>
                ))}
              </NativeSelect>
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
    <Alert
      variant={isSaved ? "success" : "info"}
      role="status"
      data-testid="settings-page-mcp-servers-action-result"
      data-kind={action.kind}
    >
      <Check className="size-3.5" />
      <AlertDescription className="text-xs">{message}</AlertDescription>
      <AlertAction>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          onClick={onDismiss}
          data-testid="settings-page-mcp-servers-action-result-dismiss"
        >
          <X className="size-3.5" />
        </Button>
      </AlertAction>
    </Alert>
  );
}
