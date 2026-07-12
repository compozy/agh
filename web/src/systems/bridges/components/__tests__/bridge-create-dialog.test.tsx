import { act, render, screen, waitFor, within } from "@testing-library/react";
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

function createDeferredRequest() {
  let resolve!: () => void;
  const promise = new Promise<void>(resolvePromise => {
    resolve = resolvePromise;
  });
  return { promise, resolve };
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

  it("Should block Cancel and dialog dismissal while the create request is pending", async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();
    const onSubmit = vi.fn();
    const request = createDeferredRequest();

    function Wrapper() {
      const [draft, setDraft] = useState<BridgeCreateDraft>({
        ...baseDraft,
        displayName: "Telegram",
        selectedProviderKey: "ext-telegram::telegram",
      });
      const [isPending, setIsPending] = useState(false);
      const [open, setOpen] = useState(true);

      const handleSubmit = async () => {
        onSubmit();
        setIsPending(true);
        await request.promise;
        setIsPending(false);
      };

      return (
        <BridgeCreateDialog
          activeWorkspaceId="ws_test"
          activeWorkspaceName="test-workspace"
          draft={draft}
          isPending={isPending}
          onDraftChange={setDraft}
          onOpenChange={next => {
            onOpenChange(next);
            setOpen(next);
          }}
          onSubmit={handleSubmit}
          open={open}
          providers={[makeProvider()]}
        />
      );
    }

    render(<Wrapper />);

    await user.click(screen.getByTestId("bridge-wizard-next"));
    await user.click(screen.getByTestId("bridge-wizard-next"));
    await user.click(screen.getByTestId("submit-bridge-create"));

    expect(onSubmit).toHaveBeenCalledTimes(1);
    expect(screen.getByTestId("submit-bridge-create")).toHaveTextContent("Creating…");
    expect(screen.getByTestId("bridge-wizard-cancel")).toBeDisabled();

    await user.keyboard("{Escape}");
    expect(screen.getByTestId("bridge-create-dialog")).toBeInTheDocument();
    expect(onOpenChange).not.toHaveBeenCalled();

    await user.click(screen.getByTestId("bridge-wizard-cancel"));
    expect(onOpenChange).not.toHaveBeenCalled();

    await act(async () => {
      request.resolve();
      await request.promise;
    });
    await waitFor(() => expect(screen.getByTestId("bridge-wizard-cancel")).toBeEnabled());

    await user.click(screen.getByTestId("bridge-wizard-cancel"));
    expect(onOpenChange).toHaveBeenCalledWith(false);
    expect(screen.queryByTestId("bridge-create-dialog")).not.toBeInTheDocument();
  });

  it("Should write a complete progress override and remove it when provider default is restored", async () => {
    const user = userEvent.setup();

    function Wrapper() {
      const [draft, setDraft] = useState<BridgeCreateDraft>({
        ...baseDraft,
        displayName: "Telegram",
        selectedProviderKey: "ext-telegram::telegram",
      });

      return (
        <>
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
          <output data-testid="bridge-create-progress-draft">
            {draft.deliveryDefaults.progress
              ? JSON.stringify(draft.deliveryDefaults.progress)
              : "provider-default"}
          </output>
        </>
      );
    }

    render(<Wrapper />);
    await user.click(screen.getByTestId("bridge-wizard-next"));
    await user.click(screen.getByTestId("bridge-wizard-next"));

    await user.selectOptions(screen.getByTestId("bridge-delivery-progress-mode-select"), "all");
    const groupingSelect = screen.getByTestId("bridge-delivery-progress-grouping-select");
    expect(within(groupingSelect).queryByRole("option", { name: "Provider default" })).toBeNull();
    expect(groupingSelect.querySelector('option[value=""]')).toBeNull();
    await user.selectOptions(groupingSelect, "separate");
    await user.selectOptions(screen.getByTestId("bridge-delivery-progress-typing-select"), "true");
    await user.selectOptions(
      screen.getByTestId("bridge-delivery-progress-reactions-select"),
      "false"
    );

    expect(
      JSON.parse(screen.getByTestId("bridge-create-progress-draft").textContent ?? "")
    ).toEqual({
      grouping: "separate",
      reactions: false,
      tool_progress: "all",
      typing: true,
    });

    await user.selectOptions(screen.getByTestId("bridge-delivery-progress-mode-select"), "");

    expect(screen.getByTestId("bridge-create-progress-draft")).toHaveTextContent(
      "provider-default"
    );
    expect(screen.getByTestId("bridge-delivery-progress-grouping-select")).toBeDisabled();
    expect(screen.getByTestId("bridge-delivery-progress-typing-select")).toBeDisabled();
    expect(screen.getByTestId("bridge-delivery-progress-reactions-select")).toBeDisabled();
  });

  it("Should advertise a post-create manifest only when the selected provider supports it", () => {
    const slackProvider = makeProvider({
      display_name: "Slack",
      extension_name: "ext-slack",
      platform: "slack",
    });
    const props = {
      activeWorkspaceId: "ws_test",
      activeWorkspaceName: "test-workspace",
      draft: {
        ...baseDraft,
        displayName: "Slack",
        selectedProviderKey: "ext-slack::slack",
      },
      isPending: false,
      onDraftChange: vi.fn(),
      onOpenChange: vi.fn(),
      onSubmit: vi.fn(),
      open: true,
      providers: [slackProvider],
    };

    const { rerender } = render(<BridgeCreateDialog {...props} supportsManifest />);

    expect(screen.getByTestId("bridge-manifest-precreate-hint")).toHaveTextContent(
      "Slack manifest available after creation"
    );
    expect(screen.queryByTestId("bridge-manifest-json")).not.toBeInTheDocument();

    rerender(
      <BridgeCreateDialog
        {...props}
        manifestState={{
          bridgeId: "brg_slack",
          isLoading: false,
          manifestJSON: '{"name":"Slack"}',
          onOpenBridge: vi.fn(),
          onRetry: vi.fn(),
        }}
        supportsManifest={false}
      />
    );

    expect(screen.queryByTestId("bridge-manifest-precreate-hint")).not.toBeInTheDocument();
    expect(screen.queryByTestId("bridge-manifest-handoff")).not.toBeInTheDocument();
  });

  it("Should render the committed Slack manifest with copy and dashboard handoff", async () => {
    const user = userEvent.setup();
    const manifestJSON = JSON.stringify({ display_information: { name: "AGH Support" } }, null, 2);
    const onOpenBridge = vi.fn();
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: { writeText },
    });

    render(
      <BridgeCreateDialog
        draft={{
          ...baseDraft,
          displayName: "Slack",
          selectedProviderKey: "ext-slack::slack",
        }}
        isPending={false}
        manifestState={{
          bridgeId: "brg_slack",
          isLoading: false,
          manifestJSON,
          onOpenBridge,
          onRetry: vi.fn(),
        }}
        onDraftChange={vi.fn()}
        onOpenChange={vi.fn()}
        onSubmit={vi.fn()}
        open
        providers={[
          makeProvider({
            display_name: "Slack",
            extension_name: "ext-slack",
            platform: "slack",
          }),
        ]}
        supportsManifest
      />
    );

    expect(screen.getByRole("heading", { name: "Set up Slack app" })).toBeInTheDocument();
    expect(screen.getByTestId("bridge-manifest-json")).toHaveTextContent("AGH Support");
    expect(screen.getByRole("link", { name: "Open Slack app dashboard" })).toHaveAttribute(
      "href",
      "https://api.slack.com/apps"
    );
    expect(screen.queryByTestId("bridge-wizard-back")).not.toBeInTheDocument();
    expect(screen.queryByTestId("bridge-wizard-cancel")).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Copy Slack app manifest" }));
    expect(writeText).toHaveBeenCalledWith(manifestJSON);

    await user.click(screen.getByTestId("bridge-manifest-open-bridge"));
    expect(onOpenBridge).toHaveBeenCalledTimes(1);
  });

  it("Should preserve recovery actions when the committed manifest fails", async () => {
    const user = userEvent.setup();
    const onRetry = vi.fn();
    const onOpenBridge = vi.fn();

    render(
      <BridgeCreateDialog
        draft={{
          ...baseDraft,
          displayName: "Slack",
          selectedProviderKey: "ext-slack::slack",
        }}
        isPending={false}
        manifestState={{
          bridgeId: "brg_slack",
          error: "Saved webhook URL is invalid.",
          isLoading: false,
          onOpenBridge,
          onRetry,
        }}
        onDraftChange={vi.fn()}
        onOpenChange={vi.fn()}
        onSubmit={vi.fn()}
        open
        providers={[
          makeProvider({
            display_name: "Slack",
            extension_name: "ext-slack",
            platform: "slack",
          }),
        ]}
        supportsManifest
      />
    );

    expect(screen.getByTestId("bridge-manifest-error")).toHaveTextContent(
      "Saved webhook URL is invalid."
    );
    expect(screen.queryByTestId("bridge-manifest-json")).not.toBeInTheDocument();
    expect(
      screen.queryByRole("link", { name: "Open Slack app dashboard" })
    ).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Retry" }));
    await user.click(screen.getByTestId("bridge-manifest-open-bridge"));

    expect(onRetry).toHaveBeenCalledTimes(1);
    expect(onOpenBridge).toHaveBeenCalledTimes(1);
  });
});
