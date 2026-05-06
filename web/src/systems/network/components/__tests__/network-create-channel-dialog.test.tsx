import type { ReactElement } from "react";
import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { UIProvider } from "@agh/ui";
import { agentFixtures } from "@/systems/agent/mocks";

import { createNetworkChannelDraft } from "../../lib/network-formatters";
import { NetworkCreateChannelDialog } from "../network-create-channel-dialog";

function renderDialog(ui: ReactElement) {
  return render(<UIProvider reducedMotion="always">{ui}</UIProvider>);
}

describe("NetworkCreateChannelDialog", () => {
  it("Should render the submit button disabled and not fire onSubmit when canSubmit=false", () => {
    const onSubmit = vi.fn();
    renderDialog(
      <NetworkCreateChannelDialog
        agents={agentFixtures}
        canSubmit={false}
        draft={createNetworkChannelDraft()}
        isSubmitting={false}
        onChannelNameChange={() => undefined}
        onOpenChange={() => undefined}
        onPurposeChange={() => undefined}
        onAgentSelectionChange={() => undefined}
        onSubmit={onSubmit}
        open
        workspaceName="polybot"
      />
    );

    const submit = screen.getByTestId("network-create-channel-submit");
    expect(submit).toBeDisabled();
    fireEvent.click(submit);
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it("Should surface the channel-name input wired to onChannelNameChange", () => {
    const onChannelNameChange = vi.fn();
    renderDialog(
      <NetworkCreateChannelDialog
        agents={agentFixtures}
        canSubmit
        draft={createNetworkChannelDraft()}
        isSubmitting={false}
        onChannelNameChange={onChannelNameChange}
        onOpenChange={() => undefined}
        onPurposeChange={() => undefined}
        onAgentSelectionChange={() => undefined}
        onSubmit={() => undefined}
        open
        workspaceName="polybot"
      />
    );

    fireEvent.change(screen.getByTestId("network-channel-name-input"), {
      target: { value: "deployments" },
    });

    expect(onChannelNameChange).toHaveBeenCalledWith("deployments");
  });

  it("Should surface the purpose input wired to onPurposeChange", () => {
    const onPurposeChange = vi.fn();
    renderDialog(
      <NetworkCreateChannelDialog
        agents={agentFixtures}
        canSubmit
        draft={createNetworkChannelDraft()}
        isSubmitting={false}
        onChannelNameChange={() => undefined}
        onOpenChange={() => undefined}
        onPurposeChange={onPurposeChange}
        onAgentSelectionChange={() => undefined}
        onSubmit={() => undefined}
        open
        workspaceName="polybot"
      />
    );

    fireEvent.change(screen.getByTestId("network-channel-purpose-input"), {
      target: { value: "Coordinate deploy verification" },
    });

    expect(screen.getByTestId("network-channel-purpose-input")).toBeRequired();
    expect(screen.getByTestId("network-channel-purpose-input")).toHaveAttribute(
      "aria-required",
      "true"
    );
    expect(onPurposeChange).toHaveBeenCalledWith("Coordinate deploy verification");
  });

  it("Should toggle an agent when its row is clicked from the multi-select popover", async () => {
    const user = userEvent.setup();
    const onAgentSelectionChange = vi.fn();
    renderDialog(
      <NetworkCreateChannelDialog
        agents={agentFixtures}
        canSubmit
        draft={{ ...createNetworkChannelDraft(), channelName: "x" }}
        isSubmitting={false}
        onChannelNameChange={() => undefined}
        onOpenChange={() => undefined}
        onPurposeChange={() => undefined}
        onAgentSelectionChange={onAgentSelectionChange}
        onSubmit={() => undefined}
        open
        workspaceName="polybot"
      />
    );

    const firstAgent = agentFixtures[0]!;
    await user.click(screen.getByTestId("network-create-channel-agent-trigger"));
    await user.click(screen.getByTestId(`network-agent-option-${firstAgent.name}`));
    expect(onAgentSelectionChange).toHaveBeenCalledWith([firstAgent.name]);
  });

  it("Should surface the Empty agents state and a workspace warning when the active workspace is missing", () => {
    renderDialog(
      <NetworkCreateChannelDialog
        agents={[]}
        canSubmit={false}
        draft={createNetworkChannelDraft()}
        isSubmitting={false}
        onChannelNameChange={() => undefined}
        onOpenChange={() => undefined}
        onPurposeChange={() => undefined}
        onAgentSelectionChange={() => undefined}
        onSubmit={() => undefined}
        open
        workspaceName={null}
      />
    );

    expect(screen.getByText("No agents available")).toBeInTheDocument();
    expect(
      screen.getByText("Select an active workspace before creating a channel.")
    ).toBeInTheDocument();
  });

  it("Should fire onSubmit once when the form is submitted with canSubmit=true", () => {
    const onSubmit = vi.fn();
    renderDialog(
      <NetworkCreateChannelDialog
        agents={agentFixtures}
        canSubmit
        draft={{
          ...createNetworkChannelDraft(),
          channelName: "deploy",
          purpose: "Coordinate deploy verification",
          selectedAgentNames: [agentFixtures[0]!.name],
        }}
        isSubmitting={false}
        onChannelNameChange={() => undefined}
        onOpenChange={() => undefined}
        onPurposeChange={() => undefined}
        onAgentSelectionChange={() => undefined}
        onSubmit={onSubmit}
        open
        workspaceName="polybot"
      />
    );

    fireEvent.click(screen.getByTestId("network-create-channel-submit"));
    expect(onSubmit).toHaveBeenCalledTimes(1);
  });

  it("Should call onOpenChange when the Cancel button is pressed", () => {
    const onOpenChange = vi.fn();
    renderDialog(
      <NetworkCreateChannelDialog
        agents={agentFixtures}
        canSubmit
        draft={createNetworkChannelDraft()}
        isSubmitting={false}
        onChannelNameChange={() => undefined}
        onOpenChange={onOpenChange}
        onPurposeChange={() => undefined}
        onAgentSelectionChange={() => undefined}
        onSubmit={() => undefined}
        open
        workspaceName="polybot"
      />
    );

    fireEvent.click(screen.getByText("Cancel"));
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });
});
