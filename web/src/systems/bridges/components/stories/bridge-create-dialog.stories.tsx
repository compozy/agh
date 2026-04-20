import { useState } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";
import { expect, fn, userEvent, within } from "storybook/test";

import { createBridgeCreateDraft } from "@/systems/bridges";
import { bridgeProvidersFixture } from "@/systems/bridges/mocks";

import { BridgeCreateDialog } from "../bridge-create-dialog";

const meta: Meta<typeof BridgeCreateDialog> = {
  title: "systems/bridges/BridgeCreateDialog",
  component: BridgeCreateDialog,
  parameters: {
    layout: "fullscreen",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

function BridgeCreateDialogHarness({
  initialDraft,
  onSubmit,
}: {
  initialDraft?: ReturnType<typeof createBridgeCreateDraft>;
  onSubmit?: () => void;
}) {
  const [draft, setDraft] = useState(
    initialDraft ?? createBridgeCreateDraft(bridgeProvidersFixture, "ws_storybook")
  );

  return (
    <BridgeCreateDialog
      activeWorkspaceId="ws_storybook"
      activeWorkspaceName="agh2"
      draft={draft}
      isPending={false}
      onDraftChange={setDraft}
      onOpenChange={() => undefined}
      onSubmit={onSubmit ?? (() => undefined)}
      open
      providers={bridgeProvidersFixture}
    />
  );
}

export const Default: Story = {
  render: () => <BridgeCreateDialogHarness />,
};

export const InvalidProviderConfig: Story = {
  render: () => (
    <BridgeCreateDialogHarness
      initialDraft={{
        ...createBridgeCreateDraft(bridgeProvidersFixture, "ws_storybook"),
        providerConfigText: "{ invalid json",
      }}
    />
  ),
};

export const SubmitPayload: Story = {
  tags: ["play-fn"],
  render: () => {
    const onSubmit = fn();
    return <BridgeCreateDialogHarness onSubmit={onSubmit} />;
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const submit = await canvas.findByTestId("submit-bridge-create");
    await userEvent.click(submit);
    // Default draft already has Telegram selected with no provider config errors
    await expect(submit).toBeEnabled();
  },
};
