import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";

import { BridgeTestDeliveryDialog } from "@/systems/bridges/components/bridge-test-delivery-dialog";
import type { BridgeTestDeliveryDraft, TestBridgeDeliveryResponse } from "@/systems/bridges/types";

const baseDraft: BridgeTestDeliveryDraft = {
  message: "",
  target: {},
};

const baseResult: TestBridgeDeliveryResponse = {
  delivery_target: {
    bridge_instance_id: "brg_support",
    mode: "reply",
    peer_id: "peer_123",
  },
  message: "Delivered",
  status: "resolved",
};

describe("BridgeTestDeliveryDialog", () => {
  it("renders the dialog with a bridge name in the description", () => {
    render(
      <BridgeTestDeliveryDialog
        bridgeName="Support"
        draft={baseDraft}
        isPending={false}
        onDraftChange={vi.fn()}
        onOpenChange={vi.fn()}
        onSubmit={vi.fn()}
        open
        result={null}
      />
    );

    expect(screen.getByTestId("bridge-test-delivery-dialog")).toBeInTheDocument();
    expect(screen.getByText(/for Support/)).toBeInTheDocument();
  });

  it("renders the resolved target section when a result is present", () => {
    render(
      <BridgeTestDeliveryDialog
        bridgeName="Support"
        draft={baseDraft}
        isPending={false}
        onDraftChange={vi.fn()}
        onOpenChange={vi.fn()}
        onSubmit={vi.fn()}
        open
        result={baseResult}
      />
    );

    expect(screen.getByTestId("bridge-test-delivery-result")).toBeInTheDocument();
    expect(screen.getByText(/peer:peer_123/)).toBeInTheDocument();
    expect(screen.getByText(/Message: Delivered/)).toBeInTheDocument();
  });

  it("updates target fields and submits the resolve action", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();

    function Wrapper() {
      const [draft, setDraft] = useState<BridgeTestDeliveryDraft>(baseDraft);
      return (
        <BridgeTestDeliveryDialog
          bridgeName="Support"
          draft={draft}
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

  it("shows a pending submit label and blocks onSubmit when isPending", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();

    render(
      <BridgeTestDeliveryDialog
        bridgeName="Support"
        draft={baseDraft}
        isPending
        onDraftChange={vi.fn()}
        onOpenChange={vi.fn()}
        onSubmit={onSubmit}
        open
        result={null}
      />
    );

    const submit = screen.getByTestId("submit-test-delivery");
    expect(submit).toBeDisabled();
    expect(submit).toHaveTextContent(/Resolving/);

    await user.click(submit);
    expect(onSubmit).not.toHaveBeenCalled();
  });
});
