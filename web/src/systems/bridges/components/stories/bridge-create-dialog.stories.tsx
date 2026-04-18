import { useState } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";

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
}: {
  initialDraft?: ReturnType<typeof createBridgeCreateDraft>;
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
      onSubmit={() => undefined}
      open
      providers={bridgeProvidersFixture}
    />
  );
}

export const Default: Story = {
  render: () => <BridgeCreateDialogHarness />,
};

export const Error: Story = {
  render: () => (
    <BridgeCreateDialogHarness
      initialDraft={{
        ...createBridgeCreateDraft(bridgeProvidersFixture, "ws_storybook"),
        providerConfigText: "{ invalid json",
      }}
    />
  ),
};
