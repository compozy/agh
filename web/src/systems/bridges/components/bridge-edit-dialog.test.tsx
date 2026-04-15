import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";

import { BridgeEditDialog } from "@/systems/bridges/components/bridge-edit-dialog";
import type { BridgeProvider, BridgeUpdateDraft } from "@/systems/bridges/types";

const baseDraft: BridgeUpdateDraft = {
  deliveryDefaults: {},
  dmPolicy: "allowlist",
  displayName: "Support",
  providerConfigText: '{\n  "mode": "bot"\n}',
  routingPolicy: { include_group: true, include_peer: true, include_thread: true },
};

function makeProvider(overrides: Partial<BridgeProvider> = {}): BridgeProvider {
  return {
    config_schema: {
      schema: "provider-config",
      version: "2026-04-15",
    },
    display_name: "Telegram",
    enabled: true,
    extension_name: "ext-telegram",
    health: "healthy",
    platform: "telegram",
    secret_slots: [
      {
        description: "Bot token",
        name: "bot_token",
        required: true,
      },
    ],
    state: "active",
    ...overrides,
  };
}

describe("BridgeEditDialog", () => {
  it("blocks submission when provider config JSON is invalid", () => {
    render(
      <BridgeEditDialog
        allowProviderDefaultDmPolicy={false}
        bridgeName="Support"
        draft={{
          ...baseDraft,
          providerConfigText: "{invalid",
        }}
        isPending={false}
        onDraftChange={vi.fn()}
        onOpenChange={vi.fn()}
        onSubmit={vi.fn()}
        open
        provider={makeProvider()}
      />
    );

    expect(screen.getByTestId("bridge-edit-provider-config-error")).toHaveTextContent(
      "Provider configuration must be valid JSON."
    );
    expect(screen.getByTestId("submit-bridge-edit")).toBeDisabled();
  });

  it("updates the mutable bridge fields", async () => {
    const user = userEvent.setup();

    function Wrapper() {
      const [draft, setDraft] = useState<BridgeUpdateDraft>(baseDraft);

      return (
        <BridgeEditDialog
          allowProviderDefaultDmPolicy={false}
          bridgeName="Support"
          draft={draft}
          isPending={false}
          onDraftChange={setDraft}
          onOpenChange={vi.fn()}
          onSubmit={vi.fn()}
          open
          provider={makeProvider()}
        />
      );
    }

    render(<Wrapper />);

    await user.clear(screen.getByTestId("bridge-edit-display-name-input"));
    await user.type(screen.getByTestId("bridge-edit-display-name-input"), "Support Ops");
    await user.selectOptions(screen.getByTestId("bridge-edit-dm-policy-select"), "pairing");
    await user.click(screen.getAllByRole("switch")[0]);
    await user.selectOptions(screen.getByTestId("bridge-edit-delivery-mode-select"), "reply");
    await user.type(screen.getByTestId("bridge-edit-delivery-peer-input"), "peer_123");

    expect(screen.getByTestId("bridge-edit-display-name-input")).toHaveValue("Support Ops");
    expect(screen.getByTestId("bridge-edit-dm-policy-select")).toHaveValue("pairing");
    expect(screen.getAllByRole("switch")[0]).toHaveAttribute("aria-checked", "false");
    expect(screen.getByTestId("bridge-edit-delivery-mode-select")).toHaveValue("reply");
    expect(screen.getByTestId("bridge-edit-delivery-peer-input")).toHaveValue("peer_123");
  });
});
