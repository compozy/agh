import { fireEvent, render, screen } from "@testing-library/react";
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

function readDialogWidth(): string {
  const dialog = screen.getByTestId("bridge-create-dialog");
  return dialog.className;
}

describe("BridgeCreateDialog", () => {
  it("Should anchor the dialog to the 880 px modal width token (--width-modal-lg)", () => {
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

    expect(readDialogWidth()).toContain("w-(--width-modal-lg)");
    expect(readDialogWidth()).toContain("sm:max-w-(--width-modal-lg)");
  });

  it("Should render an empty state on the provider step when no providers are available", () => {
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
    expect(screen.getByTestId("bridge-wizard-next")).toBeDisabled();
  });

  it("Should advance through provider → runtime → delivery steps and reveal the create button", async () => {
    const user = userEvent.setup();

    function Wrapper() {
      const [draft, setDraft] = useState<BridgeCreateDraft>({
        ...baseDraft,
        selectedProviderKey: "ext-telegram::telegram",
        displayName: "Telegram",
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
          providers={[makeProvider()]}
        />
      );
    }

    render(<Wrapper />);

    expect(screen.getByTestId("bridge-wizard-stepper")).toBeInTheDocument();
    expect(screen.getByTestId("bridge-wizard-progress")).toHaveTextContent("Step 1 of 3");

    await user.click(screen.getByTestId("bridge-wizard-next"));
    expect(screen.getByTestId("bridge-wizard-progress")).toHaveTextContent("Step 2 of 3");
    expect(screen.getByTestId("bridge-display-name-input")).toHaveValue("Telegram");

    await user.click(screen.getByTestId("bridge-wizard-next"));
    expect(screen.getByTestId("bridge-wizard-progress")).toHaveTextContent("Step 3 of 3");
    expect(screen.getByTestId("submit-bridge-create")).toBeInTheDocument();
  });

  it("Should select a provider card on click and update provider runtime metadata when the runtime step is revealed", async () => {
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
              config_schema: { schema: "provider-config", version: "2026-04-16" },
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

    await user.click(screen.getByTestId("bridge-provider-card-ext-slack::slack"));
    await user.click(screen.getByTestId("bridge-wizard-next"));

    expect(screen.getByTestId("bridge-provider-config-schema")).toHaveTextContent(
      "provider-config · v2026-04-16"
    );
    expect(screen.getByTestId("bridge-provider-secret-slots")).toHaveTextContent("signing_secret");
  });

  it("Should block the wizard on the runtime step when provider config is invalid JSON", async () => {
    const user = userEvent.setup();

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

    await user.click(screen.getByTestId("bridge-wizard-next"));

    expect(screen.getByTestId("bridge-provider-config-error")).toHaveTextContent(
      "Provider configuration must be valid JSON."
    );
    expect(screen.getByTestId("bridge-wizard-next")).toBeDisabled();
  });

  it("Should preserve delivery edits across step navigation and surface the pending state on submit", async () => {
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

    await user.click(screen.getByTestId("bridge-wizard-next"));
    await user.click(screen.getByTestId("bridge-wizard-next"));

    fireEvent.change(screen.getByTestId("bridge-delivery-mode-select"), {
      target: { value: "direct-send" },
    });
    fireEvent.change(screen.getByTestId("bridge-delivery-peer-input"), {
      target: { value: "peer_123" },
    });
    fireEvent.change(screen.getByTestId("bridge-delivery-thread-input"), {
      target: { value: "thread_123" },
    });
    fireEvent.change(screen.getByTestId("bridge-delivery-group-input"), {
      target: { value: "group_123" },
    });
    fireEvent.click(screen.getAllByRole("switch")[1]);

    expect(screen.getByTestId("bridge-delivery-mode-select")).toHaveValue("direct-send");
    expect(screen.getByTestId("bridge-delivery-peer-input")).toHaveValue("peer_123");
    expect(screen.getByTestId("bridge-delivery-thread-input")).toHaveValue("thread_123");
    expect(screen.getByTestId("bridge-delivery-group-input")).toHaveValue("group_123");
    expect(screen.getAllByRole("switch")[1]).toHaveAttribute("aria-checked", "false");
    expect(screen.getByTestId("submit-bridge-create")).toHaveTextContent("Creating…");

    fireEvent.click(screen.getByTestId("bridge-wizard-cancel"));
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });
});
