import { FormSection, Input } from "@agh/ui";

import type { MCPDraft, MCPDraftErrors } from "../lib/mcp-editor-model";
import type { SettingsMCPServerTarget } from "../types";

import { MCPFieldLabel, MCPNameField, MCPTargetField } from "./mcp-editor-fields";
import { MCPEditorOAuthSection } from "./mcp-editor-oauth-section";
import type { MCPVaultInventory } from "./mcp-secret-binding";

export interface MCPEditorRemoteSectionProps {
  draft: MCPDraft;
  errors: MCPDraftErrors;
  vaultInventory: MCPVaultInventory;
  isCreate: boolean;
  target: SettingsMCPServerTarget;
  availableTargets: SettingsMCPServerTarget[];
  onChange: (updater: (draft: MCPDraft) => MCPDraft) => void;
  onTargetChange: (target: SettingsMCPServerTarget) => void;
}

export function MCPEditorRemoteSection({
  draft,
  errors,
  vaultInventory,
  isCreate,
  target,
  availableTargets,
  onChange,
  onTargetChange,
}: MCPEditorRemoteSectionProps) {
  return (
    <>
      <FormSection
        title="Remote endpoint"
        rightLabel={draft.transport}
        data-testid="settings-mcp-editor-remote"
      >
        <div className="flex flex-col gap-3">
          <MCPNameField
            name={draft.name}
            isCreate={isCreate}
            error={errors.name}
            onChange={name => onChange(current => ({ ...current, name }))}
          />
          <div>
            <MCPFieldLabel htmlFor="mcp-editor-url" hint="required">
              URL
            </MCPFieldLabel>
            <Input
              id="mcp-editor-url"
              className="font-mono"
              value={draft.url}
              placeholder="https://mcp.linear.app/mcp"
              aria-invalid={errors.url ? true : undefined}
              onChange={event => onChange(current => ({ ...current, url: event.target.value }))}
              data-testid="settings-mcp-servers-editor-url-input"
            />
            {errors.url ? <p className="mt-1.5 text-caption text-danger">{errors.url}</p> : null}
          </div>
          <MCPTargetField
            target={target}
            availableTargets={availableTargets}
            onChange={onTargetChange}
          />
          <p
            className="rounded-md bg-info-tint p-2.5 text-caption text-info"
            data-testid="settings-mcp-editor-remote-notice"
          >
            <strong className="font-semibold">Remote exclusivity.</strong> HTTP and SSE definitions
            accept URL and optional OAuth fields only. Command, args, env, and secret_env are
            omitted.
          </p>
          {errors.remoteFields ? (
            <p className="text-caption text-danger" data-testid="settings-mcp-editor-remote-error">
              {errors.remoteFields}
            </p>
          ) : null}
        </div>
      </FormSection>

      <MCPEditorOAuthSection
        oauth={draft.oauth}
        vaultInventory={vaultInventory}
        errors={errors}
        onChange={oauth => onChange(current => ({ ...current, oauth }))}
      />
    </>
  );
}
