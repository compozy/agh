import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import type { BridgeCreateDraft } from "@/systems/bridges/types";
import { BridgeCreateDialog } from "@/systems/bridges/components/bridge-create-dialog";

const baseDraft: BridgeCreateDraft = {
  deliveryDefaults: {},
  displayName: "",
  routingPolicy: { include_group: true, include_peer: true, include_thread: true },
  scope: "global",
  selectedProviderKey: "",
};

describe("BridgeCreateDialog", () => {
  it("renders an explicit empty state when no providers are available", () => {
    render(
      <BridgeCreateDialog
        activeWorkspaceId="ws_test"
        activeWorkspaceName="test-workspace"
        draft={baseDraft}
        isPending={false}
        onDraftChange={vi.fn()}
        onOpenChange={vi.fn()}
        onSubmit={vi.fn()}
        open
        providers={[]}
      />
    );

    expect(screen.getByTestId("bridge-provider-empty")).toHaveTextContent(
      "No bridge providers are currently available."
    );
    expect(screen.getByTestId("submit-bridge-create")).toBeDisabled();
  });
});
