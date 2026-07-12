import { Plus, Trash2 } from "lucide-react";

import type { MCPDraft, MCPEditorState, MCPEnvPair } from "@/hooks/routes/use-mcp-page";
import { useLocalRowKeys } from "@/hooks/use-local-row-keys";
import type { SettingsMCPServerEntry, SettingsMCPServerTarget } from "@/systems/settings";
import { Button, Input, NativeSelect, NativeSelectOption, Pill } from "@agh/ui";

import { mcpTargetLabel, mcpWriteTargetLabel } from "./mcp-server-labels";
import { SettingsEditorDialog } from "./settings-editor-dialog";
import { SettingsFieldRow } from "./settings-field-row";
import { SettingsSourceBadge } from "./settings-source-badge";

export interface MCPServerEditorProps {
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

export function MCPServerEditor({
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
          variant="modal"
          data-testid="settings-mcp-servers-editor-name"
          label="Name"
          description={
            isCreate
              ? "Lower-case identifier injected into agents as the MCP server name."
              : "Name is immutable -- remove the server and add a new one to rename."
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
          variant="modal"
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
  const description = (() => {
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
  })();

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
                {mcpTargetLabel(candidate)}
              </NativeSelectOption>
            ))}
          </NativeSelect>
          {entry ? (
            <div
              className="eyebrow flex flex-wrap items-center gap-1 text-muted"
              data-testid="settings-mcp-servers-editor-available-targets"
            >
              <span>allowed:</span>
              {entry.source_metadata.available_targets.map(available => (
                <Pill mono key={available} tone="neutral">
                  {mcpWriteTargetLabel(available)}
                </Pill>
              ))}
            </div>
          ) : null}
        </div>
      }
    />
  );
}

function ArgsEditor({ args, onChange }: { args: string[]; onChange: (next: string[]) => void }) {
  const rowKeys = useLocalRowKeys(args.length, "mcp-arg");
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
            <div key={rowKeys.keys[index]} className="flex items-center gap-2">
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
                onClick={() => {
                  rowKeys.remove(index);
                  onChange(args.filter((_, i) => i !== index));
                }}
                aria-label={`Remove arg ${index}`}
                data-testid={`settings-mcp-servers-editor-args-remove-${index}`}
              >
                <Trash2 className="size-3" />
              </Button>
            </div>
          ))}
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={() => {
              rowKeys.append();
              onChange([...args, ""]);
            }}
            data-testid="settings-mcp-servers-editor-args-add"
          >
            <Plus className="size-3" />
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
  const rowKeys = useLocalRowKeys(env.length, "mcp-env");
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
            <div key={rowKeys.keys[index]} className="flex items-center gap-2">
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
                onClick={() => {
                  rowKeys.remove(index);
                  onChange(env.filter((_, i) => i !== index));
                }}
                aria-label={`Remove env ${index}`}
                data-testid={`settings-mcp-servers-editor-env-remove-${index}`}
              >
                <Trash2 className="size-3" />
              </Button>
            </div>
          ))}
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={() => {
              rowKeys.append();
              onChange([...env, { key: "", value: "" }]);
            }}
            data-testid="settings-mcp-servers-editor-env-add"
          >
            <Plus className="size-3" />
            Add variable
          </Button>
        </div>
      }
    />
  );
}
