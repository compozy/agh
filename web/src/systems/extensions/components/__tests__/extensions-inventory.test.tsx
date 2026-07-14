import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { bundleActivationFixtures, extensionFixtures } from "../../mocks/fixtures";
import type { BundleActivation } from "../../types";
import { marketplaceListings } from "@/systems/marketplace/mocks/fixtures";

const mocks = vi.hoisted(() => ({
  bundleActivations: {
    data: [] as BundleActivation[] | undefined,
    error: null as Error | null,
    isLoading: false,
    refetch: vi.fn(),
  },
  deactivate: vi.fn(),
  extensionInventory: { data: [] as unknown[], error: null as Error | null, isLoading: false },
  toggle: vi.fn(),
  updateBundle: vi.fn(),
  updateExtension: vi.fn(),
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
}));

vi.mock("../../hooks/use-extensions", () => ({
  useBundleActivations: () => mocks.bundleActivations,
  useExtensionInventory: () => mocks.extensionInventory,
  useExtensionProvenance: () => ({ data: null, error: null, isLoading: false }),
}));

vi.mock("../../hooks/use-extension-actions", () => ({
  useDeactivateBundle: () => ({
    error: null,
    isPending: false,
    mutateAsync: mocks.deactivate,
  }),
  useRemoveExtension: () => ({ error: null, isPending: false, mutateAsync: vi.fn() }),
  useToggleExtension: () => ({ isPending: false, mutate: mocks.toggle, variables: undefined }),
  useUpdateBundleActivation: () => ({
    isPending: false,
    mutate: mocks.updateBundle,
    variables: undefined,
  }),
  useUpdateExtension: () => ({
    isPending: false,
    mutate: mocks.updateExtension,
    variables: undefined,
  }),
}));

import { ExtensionsInventory } from "../extensions-inventory";

beforeEach(() => {
  vi.clearAllMocks();
  mocks.deactivate.mockResolvedValue(undefined);
  mocks.bundleActivations.data = bundleActivationFixtures;
  mocks.bundleActivations.error = null;
  mocks.bundleActivations.isLoading = false;
  mocks.extensionInventory = {
    data: [
      {
        extension: extensionFixtures[0],
        listing: marketplaceListings.extension[0],
        updateAvailable: true,
      },
      {
        extension: extensionFixtures[1],
        listing: marketplaceListings.extension[1],
        updateAvailable: false,
      },
    ],
    error: null,
    isLoading: false,
  };
});

describe("ExtensionsInventory", () => {
  it("Should render Update only for a feed-joined signal and expose the lifecycle overflow", async () => {
    const user = userEvent.setup();
    render(<ExtensionsInventory tab="extensions" />);

    const updates = screen.getAllByRole("button", { name: "Update" });
    expect(updates).toHaveLength(1);
    await user.click(updates[0]!);
    expect(mocks.updateExtension).toHaveBeenCalledWith("otel-bridge");

    await user.click(screen.getByRole("button", { name: "Actions for otel-bridge" }));
    expect(await screen.findByRole("menuitem", { name: "Provenance" })).toBeInTheDocument();
    expect(screen.getByRole("menuitem", { name: "Remove…" })).toHaveClass("text-danger");

    await user.click(screen.getByRole("switch", { name: "Disable otel-bridge" }));
    expect(mocks.toggle).toHaveBeenCalledWith({ enabled: false, name: "otel-bridge" });
  });

  it("Should show bundle Update only for spec drift and send a plain reapply PATCH", async () => {
    const user = userEvent.setup();
    render(<ExtensionsInventory tab="bundles" />);

    const update = screen.getByRole("button", { name: "Update" });
    await user.click(update);
    expect(mocks.updateBundle).toHaveBeenCalledWith({
      body: {},
      id: "activation-dep-kit",
    });
    expect(screen.getByTestId("bundle-row-activation-ops-starter")).not.toHaveTextContent("Update");
  });

  it("Should open the real deactivate modal instead of using a browser confirm", async () => {
    const user = userEvent.setup();
    const confirmSpy = vi.spyOn(globalThis, "confirm");
    render(<ExtensionsInventory tab="bundles" />);

    await user.click(screen.getByRole("button", { name: "Actions for ops-starter" }));
    await user.click(await screen.findByRole("menuitem", { name: "Deactivate…" }));

    expect(screen.getByTestId("deactivate-bundle-dialog")).toBeInTheDocument();
    expect(confirmSpy).not.toHaveBeenCalled();
    await user.click(screen.getByRole("button", { name: "Deactivate" }));
    expect(mocks.deactivate).toHaveBeenCalledWith("activation-ops-starter");
  });

  it("Should render extension loading, failure, empty, and clearable search states", async () => {
    const user = userEvent.setup();
    mocks.extensionInventory = { data: [], error: null, isLoading: true };
    const { rerender } = render(<ExtensionsInventory tab="extensions" />);
    expect(screen.getByTestId("extensions-loading")).toBeInTheDocument();

    mocks.extensionInventory = {
      data: [],
      error: new Error("extensions unavailable"),
      isLoading: false,
    };
    rerender(<ExtensionsInventory tab="extensions" />);
    expect(screen.getByText("Unable to load extensions")).toBeInTheDocument();
    expect(screen.getByText("extensions unavailable")).toBeInTheDocument();

    mocks.extensionInventory = { data: [], error: null, isLoading: false };
    rerender(<ExtensionsInventory tab="extensions" />);
    expect(screen.getByText("No extensions installed")).toBeInTheDocument();

    mocks.extensionInventory = {
      data: [
        {
          extension: extensionFixtures[0],
          listing: marketplaceListings.extension[0],
          updateAvailable: true,
        },
      ],
      error: null,
      isLoading: false,
    };
    rerender(<ExtensionsInventory tab="extensions" />);
    await user.type(screen.getByRole("searchbox", { name: "Search installed extensions" }), "none");
    expect(screen.getByText("No matching extensions")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Clear search" }));
    expect(screen.getByTestId("extension-row-otel-bridge")).toBeInTheDocument();
  });

  it("Should keep inventory removal blocked when bundle activity fails", async () => {
    const user = userEvent.setup();
    mocks.bundleActivations.data = undefined;
    mocks.bundleActivations.error = new Error("bundle inventory unavailable");
    render(<ExtensionsInventory tab="extensions" />);

    await user.click(screen.getByRole("button", { name: "Actions for otel-bridge" }));
    await user.click(await screen.findByRole("menuitem", { name: "Remove…" }));
    await user.type(screen.getByLabelText("Type to confirm"), "otel-bridge");

    expect(screen.getByRole("note")).toHaveTextContent("bundle inventory unavailable");
    expect(screen.getByTestId("remove-extension-confirm")).toBeDisabled();
    await user.click(screen.getByRole("button", { name: "Retry bundle activity" }));
    expect(mocks.bundleActivations.refetch).toHaveBeenCalled();
  });

  it("Should render bundle loading, failure, empty, and clearable search states", async () => {
    const user = userEvent.setup();
    mocks.bundleActivations.data = [];
    mocks.bundleActivations.isLoading = true;
    const { rerender } = render(<ExtensionsInventory tab="bundles" />);
    expect(screen.getByTestId("extensions-loading")).toBeInTheDocument();

    mocks.bundleActivations.isLoading = false;
    mocks.bundleActivations.error = new Error("bundles unavailable");
    rerender(<ExtensionsInventory tab="bundles" />);
    expect(screen.getByText("Unable to load bundle activations")).toBeInTheDocument();
    expect(screen.getByText("bundles unavailable")).toBeInTheDocument();

    mocks.bundleActivations.error = null;
    rerender(<ExtensionsInventory tab="bundles" />);
    expect(screen.getByText("No bundles activated")).toBeInTheDocument();

    mocks.bundleActivations.data = bundleActivationFixtures;
    rerender(<ExtensionsInventory tab="bundles" />);
    await user.type(
      screen.getByRole("searchbox", { name: "Search bundle activations" }),
      "missing"
    );
    expect(screen.getByText("No matching bundles")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Clear search" }));
    expect(screen.getByTestId("bundle-row-activation-ops-starter")).toBeInTheDocument();
  });
});
