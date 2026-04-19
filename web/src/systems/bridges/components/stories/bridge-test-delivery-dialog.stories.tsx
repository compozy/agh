import { useState } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";
import { expect, userEvent, within } from "storybook/test";

import { createBridgeTestDeliveryDraft } from "@/systems/bridges";
import { bridgeDetailFixture, testBridgeDeliveryFixture } from "@/systems/bridges/mocks";

import { BridgeTestDeliveryDialog } from "../bridge-test-delivery-dialog";

const meta: Meta<typeof BridgeTestDeliveryDialog> = {
  title: "systems/bridges/BridgeTestDeliveryDialog",
  component: BridgeTestDeliveryDialog,
  parameters: {
    layout: "fullscreen",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

function BridgeTestDeliveryDialogHarness({ includeResult = true }: { includeResult?: boolean }) {
  const [draft, setDraft] = useState(createBridgeTestDeliveryDraft(bridgeDetailFixture.bridge));

  return (
    <BridgeTestDeliveryDialog
      bridgeName={bridgeDetailFixture.bridge.display_name}
      draft={draft}
      isPending={false}
      onDraftChange={setDraft}
      onOpenChange={() => undefined}
      onSubmit={() => undefined}
      open
      result={includeResult ? testBridgeDeliveryFixture : null}
    />
  );
}

export const Default: Story = {
  render: () => <BridgeTestDeliveryDialogHarness includeResult={false} />,
};

export const WithResolvedTarget: Story = {
  render: () => <BridgeTestDeliveryDialogHarness includeResult />,
};

export const OpenFlow: Story = {
  tags: ["play-fn"],
  render: () => <BridgeTestDeliveryDialogHarness includeResult={false} />,
  play: async ({ canvasElement }) => {
    const body = within(document.body);
    const message = await body.findByTestId("test-delivery-message");
    await userEvent.type(message, "Ping", { delay: null });
    await expect(message).toHaveValue("Ping");
    const dialog = await body.findByTestId("bridge-test-delivery-dialog");
    await expect(dialog).toBeVisible();
    void canvasElement;
  },
};
