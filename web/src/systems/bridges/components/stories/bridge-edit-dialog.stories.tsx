import { useState } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { createBridgeUpdateDraft } from "@/systems/bridges";
import { bridgeDetailFixture, bridgeProvidersFixture } from "@/systems/bridges/mocks";
import { BridgeEditDialog } from "@/systems/bridges/components/bridge-edit-dialog";

const meta: Meta<typeof BridgeEditDialog> = {
  title: "systems/bridges/components/BridgeEditDialog",
  component: BridgeEditDialog,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component:
          "Edits mutable bridge runtime, routing, delivery target, and progress override settings.",
      },
    },
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

/** Shows the bridge's current mutable settings. */
export const Default: Story = {
  args: {},
  render: () => <BridgeEditDialogHarness />,
};

/** Shows provider configuration validation without hiding the rest of the editor. */
export const InvalidProviderConfig: Story = {
  args: {},
  render: () => (
    <BridgeEditDialogHarness
      initialDraft={{
        ...createBridgeUpdateDraft(bridgeDetailFixture.bridge),
        providerConfigText: "{ invalid json",
      }}
    />
  ),
};

/** Shows an explicit progress override hydrated from persisted delivery defaults. */
export const ProgressOverride: Story = {
  args: {},
  render: () => (
    <BridgeEditDialogHarness
      initialDraft={{
        ...createBridgeUpdateDraft(bridgeDetailFixture.bridge),
        deliveryDefaults: {
          ...createBridgeUpdateDraft(bridgeDetailFixture.bridge).deliveryDefaults,
          progress: {
            grouping: "separate",
            reactions: false,
            tool_progress: "verbose",
            typing: true,
          },
        },
      }}
    />
  ),
};
