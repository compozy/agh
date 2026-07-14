import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type {
  MarketplaceEntryResponse,
  MarketplaceSearchResponse,
  MCPInstallResponse,
} from "../../types";
import {
  marketplaceBundlePreviewFixture,
  marketplaceDetails,
  marketplaceListings,
  marketplaceSearchFixture,
} from "../../mocks";
import { BundleActivationDialog } from "../bundle-activation-dialog";
import { ExtensionTrustDialog } from "../extension-trust-dialog";
import { MarketplaceCard } from "../marketplace-card";
import { MarketplaceDetail } from "../marketplace-detail";
import { MarketplaceKindView } from "../marketplace-kind-view";
import { MarketplaceLanding } from "../marketplace-landing";
import { MCPInstallDialog } from "../mcp-install-dialog";
import { buildMCPInstallRequest } from "../mcp-install-model";

const mocks = vi.hoisted(() => ({
  activate: vi.fn(),
  activateError: null as Error | null,
  activateReset: vi.fn(),
  createSecret: vi.fn(),
  preview: vi.fn(),
  previewData: undefined as unknown,
  previewError: null as Error | null,
  vaultError: null as Error | null,
  vaultSecrets: [] as Array<{
    created_at: string;
    kind: string;
    namespace: string;
    present: boolean;
    ref: string;
    updated_at: string;
  }>,
}));

vi.mock("@tanstack/react-router", () => ({
  Link: ({ children, ...props }: { children: ReactNode; [key: string]: unknown }) => {
    const { params: _params, search, to, ...anchorProps } = props;
    const searchParams = new URLSearchParams();
    if (search && typeof search === "object") {
      for (const [key, value] of Object.entries(search)) {
        if (value !== undefined) searchParams.set(key, String(value));
      }
    }
    const path = typeof to === "string" ? to : "#marketplace";
    const href = searchParams.size ? `${path}?${searchParams.toString()}` : path;
    return (
      <a href={href} {...anchorProps}>
        {children}
      </a>
    );
  },
}));

vi.mock("@/systems/vault", () => ({
  usePutVaultSecret: () => ({ isPending: false, mutateAsync: mocks.createSecret }),
  useVaultSecrets: () => ({
    data: mocks.vaultSecrets,
    error: mocks.vaultError,
    isLoading: false,
  }),
}));

vi.mock("../../hooks/use-marketplace-actions", () => ({
  useActivateMarketplaceBundle: () => ({
    error: mocks.activateError,
    isPending: false,
    mutateAsync: mocks.activate,
    reset: mocks.activateReset,
  }),
  usePreviewMarketplaceBundle: () => ({
    data: mocks.previewData,
    error: mocks.previewError,
    isPending: false,
    mutate: mocks.preview,
  }),
}));

const noop = () => undefined;

beforeEach(() => {
  mocks.activate.mockReset();
  mocks.activateError = null;
  mocks.activateReset.mockReset();
  mocks.createSecret.mockReset();
  mocks.preview.mockReset();
  mocks.previewData = marketplaceBundlePreviewFixture;
  mocks.previewError = null;
  mocks.vaultError = null;
  mocks.vaultSecrets = [];
  mocks.createSecret.mockResolvedValue({ present: true });
});

describe("Marketplace landing and kind views", () => {
  it("Should keep sibling sections usable when one source fails", () => {
    const partial: MarketplaceSearchResponse = {
      ...marketplaceSearchFixture,
      query: "run",
      kinds: marketplaceSearchFixture.kinds.map(result =>
        result.kind === "extension"
          ? { ...result, error: "source unavailable", items: [], total: null }
          : result
      ),
    };

    render(
      <MarketplaceLanding
        data={partial}
        onAction={noop}
        onClearSearch={noop}
        onRetry={noop}
        onSearchChange={noop}
        query="run"
      />
    );

    const alert = screen.getByRole("alert");
    expect(alert).toHaveTextContent("Extensions marketplace is unreachable");
    expect(alert).toHaveAttribute("data-variant", "danger");
    expect(screen.getByTestId("marketplace-card-git-flow")).toBeInTheDocument();
    expect(screen.getByTestId("marketplace-card-dep-kit")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "View all skills results" })).toHaveAttribute(
      "href",
      "/marketplace?kind=skills&q=run"
    );
    expect(screen.getByRole("link", { name: "View all bundles results" })).toHaveAttribute(
      "href",
      "/marketplace?kind=bundles&q=run"
    );
  });

  it("Should keep a stale source's cached items visible while flagging staleness", async () => {
    const user = userEvent.setup();
    const onRetry = vi.fn();
    const stale: MarketplaceSearchResponse = {
      ...marketplaceSearchFixture,
      query: "run",
      kinds: marketplaceSearchFixture.kinds.map(result =>
        result.kind === "extension"
          ? { ...result, error: "feed refresh failed", stale: true }
          : result
      ),
    };

    render(
      <MarketplaceLanding
        data={stale}
        onAction={noop}
        onClearSearch={noop}
        onRetry={onRetry}
        onSearchChange={noop}
        query="run"
      />
    );

    expect(screen.getByTestId("marketplace-card-otel-bridge")).toBeInTheDocument();
    const alert = screen.getByRole("alert");
    expect(alert).toHaveTextContent("Extensions results may be out of date");
    expect(alert).toHaveAttribute("data-variant", "warning");
    await user.click(within(alert).getByRole("button", { name: "Retry" }));
    expect(onRetry).toHaveBeenCalledOnce();
  });

  it("Should surface one page-level stale notice when a background refresh fails", async () => {
    const user = userEvent.setup();
    const onRetry = vi.fn();
    render(
      <MarketplaceLanding
        data={marketplaceSearchFixture}
        error={new Error("refresh failed")}
        onAction={noop}
        onClearSearch={noop}
        onRetry={onRetry}
        onSearchChange={noop}
        query=""
      />
    );

    const alerts = screen.getAllByRole("alert");
    expect(alerts).toHaveLength(1);
    const alert = alerts[0]!;
    expect(alert).toHaveTextContent("Marketplace results may be out of date");
    expect(alert).toHaveAttribute("data-variant", "warning");
    expect(screen.getByTestId("marketplace-card-git-flow")).toBeInTheDocument();
    expect(screen.getByTestId("marketplace-card-otel-bridge")).toBeInTheDocument();
    await user.click(within(alert).getByRole("button", { name: "Retry" }));
    expect(onRetry).toHaveBeenCalledOnce();
  });

  it("Should collapse all-zero search into one clearable empty state", async () => {
    const user = userEvent.setup();
    const onClear = vi.fn();
    const empty: MarketplaceSearchResponse = {
      query: "missing",
      kinds: marketplaceSearchFixture.kinds.map(result => ({ ...result, items: [], total: 0 })),
    };
    render(
      <MarketplaceLanding
        data={empty}
        onAction={noop}
        onClearSearch={onClear}
        onRetry={noop}
        onSearchChange={noop}
        query="missing"
      />
    );

    expect(screen.getByText("No matches in the marketplace")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Clear search" }));
    expect(onClear).toHaveBeenCalledOnce();
  });

  it("Should offer Most downloaded only when the response exposes downloads", () => {
    const { rerender } = render(
      <MarketplaceKindView
        data={{ items: marketplaceListings.skill, kind: "skill", stale: false, total: 12 }}
        kind="skill"
        onAction={noop}
        onClearSearch={noop}
        onRetry={noop}
        onSearchChange={noop}
        onSortChange={noop}
        query=""
        sort="relevance"
      />
    );

    expect(screen.getByRole("option", { name: "Sort: Most downloaded" })).toBeInTheDocument();
    expect(screen.getByText("Showing 4 of 12")).toBeInTheDocument();

    rerender(
      <MarketplaceKindView
        data={{ items: marketplaceListings.bundle, kind: "bundle", stale: false }}
        kind="bundle"
        onAction={noop}
        onClearSearch={noop}
        onRetry={noop}
        onSearchChange={noop}
        onSortChange={noop}
        query=""
        sort="downloads"
      />
    );
    expect(screen.queryByRole("option", { name: "Sort: Most downloaded" })).not.toBeInTheDocument();
    expect(screen.getByRole("combobox", { name: "Sort bundles" })).toHaveValue("relevance");
    expect(screen.queryByText(/Showing .* of/)).not.toBeInTheDocument();
  });

  it("Should expose retry behavior for top-level and kind catalog failures", async () => {
    const user = userEvent.setup();
    const onRetry = vi.fn();
    const { rerender } = render(
      <MarketplaceLanding
        error={new Error("catalog offline")}
        onAction={noop}
        onClearSearch={noop}
        onRetry={onRetry}
        onSearchChange={noop}
        query=""
      />
    );

    expect(screen.getByRole("heading", { level: 1, name: "Marketplace" })).toBeInTheDocument();
    expect(screen.getByRole("searchbox", { name: "Search the marketplace" })).toBeInTheDocument();
    expect(screen.getByText("Unable to load the marketplace")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Retry" }));
    expect(onRetry).toHaveBeenCalledOnce();

    rerender(
      <MarketplaceKindView
        error={new Error("skills offline")}
        kind="skill"
        onAction={noop}
        onClearSearch={noop}
        onRetry={onRetry}
        onSearchChange={noop}
        onSortChange={noop}
        query=""
        sort="relevance"
      />
    );
    expect(screen.getByRole("heading", { level: 1, name: "Marketplace" })).toBeInTheDocument();
    expect(screen.getByRole("searchbox", { name: "Search skills" })).toBeInTheDocument();
    expect(screen.getByText("Unable to load skills")).toBeInTheDocument();
  });

  it("Should keep the landing browse frame while loading and omit unavailable section actions", () => {
    render(
      <MarketplaceLanding
        isLoading
        onAction={noop}
        onClearSearch={noop}
        onRetry={noop}
        onSearchChange={noop}
        query=""
      />
    );

    expect(screen.getByRole("heading", { level: 1, name: "Marketplace" })).toBeInTheDocument();
    expect(screen.getAllByRole("status", { name: "Loading marketplace entries" })).toHaveLength(4);
    expect(screen.queryByRole("link", { name: /view all/i })).not.toBeInTheDocument();
  });

  it("Should keep loading geometry and empty kind recovery truthful", async () => {
    const user = userEvent.setup();
    const onClear = vi.fn();
    const { rerender } = render(
      <MarketplaceKindView
        isLoading
        kind="mcp"
        onAction={noop}
        onClearSearch={onClear}
        onRetry={noop}
        onSearchChange={noop}
        onSortChange={noop}
        query=""
        sort="relevance"
      />
    );
    expect(screen.getByRole("status", { name: "Loading marketplace entries" })).toBeInTheDocument();

    rerender(
      <MarketplaceKindView
        data={{ items: [], kind: "mcp", stale: false, total: 0 }}
        kind="mcp"
        onAction={noop}
        onClearSearch={onClear}
        onRetry={noop}
        onSearchChange={noop}
        onSortChange={noop}
        query="missing"
        sort="relevance"
      />
    );
    expect(screen.getByText("No mcp servers match this query")).toBeInTheDocument();
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Clear search" }));
    expect(onClear).toHaveBeenCalledOnce();
  });

  it("Should flag a stale kind while keeping its cached rows operable", async () => {
    const user = userEvent.setup();
    const onRetry = vi.fn();
    render(
      <MarketplaceKindView
        data={{
          error: "feed refresh failed",
          items: marketplaceListings.extension,
          kind: "extension",
          stale: true,
          total: marketplaceListings.extension.length,
        }}
        kind="extension"
        onAction={noop}
        onClearSearch={noop}
        onRetry={onRetry}
        onSearchChange={noop}
        onSortChange={noop}
        query=""
        sort="relevance"
      />
    );

    expect(screen.getByTestId("marketplace-card-otel-bridge")).toBeInTheDocument();
    const alert = screen.getByRole("alert");
    expect(alert).toHaveTextContent("Extensions results may be out of date");
    expect(alert).toHaveAttribute("data-variant", "warning");
    await user.click(within(alert).getByRole("button", { name: "Retry" }));
    expect(onRetry).toHaveBeenCalledOnce();
  });

  it("Should surface an unreachable kind source as a danger banner", () => {
    render(
      <MarketplaceKindView
        data={{
          error: "source unavailable",
          items: [],
          kind: "extension",
          stale: false,
          total: null,
        }}
        kind="extension"
        onAction={noop}
        onClearSearch={noop}
        onRetry={noop}
        onSearchChange={noop}
        onSortChange={noop}
        query=""
        sort="relevance"
      />
    );

    const alert = screen.getByRole("alert");
    expect(alert).toHaveTextContent("Extensions marketplace is unreachable");
    expect(alert).toHaveAttribute("data-variant", "danger");
    expect(screen.queryByTestId("marketplace-card-otel-bridge")).not.toBeInTheDocument();
    expect(screen.queryByText("No extensions")).not.toBeInTheDocument();
  });

  it("Should keep cached rows visible when a background refresh errors", async () => {
    const user = userEvent.setup();
    const onRetry = vi.fn();
    render(
      <MarketplaceKindView
        data={{
          items: marketplaceListings.extension,
          kind: "extension",
          stale: false,
          total: marketplaceListings.extension.length,
        }}
        error={new Error("refresh failed")}
        kind="extension"
        onAction={noop}
        onClearSearch={noop}
        onRetry={onRetry}
        onSearchChange={noop}
        onSortChange={noop}
        query=""
        sort="relevance"
      />
    );

    expect(screen.getByTestId("marketplace-card-otel-bridge")).toBeInTheDocument();
    const alert = screen.getByRole("alert");
    expect(alert).toHaveTextContent("Extensions results may be out of date");
    await user.click(within(alert).getByRole("button", { name: "Retry" }));
    expect(onRetry).toHaveBeenCalledOnce();
  });
});

describe("Marketplace cards and trust", () => {
  it("Should render installed Manage, update, and focusable policy-block actions", async () => {
    const user = userEvent.setup();
    const { rerender } = render(
      <MarketplaceCard entry={marketplaceListings.skill[0]!} onAction={noop} />
    );
    expect(screen.getByRole("link", { name: "Manage git-flow" })).toHaveAttribute(
      "href",
      "/skills/git-flow"
    );

    rerender(<MarketplaceCard entry={marketplaceListings.skill[2]!} onAction={noop} />);
    expect(screen.getByRole("button", { name: "Update qa-bootstrap" })).toBeInTheDocument();

    rerender(<MarketplaceCard entry={marketplaceListings.extension[2]!} onAction={noop} />);
    const blocked = screen.getByRole("button", { name: /blocked by extensions policy/i });
    expect(blocked).toHaveAttribute("aria-disabled", "true");
    expect(blocked).not.toBeDisabled();
    await user.tab();
    await user.tab();
    expect(blocked).toHaveFocus();
    expect(await screen.findByText("Blocked by extensions policy")).toBeInTheDocument();
  });

  it("Should list the daemon trust warnings in the unverified confirmation", () => {
    render(
      <ExtensionTrustDialog
        entry={marketplaceListings.extension[1]!}
        onConfirm={noop}
        onOpenChange={noop}
        open
      />
    );
    expect(screen.getByText("Unsigned package")).toBeInTheDocument();
    expect(screen.getByText("Network egress")).toBeInTheDocument();
  });

  it("Should expose every acquisition status and pending action without losing card identity", async () => {
    const user = userEvent.setup();
    const onAction = vi.fn();
    const { rerender } = render(
      <MarketplaceCard entry={marketplaceListings.extension[0]!} onAction={onAction} />
    );

    expect(screen.getAllByText("verified")).toHaveLength(2);

    rerender(<MarketplaceCard entry={marketplaceListings.extension[1]!} onAction={onAction} />);
    expect(screen.getByText("unverified · 2")).toBeInTheDocument();

    rerender(<MarketplaceCard entry={marketplaceListings.mcp[0]!} onAction={onAction} />);
    expect(screen.getByText("curated")).toBeInTheDocument();

    rerender(<MarketplaceCard entry={marketplaceListings.skill[1]!} onAction={onAction} />);
    await user.click(screen.getByRole("button", { name: "Install docs-sync" }));
    expect(onAction).toHaveBeenCalledWith(marketplaceListings.skill[1]);

    rerender(<MarketplaceCard entry={marketplaceListings.skill[1]!} onAction={onAction} pending />);
    expect(screen.getByRole("button", { name: "Install docs-sync" })).toHaveTextContent(
      "Installing…"
    );
    expect(screen.getByRole("link", { name: "View docs-sync details" })).toHaveAttribute(
      "aria-disabled",
      "true"
    );

    rerender(<MarketplaceCard entry={marketplaceListings.skill[2]!} onAction={onAction} pending />);
    expect(screen.getByRole("button", { name: "Update qa-bootstrap" })).toHaveTextContent(
      "Updating…"
    );

    rerender(
      <MarketplaceCard entry={marketplaceListings.bundle[0]!} onAction={onAction} pending />
    );
    expect(screen.getByRole("button", { name: "Activate dep-kit" })).toHaveTextContent(
      "Activating…"
    );

    rerender(
      <MarketplaceCard
        entry={{ ...marketplaceListings.skill[2]!, version: undefined }}
        onAction={onAction}
      />
    );
    expect(screen.getByText("update available")).toBeInTheDocument();
  });

  it("Should expose the policy settings bridge on blocked detail", () => {
    render(
      <MarketplaceDetail data={marketplaceDetails["extension:policy-blocked"]!} onAction={noop} />
    );
    expect(screen.getByText("Settings › Extensions")).toHaveAttribute(
      "href",
      "/settings/extensions"
    );
    const blockedAction = screen.getByRole("button", { name: "Install policy-blocked" });
    expect(blockedAction).toHaveAttribute("aria-disabled", "true");
    expect(blockedAction).not.toBeDisabled();
  });

  it("Should render installed skill readme and management metadata without a duplicate history", () => {
    render(<MarketplaceDetail data={marketplaceDetails["skill:git-flow"]!} onAction={noop} />);

    expect(screen.getAllByRole("heading", { level: 1 })).toHaveLength(1);
    expect(screen.getByRole("link", { name: "Manage git-flow" })).toHaveAttribute(
      "href",
      "/skills/git-flow"
    );
    expect(screen.getByText("What it does")).toBeInTheDocument();
    expect(screen.queryByText("v1.4.1")).not.toBeInTheDocument();
    expect(screen.getByText("MIT")).toBeInTheDocument();
    expect(screen.getByText("3.4K")).toBeInTheDocument();
  });

  it("Should render extension provenance and route its primary action", async () => {
    const user = userEvent.setup();
    const onAction = vi.fn();
    render(
      <MarketplaceDetail data={marketplaceDetails["extension:slack-notify"]!} onAction={onAction} />
    );

    expect(screen.getByText("Provenance")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Repository" })).toHaveAttribute(
      "href",
      "https://github.com/community/slack-notify"
    );
    expect(screen.getByRole("link", { name: "Pinned archive" })).toHaveAttribute(
      "href",
      "https://github.com/community/slack-notify/releases/download/v1.1.4/slack-notify-v1.1.4.tar.gz"
    );
    expect(
      screen.getByText("9be4d36a7bf48f5df88bb0ee3564561eaeaaeef2bcf76c2e2ad19856c045ef98")
    ).toBeInTheDocument();
    expect(screen.getByText("Unsigned package")).toBeInTheDocument();
    expect(screen.getByText("Network egress")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Install slack-notify" }));
    expect(onAction).toHaveBeenCalledWith(marketplaceListings.extension[1]);
  });

  it("Should render bundle profiles and MCP transport-specific configuration", async () => {
    const user = userEvent.setup();
    const onAction = vi.fn();
    const { rerender } = render(
      <MarketplaceDetail data={marketplaceDetails["bundle:dep-kit"]!} onAction={onAction} />
    );

    expect(screen.getByText("Dependency review with release notifications.")).toBeInTheDocument();
    expect(screen.getByText(/2 jobs · 2 triggers/)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Activate dep-kit" }));
    expect(onAction).toHaveBeenCalledWith(marketplaceListings.bundle[0]);

    rerender(<MarketplaceDetail data={marketplaceDetails["mcp:github"]!} onAction={noop} />);
    expect(screen.getByText("npx -y @modelcontextprotocol/server-github")).toBeInTheDocument();
    expect(screen.getByText("GITHUB_TOKEN")).toBeInTheDocument();

    rerender(<MarketplaceDetail data={marketplaceDetails["mcp:linear"]!} onAction={noop} />);
    expect(screen.getByText("https://mcp.linear.app/sse")).toBeInTheDocument();
    expect(screen.getByText("OAuth 2 PKCE")).toBeInTheDocument();
    expect(screen.getByText("read, issues:create")).toBeInTheDocument();
  });

  it("Should render sparse catalog details and all pending detail actions truthfully", () => {
    const sparseMCP: MarketplaceEntryResponse = {
      entry: {
        description: "A transport-only MCP entry.",
        entry_id: "transport-only",
        installed: false,
        kind: "mcp",
        name: "transport-only",
        source: "curated",
        update_available: false,
      },
      mcp: {
        env: [{ name: "OPTIONAL_PATH", required: false, secret: false }],
        transport: "stdio",
      },
    };
    const { rerender } = render(<MarketplaceDetail data={sparseMCP} onAction={noop} pending />);

    expect(screen.getByRole("button", { name: "Install transport-only" })).toHaveTextContent(
      "Installing…"
    );
    expect(screen.getByText("Optional")).toBeInTheDocument();

    rerender(
      <MarketplaceDetail
        data={{
          ...marketplaceDetails["bundle:dep-kit"]!,
          bundle: {
            extension_name: "agh-foundations",
            profiles: [
              {
                agents: 0,
                bridges: 0,
                channels: 0,
                jobs: 0,
                name: "minimal",
                triggers: 0,
              },
            ],
          },
        }}
        onAction={noop}
        pending
      />
    );
    expect(screen.getByRole("button", { name: "Activate dep-kit" })).toHaveTextContent(
      "Activating…"
    );
    expect(screen.getByText(/0 agents · 0 jobs/)).toBeInTheDocument();

    rerender(
      <MarketplaceDetail
        data={{
          ...marketplaceDetails["skill:git-flow"]!,
          entry: marketplaceListings.skill[2]!,
        }}
        onAction={noop}
        pending
      />
    );
    expect(screen.getByRole("button", { name: "Update qa-bootstrap" })).toHaveTextContent(
      "Updating…"
    );
  });

  it("Should distinguish verified checksums and error-level trust guidance", () => {
    const verifiedDetail: MarketplaceEntryResponse = {
      ...marketplaceDetails["extension:slack-notify"]!,
      entry: {
        ...marketplaceListings.extension[0]!,
        trust: {
          allow_unverified: false,
          checksum_verified: true,
          decision: "verified",
          registry_tier: "verified",
          warnings: [
            {
              category: "supply_chain",
              code: "extension_checksum_mismatch",
              data_freshness: "live",
              id: "extension_checksum_mismatch",
              message: "The downloaded archive did not match the curated digest.",
              severity: "error",
              suggested_command: "agh extensions marketplace diagnostics otel-bridge",
              title: "Checksum mismatch",
            },
          ],
        },
      },
    };

    render(<MarketplaceDetail data={verifiedDetail} onAction={noop} />);

    const checksumTerm = screen
      .getAllByRole("term")
      .find(element => element.textContent === "Checksum");
    expect(checksumTerm).toBeDefined();
    const checksumRow = checksumTerm!.parentElement;
    expect(checksumRow).not.toBeNull();
    expect(within(checksumRow!).getByRole("definition")).toHaveTextContent("verified");
    expect(screen.getByText("Checksum mismatch")).toBeInTheDocument();
    expect(
      screen.getByText("agh extensions marketplace diagnostics otel-bridge")
    ).toBeInTheDocument();
  });
});

describe("MCP guided install", () => {
  it("Should keep Install disabled until the required typed value is present", async () => {
    const user = userEvent.setup();
    const onInstall = vi.fn().mockResolvedValue({} as MCPInstallResponse);
    render(
      <MCPInstallDialog
        data={marketplaceDetails["mcp:github"]!}
        onInstall={onInstall}
        onOpenChange={noop}
        open
        workspaceId="ws_story_fintech"
      />
    );
    const confirm = screen.getByTestId("mcp-install-confirm");
    expect(confirm).toBeDisabled();
    await user.type(screen.getByLabelText(/^GITHUB_TOKEN\*?$/), "github-secret");
    expect(confirm).toBeEnabled();
    await user.click(confirm);
    await waitFor(() => expect(onInstall).toHaveBeenCalledOnce());
    expect(onInstall.mock.calls[0]?.[0]).toMatchObject({
      scope: "workspace",
      values: { env: { GITHUB_TOKEN: { value: "github-secret" } } },
      workspace_id: "ws_story_fintech",
    });
  });

  it("Should bind a selected present Vault ref without reading its value", async () => {
    const user = userEvent.setup();
    const onInstall = vi.fn().mockResolvedValue({} as MCPInstallResponse);
    mocks.vaultSecrets = [
      {
        created_at: "2026-07-14T11:00:00Z",
        kind: "mcp_env",
        namespace: "mcp",
        present: true,
        ref: "vault:mcp/ws/ws_story_fintech/github/env/GITHUB_TOKEN",
        updated_at: "2026-07-14T11:00:00Z",
      },
    ];
    render(
      <MCPInstallDialog
        data={marketplaceDetails["mcp:github"]!}
        onInstall={onInstall}
        onOpenChange={noop}
        open
        workspaceId="ws_story_fintech"
      />
    );
    await user.click(screen.getByRole("button", { name: "Use Vault" }));
    expect(screen.getByText("Choose existing secret")).toBeInTheDocument();
    expect(screen.getByText("namespace=mcp")).toBeInTheDocument();
    expect(screen.getByText("Values stay hidden.")).toBeInTheDocument();
    const vaultRadio = screen.getByRole("radio", {
      name: /vault:mcp\/ws\/ws_story_fintech\/github\/env\/GITHUB_TOKEN/,
    });
    await user.click(vaultRadio);
    expect(vaultRadio).toHaveAttribute("aria-checked", "true");
    await user.click(screen.getByTestId("mcp-install-confirm"));
    await waitFor(() => expect(onInstall).toHaveBeenCalledOnce());
    expect(onInstall.mock.calls[0]?.[0]).toMatchObject({
      values: {
        env: {
          GITHUB_TOKEN: {
            vault_ref: "vault:mcp/ws/ws_story_fintech/github/env/GITHUB_TOKEN",
          },
        },
      },
    });
  });

  it("Should create an inline Vault secret and bind the canonical ref", async () => {
    const user = userEvent.setup();
    const onInstall = vi.fn().mockResolvedValue({} as MCPInstallResponse);
    render(
      <MCPInstallDialog
        data={marketplaceDetails["mcp:github"]!}
        onInstall={onInstall}
        onOpenChange={noop}
        open
        workspaceId="ws_story_fintech"
      />
    );
    await user.click(screen.getByRole("button", { name: "Use Vault" }));
    await user.click(screen.getByRole("button", { name: "Create Vault secret" }));
    await user.type(screen.getByLabelText("New Vault value for GITHUB_TOKEN"), "created-secret");
    await user.click(screen.getByTestId("mcp-create-secret-GITHUB_TOKEN"));
    await waitFor(() => expect(mocks.createSecret).toHaveBeenCalledOnce());
    expect(mocks.createSecret).toHaveBeenCalledWith({
      kind: "mcp_env",
      ref: "vault:mcp/ws/ws_story_fintech/github/env/GITHUB_TOKEN",
      secret_value: "created-secret",
    });
    await user.click(screen.getByTestId("mcp-install-confirm"));
    await waitFor(() => expect(onInstall).toHaveBeenCalledOnce());
  });

  it("Should preserve field values and show the daemon error after submit failure", async () => {
    const user = userEvent.setup();
    const onInstall = vi.fn().mockRejectedValue(new Error("Config write rejected"));
    render(
      <MCPInstallDialog
        data={marketplaceDetails["mcp:github"]!}
        onInstall={onInstall}
        onOpenChange={noop}
        open
        workspaceId="ws_story_fintech"
      />
    );
    const input = screen.getByLabelText(/^GITHUB_TOKEN\*?$/);
    await user.type(input, "keep-this-value");
    await user.click(screen.getByTestId("mcp-install-confirm"));
    expect(await screen.findByRole("alert")).toHaveTextContent("Config write rejected");
    expect(input).toHaveValue("keep-this-value");
  });

  it("Should render remote auth without secret fields and submit null values", async () => {
    const user = userEvent.setup();
    const onInstall = vi.fn().mockResolvedValue({} as MCPInstallResponse);
    const remote = marketplaceDetails["mcp:linear"]!;
    render(
      <MCPInstallDialog
        data={remote}
        onInstall={onInstall}
        onOpenChange={noop}
        open
        workspaceId="ws_story_fintech"
      />
    );
    expect(screen.getByText("Authorization · OAuth 2 PKCE")).toBeInTheDocument();
    expect(screen.queryByText("Required configuration")).not.toBeInTheDocument();
    await user.click(screen.getByTestId("mcp-install-confirm"));
    await waitFor(() => expect(onInstall).toHaveBeenCalledOnce());
    expect(onInstall.mock.calls[0]?.[0]).toMatchObject({ entry_id: "linear", values: null });
    expect(buildMCPInstallRequest(remote, "global", null, {}, true)).not.toHaveProperty(
      "values.env"
    );
  });
});

describe("Bundle activation preview", () => {
  it("Should rerun preview when the selected profile changes", async () => {
    const user = userEvent.setup();
    render(
      <BundleActivationDialog
        data={marketplaceDetails["bundle:dep-kit"]!}
        onOpenChange={noop}
        open
        workspaceId="ws_story_fintech"
      />
    );
    await waitFor(() => expect(mocks.preview).toHaveBeenCalled());
    await user.click(screen.getByRole("radio", { name: /release/i }));
    await waitFor(() =>
      expect(mocks.preview).toHaveBeenLastCalledWith(
        expect.objectContaining({ profile_name: "release" })
      )
    );
  });

  it("Should name the daemon conflict and keep Activate disabled", () => {
    mocks.previewData = undefined;
    mocks.previewError = new Error("Primary channel already belongs to another activation");
    render(
      <BundleActivationDialog
        data={marketplaceDetails["bundle:dep-kit"]!}
        onOpenChange={noop}
        open
        workspaceId="ws_story_fintech"
      />
    );
    expect(screen.getByTestId("bundle-preview-error")).toHaveTextContent(
      "Primary channel already belongs to another activation"
    );
    expect(screen.getByTestId("bundle-activate-confirm")).toBeDisabled();
  });

  it("Should clear an activation error before retrying the preview", async () => {
    const user = userEvent.setup();
    mocks.activateError = new Error("Activation conflict");
    render(
      <BundleActivationDialog
        data={marketplaceDetails["bundle:dep-kit"]!}
        onOpenChange={noop}
        open
        workspaceId="ws_story_fintech"
      />
    );
    await waitFor(() => expect(mocks.preview).toHaveBeenCalled());
    const previewCalls = mocks.preview.mock.calls.length;

    await user.click(screen.getByRole("button", { name: "Retry bundle preview" }));

    expect(mocks.activateReset).toHaveBeenCalledOnce();
    expect(mocks.preview).toHaveBeenCalledTimes(previewCalls + 1);
  });

  it("Should submit the selected global scope and Live network confirmation", async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();
    render(
      <BundleActivationDialog
        data={marketplaceDetails["bundle:dep-kit"]!}
        onOpenChange={onOpenChange}
        open
        workspaceId="ws_story_fintech"
      />
    );

    await user.click(screen.getByRole("radio", { name: /Global/ }));
    await user.click(screen.getByRole("switch", { name: "Confirm Live network participation" }));
    await user.click(screen.getByTestId("bundle-activate-confirm"));

    await waitFor(() =>
      expect(mocks.activate).toHaveBeenCalledWith(
        expect.objectContaining({
          confirm_network_requirement: true,
          scope: "global",
        })
      )
    );
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });
});
