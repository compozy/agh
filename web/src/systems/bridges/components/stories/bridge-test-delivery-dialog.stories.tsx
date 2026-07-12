import { useState } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";
import { expect, userEvent, within } from "storybook/test";

import { createBridgeTestDeliveryDraft } from "@/systems/bridges";
import { bridgeDetailFixture, testBridgeDeliveryFixture } from "@/systems/bridges/mocks";
import type { SendBridgeTestResponse } from "@/systems/bridges/types";

import { BridgeTestDeliveryDialog } from "../bridge-test-delivery-dialog";

const storyDraft = createBridgeTestDeliveryDraft(bridgeDetailFixture.bridge);
const noop = () => undefined;

const meta = {
  args: {
    bridgeName: bridgeDetailFixture.bridge.display_name,
    draft: storyDraft,
    intent: "dry-run",
    isPending: false,
    onDraftChange: noop,
    onOpenChange: noop,
    onSubmit: noop,
    open: true,
    result: null,
  },
  component: BridgeTestDeliveryDialog,
  parameters: { layout: "fullscreen" },
  title: "systems/bridges/components/BridgeTestDeliveryDialog",
} satisfies Meta<typeof BridgeTestDeliveryDialog>;

export default meta;
type Story = StoryObj<typeof meta>;

const sendResult: SendBridgeTestResponse = {
  bridge_instance_id: bridgeDetailFixture.bridge.id,
  delivery_id: "delivery_story_001",
  delivery_target: {
    bridge_instance_id: bridgeDetailFixture.bridge.id,
    mode: "direct-send",
    peer_id: "launch-room",
  },
  remote_message_id: "telegram_2048",
  status: "sent",
};

function DryRunHarness({ includeResult = true }: { includeResult?: boolean }) {
  const [draft, setDraft] = useState(createBridgeTestDeliveryDraft(bridgeDetailFixture.bridge));
  return (
    <BridgeTestDeliveryDialog
      bridgeName={bridgeDetailFixture.bridge.display_name}
      draft={draft}
      intent="dry-run"
      isPending={false}
      onDraftChange={setDraft}
      onOpenChange={() => undefined}
      onSubmit={() => undefined}
      open
      result={includeResult ? testBridgeDeliveryFixture : null}
    />
  );
}

function SendTestHarness({ includeResult = true }: { includeResult?: boolean }) {
  const initialDraft = createBridgeTestDeliveryDraft(bridgeDetailFixture.bridge);
  const [draft, setDraft] = useState({ ...initialDraft, message: "Bridge setup is ready." });
  return (
    <BridgeTestDeliveryDialog
      bridgeName={bridgeDetailFixture.bridge.display_name}
      draft={draft}
      intent="send-test"
      isPending={false}
      onDraftChange={setDraft}
      onOpenChange={() => undefined}
      onSubmit={() => undefined}
      open
      result={includeResult ? sendResult : null}
    />
  );
}

export const DryRun: Story = { render: () => <DryRunHarness includeResult={false} /> };
export const DryRunResolved: Story = { render: () => <DryRunHarness /> };
export const SendTest: Story = { render: () => <SendTestHarness includeResult={false} /> };
export const SendTestDelivered: Story = { render: () => <SendTestHarness /> };

export const SendTestFlow: Story = {
  render: () => <SendTestHarness includeResult={false} />,
  tags: ["play-fn"],
  play: async ({ canvasElement }) => {
    const body = within(document.body);
    const message = await body.findByTestId("test-delivery-message");
    await userEvent.clear(message);
    await userEvent.type(message, "Operator ping", { delay: null });
    await expect(message).toHaveValue("Operator ping");
    await expect(body.getByTestId("bridge-send-test-dialog")).toBeVisible();
    void canvasElement;
  },
};
