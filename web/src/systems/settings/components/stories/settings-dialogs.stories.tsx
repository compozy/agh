import type { Meta, StoryObj } from "@storybook/react-vite";
import { KeyRound, Plug, Settings2, Trash2 } from "lucide-react";
import { fn } from "storybook/test";

import { ConfirmDialog, Input, Pill, SecretField } from "@agh/ui";

import { SettingsEditorDialog } from "../settings-editor-dialog";
import { SettingsFieldRow } from "../settings-field-row";

const meta: Meta<typeof SettingsEditorDialog> = {
  title: "systems/settings/components/SettingsDialogs",
  component: SettingsEditorDialog,
  parameters: {
    layout: "centered",
    docs: {
      description: {
        component:
          "Reusable settings create/edit and delete dialogs. `SettingsEditorDialog` pins the shared modal shell — ruled `EntityDialogHeader`, host size token, and `EntityDialogFooter` with its consequence hint — so vault and sandbox inherit chrome without per-page header forks.",
      },
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Editor dialog shows metadata, field rows, and warning feedback.
 */
export const Editor: Story = {
  args: {},
  render: () => (
    <SettingsEditorDialog
      open
      mode="edit"
      icon={Settings2}
      eyebrow="Settings · Provider"
      size="md"
      slug="providers"
      title="Edit provider"
      description="Update command and model defaults for this provider overlay."
      hint="Saved overlays apply to new sessions in this workspace."
      metadata={<Pill tone="info">workspace override</Pill>}
      warnings={["Changing the command requires a daemon restart."]}
      canSave
      isSaving={false}
      onSave={fn()}
      onOpenChange={fn()}
    >
      <SettingsFieldRow label="Command" control={<Input defaultValue="codex" />} />
      <SettingsFieldRow label="Default model" control={<Input defaultValue="gpt-5.4" />} />
    </SettingsEditorDialog>
  ),
};

/**
 * Delete dialog renders fallback guidance and destructive confirmation.
 */
export const Delete: Story = {
  args: {},
  render: () => (
    <ConfirmDialog
      open
      title="Delete provider overlay"
      description="This removes the workspace override; built-in provider defaults remain available."
      note="The provider falls back to the built-in config after deletion."
      isPending={false}
      cancelLabel="Cancel"
      onConfirm={fn()}
      onOpenChange={fn()}
      confirmIcon={Trash2}
      confirmLabel="Delete"
      contentProps={{ "data-testid": "settings-providers-delete" }}
    />
  ),
};

/**
 * Dialog feedback states are visible without relying on a real route mutation.
 */
export const SavingAndError: Story = {
  args: {},
  render: () => (
    <SettingsEditorDialog
      open
      mode="create"
      icon={Plug}
      eyebrow="System · MCP server"
      size="md"
      slug="mcp"
      title="Add MCP server"
      error="Server command failed validation."
      canSave={false}
      isSaving={false}
      onSave={fn()}
      onOpenChange={fn()}
    >
      <SettingsFieldRow label="Name" error="Required" control={<Input aria-invalid />} />
    </SettingsEditorDialog>
  ),
};

/**
 * Vault create runs through the same shell. This story is the shell pin for the
 * modal-redesign visual contract: header, host token, and footer only — the
 * field body is owned by the vault body migration.
 */
export const VaultCreate: Story = {
  args: {},
  render: () => (
    <SettingsEditorDialog
      open
      mode="create"
      icon={KeyRound}
      eyebrow="System · Vault"
      size="sm"
      slug="vault"
      title="Add vault secret"
      description="Stores a write-only secret value and returns redacted metadata."
      hint={
        <>
          Bind it from providers, bridges, or sandboxes as{" "}
          <b className="font-medium text-muted">vault:&lt;reference&gt;</b>.
        </>
      }
      saveLabel="Store secret"
      canSave
      isSaving={false}
      onSave={fn()}
      onOpenChange={fn()}
    >
      <SettingsFieldRow
        label="Ref"
        description="Daemon-owned vault reference."
        control={<Input className="font-mono" defaultValue="vault:mcp/github-token" />}
      />
      <SettingsFieldRow
        label="Kind"
        description="Metadata label returned on public Vault surfaces."
        control={<Input className="w-48 font-mono" defaultValue="api_key" />}
      />
      <SecretField
        description="Write-only payload. The daemon never returns this value."
        id="vault-secret-value"
        label="Secret value"
        onValueChange={fn()}
        required
        value=""
      />
    </SettingsEditorDialog>
  ),
};
