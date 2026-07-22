import { Input, Textarea } from "@agh/ui";

import type { ProviderDraft } from "../types";
import type { ProviderDraftChange } from "./provider-edit-form";
import { ModalSettingsFieldRow } from "./settings-field-row";

interface ProviderGeneralFieldsProps {
  mode: "create" | "edit";
  draft: ProviderDraft;
  onChange: ProviderDraftChange;
}

export function ProviderGeneralFields({ mode, draft, onChange }: ProviderGeneralFieldsProps) {
  const isCreate = mode === "create";

  return (
    <>
      <ModalSettingsFieldRow
        data-testid="settings-providers-editor-name"
        label="Name"
        description={
          isCreate
            ? "Lower-case identifier used in agent frontmatter and CLI flags."
            : "Name is immutable -- create a new provider to rename."
        }
        control={
          <Input
            className="w-56 font-mono disabled:opacity-60"
            data-testid="settings-providers-editor-name-input"
            value={draft.name}
            placeholder="e.g. claude"
            disabled={!isCreate}
            onChange={event => onChange(current => ({ ...current, name: event.target.value }))}
          />
        }
      />
      <ModalSettingsFieldRow
        data-testid="settings-providers-editor-command"
        label="Command"
        description="Executable used to launch the ACP subprocess."
        control={
          <Input
            className="w-72 font-mono"
            data-testid="settings-providers-editor-command-input"
            value={draft.command}
            placeholder="npx @agentclientprotocol/claude-agent-acp@latest"
            onChange={event => onChange(current => ({ ...current, command: event.target.value }))}
          />
        }
      />
      <ModalSettingsFieldRow
        data-testid="settings-providers-editor-display-name"
        label="Display name"
        description="Operator-facing label shown beside the provider id."
        control={
          <Input
            className="w-56"
            data-testid="settings-providers-editor-display-name-input"
            value={draft.display_name}
            placeholder="OpenRouter"
            onChange={event =>
              onChange(current => ({ ...current, display_name: event.target.value }))
            }
          />
        }
      />
      <ModalSettingsFieldRow
        data-testid="settings-providers-editor-model"
        label="Default model"
        description="Sent to the provider when an agent does not specify one."
        control={
          <Input
            className="w-56 font-mono"
            data-testid="settings-providers-editor-model-input"
            value={draft.model_default}
            placeholder="Leave blank to use the provider default"
            onChange={event =>
              onChange(current => ({ ...current, model_default: event.target.value }))
            }
          />
        }
      />
      <ModalSettingsFieldRow
        data-testid="settings-providers-editor-curated-models"
        label="Curated models"
        description="Provider-scoped model IDs stored under models.curated."
        control={
          <Textarea
            className="min-h-24 w-72 font-mono text-xs"
            data-testid="settings-providers-editor-curated-models-input"
            value={draft.curated_models}
            placeholder="One model ID per line"
            onChange={event =>
              onChange(current => ({ ...current, curated_models: event.target.value }))
            }
          />
        }
      />
      <ModalSettingsFieldRow
        data-testid="settings-providers-editor-harness"
        label="Harness"
        description="Runtime adapter used to launch the provider."
        control={
          <Input
            className="w-40 font-mono"
            data-testid="settings-providers-editor-harness-input"
            value={draft.harness}
            placeholder="acp or pi_acp"
            onChange={event => onChange(current => ({ ...current, harness: event.target.value }))}
          />
        }
      />
      <ModalSettingsFieldRow
        data-testid="settings-providers-editor-runtime-provider"
        label="Runtime provider"
        description="Downstream provider id used by the selected harness."
        control={
          <Input
            className="w-56 font-mono"
            data-testid="settings-providers-editor-runtime-provider-input"
            value={draft.runtime_provider}
            placeholder="openrouter"
            onChange={event =>
              onChange(current => ({ ...current, runtime_provider: event.target.value }))
            }
          />
        }
      />
      <ModalSettingsFieldRow
        data-testid="settings-providers-editor-transport"
        label="Transport"
        description="Provider API family or Pi models override transport."
        control={
          <Input
            className="w-56 font-mono"
            data-testid="settings-providers-editor-transport-input"
            value={draft.transport}
            placeholder="openai"
            onChange={event => onChange(current => ({ ...current, transport: event.target.value }))}
          />
        }
      />
      <ModalSettingsFieldRow
        data-testid="settings-providers-editor-base-url"
        label="Base URL"
        description="Custom API base URL for Pi-backed model overrides."
        control={
          <Input
            className="w-72 font-mono"
            data-testid="settings-providers-editor-base-url-input"
            value={draft.base_url}
            placeholder="https://openrouter.ai/api/v1"
            onChange={event => onChange(current => ({ ...current, base_url: event.target.value }))}
          />
        }
      />
    </>
  );
}
