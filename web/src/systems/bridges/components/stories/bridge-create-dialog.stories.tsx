import { useState } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";
import { expect, fn, userEvent, within } from "storybook/test";

import { storyDefaultWorkspaceId, storyDefaultWorkspaceName } from "@/storybook/fintech-scenario";
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
    initialDraft ?? createBridgeCreateDraft(bridgeProvidersFixture, storyDefaultWorkspaceId)
  );

  return (
    <BridgeCreateDialog
      activeWorkspaceId={storyDefaultWorkspaceId}
      activeWorkspaceName={storyDefaultWorkspaceName}
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

export const ProviderStep: Story = {
  tags: ["play-fn"],
  render: () => <BridgeCreateDialogHarness />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(await canvas.findByTestId("bridge-wizard-stepper")).toBeInTheDocument();
    await expect(canvas.getByTestId("bridge-wizard-progress")).toHaveTextContent("Step 1 of 3");
  },
};

export const RuntimeStep: Story = {
  tags: ["play-fn"],
  render: () => <BridgeCreateDialogHarness />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const next = await canvas.findByTestId("bridge-wizard-next");
    await userEvent.click(next);
    await expect(canvas.getByTestId("bridge-wizard-progress")).toHaveTextContent("Step 2 of 3");
    await expect(canvas.getByTestId("bridge-display-name-input")).toBeInTheDocument();
  },
};

export const DeliveryStep: Story = {
  tags: ["play-fn"],
  render: () => <BridgeCreateDialogHarness />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const next = await canvas.findByTestId("bridge-wizard-next");
    await userEvent.click(next);
    await userEvent.click(await canvas.findByTestId("bridge-wizard-next"));
    await expect(canvas.getByTestId("bridge-wizard-progress")).toHaveTextContent("Step 3 of 3");
    await expect(canvas.getByTestId("submit-bridge-create")).toBeInTheDocument();
  },
};

export const InvalidProviderConfig: Story = {
  render: () => (
    <BridgeCreateDialogHarness
      initialDraft={{
        ...createBridgeCreateDraft(bridgeProvidersFixture, storyDefaultWorkspaceId),
        providerConfigText: "{ invalid json",
      }}
    />
  ),
  tags: ["play-fn"],
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const next = await canvas.findByTestId("bridge-wizard-next");
    await userEvent.click(next);
    await expect(canvas.getByTestId("bridge-provider-config-error")).toBeInTheDocument();
    await expect(canvas.getByTestId("bridge-wizard-next")).toBeDisabled();
  },
};

export const SubmitPayload: Story = {
  tags: ["play-fn"],
  render: () => {
    const onSubmit = fn();
    return <BridgeCreateDialogHarness onSubmit={onSubmit} />;
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(await canvas.findByTestId("bridge-wizard-next"));
    await userEvent.click(await canvas.findByTestId("bridge-wizard-next"));
    const submit = await canvas.findByTestId("submit-bridge-create");
    await expect(submit).toBeEnabled();
    await userEvent.click(submit);
  },
};
