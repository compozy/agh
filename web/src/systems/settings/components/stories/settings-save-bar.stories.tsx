import { useState } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";
import { expect, fn, userEvent, within } from "storybook/test";

import { SettingsSaveBar } from "../settings-save-bar";

const meta: Meta<typeof SettingsSaveBar> = {
  title: "systems/settings/SettingsSaveBar",
  component: SettingsSaveBar,
  parameters: {
    layout: "padded",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Clean: Story = {
  args: {
    slug: "general",
    isDirty: false,
    isSaving: false,
    onSave: fn(),
    onReset: fn(),
  },
};

export const Dirty: Story = {
  args: {
    slug: "general",
    isDirty: true,
    isSaving: false,
    onSave: fn(),
    onReset: fn(),
  },
};

export const Saving: Story = {
  args: {
    slug: "general",
    isDirty: true,
    isSaving: true,
    onSave: fn(),
    onReset: fn(),
  },
};

export const Invalid: Story = {
  args: {
    slug: "general",
    isDirty: true,
    isSaving: false,
    isInvalid: true,
    onSave: fn(),
    onReset: fn(),
  },
};

export const WithError: Story = {
  args: {
    slug: "general",
    isDirty: true,
    isSaving: false,
    error: "Could not reach the daemon",
    onSave: fn(),
    onReset: fn(),
  },
};

export const WithWarnings: Story = {
  args: {
    slug: "general",
    isDirty: true,
    isSaving: false,
    warnings: ["Restart required", "Environment mismatch"],
    onSave: fn(),
    onReset: fn(),
  },
};

export const LastApplied: Story = {
  args: {
    slug: "general",
    isDirty: false,
    isSaving: false,
    lastAppliedLabel: "Applied 2 minutes ago",
    onSave: fn(),
    onReset: fn(),
  },
};

/**
 * Interaction test: flips isDirty true → false via Discard and asserts the
 * buttons disable and the placeholder copy returns.
 */
export const DirtyToCleanInteraction: Story = {
  tags: ["play-fn"],
  render: () => {
    function Harness() {
      const [isDirty, setIsDirty] = useState(true);
      return (
        <SettingsSaveBar
          slug="general"
          isDirty={isDirty}
          isSaving={false}
          onSave={() => setIsDirty(false)}
          onReset={() => setIsDirty(false)}
        />
      );
    }
    return <Harness />;
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const reset = canvas.getByTestId("settings-page-general-reset");
    const save = canvas.getByTestId("settings-page-general-save");
    await expect(reset).not.toBeDisabled();
    await userEvent.click(reset);
    await expect(save).toBeDisabled();
    await expect(reset).toBeDisabled();
    await expect(canvas.getByText(/No unsaved changes/i)).toBeInTheDocument();
  },
};
