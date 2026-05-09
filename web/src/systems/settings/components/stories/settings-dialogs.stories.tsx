import type { Meta, StoryObj } from "@storybook/react-vite";
import { Trash2 } from "lucide-react";
import { fn } from "storybook/test";

import { ConfirmDialog, Input, Pill } from "@agh/ui";

import { SettingsEditorDialog } from "../settings-editor-dialog";
import { SettingsFieldRow } from "../settings-field-row";

const meta: Meta<typeof SettingsEditorDialog> = {
  title: "systems/settings/SettingsDialogs",
  component: SettingsEditorDialog,
  parameters: {
    layout: "centered",
    docs: {
      description: {
        component: "Reusable settings create/edit and delete dialogs with inline feedback.",
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
      slug="providers"
      title="Edit provider"
      description="Update command and model defaults for this provider overlay."
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
