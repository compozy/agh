import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { agentFixtures } from "@/systems/agent/mocks";

import { createNetworkChannelDraft } from "../lib/network-formatters";
import { NetworkCreateChannelDialog } from "./network-create-channel-dialog";

describe("NetworkCreateChannelDialog", () => {
  it("Should render the submit button disabled and not fire onSubmit when canSubmit=false", () => {
    const onSubmit = vi.fn();
    render(
      <NetworkCreateChannelDialog
        agents={agentFixtures}
        canSubmit={false}
        draft={createNetworkChannelDraft()}
        isSubmitting={false}
        onChannelNameChange={() => undefined}
        onOpenChange={() => undefined}
        onPurposeChange={() => undefined}
        onSubmit={onSubmit}
        onToggleAgent={() => undefined}
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
    render(
      <NetworkCreateChannelDialog
        agents={agentFixtures}
        canSubmit
        draft={createNetworkChannelDraft()}
        isSubmitting={false}
        onChannelNameChange={onChannelNameChange}
        onOpenChange={() => undefined}
        onPurposeChange={() => undefined}
        onSubmit={() => undefined}
        onToggleAgent={() => undefined}
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
    render(
      <NetworkCreateChannelDialog
        agents={agentFixtures}
        canSubmit
        draft={createNetworkChannelDraft()}
        isSubmitting={false}
        onChannelNameChange={() => undefined}
        onOpenChange={() => undefined}
        onPurposeChange={onPurposeChange}
        onSubmit={() => undefined}
        onToggleAgent={() => undefined}
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

  it("Should toggle an agent when its row is clicked", () => {
    const onToggleAgent = vi.fn();
    render(
      <NetworkCreateChannelDialog
        agents={agentFixtures}
        canSubmit
        draft={{ ...createNetworkChannelDraft(), channelName: "x" }}
        isSubmitting={false}
        onChannelNameChange={() => undefined}
        onOpenChange={() => undefined}
        onPurposeChange={() => undefined}
        onSubmit={() => undefined}
        onToggleAgent={onToggleAgent}
        open
        workspaceName="polybot"
      />
    );

    const firstAgent = agentFixtures[0]!;
    fireEvent.click(screen.getByTestId(`network-agent-option-${firstAgent.name}`));
    expect(onToggleAgent).toHaveBeenCalledWith(firstAgent.name);
  });

  it("Should surface the Empty agents state and a workspace warning when the active workspace is missing", () => {
    render(
      <NetworkCreateChannelDialog
        agents={[]}
        canSubmit={false}
        draft={createNetworkChannelDraft()}
        isSubmitting={false}
        onChannelNameChange={() => undefined}
        onOpenChange={() => undefined}
        onPurposeChange={() => undefined}
        onSubmit={() => undefined}
        onToggleAgent={() => undefined}
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
    render(
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
        onSubmit={onSubmit}
        onToggleAgent={() => undefined}
        open
        workspaceName="polybot"
      />
    );

    fireEvent.click(screen.getByTestId("network-create-channel-submit"));
    expect(onSubmit).toHaveBeenCalledTimes(1);
  });

  it("Should call onOpenChange when the Cancel button is pressed", () => {
    const onOpenChange = vi.fn();
    render(
      <NetworkCreateChannelDialog
        agents={agentFixtures}
        canSubmit
        draft={createNetworkChannelDraft()}
        isSubmitting={false}
        onChannelNameChange={() => undefined}
        onOpenChange={onOpenChange}
        onPurposeChange={() => undefined}
        onSubmit={() => undefined}
        onToggleAgent={() => undefined}
        open
        workspaceName="polybot"
      />
    );

    fireEvent.click(screen.getByText("Cancel"));
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });
});
