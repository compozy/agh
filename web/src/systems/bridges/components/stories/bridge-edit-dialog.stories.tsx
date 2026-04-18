import { useState } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { createBridgeUpdateDraft } from "@/systems/bridges";
import { bridgeDetailFixture, bridgeProvidersFixture } from "@/systems/bridges/mocks";

import { BridgeEditDialog } from "../bridge-edit-dialog";

const meta: Meta<typeof BridgeEditDialog> = {
  title: "systems/bridges/BridgeEditDialog",
  component: BridgeEditDialog,
  parameters: {
    layout: "fullscreen",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

function BridgeEditDialogHarness({
  initialDraft,
}: {
  initialDraft?: ReturnType<typeof createBridgeUpdateDraft>;
}) {
  const [draft, setDraft] = useState(
    initialDraft ?? createBridgeUpdateDraft(bridgeDetailFixture.bridge)
  );

  return (
    <BridgeEditDialog
      allowProviderDefaultDmPolicy={false}
      bridgeName={bridgeDetailFixture.bridge.display_name}
      draft={draft}
      isPending={false}
      onDraftChange={setDraft}
      onOpenChange={() => undefined}
      onSubmit={() => undefined}
      open
      provider={bridgeProvidersFixture[0]}
    />
  );
}

export const Default: Story = {
  render: () => <BridgeEditDialogHarness />,
};

export const Error: Story = {
  render: () => (
    <BridgeEditDialogHarness
      initialDraft={{
        ...createBridgeUpdateDraft(bridgeDetailFixture.bridge),
        providerConfigText: "{ invalid json",
      }}
    />
  ),
};
