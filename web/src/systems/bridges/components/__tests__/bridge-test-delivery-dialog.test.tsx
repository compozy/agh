import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";

import { BridgeTestDeliveryDialog } from "@/systems/bridges/components/bridge-test-delivery-dialog";
import type {
  BridgeTestDeliveryDraft,
  SendBridgeTestResponse,
  TestBridgeDeliveryResponse,
} from "@/systems/bridges/types";

const baseDraft: BridgeTestDeliveryDraft = { message: "", target: {} };
const dryRunResult: TestBridgeDeliveryResponse = {
  delivery_target: { bridge_instance_id: "brg_support", mode: "reply", peer_id: "peer_123" },
  message: "Delivered",
  status: "resolved",
};
const sendResult: SendBridgeTestResponse = {
  bridge_instance_id: "brg_support",
  delivery_id: "delivery_123",
  delivery_target: {
    bridge_instance_id: "brg_support",
    mode: "direct-send",
    peer_id: "peer_abc",
  },
  remote_message_id: "remote_456",
  status: "delivered",
};

const committedResultUnavailable: SendBridgeTestResponse = {
  bridge_instance_id: "brg_support",
  delivery_id: "delivery_ambiguous",
  delivery_target: {
    bridge_instance_id: "brg_support",
    mode: "direct-send",
    peer_id: "peer_abc",
  },
  error: {
    message: "The provider accepted the mutation but did not return a remote message ID.",
  },
  status: "committed_result_unavailable",
};

describe("BridgeTestDeliveryDialog", () => {
  it("Should render dry-run copy and resolved target without provider side effects", () => {
    render(
      <BridgeTestDeliveryDialog
        bridgeName="Support"
        draft={baseDraft}
        intent="dry-run"
        isPending={false}
        onDraftChange={vi.fn()}
        onOpenChange={vi.fn()}
        onSubmit={vi.fn()}
        open
        result={dryRunResult}
      />
    );

    expect(screen.getByTestId("bridge-test-delivery-dialog")).toHaveAttribute(
      "data-intent",
      "dry-run"
    );
    expect(screen.getByRole("heading", { name: "Check delivery target" })).toBeInTheDocument();
    expect(screen.getByText(/without sending a provider message/)).toBeInTheDocument();
    expect(screen.getByTestId("bridge-test-delivery-result")).toHaveTextContent("resolved");
    expect(screen.getByTestId("bridge-test-delivery-result")).toHaveTextContent("peer:peer_123");
    expect(screen.getByTestId("bridge-test-delivery-result")).toHaveTextContent(
      "Message: Delivered"
    );
    expect(screen.getByTestId("submit-test-delivery")).toBeEnabled();
  });

  it("Should update dry-run target fields and submit the check", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();

    function Wrapper() {
      const [draft, setDraft] = useState<BridgeTestDeliveryDraft>(baseDraft);
      return (
        <BridgeTestDeliveryDialog
          bridgeName="Support"
          draft={draft}
          intent="dry-run"
          isPending={false}
          onDraftChange={setDraft}
          onOpenChange={vi.fn()}
          onSubmit={onSubmit}
          open
          result={null}
        />
      );
    }

    render(<Wrapper />);
    await user.type(screen.getByTestId("test-delivery-message"), "Ping");
    await user.selectOptions(screen.getByTestId("test-delivery-mode-select"), "direct-send");
    await user.type(screen.getByTestId("test-delivery-peer-input"), "peer_abc");
    await user.type(screen.getByTestId("test-delivery-thread-input"), "thread_def");
    await user.type(screen.getByTestId("test-delivery-group-input"), "group_xyz");
    await user.click(screen.getByTestId("submit-test-delivery"));

    expect(onSubmit).toHaveBeenCalledTimes(1);
    expect(screen.getByTestId("test-delivery-message")).toHaveValue("Ping");
    expect(screen.getByTestId("test-delivery-peer-input")).toHaveValue("peer_abc");
    expect(screen.getByTestId("test-delivery-mode-select")).toHaveValue("direct-send");
  });

  it("Should require a message for send-test and submit after content is entered", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();

    function Wrapper() {
      const [draft, setDraft] = useState<BridgeTestDeliveryDraft>(baseDraft);
      return (
        <BridgeTestDeliveryDialog
          bridgeName="Support"
          draft={draft}
          intent="send-test"
          isPending={false}
          onDraftChange={setDraft}
          onOpenChange={vi.fn()}
          onSubmit={onSubmit}
          open
          result={null}
        />
      );
    }

    render(<Wrapper />);
    expect(screen.getByTestId("bridge-send-test-dialog")).toHaveAttribute(
      "data-intent",
      "send-test"
    );
    expect(screen.getByRole("heading", { name: "Send test message" })).toBeInTheDocument();
    expect(screen.getByText(/one real provider message/)).toBeInTheDocument();
    expect(screen.getByTestId("submit-send-test")).toBeDisabled();

    await user.type(screen.getByTestId("test-delivery-message"), "Operator ping");
    expect(screen.getByTestId("submit-send-test")).toBeEnabled();
    await user.click(screen.getByTestId("submit-send-test"));
    expect(onSubmit).toHaveBeenCalledTimes(1);
  });

  it("Should render send-test status, delivery id, target, and optional remote id", () => {
    render(
      <BridgeTestDeliveryDialog
        bridgeName="Support"
        draft={{ ...baseDraft, message: "Operator ping" }}
        intent="send-test"
        isPending={false}
        onDraftChange={vi.fn()}
        onOpenChange={vi.fn()}
        onSubmit={vi.fn()}
        open
        result={sendResult}
      />
    );

    const result = screen.getByTestId("bridge-send-test-result");
    expect(result).toHaveTextContent("delivered");
    expect(result).toHaveTextContent("delivery_123");
    expect(result).toHaveTextContent("peer:peer_abc");
    expect(result).toHaveTextContent("remote_456");
  });

  it("Should warn when a provider committed the mutation without returning its result", () => {
    render(
      <BridgeTestDeliveryDialog
        bridgeName="Support"
        draft={{ ...baseDraft, message: "Operator ping" }}
        intent="send-test"
        isPending={false}
        onDraftChange={vi.fn()}
        onOpenChange={vi.fn()}
        onSubmit={vi.fn()}
        open
        result={committedResultUnavailable}
      />
    );

    const result = screen.getByTestId("bridge-send-test-result");
    expect(result).toHaveTextContent("committed_result_unavailable");
    expect(result).toHaveTextContent(
      "The provider accepted the mutation but did not return a remote message ID."
    );
    expect(result).not.toHaveTextContent("Remote message ID");
    expect(screen.getByRole("status")).toHaveAttribute("data-tone", "warning");
  });

  it("Should block pending submits with intent-specific labels", () => {
    const { rerender } = render(
      <BridgeTestDeliveryDialog
        bridgeName="Support"
        draft={baseDraft}
        intent="dry-run"
        isPending
        onDraftChange={vi.fn()}
        onOpenChange={vi.fn()}
        onSubmit={vi.fn()}
        open
        result={null}
      />
    );
    expect(screen.getByTestId("submit-test-delivery")).toBeDisabled();
    expect(screen.getByTestId("submit-test-delivery")).toHaveTextContent("Checking…");

    rerender(
      <BridgeTestDeliveryDialog
        bridgeName="Support"
        draft={{ ...baseDraft, message: "Operator ping" }}
        intent="send-test"
        isPending
        onDraftChange={vi.fn()}
        onOpenChange={vi.fn()}
        onSubmit={vi.fn()}
        open
        result={null}
      />
    );
    expect(screen.getByTestId("submit-send-test")).toBeDisabled();
    expect(screen.getByTestId("submit-send-test")).toHaveTextContent("Sending…");
  });
});
