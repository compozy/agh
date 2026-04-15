import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";

import { BridgeCreateDialog } from "@/systems/bridges/components/bridge-create-dialog";
import type { BridgeCreateDraft, BridgeProvider } from "@/systems/bridges/types";

const baseDraft: BridgeCreateDraft = {
  deliveryDefaults: {},
  dmPolicy: "",
  displayName: "",
  providerConfigText: "",
  routingPolicy: { include_group: true, include_peer: true, include_thread: true },
  scope: "global",
  selectedProviderKey: "",
};

function makeProvider(overrides: Partial<BridgeProvider> = {}): BridgeProvider {
  return {
    config_schema: {
      schema: "provider-config",
      version: "2026-04-15",
    },
    description: "Provider-specific runtime settings",
    display_name: "Telegram",
    enabled: true,
    extension_name: "ext-telegram",
    health: "healthy",
    platform: "telegram",
    secret_slots: [
      {
        description: "Bot API token",
        name: "bot_token",
        required: true,
      },
    ],
    state: "active",
    ...overrides,
  };
}

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

  it("updates provider hints without clobbering routing state", async () => {
    const user = userEvent.setup();

    function Wrapper() {
      const [draft, setDraft] = useState<BridgeCreateDraft>({
        ...baseDraft,
        selectedProviderKey: "ext-telegram::telegram",
      });

      return (
        <BridgeCreateDialog
          activeWorkspaceId="ws_test"
          activeWorkspaceName="test-workspace"
          draft={draft}
          isPending={false}
          onDraftChange={setDraft}
          onOpenChange={vi.fn()}
          onSubmit={vi.fn()}
          open
          providers={[
            makeProvider(),
            makeProvider({
              config_schema: {
                schema: "provider-config",
                version: "2026-04-16",
              },
              display_name: "Slack",
              extension_name: "ext-slack",
              platform: "slack",
              secret_slots: [
                {
                  description: "Webhook signing secret",
                  name: "signing_secret",
                  required: true,
                },
              ],
            }),
          ]}
        />
      );
    }

    render(<Wrapper />);

    const switches = screen.getAllByRole("switch");
    await user.click(switches[0]);

    expect(switches[0]).toHaveAttribute("aria-checked", "false");

    await user.click(screen.getByTestId("bridge-provider-card-ext-slack::slack"));

    expect(screen.getByTestId("bridge-provider-config-schema")).toHaveTextContent(
      "provider-config · v2026-04-16"
    );
    expect(screen.getByTestId("bridge-provider-secret-slots")).toHaveTextContent("signing_secret");
    expect(screen.getAllByRole("switch")[0]).toHaveAttribute("aria-checked", "false");
  });

  it("blocks submission when provider config is invalid json", async () => {
    render(
      <BridgeCreateDialog
        activeWorkspaceId="ws_test"
        activeWorkspaceName="test-workspace"
        draft={{
          ...baseDraft,
          displayName: "Telegram",
          providerConfigText: "{invalid",
          selectedProviderKey: "ext-telegram::telegram",
        }}
        isPending={false}
        onDraftChange={vi.fn()}
        onOpenChange={vi.fn()}
        onSubmit={vi.fn()}
        open
        providers={[makeProvider()]}
      />
    );

    expect(screen.getByTestId("bridge-provider-config-error")).toHaveTextContent(
      "Provider configuration must be valid JSON."
    );
    expect(screen.getByTestId("submit-bridge-create")).toBeDisabled();
  });

  it("updates the generic form controls and supports pending plus cancel states", async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();

    function Wrapper() {
      const [draft, setDraft] = useState<BridgeCreateDraft>({
        ...baseDraft,
        displayName: "Telegram",
        selectedProviderKey: "ext-telegram::telegram",
      });

      return (
        <BridgeCreateDialog
          activeWorkspaceId="ws_test"
          activeWorkspaceName="test-workspace"
          draft={draft}
          isPending
          onDraftChange={setDraft}
          onOpenChange={onOpenChange}
          onSubmit={vi.fn()}
          open
          providers={[makeProvider()]}
        />
      );
    }

    render(<Wrapper />);

    await user.clear(screen.getByTestId("bridge-display-name-input"));
    await user.type(screen.getByTestId("bridge-display-name-input"), "Ops bridge");
    await user.selectOptions(screen.getByTestId("bridge-scope-select"), "global");
    await user.click(screen.getAllByRole("switch")[1]);
    await user.selectOptions(screen.getByTestId("bridge-delivery-mode-select"), "direct-send");
    await user.type(screen.getByTestId("bridge-delivery-peer-input"), "peer_123");
    await user.type(screen.getByTestId("bridge-delivery-thread-input"), "thread_123");
    await user.type(screen.getByTestId("bridge-delivery-group-input"), "group_123");
    await user.click(screen.getByRole("button", { name: "Cancel" }));

    expect(screen.getByTestId("bridge-display-name-input")).toHaveValue("Ops bridge");
    expect(screen.getByTestId("bridge-scope-select")).toHaveValue("global");
    expect(screen.getAllByRole("switch")[1]).toHaveAttribute("aria-checked", "false");
    expect(screen.getByTestId("bridge-delivery-mode-select")).toHaveValue("direct-send");
    expect(screen.getByTestId("bridge-delivery-peer-input")).toHaveValue("peer_123");
    expect(screen.getByTestId("bridge-delivery-thread-input")).toHaveValue("thread_123");
    expect(screen.getByTestId("bridge-delivery-group-input")).toHaveValue("group_123");
    expect(screen.getByTestId("submit-bridge-create")).toHaveTextContent("Creating…");
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });
});
