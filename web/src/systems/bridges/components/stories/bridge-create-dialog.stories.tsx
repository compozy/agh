import { useState } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";
import { expect, userEvent, within } from "storybook/test";

import { storyDefaultWorkspaceId, storyDefaultWorkspaceName } from "@/storybook/fintech-scenario";
import { createBridgeCreateDraft } from "@/systems/bridges";
import { bridgeProvidersFixture, slackBridgeManifestFixture } from "@/systems/bridges/mocks";

import { BridgeCreateDialog } from "../bridge-create-dialog";
import type { BridgeManifestCommittedState } from "../bridge-manifest-handoff";

const manifestJSON = JSON.stringify(slackBridgeManifestFixture.manifest, null, 2);

const meta: Meta<typeof BridgeCreateDialog> = {
  title: "systems/bridges/components/BridgeCreateDialog",
  component: BridgeCreateDialog,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component:
          "Creates a bridge through provider, runtime, and delivery steps, then presents a committed Slack manifest handoff when supported.",
      },
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

function BridgeCreateDialogHarness({
  manifestState,
  supportsManifest = false,
}: {
  manifestState?: BridgeManifestCommittedState;
  supportsManifest?: boolean;
}) {
  const [draft, setDraft] = useState(
    createBridgeCreateDraft(bridgeProvidersFixture, storyDefaultWorkspaceId)
  );

  return (
    <BridgeCreateDialog
      activeWorkspaceId={storyDefaultWorkspaceId}
      activeWorkspaceName={storyDefaultWorkspaceName}
      draft={draft}
      isPending={false}
      manifestState={manifestState}
      onDraftChange={setDraft}
      onOpenChange={() => undefined}
      onSubmit={() => undefined}
      open
      providers={bridgeProvidersFixture}
      supportsManifest={supportsManifest}
    />
  );
}

/** Shows provider selection before any bridge has been persisted. */
export const Default: Story = {
  args: {},
  render: () => <BridgeCreateDialogHarness supportsManifest />,
};

/** Advances to provider-owned runtime configuration. */
export const RuntimeStep: Story = {
  args: {},
  tags: ["play-fn"],
  render: () => <BridgeCreateDialogHarness supportsManifest />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement.ownerDocument.body);
    await userEvent.click(await canvas.findByTestId("bridge-wizard-next"));
    await expect(canvas.getByTestId("bridge-wizard-progress")).toHaveTextContent("Step 2 of 3");
    await expect(canvas.getByTestId("bridge-display-name-input")).toBeInTheDocument();
  },
};

/** Shows routing, delivery target defaults, and the optional progress override. */
export const DeliveryStep: Story = {
  args: {},
  tags: ["play-fn"],
  render: () => <BridgeCreateDialogHarness supportsManifest />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement.ownerDocument.body);
    await userEvent.click(await canvas.findByTestId("bridge-wizard-next"));
    await userEvent.click(await canvas.findByTestId("bridge-wizard-next"));
    await expect(canvas.getByTestId("bridge-wizard-progress")).toHaveTextContent("Step 3 of 3");
    await expect(canvas.getByTestId("bridge-delivery-progress-mode-select")).toBeInTheDocument();
  },
};

/** Shows the deterministic post-create Slack manifest handoff. */
export const SlackManifestHandoff: Story = {
  args: {},
  render: () => (
    <BridgeCreateDialogHarness
      manifestState={{
        bridgeId: "brg_launch_room",
        isLoading: false,
        manifestJSON,
        onOpenBridge: () => undefined,
        onRetry: () => undefined,
      }}
      supportsManifest
    />
  ),
};

/** Shows recovery after the bridge persisted but manifest retrieval failed. */
export const SlackManifestError: Story = {
  args: {},
  render: () => (
    <BridgeCreateDialogHarness
      manifestState={{
        bridgeId: "brg_launch_room",
        error: "The saved webhook URL is not valid for a Slack manifest.",
        isLoading: false,
        onOpenBridge: () => undefined,
        onRetry: () => undefined,
      }}
      supportsManifest
    />
  ),
};
