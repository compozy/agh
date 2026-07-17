import {
  Alert,
  AlertDescription,
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Eyebrow,
  RadioCard,
  Spinner,
} from "@agh/ui";
import { AlertCircle } from "lucide-react";

import {
  withTransport,
  type MCPDraft,
  type MCPDraftErrors,
  type MCPTransport,
} from "../lib/mcp-editor-model";
import type { SettingsMCPServerEntry, SettingsMCPServerTarget } from "../types";

import { MCPEditorRemoteSection } from "./mcp-editor-remote-section";
import { MCPEditorStdioSection } from "./mcp-editor-stdio-section";
import { SettingsSourceBadge } from "./settings-source-badge";

const TRANSPORTS: { value: MCPTransport; description: string }[] = [
  { value: "stdio", description: "Local subprocess" },
  { value: "http", description: "Remote HTTP endpoint" },
  { value: "sse", description: "Remote SSE endpoint" },
];

export interface MCPServerEditorProps {
  open: boolean;
  mode: "create" | "edit";
  draft: MCPDraft;
  scope: "global" | "workspace";
  errors: MCPDraftErrors;
  isValid: boolean;
  isSaving: boolean;
  saveError: string | null;
  warnings?: string[];
  vaultRefs: string[];
  target: SettingsMCPServerTarget;
  availableTargets: SettingsMCPServerTarget[];
  entry: SettingsMCPServerEntry | null;
  onChange: (updater: (draft: MCPDraft) => MCPDraft) => void;
  onTargetChange: (target: SettingsMCPServerTarget) => void;
  onClose: () => void;
  onSave: () => void;
  onRemove?: () => void;
}

export function MCPServerEditor({
  open,
  mode,
  draft,
  scope,
  errors,
  isValid,
  isSaving,
  saveError,
  warnings,
  vaultRefs,
  target,
  availableTargets,
  entry,
  onChange,
  onTargetChange,
  onClose,
  onSave,
  onRemove,
}: MCPServerEditorProps) {
  if (!open) return null;

  const isCreate = mode === "create";
  const title = isCreate ? "Add MCP server" : `Edit ${draft.name}`;
  const description = isCreate
    ? "Saving writes a full replacement of the named definition in the selected target."
    : "Saving replaces the complete server definition in the selected target.";

  return (
    <Dialog
      open={open}
      onOpenChange={next => {
        if (!next) onClose();
      }}
    >
      <DialogContent
        className="grid w-[calc(100%-2rem)] max-w-xl gap-4 sm:max-w-3xl"
        unframed
        data-testid="settings-mcp-servers-editor"
        data-mode={mode}
        data-transport={draft.transport}
      >
        <DialogHeader variant="ruled">
          <Eyebrow className="text-muted">MCP server · {scope}</Eyebrow>
          <DialogTitle data-testid="settings-mcp-servers-editor-title">{title}</DialogTitle>
          <DialogDescription>{description}</DialogDescription>
          {entry ? (
            <div className="pt-1">
              <SettingsSourceBadge
                data-testid="settings-mcp-servers-editor-source"
                source={entry.source_metadata.effective_source}
                shadowed={entry.source_metadata.shadowed_sources ?? []}
              />
            </div>
          ) : null}
        </DialogHeader>

        <div
          className="flex max-h-[62vh] flex-col gap-4 overflow-y-auto px-5 py-4"
          data-testid="settings-mcp-servers-editor-body"
        >
          <div>
            <Eyebrow className="mb-1.5 block text-muted">Transport</Eyebrow>
            <div
              className="grid grid-cols-3 gap-2"
              role="radiogroup"
              aria-label="Transport"
              data-testid="settings-mcp-servers-editor-transport"
            >
              {TRANSPORTS.map(option => (
                <RadioCard
                  key={option.value}
                  selected={draft.transport === option.value}
                  title={option.value}
                  description={option.description}
                  onSelect={() => onChange(current => withTransport(current, option.value))}
                  data-testid={`settings-mcp-servers-editor-transport-${option.value}`}
                />
              ))}
            </div>
          </div>

          <div className="grid grid-cols-1 gap-3 lg:grid-cols-2">
            {draft.transport === "stdio" ? (
              <MCPEditorStdioSection
                draft={draft}
                errors={errors}
                vaultRefs={vaultRefs}
                isCreate={isCreate}
                target={target}
                availableTargets={availableTargets}
                onChange={onChange}
                onTargetChange={onTargetChange}
              />
            ) : (
              <MCPEditorRemoteSection
                draft={draft}
                errors={errors}
                vaultRefs={vaultRefs}
                isCreate={isCreate}
                target={target}
                availableTargets={availableTargets}
                onChange={onChange}
                onTargetChange={onTargetChange}
              />
            )}
          </div>

          {saveError ? (
            <Alert variant="danger" data-testid="settings-mcp-servers-editor-error">
              <AlertCircle className="mt-0.5 size-3 shrink-0" />
              <AlertDescription className="text-xs">{saveError}</AlertDescription>
            </Alert>
          ) : null}
          {!saveError && warnings && warnings.length > 0 ? (
            <Alert variant="warning" data-testid="settings-mcp-servers-editor-warnings">
              <AlertCircle className="mt-0.5 size-3 shrink-0" />
              <AlertDescription>
                <ul className="flex flex-col gap-1 text-xs">
                  {warnings.map(warning => (
                    <li key={warning}>{warning}</li>
                  ))}
                </ul>
              </AlertDescription>
            </Alert>
          ) : null}
        </div>

        <DialogFooter variant="ruled">
          {!isCreate && onRemove ? (
            <Button
              type="button"
              variant="destructive"
              size="sm"
              className="mr-auto"
              onClick={onRemove}
              disabled={isSaving}
              data-testid="settings-mcp-servers-editor-remove"
            >
              Remove server
            </Button>
          ) : (
            <span className="mr-auto text-caption text-muted">
              Plaintext secrets are write-only.
            </span>
          )}
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={onClose}
            disabled={isSaving}
            data-testid="settings-mcp-servers-editor-cancel"
          >
            Cancel
          </Button>
          <Button
            type="button"
            size="sm"
            onClick={onSave}
            disabled={!isValid || isSaving}
            data-testid="settings-mcp-servers-editor-save"
          >
            {isSaving ? <Spinner className="size-3" /> : null}
            {isSaving ? "Saving..." : "Save changes"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
