import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import type { MarketplaceListing } from "../../types";
import {
  marketplaceBundlePreviewFixture,
  marketplaceDetails,
  marketplaceListings,
} from "../../mocks";
import { useMarketplaceActionController } from "../use-marketplace-action-controller";

const mocks = vi.hoisted(() => ({
  activateBundle: vi.fn(),
  installExtension: vi.fn(),
  installMCP: vi.fn(),
  installSkill: vi.fn(),
  locationAssign: vi.fn(),
  previewBundle: vi.fn(),
  previewData: undefined as unknown,
  toastError: vi.fn(),
  toastSuccess: vi.fn(),
  updateSkill: vi.fn(),
  updateExtension: vi.fn(),
}));

vi.mock("sonner", () => ({
  toast: { error: mocks.toastError, success: mocks.toastSuccess },
}));

vi.mock("../../hooks/use-marketplace-actions", () => ({
  useActivateMarketplaceBundle: () => ({
    error: null,
    isPending: false,
    mutateAsync: mocks.activateBundle,
  }),
  useInstallMarketplaceExtension: () => ({
    isPending: false,
    mutateAsync: mocks.installExtension,
  }),
  useInstallMarketplaceMCP: () => ({ mutateAsync: mocks.installMCP }),
  useInstallMarketplaceSkill: () => ({ mutateAsync: mocks.installSkill }),
  usePreviewMarketplaceBundle: () => ({
    data: mocks.previewData,
    error: null,
    isPending: false,
    mutate: mocks.previewBundle,
  }),
  useUpdateMarketplaceSkill: () => ({ mutateAsync: mocks.updateSkill }),
  useUpdateMarketplaceExtension: () => ({
    isPending: false,
    mutateAsync: mocks.updateExtension,
  }),
}));

function ActionHarness({ entry }: { entry: MarketplaceListing }) {
  const controller = useMarketplaceActionController("ws-a");
  return (
    <>
      <button onClick={() => controller.handleAction(entry)} type="button">
        Run action
      </button>
      <output aria-label="Pending entry">
        {controller.isEntryPending(entry) ? "pending" : "idle"}
      </output>
      {controller.dialogs}
    </>
  );
}

function ConcurrentActionHarness({
  first,
  second,
}: {
  first: MarketplaceListing;
  second: MarketplaceListing;
}) {
  const controller = useMarketplaceActionController("ws-a");
  return (
    <>
      <button onClick={() => controller.handleAction(first)} type="button">
        Run first
      </button>
      <button onClick={() => controller.handleAction(second)} type="button">
        Run second
      </button>
      <output aria-label="First pending">
        {controller.isEntryPending(first) ? "pending" : "idle"}
      </output>
      <output aria-label="Second pending">
        {controller.isEntryPending(second) ? "pending" : "idle"}
      </output>
      {controller.dialogs}
    </>
  );
}

function setup(entry: MarketplaceListing) {
  const client = new QueryClient({
    defaultOptions: { mutations: { retry: false }, queries: { retry: false } },
  });
  const result = render(
    <QueryClientProvider client={client}>
      <ActionHarness entry={entry} />
    </QueryClientProvider>
  );
  return { client, ...result };
}

beforeEach(() => {
  vi.clearAllMocks();
  vi.stubGlobal("location", { assign: mocks.locationAssign });
  mocks.previewData = marketplaceBundlePreviewFixture;
  mocks.activateBundle.mockResolvedValue({});
  mocks.installExtension.mockResolvedValue({
    extension: {
      daemon_running: false,
      enabled: true,
      name: "installed-extension",
      source: "marketplace",
      state: "installed",
      type: "native",
      version: "1.0.0",
    },
  });
  mocks.installMCP.mockResolvedValue({});
  mocks.installSkill.mockResolvedValue({
    skill: {
      hash: "sha256:installed-skill",
      name: "installed-skill",
      path: "/skills/installed-skill",
      registry: "agh",
      slug: "agh/installed-skill",
      status: "installed",
      version: "1.0.0",
    },
  });
  mocks.updateExtension.mockResolvedValue(undefined);
  mocks.updateSkill.mockResolvedValue({});
});

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("useMarketplaceActionController", () => {
  it("Should install and update skills while clearing pending state", async () => {
    const user = userEvent.setup();
    const { rerender } = setup(marketplaceListings.skill[1]!);

    await user.click(screen.getByRole("button", { name: "Run action" }));
    await waitFor(() =>
      expect(mocks.installSkill).toHaveBeenCalledWith({
        slug: "agh/docs-sync",
        version: "0.9.1",
      })
    );
    expect(mocks.toastSuccess).toHaveBeenCalledWith(
      "docs-sync installed",
      expect.objectContaining({ action: expect.objectContaining({ label: "Manage →" }) })
    );
    const skillToast = mocks.toastSuccess.mock.calls.find(
      call => call[0] === "docs-sync installed"
    );
    skillToast?.[1].action.onClick();
    expect(mocks.locationAssign).toHaveBeenCalledWith("/skills/installed-skill");
    expect(screen.getByRole("status", { name: "Pending entry" })).toHaveTextContent("idle");

    rerender(
      <QueryClientProvider client={new QueryClient()}>
        <ActionHarness entry={marketplaceListings.skill[2]!} />
      </QueryClientProvider>
    );
    await user.click(screen.getByRole("button", { name: "Run action" }));
    await waitFor(() => expect(mocks.updateSkill).toHaveBeenCalledWith({ name: "qa-bootstrap" }));
    expect(mocks.toastSuccess).toHaveBeenCalledWith("qa-bootstrap updated");
  });

  it("Should update extensions by installed identity through PUT semantics", async () => {
    const user = userEvent.setup();
    const verifiedUpdate: MarketplaceListing = {
      ...marketplaceListings.extension[0]!,
      installed: true,
      installed_name: "manifest-otel-bridge",
      installed_version: "0.5.0",
      name: "OpenTelemetry Bridge",
      update_available: true,
    };
    const { rerender } = setup(verifiedUpdate);

    await user.click(screen.getByRole("button", { name: "Run action" }));

    await waitFor(() =>
      expect(mocks.updateExtension).toHaveBeenCalledWith({
        body: { allow_unverified: false, version: "0.6.0" },
        name: "manifest-otel-bridge",
      })
    );
    expect(mocks.installExtension).not.toHaveBeenCalled();

    const unverifiedUpdate: MarketplaceListing = {
      ...marketplaceListings.extension[1]!,
      installed: true,
      installed_name: "manifest-slack-notify",
      installed_version: "1.0.0",
      name: "Slack Notifications",
      update_available: true,
    };
    rerender(
      <QueryClientProvider client={new QueryClient()}>
        <ActionHarness entry={unverifiedUpdate} />
      </QueryClientProvider>
    );
    await user.click(screen.getByRole("button", { name: "Run action" }));
    await user.click(await screen.findByTestId("extension-trust-confirm"));

    await waitFor(() =>
      expect(mocks.updateExtension).toHaveBeenLastCalledWith({
        body: { allow_unverified: true, version: "1.1.4" },
        name: "manifest-slack-notify",
      })
    );
    expect(mocks.installExtension).not.toHaveBeenCalled();
  });

  it("Should surface acquisition failures and always release pending state", async () => {
    const user = userEvent.setup();
    mocks.installSkill.mockRejectedValue(new Error("registry unavailable"));
    setup(marketplaceListings.skill[1]!);

    await user.click(screen.getByRole("button", { name: "Run action" }));

    await waitFor(() => expect(mocks.toastError).toHaveBeenCalledWith("registry unavailable"));
    expect(screen.getByRole("status", { name: "Pending entry" })).toHaveTextContent("idle");
  });

  it("Should enforce blocked, verified, and warning-confirm extension decisions", async () => {
    const user = userEvent.setup();
    const { rerender } = setup(marketplaceListings.extension[2]!);

    await user.click(screen.getByRole("button", { name: "Run action" }));
    expect(mocks.installExtension).not.toHaveBeenCalled();
    expect(screen.queryByTestId("extension-trust-dialog")).not.toBeInTheDocument();

    rerender(
      <QueryClientProvider client={new QueryClient()}>
        <ActionHarness entry={marketplaceListings.extension[0]!} />
      </QueryClientProvider>
    );
    await user.click(screen.getByRole("button", { name: "Run action" }));
    await waitFor(() =>
      expect(mocks.installExtension).toHaveBeenCalledWith({
        allow_unverified: false,
        slug: "agh/otel-bridge",
        version: "0.6.0",
      })
    );
    const verifiedToast = mocks.toastSuccess.mock.calls.find(
      call => call[0] === "otel-bridge installed"
    );
    verifiedToast?.[1].action.onClick();
    expect(mocks.locationAssign).toHaveBeenCalledWith("/extensions/installed-extension");

    rerender(
      <QueryClientProvider client={new QueryClient()}>
        <ActionHarness entry={marketplaceListings.extension[1]!} />
      </QueryClientProvider>
    );
    await user.click(screen.getByRole("button", { name: "Run action" }));
    expect(await screen.findByTestId("extension-trust-dialog")).toBeInTheDocument();
    mocks.installExtension.mockRejectedValueOnce(new Error("policy changed"));
    await user.click(screen.getByTestId("extension-trust-confirm"));
    expect(await screen.findByRole("alert")).toHaveTextContent("policy changed");

    await user.click(screen.getByTestId("extension-trust-confirm"));
    await waitFor(() =>
      expect(screen.queryByTestId("extension-trust-dialog")).not.toBeInTheDocument()
    );
    expect(mocks.installExtension).toHaveBeenLastCalledWith({
      allow_unverified: true,
      slug: "community/slack-notify",
      version: "1.1.4",
    });
    const warningToast = mocks.toastSuccess.mock.calls.find(
      call => call[0] === "slack-notify installed"
    );
    warningToast?.[1].action.onClick();
    expect(mocks.locationAssign).toHaveBeenLastCalledWith("/extensions/installed-extension");
  });

  it("Should track overlapping entries by full kind and entry identity", async () => {
    const user = userEvent.setup();
    let resolveSkill: ((value: unknown) => void) | undefined;
    let resolveExtension: ((value: unknown) => void) | undefined;
    mocks.installSkill.mockReturnValueOnce(
      new Promise(resolve => {
        resolveSkill = resolve;
      })
    );
    mocks.installExtension.mockReturnValueOnce(
      new Promise(resolve => {
        resolveExtension = resolve;
      })
    );
    const skill = { ...marketplaceListings.skill[1]!, entry_id: "shared-entry" };
    const extension = { ...marketplaceListings.extension[0]!, entry_id: "shared-entry" };
    const client = new QueryClient({
      defaultOptions: { mutations: { retry: false }, queries: { retry: false } },
    });
    render(
      <QueryClientProvider client={client}>
        <ConcurrentActionHarness first={skill} second={extension} />
      </QueryClientProvider>
    );

    await user.click(screen.getByRole("button", { name: "Run first" }));
    await user.click(screen.getByRole("button", { name: "Run second" }));
    await waitFor(() =>
      expect(screen.getByRole("status", { name: "First pending" })).toHaveTextContent("pending")
    );
    expect(screen.getByRole("status", { name: "Second pending" })).toHaveTextContent("pending");

    await act(async () => {
      resolveSkill?.({
        skill: {
          hash: "sha256:skill",
          name: "installed-skill",
          path: "/skills/installed-skill",
          registry: "agh",
          slug: "agh/installed-skill",
          status: "installed",
          version: "1.0.0",
        },
      });
    });
    await waitFor(() =>
      expect(screen.getByRole("status", { name: "First pending" })).toHaveTextContent("idle")
    );
    expect(screen.getByRole("status", { name: "Second pending" })).toHaveTextContent("pending");

    await act(async () => {
      resolveExtension?.({
        extension: {
          daemon_running: false,
          enabled: true,
          name: "installed-extension",
          source: "marketplace",
          state: "installed",
          type: "native",
          version: "1.0.0",
        },
      });
    });
    await waitFor(() =>
      expect(screen.getByRole("status", { name: "Second pending" })).toHaveTextContent("idle")
    );
  });

  it("Should fetch detail before opening MCP and bundle acquisition dialogs", async () => {
    const user = userEvent.setup();
    const { client, rerender } = setup(marketplaceListings.mcp[0]!);
    const fetchDetail = vi.spyOn(client, "fetchQuery");
    fetchDetail.mockResolvedValueOnce(marketplaceDetails["mcp:github"]!);

    await user.click(screen.getByRole("button", { name: "Run action" }));
    expect(await screen.findByTestId("mcp-install-dialog")).toBeInTheDocument();
    expect(fetchDetail).toHaveBeenCalledWith(
      expect.objectContaining({ queryKey: expect.arrayContaining(["mcp", "github", "ws-a"]) })
    );

    await user.keyboard("{Escape}");
    await waitFor(() => expect(screen.queryByTestId("mcp-install-dialog")).not.toBeInTheDocument());
    fetchDetail.mockResolvedValueOnce(marketplaceDetails["bundle:dep-kit"]!);
    rerender(
      <QueryClientProvider client={client}>
        <ActionHarness entry={marketplaceListings.bundle[0]!} />
      </QueryClientProvider>
    );
    await user.click(screen.getByRole("button", { name: "Run action" }));
    expect(await screen.findByTestId("bundle-activation-dialog")).toBeInTheDocument();
  });
});
