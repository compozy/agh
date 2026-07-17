import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import {
  bundleActivationFixtures,
  extensionFixtures,
  extensionProvenanceFixtures,
} from "../../mocks/fixtures";
import type { BundleActivation, InstalledExtensionView } from "../../types";
import { marketplaceListings } from "@/systems/marketplace/mocks/fixtures";

const mocks = vi.hoisted(() => ({
  bundles: {
    data: [] as BundleActivation[] | undefined,
    error: null as Error | null,
    isLoading: false,
    refetch: vi.fn(),
  },
  detail: {
    data: null as InstalledExtensionView | null,
    error: null as Error | null,
    isLoading: false,
  },
  navigate: vi.fn(),
  provenance: {
    data: null as (typeof extensionProvenanceFixtures)[string] | null,
    error: null,
    isLoading: false,
  },
  remove: vi.fn(),
  toggle: vi.fn(),
  update: vi.fn(),
}));

vi.mock("@tanstack/react-router", () => ({
  Link: ({
    children,
    params,
    to,
    ...props
  }: {
    children?: ReactNode;
    params?: Record<string, string>;
    to: string;
  }) => {
    const href = Object.entries(params ?? {}).reduce(
      (path, [key, value]) => path.replace(`$${key}`, value),
      to
    );
    return (
      <a href={href} {...props}>
        {children}
      </a>
    );
  },
  useNavigate: () => mocks.navigate,
}));

vi.mock("../../hooks/use-extensions", () => ({
  useBundleActivations: () => mocks.bundles,
  useExtensionDetail: () => mocks.detail,
  useExtensionProvenance: () => mocks.provenance,
}));

vi.mock("../../hooks/use-extension-actions", () => ({
  useRemoveExtension: () => ({ error: null, isPending: false, mutateAsync: mocks.remove }),
  useToggleExtension: () => ({ isPending: false, mutate: mocks.toggle }),
  useUpdateExtension: () => ({ isPending: false, mutate: mocks.update }),
}));

import { ExtensionDetail } from "../extension-detail";

beforeEach(() => {
  vi.clearAllMocks();
  mocks.bundles.data = bundleActivationFixtures;
  mocks.bundles.error = null;
  mocks.bundles.isLoading = false;
  mocks.detail.data = {
    extension: extensionFixtures[1]!,
    listing: marketplaceListings.extension[1]!,
    updateAvailable: false,
  };
  mocks.detail.error = null;
  mocks.detail.isLoading = false;
  mocks.provenance.data = extensionProvenanceFixtures["slack-notify"]!;
  mocks.remove.mockResolvedValue(undefined);
});

describe("ExtensionDetail", () => {
  it("Should render missing environment, danger error, and active bundle navigation", () => {
    render(<ExtensionDetail name="slack-notify" />);

    expect(screen.getAllByText("SLACK_BOT_TOKEN").length).toBeGreaterThan(0);
    expect(screen.getByText("missing")).toBeInTheDocument();
    expect(screen.getByTestId("extension-last-error")).toHaveTextContent(
      "SLACK_BOT_TOKEN is not configured"
    );
    expect(screen.getByRole("button", { name: "Open active bundle →" })).toHaveAttribute(
      "href",
      "/extensions/bundles/activation-ops-starter"
    );
  });

  it("Should replace stale extension content with loading, failure, and not-found states", () => {
    mocks.detail.data = null;
    mocks.detail.isLoading = true;
    const { rerender } = render(<ExtensionDetail name="missing" />);
    expect(screen.queryByTestId("extension-detail")).not.toBeInTheDocument();
    expect(screen.getByRole("status")).toHaveTextContent("Loading extension");

    mocks.detail.isLoading = false;
    mocks.detail.error = new Error("extension inventory unavailable");
    rerender(<ExtensionDetail name="missing" />);
    expect(screen.getByText("Unable to load extension")).toBeInTheDocument();
    expect(screen.getByText("extension inventory unavailable")).toBeInTheDocument();

    mocks.detail.error = null;
    rerender(<ExtensionDetail name="missing" />);
    expect(screen.getByText("Extension not found")).toBeInTheDocument();
    expect(screen.getByText("No installed extension is named missing.")).toBeInTheDocument();
  });

  it("Should keep bundle activity unresolved until the dependency query succeeds", async () => {
    const user = userEvent.setup();
    mocks.bundles.data = undefined;
    mocks.bundles.isLoading = true;
    mocks.detail.data = {
      extension: extensionFixtures[0]!,
      listing: marketplaceListings.extension[0]!,
      updateAvailable: false,
    };
    const { rerender } = render(<ExtensionDetail name="otel-bridge" />);

    expect(screen.getByRole("status", { name: "Checking active bundles" })).toBeInTheDocument();
    expect(screen.queryByText("inactive")).not.toBeInTheDocument();

    mocks.bundles.isLoading = false;
    mocks.bundles.error = new Error("bundle inventory unavailable");
    rerender(<ExtensionDetail name="otel-bridge" />);
    expect(screen.getByText("Bundle activity could not be loaded")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Retry bundle activity" }));
    expect(mocks.bundles.refetch).toHaveBeenCalled();

    await user.click(screen.getByRole("button", { name: "Actions for otel-bridge" }));
    await user.click(await screen.findByRole("menuitem", { name: "Remove…" }));
    await user.type(screen.getByLabelText("Type to confirm"), "otel-bridge");
    expect(screen.getByTestId("remove-extension-confirm")).toBeDisabled();
  });

  it("Should wire healthy extension toggle, update, provenance, and removal controls", async () => {
    const user = userEvent.setup();
    mocks.bundles.data = [];
    mocks.detail.data = {
      extension: extensionFixtures[0]!,
      listing: marketplaceListings.extension[0]!,
      updateAvailable: true,
    };
    mocks.provenance.data = extensionProvenanceFixtures["otel-bridge"]!;
    render(<ExtensionDetail name="otel-bridge" />);

    await user.click(screen.getByRole("switch", { name: "Disable otel-bridge" }));
    expect(mocks.toggle).toHaveBeenCalledWith({ enabled: false, name: "otel-bridge" });
    await user.click(screen.getByRole("button", { name: "Update" }));
    expect(mocks.update).toHaveBeenCalledWith("otel-bridge");
    expect(screen.getByRole("button", { name: "View in marketplace →" })).toHaveAttribute(
      "href",
      "/marketplace/extension/otel-bridge"
    );

    await user.click(screen.getByRole("button", { name: "Actions for otel-bridge" }));
    await user.click(await screen.findByRole("menuitem", { name: "Provenance" }));
    expect(screen.getByTestId("extension-provenance-content")).toHaveTextContent(
      "sha256:otel-bridge-060"
    );
    await user.click(screen.getByRole("button", { name: "Done" }));

    await user.click(screen.getByRole("button", { name: "Actions for otel-bridge" }));
    await user.click(await screen.findByRole("menuitem", { name: "Remove…" }));
    await user.type(screen.getByLabelText("Type to confirm"), "otel-bridge");
    await user.click(screen.getByTestId("remove-extension-confirm"));
    await waitFor(() => expect(mocks.remove).toHaveBeenCalledWith("otel-bridge"));
    expect(mocks.navigate).toHaveBeenCalledWith({ to: "/extensions" });
  });

  it("Should render honest empty fallbacks and preserve zero uptime", () => {
    mocks.bundles.data = [];
    mocks.detail.data = {
      extension: {
        ...extensionFixtures[0]!,
        actions: undefined,
        bundles: undefined,
        capabilities: undefined,
        diagnostics: undefined,
        health: undefined,
        health_message: undefined,
        missing_env: undefined,
        pid: undefined,
        provenance: undefined,
        requires_env: undefined,
        trust: undefined,
        uptime_seconds: 0,
      },
      listing: null,
      updateAvailable: false,
    };
    const { container } = render(<ExtensionDetail name="otel-bridge" />);

    const meta = container.querySelector<HTMLElement>('[data-slot="detail-header-meta"]');
    expect(meta).not.toBeNull();
    expect(within(meta!).getAllByText("marketplace")).toHaveLength(1);
    expect(screen.getByText("Uptime").parentElement).toHaveTextContent("0s");
    expect(
      screen.getByText("No catalog description is available for this installed extension.")
    ).toBeInTheDocument();
    expect(screen.getAllByText("None registered.")).toHaveLength(2);
    expect(screen.getByText("No environment variables required.")).toBeInTheDocument();
    expect(screen.getByText("No diagnostics.")).toBeInTheDocument();
    expect(screen.getByText("None.")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "View in marketplace →" })).not.toBeInTheDocument();
  });

  it("Should render runtime rails with human uptime and preserved provenance values", () => {
    mocks.bundles.data = [];
    mocks.detail.data = {
      extension: extensionFixtures[0]!,
      listing: marketplaceListings.extension[0]!,
      updateAvailable: false,
    };
    mocks.provenance.data = extensionProvenanceFixtures["otel-bridge"]!;
    render(<ExtensionDetail name="otel-bridge" />);

    const runtime = screen.getByText("Runtime").closest("section");
    expect(runtime).not.toBeNull();
    expect(within(runtime!).getByText("running")).toBeInTheDocument();
    expect(within(runtime!).getByText("4812")).toBeInTheDocument();
    expect(within(runtime!).getByText("5h 7m")).toBeInTheDocument();

    const provenance = screen.getByText("Provenance").closest("section");
    expect(provenance).not.toBeNull();
    expect(within(provenance!).getByText("marketplace_registry")).toBeInTheDocument();
    expect(
      within(provenance!).getByText("https://registry.agh.network/agh/otel-bridge")
    ).toBeInTheDocument();
    expect(within(provenance!).getAllByText("verified").length).toBeGreaterThan(0);
    expect(within(provenance!).getByText("sha256:otel-bridge-060")).toBeInTheDocument();
  });

  it("Should surface last-error content and error-severity diagnostics", () => {
    render(<ExtensionDetail name="slack-notify" />);

    const lastError = screen.getByTestId("extension-last-error");
    expect(within(lastError).getByText("error")).toBeInTheDocument();
    expect(within(lastError).getByText("SLACK_BOT_TOKEN is not configured")).toBeInTheDocument();

    expect(screen.getByText("Required environment is missing")).toBeInTheDocument();
    expect(
      screen.getByText("SLACK_BOT_TOKEN is required before this extension can start.")
    ).toBeInTheDocument();

    const runtime = screen.getByText("Runtime").closest("section");
    expect(runtime).not.toBeNull();
    expect(within(runtime!).getByText("stopped")).toBeInTheDocument();
    expect(within(runtime!).getByText("SLACK_BOT_TOKEN is not configured")).toBeInTheDocument();
  });
});
