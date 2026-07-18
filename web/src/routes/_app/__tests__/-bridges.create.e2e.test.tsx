// Suite: Bridges create setup route integration
// Invariant: the visible wizard persists one disabled bridge before fetching its daemon-owned
// Slack manifest, and serializes the complete progress override into that one create request.
// Boundary IN: Bridges route, real route hooks, TanStack Query hooks, adapters, and openapi-fetch.
// Boundary OUT: AGH daemon HTTP implementation, replaced by stateful MSW handlers.

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse, type HttpHandler } from "msw";
import { useEffect, useState, type AnchorHTMLAttributes, type ReactNode } from "react";
import { afterAll, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

import type {
  BridgeHealth,
  BridgeProvider,
  BridgeSummary,
  BridgesListResponse,
  CreateBridgeRequest,
  SlackBridgeManifestResponse,
} from "@/systems/bridges";
import { slackBridgeManifestFixture } from "@/systems/bridges/mocks";
import { createMswFetch } from "@/test/msw-fetch";
import { renderWithTopbar } from "@/test/render-with-topbar";
import { routeComponent } from "@/test/route-options";

const { clipboardWriteText, toast } = vi.hoisted(() => ({
  clipboardWriteText: vi.fn(),
  toast: {
    error: vi.fn(),
    success: vi.fn(),
  },
}));

const routerState = vi.hoisted(() => ({
  navigateMock: vi.fn(),
  searchListeners: new Set<(search: Record<string, unknown>) => void>(),
  searchParams: {} as Record<string, unknown>,
}));

interface MockLinkProps extends AnchorHTMLAttributes<HTMLAnchorElement> {
  params?: { id?: string };
  to?: string;
}

vi.mock("@tanstack/react-router", async importOriginal => {
  const actual = await importOriginal<typeof import("@tanstack/react-router")>();
  return {
    ...actual,
    Outlet: () => null,
    createFileRoute:
      () =>
      (options: {
        component: () => ReactNode;
        validateSearch?: (search: Record<string, unknown>) => Record<string, unknown>;
      }) => {
        const validate = (search: Record<string, unknown>) =>
          options.validateSearch ? options.validateSearch(search) : search;
        return {
          component: options.component,
          useSearch: () => {
            const [search, setSearch] = useState(() => validate(routerState.searchParams));
            useEffect(() => {
              const listener = (next: Record<string, unknown>) => setSearch(validate(next));
              routerState.searchListeners.add(listener);
              return () => {
                routerState.searchListeners.delete(listener);
              };
            }, []);
            return search;
          },
        };
      },
    Link: ({ children, params, to, ...props }: MockLinkProps) => (
      <a href={to ?? `/bridges/${params?.id ?? ""}`} {...props}>
        {children}
      </a>
    ),
    useChildMatches: () => [],
    useNavigate: () => async (options: { params?: { id?: string }; to: string }) => {
      routerState.navigateMock(options);
    },
  };
});

vi.mock("sonner", () => ({ toast }));

vi.mock("@/systems/bridges/hooks/use-bridge-health-stream", () => ({
  applyBridgeHealthSnapshot: vi.fn(),
  useBridgeHealthStream: vi.fn(),
}));

vi.mock("@/systems/workspace", () => ({
  useActiveWorkspace: () => ({
    activeWorkspace: {
      add_dirs: [],
      created_at: "2026-07-12T10:00:00Z",
      id: "ws_setup",
      name: "Setup lab",
      root_dir: "/workspace/setup",
      updated_at: "2026-07-12T10:00:00Z",
    },
    activeWorkspaceId: "ws_setup",
    hasWorkspaces: true,
    isError: false,
    isLoading: false,
    workspaces: [],
  }),
}));

import { Route } from "../bridges";

const BridgesPage = routeComponent(Route);
const originalFetch = globalThis.fetch;
let handlers: HttpHandler[] = [];
let createRequests: CreateBridgeRequest[] = [];
let manifestInstanceIDs: string[] = [];
let createFailuresRemaining = 0;
let manifestFailuresRemaining = 0;
let providers: BridgeProvider[] = [];

const slackProvider: BridgeProvider = {
  config_schema: { schema: "provider-config", version: "2026-07-12" },
  description: "Slack bridge adapter",
  display_name: "Slack",
  enabled: true,
  extension_name: "ext-slack",
  health: "healthy",
  platform: "slack",
  secret_slots: [
    { description: "Bot token", name: "bot_token", required: true },
    { description: "Signing secret", name: "signing_secret", required: true },
  ],
  state: "active",
};

const telegramProvider: BridgeProvider = {
  ...slackProvider,
  display_name: "Telegram",
  extension_name: "ext-telegram",
  platform: "telegram",
  secret_slots: [{ description: "Bot token", name: "bot_token", required: true }],
};

function emptyBridgeList(): BridgesListResponse {
  return {
    bridge_health: {},
    bridges: [],
    facets: {
      platforms: {},
      statuses: {
        auth_required: 0,
        degraded: 0,
        disabled: 0,
        error: 0,
        ready: 0,
        starting: 0,
      },
    },
    page: { has_more: false, limit: 50, total: 0 },
  };
}

function createdBridge(body: CreateBridgeRequest): BridgeSummary {
  const id = body.platform === "slack" ? "brg_slack_created" : `brg_${body.platform}_created`;
  return {
    created_at: "2026-07-12T10:05:00Z",
    delivery_defaults: body.delivery_defaults,
    display_name: body.display_name,
    dm_policy: body.dm_policy,
    enabled: false,
    extension_name: body.extension_name,
    id,
    notification_suppress: body.notification_suppress ?? false,
    platform: body.platform,
    provider_config: body.provider_config,
    routing_policy: body.routing_policy,
    scope: body.scope,
    status: "disabled",
    updated_at: "2026-07-12T10:05:00Z",
    workspace_id: body.workspace_id,
  };
}

function createdHealth(bridgeInstanceID: string): BridgeHealth {
  return {
    auth_failures_total: 0,
    bridge_instance_id: bridgeInstanceID,
    delivery_backlog: 0,
    delivery_dropped_total: 0,
    delivery_failures_total: 0,
    route_count: 0,
    status: "disabled",
  };
}

function bridgeHandlers(): HttpHandler[] {
  return [
    http.get("/api/bridges", () => HttpResponse.json(emptyBridgeList())),
    http.get("/api/bridges/providers", () => HttpResponse.json({ providers })),
    http.post("/api/bridges", async ({ request }) => {
      const body = (await request.json()) as CreateBridgeRequest;
      createRequests.push(body);
      if (createFailuresRemaining > 0) {
        createFailuresRemaining -= 1;
        return HttpResponse.json(
          { error: "Bridge create temporarily unavailable" },
          { status: 503 }
        );
      }
      const bridge = createdBridge(body);
      return HttpResponse.json({ bridge, health: createdHealth(bridge.id) }, { status: 201 });
    }),
    http.get("/api/bridges/providers/slack/manifest", ({ request }) => {
      const instanceID = new URL(request.url).searchParams.get("instance") ?? "";
      manifestInstanceIDs.push(instanceID);
      if (manifestFailuresRemaining > 0) {
        manifestFailuresRemaining -= 1;
        return HttpResponse.json(
          { error: "Slack manifest temporarily unavailable" },
          { status: 503 }
        );
      }
      if (instanceID !== "brg_slack_created") {
        return HttpResponse.json({ error: "Slack bridge not found" }, { status: 404 });
      }
      return HttpResponse.json(slackBridgeManifestFixture satisfies SlackBridgeManifestResponse);
    }),
  ];
}

function renderPage() {
  const queryClient = new QueryClient({
    defaultOptions: {
      mutations: { retry: false },
      queries: { retry: false },
    },
  });
  return renderWithTopbar(
    <QueryClientProvider client={queryClient}>
      <BridgesPage />
    </QueryClientProvider>
  );
}

beforeAll(() => {
  globalThis.fetch = createMswFetch(() => handlers);
});

afterAll(() => {
  globalThis.fetch = originalFetch;
});

beforeEach(() => {
  handlers = bridgeHandlers();
  createRequests = [];
  manifestInstanceIDs = [];
  createFailuresRemaining = 0;
  manifestFailuresRemaining = 0;
  providers = [slackProvider];
  clipboardWriteText.mockReset();
  clipboardWriteText.mockResolvedValue(undefined);
  routerState.navigateMock.mockReset();
  routerState.searchListeners.clear();
  routerState.searchParams = {};
  toast.error.mockReset();
  toast.success.mockReset();
});

function installClipboard() {
  Object.defineProperty(navigator, "clipboard", {
    configurable: true,
    value: { writeText: clipboardWriteText },
  });
}

async function openWizardAtDelivery(user: ReturnType<typeof userEvent.setup>) {
  await user.click(await screen.findByTestId("bridge-empty-create-btn"));
  await user.click(screen.getByTestId("bridge-wizard-next"));
  await user.click(screen.getByTestId("bridge-wizard-next"));
  expect(screen.getByTestId("bridge-wizard-progress")).toHaveTextContent("Step 3 of 3");
}

describe("Bridges Slack setup create flow", () => {
  it("Should persist one disabled bridge before fetching and copying its manifest with progress", async () => {
    const user = userEvent.setup();
    installClipboard();
    renderPage();

    await user.click(await screen.findByTestId("bridge-empty-create-btn"));
    expect(screen.getByTestId("bridge-manifest-precreate-hint")).toBeInTheDocument();
    expect(screen.queryByTestId("bridge-manifest-json")).not.toBeInTheDocument();
    expect(createRequests).toHaveLength(0);
    expect(manifestInstanceIDs).toHaveLength(0);

    await user.click(screen.getByTestId("bridge-wizard-next"));
    fireEvent.change(screen.getByTestId("bridge-provider-config-input"), {
      target: {
        value: '{"webhook":{"public_url":"https://bridge.example.test/slack/setup-lab"}}',
      },
    });
    await user.click(screen.getByTestId("bridge-wizard-next"));
    await user.selectOptions(screen.getByTestId("bridge-delivery-progress-mode-select"), "verbose");
    await user.selectOptions(
      screen.getByTestId("bridge-delivery-progress-grouping-select"),
      "separate"
    );
    await user.selectOptions(screen.getByTestId("bridge-delivery-progress-typing-select"), "false");
    await user.selectOptions(
      screen.getByTestId("bridge-delivery-progress-reactions-select"),
      "true"
    );
    await user.click(screen.getByTestId("submit-bridge-create"));

    await waitFor(() => {
      expect(createRequests).toHaveLength(1);
      expect(manifestInstanceIDs).toEqual(["brg_slack_created"]);
    });
    expect(createRequests[0]).toEqual({
      delivery_defaults: {
        progress: {
          grouping: "separate",
          reactions: true,
          tool_progress: "verbose",
          typing: false,
        },
      },
      display_name: "Slack",
      enabled: false,
      extension_name: "ext-slack",
      notification_suppress: false,
      platform: "slack",
      provider_config: {
        webhook: { public_url: "https://bridge.example.test/slack/setup-lab" },
      },
      routing_policy: {
        include_group: true,
        include_peer: true,
        include_thread: true,
      },
      scope: "workspace",
      workspace_id: "ws_setup",
    });

    expect(await screen.findByTestId("bridge-manifest-json")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Copy Slack app manifest" }));
    await waitFor(() => {
      expect(clipboardWriteText).toHaveBeenCalledWith(
        JSON.stringify(slackBridgeManifestFixture.manifest, null, 2)
      );
    });
    expect(screen.getByRole("link", { name: "Open Slack app dashboard" })).toHaveAttribute(
      "href",
      "https://api.slack.com/apps"
    );

    await user.click(screen.getByTestId("bridge-manifest-open-bridge"));
    expect(routerState.navigateMock).toHaveBeenCalledWith({
      params: { id: "brg_slack_created" },
      to: "/bridges/$id",
    });
    expect(createRequests).toHaveLength(1);
  });

  it("Should retain the draft and retry one failed create before requesting a manifest", async () => {
    createFailuresRemaining = 1;
    const user = userEvent.setup();
    renderPage();
    await openWizardAtDelivery(user);

    await user.click(screen.getByTestId("submit-bridge-create"));
    await waitFor(() => expect(createRequests).toHaveLength(1));
    expect(manifestInstanceIDs).toHaveLength(0);
    expect(screen.getByTestId("bridge-create-dialog")).toBeInTheDocument();
    expect(screen.getByTestId("bridge-wizard-progress")).toHaveTextContent("Step 3 of 3");

    await user.click(screen.getByTestId("submit-bridge-create"));
    expect(await screen.findByTestId("bridge-manifest-json")).toBeInTheDocument();
    expect(createRequests).toHaveLength(2);
    expect(createRequests[1]).toEqual(createRequests[0]);
    expect(manifestInstanceIDs).toEqual(["brg_slack_created"]);
  });

  it("Should retry a failed manifest with the persisted ID without creating another bridge", async () => {
    manifestFailuresRemaining = 1;
    const user = userEvent.setup();
    renderPage();
    await openWizardAtDelivery(user);

    await user.click(screen.getByTestId("submit-bridge-create"));
    expect(await screen.findByTestId("bridge-manifest-error")).toBeInTheDocument();
    expect(createRequests).toHaveLength(1);
    expect(manifestInstanceIDs).toEqual(["brg_slack_created"]);

    await user.click(screen.getByRole("button", { name: "Retry" }));
    expect(await screen.findByTestId("bridge-manifest-json")).toBeInTheDocument();
    expect(createRequests).toHaveLength(1);
    expect(manifestInstanceIDs).toEqual(["brg_slack_created", "brg_slack_created"]);
  });

  it("Should keep the persisted bridge accessible when its manifest remains unavailable", async () => {
    manifestFailuresRemaining = 2;
    const user = userEvent.setup();
    renderPage();
    await openWizardAtDelivery(user);

    await user.click(screen.getByTestId("submit-bridge-create"));
    expect(await screen.findByTestId("bridge-manifest-error")).toBeInTheDocument();
    await user.click(screen.getByTestId("bridge-manifest-open-bridge"));

    expect(routerState.navigateMock).toHaveBeenCalledWith({
      params: { id: "brg_slack_created" },
      to: "/bridges/$id",
    });
    expect(createRequests).toHaveLength(1);
  });

  it("Should keep Open bridge available when clipboard access fails", async () => {
    const user = userEvent.setup();
    clipboardWriteText.mockRejectedValueOnce(new Error("Clipboard permission denied"));
    installClipboard();
    renderPage();
    await openWizardAtDelivery(user);

    await user.click(screen.getByTestId("submit-bridge-create"));
    expect(await screen.findByTestId("bridge-manifest-json")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Copy Slack app manifest" }));
    expect(await screen.findByRole("button", { name: "Copy failed" })).toBeInTheDocument();

    await user.click(screen.getByTestId("bridge-manifest-open-bridge"));
    expect(routerState.navigateMock).toHaveBeenCalledWith({
      params: { id: "brg_slack_created" },
      to: "/bridges/$id",
    });
  });

  it("Should create a non-Slack provider without requesting or rendering a manifest", async () => {
    providers = [telegramProvider];
    const user = userEvent.setup();
    renderPage();
    await user.click(await screen.findByTestId("bridge-empty-create-btn"));
    expect(screen.queryByTestId("bridge-manifest-precreate-hint")).not.toBeInTheDocument();
    await user.click(screen.getByTestId("bridge-wizard-next"));
    await user.click(screen.getByTestId("bridge-wizard-next"));
    await user.click(screen.getByTestId("submit-bridge-create"));

    await waitFor(() => {
      expect(routerState.navigateMock).toHaveBeenCalledWith({
        params: { id: "brg_telegram_created" },
        to: "/bridges/$id",
      });
    });
    expect(createRequests).toHaveLength(1);
    expect(createRequests[0]).toMatchObject({
      enabled: false,
      extension_name: "ext-telegram",
      platform: "telegram",
    });
    expect(manifestInstanceIDs).toHaveLength(0);
    expect(screen.queryByTestId("bridge-manifest-json")).not.toBeInTheDocument();
  });
});
