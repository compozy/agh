import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { marketplaceKindFixture, marketplaceListings } from "../../mocks";
import { MarketplaceCard } from "../marketplace-card";
import { MarketplaceEntryAction, MarketplaceEntryStatus } from "../marketplace-entry-actions";
import { MarketplaceGrid, MarketplaceGridSkeleton } from "../marketplace-grid";
import { MarketplaceKindPage } from "../marketplace-kind-page";

const mocks = vi.hoisted(() => ({
  navigate: vi.fn(),
  marketData: null as unknown,
  marketError: null as Error | null,
  marketLoading: false,
  skills: [] as unknown[],
  extensions: [] as unknown[],
  activations: [] as unknown[],
  handleAction: vi.fn(),
  handleAuthorize: vi.fn(),
  handleUpdateBundle: vi.fn(),
  isEntryPending: vi.fn(() => false),
  isEntryFlashing: vi.fn(() => false),
  setScope: vi.fn(),
  mcpServers: [] as unknown[],
}));

vi.mock("@tanstack/react-router", async () => {
  const actual =
    await vi.importActual<typeof import("@tanstack/react-router")>("@tanstack/react-router");
  return {
    ...actual,
    Link: ({
      children,
      to,
      search,
      ...props
    }: {
      children?: React.ReactNode;
      to?: string;
      search?: Record<string, unknown>;
    } & React.AnchorHTMLAttributes<HTMLAnchorElement>) => (
      <a href={`${to ?? ""}${search?.tab ? `?tab=${String(search.tab)}` : ""}`} {...props}>
        {children}
      </a>
    ),
    useNavigate: () => mocks.navigate,
  };
});

vi.mock("@/systems/workspace", () => ({
  useActiveWorkspace: () => ({ activeWorkspaceId: "ws-a" }),
}));

vi.mock("../../hooks/use-marketplace", () => ({
  useMarketplaceKind: () => ({
    data: mocks.marketData,
    error: mocks.marketError,
    isFetching: false,
    isLoading: mocks.marketLoading,
    refetch: vi.fn(),
  }),
}));

vi.mock("@/systems/skill", async () => {
  const actual = await vi.importActual<typeof import("@/systems/skill")>("@/systems/skill");
  return {
    ...actual,
    useSkills: () => ({
      data: mocks.skills,
      error: null,
      isLoading: false,
      refetch: vi.fn(),
    }),
    useRemoveSkillMarketplace: () => ({ mutateAsync: vi.fn() }),
  };
});

vi.mock("@/systems/extensions", async () => {
  const actual =
    await vi.importActual<typeof import("@/systems/extensions")>("@/systems/extensions");
  return {
    ...actual,
    useExtensionInventory: () => ({
      data: mocks.extensions,
      error: null,
      isLoading: false,
      refetch: vi.fn(),
    }),
    useBundleActivations: () => ({
      data: mocks.activations,
      error: null,
      isLoading: false,
      refetch: vi.fn(),
    }),
    useDeactivateBundle: () => ({ mutateAsync: vi.fn() }),
    useRemoveExtension: () => ({ mutateAsync: vi.fn() }),
    useToggleExtension: () => ({ mutateAsync: vi.fn() }),
  };
});

vi.mock("@/systems/settings/hooks/use-settings-collections", async () => {
  const actual = await vi.importActual<
    typeof import("@/systems/settings/hooks/use-settings-collections")
  >("@/systems/settings/hooks/use-settings-collections");
  return {
    ...actual,
    useSettingsMCPServers: () => ({
      data: { mcp_servers: mocks.mcpServers },
      error: null,
      isLoading: false,
      isFetching: false,
      refetch: vi.fn(),
    }),
  };
});

vi.mock("@/systems/settings/hooks/use-settings-mutations", async () => {
  const actual = await vi.importActual<
    typeof import("@/systems/settings/hooks/use-settings-mutations")
  >("@/systems/settings/hooks/use-settings-mutations");
  return {
    ...actual,
    useDeleteSettingsMCPServer: () => ({ mutateAsync: vi.fn() }),
  };
});

vi.mock("../use-marketplace-action-controller", () => ({
  useMarketplaceActionController: () => ({
    dialogs: null,
    handleAction: mocks.handleAction,
    handleAuthorize: mocks.handleAuthorize,
    handleDeactivate: vi.fn(),
    handleRemove: vi.fn(),
    handleToggleEnabled: vi.fn(),
    handleUpdateBundle: mocks.handleUpdateBundle,
    isAuthorizing: false,
    isEntryFlashing: mocks.isEntryFlashing,
    isEntryPending: mocks.isEntryPending,
  }),
}));

function renderKindPage(
  kind: "skill" | "mcp" | "extension" | "bundle" = "skill",
  search: { tab?: "installed"; q?: string } = {}
) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      <MarketplaceKindPage kind={kind} search={search} />
    </QueryClientProvider>
  );
}

describe("MarketplaceKindPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.marketData = marketplaceKindFixture("skill");
    mocks.marketError = null;
    mocks.marketLoading = false;
    mocks.skills = [];
    mocks.extensions = [];
    mocks.activations = [];
    mocks.mcpServers = [];
  });

  it("Should render PageHead identity, scope PillGroup, and marketplace cards", () => {
    renderKindPage("skill");
    expect(screen.getByTestId("marketplace-kind-head-skill")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Skills" })).toBeInTheDocument();
    expect(screen.getByTestId("marketplace-scope-skill")).toBeInTheDocument();
    expect(screen.getByTestId("marketplace-grid")).toHaveAttribute("data-view", "cards");
    expect(screen.getByTestId("marketplace-card-git-flow")).toBeInTheDocument();
  });

  it("Should join MCP inventory into Installed cards with transport and authorize CTA", () => {
    mocks.marketData = marketplaceKindFixture("mcp");
    mocks.mcpServers = [
      {
        name: "linear",
        transport: "sse",
        catalog_entry: "linear",
        scope: "workspace",
        workspace_id: "ws-a",
        auth: { type: "oauth2_pkce", client_id: "x", client_secret_configured: false },
        auth_status: {
          server_name: "linear",
          scope: "workspace",
          status: "needs_login",
          token_present: false,
          refreshable: true,
        },
        runtime_status: {
          configured: true,
          initialized: false,
          state: "auth_required",
          probe: "skipped",
          tool_count: 0,
        },
        source_metadata: {
          available_targets: [],
          effective_source: { kind: "workspace-config", scope: "workspace" },
          shadowed_sources: [],
        },
      },
    ];
    renderKindPage("mcp", { tab: "installed" });
    expect(screen.getByTestId("marketplace-installed-card-linear")).toBeInTheDocument();
    expect(screen.getByText("sse")).toBeInTheDocument();
    expect(screen.getByText("authorize")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Authorize" })).toBeInTheDocument();
  });

  it("Should show teaching empty for Installed scope with browse CTA", async () => {
    const user = userEvent.setup();
    renderKindPage("skill", { tab: "installed" });
    expect(screen.getByTestId("marketplace-installed-empty-skill")).toBeInTheDocument();
    expect(screen.getByText(/agh skills install/)).toBeInTheDocument();
    await user.click(screen.getByTestId("marketplace-browse-market-skill"));
    expect(mocks.navigate).toHaveBeenCalled();
  });

  it("Should link bundle-managed skills to their activation", async () => {
    const user = userEvent.setup();
    mocks.skills = [
      {
        description: "Bundled skill",
        dir: "/tmp/skills/bundled-skill",
        enabled: true,
        name: "bundled-skill",
        provenance: {
          installed_from_bundle: "ops-starter/default",
          precedence_tier: "workspace",
        },
        source: "workspace",
      },
    ];
    mocks.activations = [
      {
        bundle_name: "ops-starter",
        extension_name: "ops-extension",
        id: "activation-ops-starter",
        profile_name: "default",
        scope: "workspace",
      },
    ];
    renderKindPage("skill", { tab: "installed" });

    const trigger = screen.getByRole("button", { name: "More for bundled-skill" });
    fireEvent.pointerDown(trigger, { button: 0, pointerType: "mouse" });
    await user.click(trigger);
    expect(await screen.findByRole("menuitem", { name: "Open bundle activation" })).toHaveAttribute(
      "href",
      "/marketplace/bundles/activations/$id"
    );
  });

  it("Should preserve activation identity when updating an installed bundle", async () => {
    const user = userEvent.setup();
    mocks.marketData = marketplaceKindFixture("bundle");
    mocks.activations = [
      {
        bundle_name: "dep-kit",
        created_at: "2026-07-14T12:00:00Z",
        extension_name: "agh-foundations",
        id: "activation-dep-kit",
        profile_name: "default",
        scope: "global",
        spec_drift: true,
        updated_at: "2026-07-14T12:00:00Z",
        version: 7,
      },
    ];
    renderKindPage("bundle", { tab: "installed" });

    await user.click(screen.getByRole("button", { name: "Update" }));

    expect(mocks.handleUpdateBundle).toHaveBeenCalledWith(
      expect.objectContaining({
        activationId: "activation-dep-kit",
        activationVersion: 7,
      })
    );
  });

  it("Should render query-empty with clear search", () => {
    mocks.marketData = { ...marketplaceKindFixture("skill"), items: [], total: 0 };
    renderKindPage("skill", { q: "zzzz" });
    expect(screen.getByTestId("marketplace-query-empty-skill")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Clear search" })).toBeInTheDocument();
  });
});

describe("Marketplace cards and actions", () => {
  it("Should render Manage link to Installed scope for installed entries", () => {
    render(<MarketplaceEntryAction entry={marketplaceListings.skill[0]!} onAction={vi.fn()} />);
    expect(screen.getByRole("link", { name: /Manage git-flow/i })).toHaveAttribute(
      "href",
      "/marketplace/skills?tab=installed"
    );
  });

  it("Should label installed bundles as active", () => {
    const entry = {
      ...marketplaceListings.bundle[0]!,
      installed: true,
      update_available: false,
    };
    render(<MarketplaceEntryStatus entry={entry} />);
    expect(screen.getByText("active")).toBeInTheDocument();
  });

  it("Should render cards-only grid skeleton", () => {
    const { container } = render(<MarketplaceGridSkeleton count={2} />);
    expect(container.querySelector('[data-view="rows"]')).toBeNull();
    expect(screen.getByRole("status")).toBeInTheDocument();
  });

  it("Should render marketplace card linking to API-kind detail", () => {
    render(<MarketplaceCard entry={marketplaceListings.skill[1]!} onAction={vi.fn()} />);
    expect(screen.getByTestId("marketplace-card-docs-sync")).toBeInTheDocument();
  });

  it("Should render marketplace grid of cards", () => {
    render(<MarketplaceGrid entries={marketplaceListings.skill.slice(0, 2)} onAction={vi.fn()} />);
    expect(screen.getByTestId("marketplace-grid")).toHaveAttribute("data-view", "cards");
  });
});
