import { useState } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";

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

function BridgeTestDeliveryDialogHarness({ errorMessage }: { errorMessage?: string }) {
  const [draft, setDraft] = useState(createBridgeTestDeliveryDraft(bridgeDetailFixture.bridge));

  return (
    <div className="space-y-4">
      {errorMessage ? (
        <div className="mx-auto max-w-2xl rounded-xl border border-[color:var(--color-danger)] bg-[color:var(--color-danger-tint)] px-4 py-3 text-sm text-[color:var(--color-danger)]">
          {errorMessage}
        </div>
      ) : null}
      <BridgeTestDeliveryDialog
        bridgeName={bridgeDetailFixture.bridge.display_name}
        draft={draft}
        isPending={false}
        onDraftChange={setDraft}
        onOpenChange={() => undefined}
        onSubmit={() => undefined}
        open
        result={errorMessage ? null : testBridgeDeliveryFixture}
      />
    </div>
  );
}

export const Default: Story = {
  render: () => <BridgeTestDeliveryDialogHarness />,
};

export const Error: Story = {
  render: () => (
    <BridgeTestDeliveryDialogHarness errorMessage="Failed to resolve delivery target for Support." />
  ),
};
