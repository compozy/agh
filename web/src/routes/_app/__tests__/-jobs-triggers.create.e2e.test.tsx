// Suite: automation create modal route E2E
// Invariant: jobs and triggers create modals submit the visible draft through real page hooks, query hooks, adapters, and the selected workspace scope.
// Boundary IN: jobs/triggers catalog + detail routes, editor dialog/forms, TanStack Query hooks, and openapi-fetch.
// Boundary OUT: AGH daemon HTTP implementation, replaced by MSW handlers at the fetch boundary.
// Note: this historical *.e2e.test.tsx file runs in Vitest/jsdom with MSW; browser
// dismissal parity for Dialog behavior is verified separately by visual/browser QA.

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { delay, http, HttpResponse, type HttpHandler } from "msw";
import { useEffect, useState, type AnchorHTMLAttributes, type ReactNode } from "react";
import { afterAll, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

import type {
  AutomationJob,
  AutomationRun,
  AutomationTrigger,
  CreateAutomationJobRequest,
  CreateAutomationTriggerRequest,
  UpdateAutomationJobRequest,
  UpdateAutomationTriggerRequest,
} from "@/systems/automation";
import {
  automationRunFixtures,
  primaryAutomationJobFixture,
  primaryAutomationTriggerFixture,
} from "@/systems/automation/mocks/fixtures";
import { loopCatalogFixtures } from "@/systems/loops/mocks/fixtures";
import { settingsAutomationSectionFixture } from "@/systems/settings/mocks/fixtures";
import { workspaceKeys } from "@/systems/workspace";
import { useActiveWorkspaceStore } from "@/systems/workspace/hooks/use-active-workspace-store";
import { useUserHomeDirStore } from "@/systems/workspace/hooks/use-user-home-dir-store";
import type { WorkspacePayload } from "@/systems/workspace/types";
import { createMswFetch, createStatefulMswStore } from "@/test/msw-fetch";
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
  childMatches: [] as unknown[],
  currentPath: "/jobs",
  navigateMock: vi.fn(),
  params: {} as Record<string, string>,
  searchListeners: new Set<(search: Record<string, unknown>) => void>(),
  searchParams: {} as Record<string, unknown>,
}));

interface MockLinkParams {
  id?: string;
}

interface MockLinkProps extends AnchorHTMLAttributes<HTMLAnchorElement> {
  params?: MockLinkParams;
}

interface MockNavigateOptions {
  search?:
    | Record<string, unknown>
    | ((current: Record<string, unknown>) => Record<string, unknown>);
  to: string;
}

async function navigate(options: MockNavigateOptions) {
  routerState.navigateMock(options);
  if (options.to !== routerState.currentPath) return;

  if (typeof options.search === "function") {
    routerState.searchParams = options.search(routerState.searchParams);
  } else if (options.search) {
    routerState.searchParams = options.search;
  }

  for (const listener of routerState.searchListeners) {
    listener(routerState.searchParams);
  }
}

vi.mock("@tanstack/react-router", async importOriginal => {
  const actual = await importOriginal<typeof import("@tanstack/react-router")>();

  return {
    ...actual,
    createFileRoute:
      () =>
      (opts: {
        component: () => ReactNode;
        validateSearch?: (search: Record<string, unknown>) => Record<string, unknown>;
      }) => {
        const validateSearch = (search: Record<string, unknown>) =>
          opts.validateSearch ? opts.validateSearch(search) : search;

        return {
          component: opts.component,
          useParams: () => routerState.params,
          useSearch: () => {
            const [search, setSearch] = useState(() => validateSearch(routerState.searchParams));

            useEffect(() => {
              const listener = (nextSearch: Record<string, unknown>) => {
                setSearch(validateSearch(nextSearch));
              };
              routerState.searchListeners.add(listener);
              return () => {
                routerState.searchListeners.delete(listener);
              };
            }, []);

            return search;
          },
        };
      },
    Link: ({ children, params, to, ...props }: MockLinkProps & { to?: string }) => (
      <a href={to ?? `/${params?.id ?? ""}`} {...props}>
        {children}
      </a>
    ),
    Outlet: () => null,
    useChildMatches: () => routerState.childMatches,
    useNavigate: () => navigate,
  };
});

vi.mock("sonner", () => ({
  toast,
}));

import { Route as JobsRoute } from "../jobs";
import { Route as JobDetailRoute } from "../jobs.$jobId";
import { Route as TriggersRoute } from "../triggers";
import { Route as TriggerDetailRoute } from "../triggers.$triggerId";

const JobsPage = routeComponent(JobsRoute);
const TriggersPage = routeComponent(TriggersRoute);
const JobDetailPage = routeComponent(JobDetailRoute);
const TriggerDetailPage = routeComponent(TriggerDetailRoute);
const originalFetch = globalThis.fetch;

const defaultWorkspaces: WorkspacePayload[] = [
  {
    add_dirs: [],
    created_at: "2026-04-17T10:00:00Z",
    id: "ws_alpha",
    name: "Alpha",
    root_dir: "/workspace/alpha",
    updated_at: "2026-04-17T10:00:00Z",
  },
  {
    add_dirs: [],
    created_at: "2026-04-17T10:00:00Z",
    id: "ws_beta",
    name: "Beta",
    root_dir: "/workspace/beta",
    updated_at: "2026-04-17T10:00:00Z",
  },
];

let handlers: HttpHandler[] = [];
let workspaces: WorkspacePayload[] = [];
let jobStore = createStatefulMswStore<AutomationJob>([]);
let triggerStore = createStatefulMswStore<AutomationTrigger>([]);
let runs: AutomationRun[] = [];
let createJobRequests: CreateAutomationJobRequest[] = [];
let createTriggerRequests: CreateAutomationTriggerRequest[] = [];
let updateJobRequests: UpdateAutomationJobRequest[] = [];
let updateTriggerRequests: UpdateAutomationTriggerRequest[] = [];
let createJobResponseOverride:
  | ((body: CreateAutomationJobRequest) => Response | Promise<Response>)
  | null = null;
let createTriggerResponseOverride:
  | ((body: CreateAutomationTriggerRequest) => Response | Promise<Response>)
  | null = null;
let updateJobResponseOverride:
  | ((id: string, body: UpdateAutomationJobRequest) => Response | Promise<Response>)
  | null = null;
let updateTriggerResponseOverride:
  | ((id: string, body: UpdateAutomationTriggerRequest) => Response | Promise<Response>)
  | null = null;

function createQueryClient() {
  return new QueryClient({
    defaultOptions: {
      mutations: { retry: false },
      queries: { retry: false },
    },
  });
}

function renderAutomationPage(page: "jobs" | "triggers") {
  routerState.currentPath = `/${page}`;
  const Page = page === "jobs" ? JobsPage : TriggersPage;
  const queryClient = createQueryClient();
  queryClient.setQueryData(workspaceKeys.list(), workspaces);

  return renderWithTopbar(
    <QueryClientProvider client={queryClient}>
      <Page />
    </QueryClientProvider>
  );
}

function renderAutomationDetail(kind: "jobs" | "triggers", id: string) {
  routerState.currentPath = kind === "jobs" ? `/jobs/${id}` : `/triggers/${id}`;
  routerState.params = kind === "jobs" ? { jobId: id } : { triggerId: id };
  const Page = kind === "jobs" ? JobDetailPage : TriggerDetailPage;
  const queryClient = createQueryClient();
  queryClient.setQueryData(workspaceKeys.list(), workspaces);

  return renderWithTopbar(
    <QueryClientProvider client={queryClient}>
      <Page />
    </QueryClientProvider>
  );
}

function slug(prefix: "job" | "trg", name: string) {
  return `${prefix}_${name.trim().replace(/[^a-zA-Z0-9]+/g, "_")}`;
}

function makeJob(overrides: Partial<AutomationJob> = {}): AutomationJob {
  return {
    ...primaryAutomationJobFixture,
    id: "job_existing",
    name: "existing-job",
    agent_name: "writer",
    prompt: "Summarize launch state.",
    scope: "workspace",
    workspace_id: "ws_alpha",
    source: "dynamic",
    enabled: true,
    retry: { strategy: "none", max_retries: 0, base_delay: "" },
    fire_limit: { max: 12, window: "1h" },
    created_at: "2026-04-17T10:00:00Z",
    updated_at: "2026-04-17T10:00:00Z",
    ...overrides,
  };
}

function createdJobFromBody(body: CreateAutomationJobRequest): AutomationJob {
  return makeJob({
    agent_name: body.agent_name,
    enabled: body.enabled ?? true,
    fire_limit: body.fire_limit ?? { max: 12, window: "1h" },
    id: slug("job", body.name),
    name: body.name,
    next_run: body.schedule.mode === "at" ? body.schedule.time : undefined,
    prompt: body.prompt,
    retry: body.retry ?? { strategy: "none", max_retries: 0, base_delay: "" },
    schedule: body.schedule,
    scheduler: undefined,
    scope: body.scope,
    task: body.task,
    workspace_id: body.workspace_id,
  });
}

function makeTrigger(overrides: Partial<AutomationTrigger> = {}): AutomationTrigger {
  return {
    ...primaryAutomationTriggerFixture,
    id: "trg_existing",
    name: "existing-trigger",
    agent_name: "reviewer",
    prompt: "Review stopped sessions.",
    event: "session.stopped",
    filter: {},
    scope: "workspace",
    workspace_id: "ws_alpha",
    source: "dynamic",
    enabled: true,
    retry: { strategy: "none", max_retries: 0, base_delay: "" },
    fire_limit: { max: 12, window: "1h" },
    webhook_secret_present: false,
    created_at: "2026-04-17T10:00:00Z",
    updated_at: "2026-04-17T10:00:00Z",
    ...overrides,
  };
}

function createdTriggerFromBody(body: CreateAutomationTriggerRequest): AutomationTrigger {
  return makeTrigger({
    agent_name: body.agent_name,
    enabled: body.enabled ?? true,
    endpoint_slug: body.endpoint_slug,
    event: body.event,
    filter: body.filter,
    fire_limit: body.fire_limit ?? { max: 12, window: "1h" },
    id: slug("trg", body.name),
    name: body.name,
    prompt: body.prompt,
    retry: body.retry ?? { strategy: "none", max_retries: 0, base_delay: "" },
    scope: body.scope,
    target_kind: body.target_kind,
    loop_target: body.loop_target,
    webhook_id: body.webhook_id,
    webhook_secret_present: Boolean(body.webhook_secret_value),
    workspace_id: body.workspace_id,
  });
}

function scopedFilterFromRequest(request: Request) {
  const url = new URL(request.url);
  return {
    scope: url.searchParams.get("scope"),
    workspace_id: url.searchParams.get("workspace_id"),
  };
}

function omitNullish<T extends object>(value: T): Partial<T> {
  return Object.fromEntries(
    Object.entries(value).filter(([, entry]) => entry !== null && entry !== undefined)
  ) as Partial<T>;
}

function automationCreateHandlers(): HttpHandler[] {
  const triggerLoops = loopCatalogFixtures.map(loop =>
    loop.name === "reviews-watch"
      ? {
          ...loop,
          inputs: { pr: { type: "number" as const, required: true } },
          start: [{ kind: "trigger" as const }],
        }
      : loop
  );

  return [
    http.get("/api/workspaces", () => HttpResponse.json({ workspaces })),
    http.get("/api/workspaces/:workspaceId/loops", () =>
      HttpResponse.json({
        loops: triggerLoops,
        page: { has_more: false, limit: 50, total: triggerLoops.length },
      })
    ),
    http.get("/api/settings/automation", () => HttpResponse.json(settingsAutomationSectionFixture)),
    http.get("/api/automation/jobs", ({ request }) => {
      const jobs = jobStore.listScoped(scopedFilterFromRequest(request));
      return HttpResponse.json({
        jobs,
        page: { has_more: false, limit: 50, total: jobs.length },
      });
    }),
    http.get("/api/automation/jobs/:id", ({ params }) => {
      const id = String(params.id);
      const job = jobStore.get(id);
      if (!job) {
        return HttpResponse.json({ error: `Automation job not found: ${id}` }, { status: 404 });
      }
      return HttpResponse.json({ job });
    }),
    http.get("/api/automation/jobs/:id/runs", ({ params }) =>
      HttpResponse.json({ runs: runs.filter(run => run.job_id === String(params.id)) })
    ),
    http.post("/api/automation/jobs", async ({ request }) => {
      const body = (await request.json()) as CreateAutomationJobRequest;
      createJobRequests.push(body);
      if (createJobResponseOverride) {
        return createJobResponseOverride(body);
      }

      const job = createdJobFromBody(body);
      jobStore.prepend(job);

      return HttpResponse.json({ job }, { status: 201 });
    }),
    http.patch("/api/automation/jobs/:id", async ({ params, request }) => {
      const id = String(params.id);
      const body = (await request.json()) as UpdateAutomationJobRequest;
      updateJobRequests.push(body);
      if (updateJobResponseOverride) {
        return updateJobResponseOverride(id, body);
      }
      const current = jobStore.get(id);
      if (!current) {
        return HttpResponse.json({ error: `Automation job not found: ${id}` }, { status: 404 });
      }

      const job = jobStore.patch(id, {
        ...(omitNullish(body) as Partial<AutomationJob>),
        updated_at: "2026-04-17T10:10:00Z",
      });
      return HttpResponse.json({ job });
    }),
    http.delete("/api/automation/jobs/:id", ({ params }) => {
      if (!jobStore.delete(String(params.id))) {
        return HttpResponse.json(
          { error: `Automation job not found: ${String(params.id)}` },
          { status: 404 }
        );
      }

      return new HttpResponse(null, { status: 204 });
    }),
    http.post("/api/automation/jobs/:id/trigger", ({ params }) => {
      const id = String(params.id);
      if (!jobStore.get(id)) {
        return HttpResponse.json({ error: `Automation job not found: ${id}` }, { status: 404 });
      }
      return HttpResponse.json({
        run: {
          ...automationRunFixtures[0]!,
          id: `run_${id}_manual`,
          job_id: id,
          session_id: undefined,
          status: "scheduled",
        },
      });
    }),
    http.get("/api/automation/triggers", ({ request }) => {
      const triggers = triggerStore.listScoped(scopedFilterFromRequest(request));
      return HttpResponse.json({
        page: { has_more: false, limit: 50, total: triggers.length },
        triggers,
      });
    }),
    http.get("/api/automation/triggers/:id", ({ params }) => {
      const id = String(params.id);
      const trigger = triggerStore.get(id);
      if (!trigger) {
        return HttpResponse.json({ error: `Automation trigger not found: ${id}` }, { status: 404 });
      }
      return HttpResponse.json({ trigger });
    }),
    http.get("/api/automation/triggers/:id/runs", ({ params }) =>
      HttpResponse.json({ runs: runs.filter(run => run.trigger_id === String(params.id)) })
    ),
    http.post("/api/automation/triggers", async ({ request }) => {
      const body = (await request.json()) as CreateAutomationTriggerRequest;
      createTriggerRequests.push(body);
      if (createTriggerResponseOverride) {
        return createTriggerResponseOverride(body);
      }
      if (
        body.target_kind === "loop" &&
        (!body.loop_target?.workspace_id ||
          (body.scope === "workspace" && body.loop_target.workspace_id !== body.workspace_id))
      ) {
        return HttpResponse.json(
          { error: "Loop target workspace must match the trigger workspace." },
          { status: 422 }
        );
      }

      const trigger = createdTriggerFromBody(body);
      triggerStore.prepend(trigger);

      return HttpResponse.json({ trigger }, { status: 201 });
    }),
    http.patch("/api/automation/triggers/:id", async ({ params, request }) => {
      const id = String(params.id);
      const body = (await request.json()) as UpdateAutomationTriggerRequest;
      updateTriggerRequests.push(body);
      if (updateTriggerResponseOverride) {
        return updateTriggerResponseOverride(id, body);
      }
      const current = triggerStore.get(id);
      if (!current) {
        return HttpResponse.json({ error: `Automation trigger not found: ${id}` }, { status: 404 });
      }

      const trigger = triggerStore.patch(id, {
        ...(omitNullish(body) as Partial<AutomationTrigger>),
        webhook_secret_present: current.webhook_secret_present,
        updated_at: "2026-04-17T10:10:00Z",
      });
      return HttpResponse.json({ trigger });
    }),
    http.delete("/api/automation/triggers/:id", ({ params }) => {
      if (!triggerStore.delete(String(params.id))) {
        return HttpResponse.json(
          { error: `Automation trigger not found: ${String(params.id)}` },
          { status: 404 }
        );
      }

      return new HttpResponse(null, { status: 204 });
    }),
  ];
}

beforeAll(() => {
  globalThis.fetch = createMswFetch(() => handlers);
});

afterAll(() => {
  globalThis.fetch = originalFetch;
});

beforeEach(() => {
  vi.spyOn(useActiveWorkspaceStore.persist, "hasHydrated").mockReturnValue(true);
  workspaces = [...defaultWorkspaces];
  jobStore.reset([makeJob()]);
  triggerStore.reset([makeTrigger()]);
  runs = [...automationRunFixtures];
  createJobRequests = [];
  createTriggerRequests = [];
  updateJobRequests = [];
  updateTriggerRequests = [];
  createJobResponseOverride = null;
  createTriggerResponseOverride = null;
  updateJobResponseOverride = null;
  updateTriggerResponseOverride = null;
  handlers = automationCreateHandlers();
  clipboardWriteText.mockReset();
  clipboardWriteText.mockResolvedValue(undefined);
  routerState.navigateMock.mockReset();
  routerState.searchListeners.clear();
  routerState.searchParams = {};
  routerState.childMatches = [];
  routerState.params = {};
  Object.defineProperty(navigator, "clipboard", {
    configurable: true,
    value: { writeText: clipboardWriteText },
  });
  delete (window as { __agh_job_xss?: boolean }).__agh_job_xss;
  delete (window as { __agh_trigger_xss?: boolean }).__agh_trigger_xss;
  toast.error.mockReset();
  toast.success.mockReset();
  useActiveWorkspaceStore.setState({ selectedWorkspaceId: "ws_alpha" });
  useUserHomeDirStore.getState().setUserHomeDir("/home/operator");
});

function fillJobAgentFields(name: string, agent = "writer", prompt = "Summarize docs changes.") {
  fireEvent.change(screen.getByTestId("job-name-input"), { target: { value: name } });
  fireEvent.change(screen.getByTestId("job-agent-input"), { target: { value: agent } });
  fireEvent.change(screen.getByTestId("job-prompt-input"), {
    target: { value: prompt },
  });
}

function fillJobTaskFields(name: string, title = "Materialized task") {
  fireEvent.change(screen.getByTestId("job-name-input"), { target: { value: name } });
  fireEvent.click(screen.getByTestId("job-target-task"));
  fireEvent.change(screen.getByTestId("job-task-title"), { target: { value: title } });
  fireEvent.change(screen.getByTestId("job-task-desc"), {
    target: { value: `Description for ${title}` },
  });
}

function fillTriggerFields(name: string, agent = "reviewer", prompt = "Review {{ .Kind }}.") {
  fireEvent.change(screen.getByTestId("trigger-name-input"), {
    target: { value: name },
  });
  fireEvent.change(screen.getByTestId("trigger-agent-input"), { target: { value: agent } });
  fireEvent.change(screen.getByTestId("trigger-prompt-input"), {
    target: { value: prompt },
  });
}

function localDatetimeIso(value: string): string {
  const match = value.match(/^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2})$/);
  if (!match) {
    throw new Error(`Invalid datetime-local test value: ${value}`);
  }
  const [, year, month, day, hour, minute] = match;

  return new Date(
    Date.UTC(Number(year), Number(month) - 1, Number(day), Number(hour), Number(minute), 0)
  )
    .toISOString()
    .replace(".000Z", "Z");
}

function selectJobWorkspace(workspaceId: string) {
  fireEvent.click(screen.getByTestId("job-workspace-select"));
  fireEvent.click(screen.getByTestId(`job-workspace-item-${workspaceId}`));
}

function selectTriggerWorkspace(workspaceId: string) {
  fireEvent.click(screen.getByTestId("trigger-workspace-select"));
  fireEvent.click(screen.getByTestId(`trigger-workspace-item-${workspaceId}`));
}

describe("Jobs create modal", () => {
  it("Should open, fill an agent-run cron job, submit through MSW, and show the created job", async () => {
    renderAutomationPage("jobs");

    fireEvent.click(await screen.findByTestId("create-job-btn"));
    expect(screen.getByRole("heading", { name: "Create job" })).toBeInTheDocument();

    fireEvent.change(screen.getByTestId("job-name-input"), { target: { value: "nightly-docs" } });
    fireEvent.change(screen.getByTestId("job-agent-input"), { target: { value: "writer" } });
    fireEvent.change(screen.getByTestId("job-prompt-input"), {
      target: { value: "Summarize docs changes." },
    });
    fireEvent.click(screen.getByRole("button", { name: "Custom" }));
    const cronInput = screen.getByLabelText("Cron expression");
    fireEvent.change(cronInput, { target: { value: "*/15 * * * *" } });

    fireEvent.click(screen.getByTestId("submit-job-form"));

    await waitFor(() => {
      expect(createJobRequests).toHaveLength(1);
    });
    expect(createJobRequests[0]).toEqual(
      expect.objectContaining({
        agent_name: "writer",
        name: "nightly-docs",
        prompt: "Summarize docs changes.",
        schedule: { mode: "cron", expr: "*/15 * * * *" },
        scope: "workspace",
        workspace_id: "ws_alpha",
      })
    );
    await waitFor(() => {
      expect(screen.queryByTestId("automation-editor-dialog")).not.toBeInTheDocument();
    });
    expect(await screen.findByTestId("automation-item-job_nightly_docs")).toBeInTheDocument();
    expect(routerState.navigateMock).toHaveBeenCalledWith({
      to: "/jobs/$jobId",
      params: { jobId: "job_nightly_docs" },
    });
    expect(toast.success).toHaveBeenCalledWith("Created job nightly-docs.");
  });

  it("Should configure at/every schedules and submit a task-run job payload", async () => {
    renderAutomationPage("jobs");

    fireEvent.click(await screen.findByTestId("create-job-btn"));
    fireEvent.change(screen.getByTestId("job-name-input"), {
      target: { value: "materialize-audit" },
    });

    fireEvent.click(screen.getByTestId("job-schedule-mode-at"));
    fireEvent.change(screen.getByLabelText("Run date and time"), {
      target: { value: "2099-01-01T09:00" },
    });
    expect(screen.getByLabelText("Run date and time")).toHaveValue("2099-01-01T09:00");

    fireEvent.click(screen.getByTestId("job-schedule-mode-every"));
    const intervalInput = screen.getByLabelText("Interval");
    fireEvent.change(intervalInput, { target: { value: "45m" } });

    fireEvent.click(screen.getByTestId("job-target-task"));
    fireEvent.change(screen.getByTestId("job-task-title"), {
      target: { value: "Audit payout drift" },
    });
    fireEvent.change(screen.getByTestId("job-task-desc"), {
      target: { value: "Open a task for payout drift review." },
    });
    expect(screen.queryByTestId("job-task-channel")).not.toBeInTheDocument();
    fireEvent.click(screen.getByTestId("job-workspace-select"));
    fireEvent.click(screen.getByTestId("job-workspace-item-ws_beta"));

    fireEvent.click(screen.getByTestId("submit-job-form"));

    await waitFor(() => {
      expect(createJobRequests).toHaveLength(1);
    });
    expect(createJobRequests[0]).toEqual(
      expect.objectContaining({
        name: "materialize-audit",
        retry: { strategy: "none", max_retries: 0, base_delay: "" },
        schedule: { mode: "every", interval: "45m" },
        scope: "workspace",
        task: expect.objectContaining({
          description: "Open a task for payout drift review.",
          title: "Audit payout drift",
        }),
        workspace_id: "ws_beta",
      })
    );
    expect(createJobRequests[0]?.task).not.toHaveProperty("network_channel");
  });

  it("Should keep submit disabled for empty schedules and invalid cron expressions", async () => {
    renderAutomationPage("jobs");

    fireEvent.click(await screen.findByTestId("create-job-btn"));
    expect(screen.getByTestId("submit-job-form")).toBeDisabled();

    fireEvent.change(screen.getByTestId("job-name-input"), { target: { value: "bad-schedule" } });
    fireEvent.change(screen.getByTestId("job-agent-input"), { target: { value: "writer" } });
    fireEvent.change(screen.getByTestId("job-prompt-input"), {
      target: { value: "Summarize docs changes." },
    });
    expect(screen.getByTestId("submit-job-form")).toBeEnabled();

    fireEvent.click(screen.getByTestId("job-schedule-mode-every"));
    const intervalInput = screen.getByLabelText("Interval");
    fireEvent.change(intervalInput, { target: { value: "" } });
    expect(intervalInput).toHaveAttribute("aria-invalid", "true");
    expect(screen.getByTestId("submit-job-form")).toBeDisabled();

    fireEvent.click(screen.getByTestId("job-schedule-mode-cron"));
    fireEvent.click(screen.getByRole("button", { name: "Custom" }));
    const cronInput = screen.getByLabelText("Cron expression");
    fireEvent.change(cronInput, { target: { value: "bad cron" } });
    expect(cronInput).toHaveAttribute("aria-invalid", "true");
    expect(screen.getByTestId("submit-job-form")).toBeDisabled();
    expect(createJobRequests).toEqual([]);
  });

  it("Should keep the job draft open on server failure and allow retry", async () => {
    createJobResponseOverride = () =>
      HttpResponse.json({ error: "automation job store unavailable" }, { status: 500 });
    renderAutomationPage("jobs");

    fireEvent.click(await screen.findByTestId("create-job-btn"));
    fireEvent.change(screen.getByTestId("job-name-input"), { target: { value: "job-retry" } });
    fireEvent.change(screen.getByTestId("job-agent-input"), { target: { value: "writer" } });
    fireEvent.change(screen.getByTestId("job-prompt-input"), {
      target: { value: "Summarize retry state." },
    });

    fireEvent.click(screen.getByTestId("submit-job-form"));

    await waitFor(() => {
      expect(createJobRequests).toHaveLength(1);
    });
    await waitFor(() => {
      expect(toast.error).toHaveBeenCalledWith("automation job store unavailable");
    });
    expect(screen.getByTestId("automation-editor-dialog")).toBeInTheDocument();
    expect(screen.getByTestId("job-name-input")).toHaveValue("job-retry");
    expect(screen.getByTestId("job-agent-input")).toHaveValue("writer");
    expect(screen.getByTestId("job-prompt-input")).toHaveValue("Summarize retry state.");
    expect(screen.queryByTestId("automation-item-job_job_retry")).not.toBeInTheDocument();

    createJobResponseOverride = null;
    fireEvent.click(screen.getByTestId("submit-job-form"));

    await waitFor(() => {
      expect(createJobRequests).toHaveLength(2);
    });
    await waitFor(() => {
      expect(screen.queryByTestId("automation-editor-dialog")).not.toBeInTheDocument();
    });
    expect(await screen.findByTestId("automation-item-job_job_retry")).toBeInTheDocument();
    expect(toast.success).toHaveBeenCalledWith("Created job job-retry.");
  });

  it("Should ignore a rapid second job submit while the first create is pending", async () => {
    createJobResponseOverride = async body => {
      await delay(50);
      const job = createdJobFromBody(body);
      jobStore.prepend(job);
      return HttpResponse.json({ job }, { status: 201 });
    };
    renderAutomationPage("jobs");

    fireEvent.click(await screen.findByTestId("create-job-btn"));
    fireEvent.change(screen.getByTestId("job-name-input"), { target: { value: "single-job" } });
    fireEvent.change(screen.getByTestId("job-agent-input"), { target: { value: "writer" } });
    fireEvent.change(screen.getByTestId("job-prompt-input"), {
      target: { value: "Create once." },
    });

    fireEvent.click(screen.getByTestId("submit-job-form"));
    fireEvent.click(screen.getByTestId("submit-job-form"));

    await waitFor(() => {
      expect(screen.getByTestId("submit-job-form")).toBeDisabled();
    });
    await waitFor(() => {
      expect(createJobRequests).toHaveLength(1);
    });
    await waitFor(() => {
      expect(screen.queryByTestId("automation-editor-dialog")).not.toBeInTheDocument();
    });
    expect(await screen.findByTestId("automation-item-job_single_job")).toBeInTheDocument();
  });

  it("Should create an at-mode job, persist the RFC3339 next run, and navigate to detail", async () => {
    renderAutomationPage("jobs");

    fireEvent.click(await screen.findByTestId("create-job-btn"));
    fillJobAgentFields("one-time-release", "release", "Run the one-time release.");
    fireEvent.click(screen.getByTestId("job-schedule-mode-at"));
    fireEvent.change(screen.getByLabelText("Run date and time"), {
      target: { value: "2099-01-01T09:00" },
    });

    fireEvent.click(screen.getByTestId("submit-job-form"));

    await waitFor(() => {
      expect(createJobRequests).toHaveLength(1);
    });
    const expectedTime = localDatetimeIso("2099-01-01T09:00");
    expect(createJobRequests[0]).toEqual(
      expect.objectContaining({
        name: "one-time-release",
        schedule: { mode: "at", time: expectedTime },
      })
    );
    await waitFor(() => {
      expect(screen.queryByTestId("automation-editor-dialog")).not.toBeInTheDocument();
    });
    expect(await screen.findByTestId("automation-item-job_one_time_release")).toBeInTheDocument();
    expect(jobStore.get("job_one_time_release")?.next_run).toBe(expectedTime);
    expect(routerState.navigateMock).toHaveBeenCalledWith({
      to: "/jobs/$jobId",
      params: { jobId: "job_one_time_release" },
    });
    expect(toast.success).toHaveBeenCalledWith("Created job one-time-release.");
  });

  it("Should disable submit and warn when an at-mode job time is in the past", async () => {
    renderAutomationPage("jobs");

    fireEvent.click(await screen.findByTestId("create-job-btn"));
    fillJobAgentFields("past-job", "writer", "This should not submit.");
    fireEvent.click(screen.getByTestId("job-schedule-mode-at"));
    fireEvent.change(screen.getByLabelText("Run date and time"), {
      target: { value: "2000-01-01T09:00" },
    });

    expect(screen.getByLabelText("Run date and time")).toHaveAttribute("aria-invalid", "true");
    expect(
      screen.getByText("That time is in the past — the job won't register.")
    ).toBeInTheDocument();
    expect(screen.getByTestId("submit-job-form")).toBeDisabled();
    expect(createJobRequests).toEqual([]);
  });

  it.each([
    {
      name: "cron task output",
      jobName: "cron-task-output",
      configure: () => {
        fireEvent.click(screen.getByRole("button", { name: "Custom" }));
        fireEvent.change(screen.getByLabelText("Cron expression"), {
          target: { value: "0 12 * * *" },
        });
        fireEvent.click(screen.getByTestId("job-target-task"));
        fireEvent.change(screen.getByTestId("job-task-title"), {
          target: { value: "Cron task output" },
        });
      },
      expected: {
        schedule: { mode: "cron", expr: "0 12 * * *" },
        task: expect.objectContaining({ title: "Cron task output" }),
      },
    },
    {
      name: "every agent output",
      jobName: "every-agent-output",
      configure: () => {
        fireEvent.click(screen.getByTestId("job-schedule-mode-every"));
        fireEvent.change(screen.getByLabelText("Interval"), { target: { value: "20m" } });
      },
      expected: {
        agent_name: "writer",
        prompt: "Summarize docs changes.",
        schedule: { mode: "every", interval: "20m" },
      },
    },
    {
      name: "at agent output",
      jobName: "at-agent-output",
      configure: () => {
        fireEvent.click(screen.getByTestId("job-schedule-mode-at"));
        fireEvent.change(screen.getByLabelText("Run date and time"), {
          target: { value: "2099-02-01T10:30" },
        });
      },
      expected: {
        agent_name: "writer",
        prompt: "Summarize docs changes.",
        schedule: { mode: "at", time: localDatetimeIso("2099-02-01T10:30") },
      },
    },
    {
      name: "at task output",
      jobName: "at-task-output",
      configure: () => {
        fireEvent.click(screen.getByTestId("job-schedule-mode-at"));
        fireEvent.change(screen.getByLabelText("Run date and time"), {
          target: { value: "2099-03-01T11:45" },
        });
        fireEvent.click(screen.getByTestId("job-target-task"));
        fireEvent.change(screen.getByTestId("job-task-title"), {
          target: { value: "At task output" },
        });
      },
      expected: {
        schedule: { mode: "at", time: localDatetimeIso("2099-03-01T11:45") },
        task: expect.objectContaining({ title: "At task output" }),
      },
    },
  ])(
    "Should submit the schedule/output matrix for $name",
    async ({ jobName, configure, expected }) => {
      renderAutomationPage("jobs");

      fireEvent.click(await screen.findByTestId("create-job-btn"));
      fillJobAgentFields(jobName);
      configure();
      fireEvent.click(screen.getByTestId("submit-job-form"));

      await waitFor(() => {
        expect(createJobRequests).toHaveLength(1);
      });
      expect(createJobRequests[0]).toEqual(expect.objectContaining(expected));
    }
  );

  it("Should isolate workspace-scoped jobs when the active workspace changes", async () => {
    routerState.searchParams = { scope: "workspace" };
    renderAutomationPage("jobs");

    fireEvent.click(await screen.findByTestId("create-job-btn"));
    fillJobAgentFields("alpha-only-job");
    fireEvent.click(screen.getByTestId("submit-job-form"));

    await waitFor(() => {
      expect(createJobRequests).toHaveLength(1);
    });
    expect(createJobRequests[0]).toEqual(
      expect.objectContaining({ scope: "workspace", workspace_id: "ws_alpha" })
    );
    expect(await screen.findByTestId("automation-item-job_alpha_only_job")).toBeInTheDocument();

    useActiveWorkspaceStore.setState({ selectedWorkspaceId: "ws_beta" });

    await waitFor(() => {
      expect(screen.queryByTestId("automation-item-job_alpha_only_job")).not.toBeInTheDocument();
    });

    useActiveWorkspaceStore.setState({ selectedWorkspaceId: "ws_alpha" });

    expect(await screen.findByTestId("automation-item-job_alpha_only_job")).toBeInTheDocument();
  });

  it("Should preserve and escape long HTML pasted into a job prompt", async () => {
    const unsafePrompt = `${"<script>window.__agh_job_xss = true</script>"}${"x".repeat(5_000)}`;
    renderAutomationPage("jobs");

    fireEvent.click(await screen.findByTestId("create-job-btn"));
    fillJobAgentFields("html-paste-job", "writer", unsafePrompt);
    fireEvent.click(screen.getByTestId("submit-job-form"));

    await waitFor(() => {
      expect(createJobRequests).toHaveLength(1);
    });
    expect(createJobRequests[0]?.prompt).toBe(unsafePrompt);
    expect(await screen.findByTestId("automation-item-job_html_paste_job")).toBeInTheDocument();
    // Job rows summarize schedule (not prompt); the invariant here is that the
    // pasted markup never lands in the DOM as elements or executes.
    expect(document.querySelector("script")).toBeNull();
    expect((window as { __agh_job_xss?: boolean }).__agh_job_xss).toBeUndefined();
  });

  it("Should keep a created job visible after remount with a fresh query client", async () => {
    const firstRender = renderAutomationPage("jobs");

    fireEvent.click(await screen.findByTestId("create-job-btn"));
    fillJobAgentFields("refresh-job");
    fireEvent.click(screen.getByTestId("submit-job-form"));

    expect(await screen.findByTestId("automation-item-job_refresh_job")).toBeInTheDocument();
    firstRender.unmount();

    renderAutomationPage("jobs");

    expect(await screen.findByTestId("automation-item-job_refresh_job")).toBeInTheDocument();
  });

  it("Should prefill an existing job in edit mode and patch only the changed values", async () => {
    renderAutomationDetail("jobs", "job_existing");

    expect(await screen.findByTestId("automation-detail-panel")).toBeInTheDocument();
    fireEvent.click(await screen.findByTestId("edit-automation-btn"));

    expect(screen.getByRole("heading", { name: "Edit job" })).toBeInTheDocument();
    expect(screen.getByTestId("job-name-input")).toHaveValue("existing-job");
    expect(screen.getByTestId("job-agent-input")).toHaveValue("writer");
    expect(screen.getByTestId("job-prompt-input")).toHaveValue("Summarize launch state.");

    fireEvent.change(screen.getByTestId("job-prompt-input"), {
      target: { value: "Summarize edited launch state." },
    });
    fireEvent.click(screen.getByTestId("submit-job-form"));

    await waitFor(() => {
      expect(updateJobRequests).toHaveLength(1);
    });
    expect(updateJobRequests[0]).toEqual(
      expect.objectContaining({
        name: "existing-job",
        prompt: "Summarize edited launch state.",
      })
    );
    await waitFor(() => {
      expect(screen.queryByTestId("automation-editor-dialog")).not.toBeInTheDocument();
    });
    expect(screen.getByTestId("automation-detail-panel")).toHaveTextContent(
      "Summarize edited launch state."
    );
    expect(toast.success).toHaveBeenCalledWith("Updated job existing-job.");
  });

  it("Should reset detail-local state when job and trigger route identities change", async () => {
    jobStore.reset([
      makeJob({ id: "job_alpha", name: "alpha-job" }),
      makeJob({ id: "job_beta", name: "beta-job" }),
    ]);
    runs = [];
    routerState.currentPath = "/jobs/job_alpha";
    routerState.params = { jobId: "job_alpha" };
    const queryClient = createQueryClient();
    queryClient.setQueryData(workspaceKeys.list(), workspaces);
    const jobView = renderWithTopbar(
      <QueryClientProvider client={queryClient}>
        <JobDetailPage />
      </QueryClientProvider>
    );

    expect(await screen.findByText("alpha-job")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("trigger-job-btn"));
    expect(await screen.findByTestId("automation-run-run_job_alpha_manual")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("edit-automation-btn"));
    expect(screen.getByRole("heading", { name: "Edit job" })).toBeInTheDocument();

    routerState.params = { jobId: "job_beta" };
    jobView.rerender(
      <QueryClientProvider client={queryClient}>
        <JobDetailPage />
      </QueryClientProvider>
    );

    expect(await screen.findByText("beta-job")).toBeInTheDocument();
    expect(screen.queryByTestId("automation-run-run_job_alpha_manual")).not.toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Edit job" })).not.toBeInTheDocument();
    jobView.unmount();

    triggerStore.reset([
      makeTrigger({ id: "trg_alpha", name: "alpha-trigger" }),
      makeTrigger({ id: "trg_beta", name: "beta-trigger" }),
    ]);
    routerState.currentPath = "/triggers/trg_alpha";
    routerState.params = { triggerId: "trg_alpha" };
    const triggerClient = createQueryClient();
    triggerClient.setQueryData(workspaceKeys.list(), workspaces);
    const triggerView = renderWithTopbar(
      <QueryClientProvider client={triggerClient}>
        <TriggerDetailPage />
      </QueryClientProvider>
    );

    expect(await screen.findByText("alpha-trigger")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("edit-automation-btn"));
    expect(screen.getByRole("heading", { name: "Edit trigger" })).toBeInTheDocument();

    routerState.params = { triggerId: "trg_beta" };
    triggerView.rerender(
      <QueryClientProvider client={triggerClient}>
        <TriggerDetailPage />
      </QueryClientProvider>
    );

    expect(await screen.findByText("beta-trigger")).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Edit trigger" })).not.toBeInTheDocument();
  });

  it("Should disable job task owner ref until kind is selected and submit a specific owner", async () => {
    renderAutomationPage("jobs");

    fireEvent.click(await screen.findByTestId("create-job-btn"));
    fillJobTaskFields("owned-task-job", "Owned task");

    expect(screen.getByTestId("job-owner-ref")).toBeDisabled();
    fireEvent.change(screen.getByTestId("job-owner-kind"), { target: { value: "human" } });
    expect(screen.getByTestId("job-owner-ref")).toBeEnabled();
    fireEvent.change(screen.getByTestId("job-owner-ref"), { target: { value: "pedro" } });
    fireEvent.click(screen.getByTestId("submit-job-form"));

    await waitFor(() => {
      expect(createJobRequests).toHaveLength(1);
    });
    expect(createJobRequests[0]?.task?.owner).toEqual({ kind: "human", ref: "pedro" });
  });

  it("Should submit the last-selected job workspace after filling the form first", async () => {
    renderAutomationPage("jobs");

    fireEvent.click(await screen.findByTestId("create-job-btn"));
    fillJobAgentFields("late-workspace-job");
    selectJobWorkspace("ws_beta");
    fireEvent.click(screen.getByTestId("submit-job-form"));

    await waitFor(() => {
      expect(createJobRequests).toHaveLength(1);
    });
    expect(createJobRequests[0]).toEqual(
      expect.objectContaining({
        name: "late-workspace-job",
        scope: "workspace",
        workspace_id: "ws_beta",
      })
    );
  });

  it("Should keep a job draft open through a network drop and retry without a phantom row", async () => {
    createJobResponseOverride = () => HttpResponse.error();
    renderAutomationPage("jobs");

    fireEvent.click(await screen.findByTestId("create-job-btn"));
    fillJobAgentFields("network-drop-job");
    fireEvent.click(screen.getByTestId("submit-job-form"));

    await waitFor(() => {
      expect(toast.error).toHaveBeenCalled();
    });
    expect(screen.getByTestId("automation-editor-dialog")).toBeInTheDocument();
    expect(screen.queryByTestId("automation-item-job_network_drop_job")).not.toBeInTheDocument();

    createJobResponseOverride = null;
    fireEvent.click(screen.getByTestId("submit-job-form"));

    expect(await screen.findByTestId("automation-item-job_network_drop_job")).toBeInTheDocument();
  });

  it("Should support keyboard-only job creation and Escape cancellation", async () => {
    const user = userEvent.setup();
    renderAutomationPage("jobs");

    screen.getByTestId("create-job-btn").focus();
    await user.keyboard("{Enter}");
    expect(await screen.findByRole("heading", { name: "Create job" })).toBeInTheDocument();
    await user.keyboard("{Escape}");
    await waitFor(() => {
      expect(screen.queryByTestId("automation-editor-dialog")).not.toBeInTheDocument();
    });
    expect(screen.getByTestId("create-job-btn")).toBeEnabled();

    screen.getByTestId("create-job-btn").focus();
    await user.keyboard("{Enter}");
    expect(await screen.findByRole("heading", { name: "Create job" })).toBeInTheDocument();
    screen.getByTestId("job-name-input").focus();
    await user.keyboard("keyboard-job");
    screen.getByTestId("job-agent-input").focus();
    await user.keyboard("writer");
    screen.getByTestId("job-prompt-input").focus();
    await user.keyboard("Created from keyboard.");
    screen.getByTestId("submit-job-form").focus();
    await user.keyboard("{Enter}");

    await waitFor(() => {
      expect(createJobRequests).toHaveLength(1);
    });
    expect(await screen.findByTestId("automation-item-job_keyboard_job")).toBeInTheDocument();
  });
});

describe("Triggers create modal", () => {
  it("Should open, fill a workspace-scoped extension trigger, submit through MSW, and show it", async () => {
    renderAutomationPage("triggers");

    fireEvent.click(await screen.findByTestId("create-trigger-btn"));
    expect(screen.getByRole("heading", { name: "Create trigger" })).toBeInTheDocument();

    fireEvent.change(screen.getByTestId("trigger-name-input"), {
      target: { value: "github-push-digest" },
    });
    fireEvent.click(screen.getByTestId("trigger-workspace-select"));
    fireEvent.click(screen.getByTestId("trigger-workspace-item-ws_beta"));
    fireEvent.click(screen.getByTestId("trigger-event-ext"));
    expect(screen.getByTestId("automation-editor-dialog")).toBeInTheDocument();
    expect(screen.getByTestId("trigger-ext-ext-input")).toBeInTheDocument();
    expect(screen.getByTestId("submit-trigger-form")).toBeDisabled();

    fireEvent.change(screen.getByTestId("trigger-ext-ext-input"), { target: { value: "github" } });
    fireEvent.change(screen.getByTestId("trigger-ext-event-input"), { target: { value: "push" } });
    fireEvent.click(screen.getByRole("button", { name: "Add condition" }));
    fireEvent.change(screen.getByTestId("trigger-filter-key-0"), {
      target: { value: "data.branch" },
    });
    fireEvent.change(screen.getByTestId("trigger-filter-value-0"), {
      target: { value: "main" },
    });
    fireEvent.change(screen.getByTestId("trigger-agent-input"), { target: { value: "reviewer" } });
    fireEvent.change(screen.getByTestId("trigger-prompt-input"), {
      target: { value: "Review {{ .Kind }} on main." },
    });

    fireEvent.click(screen.getByTestId("submit-trigger-form"));

    await waitFor(() => {
      expect(createTriggerRequests).toHaveLength(1);
    });
    expect(createTriggerRequests[0]).toEqual(
      expect.objectContaining({
        agent_name: "reviewer",
        event: "ext.github.push",
        filter: { "data.branch": "main" },
        name: "github-push-digest",
        prompt: "Review {{ .Kind }} on main.",
        scope: "workspace",
        workspace_id: "ws_beta",
      })
    );
    await waitFor(() => {
      expect(screen.queryByTestId("automation-editor-dialog")).not.toBeInTheDocument();
    });
    expect(await screen.findByTestId("automation-item-trg_github_push_digest")).toBeInTheDocument();
    expect(toast.success).toHaveBeenCalledWith("Created trigger github-push-digest.");
  });

  it("Should create a global webhook trigger with endpoint identity and secret payload", async () => {
    renderAutomationPage("triggers");

    fireEvent.click(await screen.findByTestId("create-trigger-btn"));
    fireEvent.change(screen.getByTestId("trigger-name-input"), {
      target: { value: "ci-webhook" },
    });
    fireEvent.click(screen.getByTestId("trigger-workspace-select"));
    fireEvent.click(screen.getByTestId("trigger-workspace-item-ws_beta"));
    fireEvent.click(screen.getByTestId("trigger-event-webhook"));
    expect(screen.queryByTestId("trigger-workspace-select")).not.toBeInTheDocument();
    expect(screen.getByTestId("submit-trigger-form")).toBeDisabled();

    fireEvent.change(screen.getByTestId("trigger-endpoint-slug-input"), {
      target: { value: "ci-deploys" },
    });
    fireEvent.change(screen.getByTestId("trigger-webhook-id-input"), {
      target: { value: "wbh_ci_deploys" },
    });
    fireEvent.change(screen.getByTestId("trigger-webhook-secret-value-input"), {
      target: { value: "whsec_ci_deploys" },
    });
    fireEvent.change(screen.getByTestId("trigger-agent-input"), { target: { value: "release" } });
    fireEvent.change(screen.getByTestId("trigger-prompt-input"), {
      target: { value: "Inspect {{ .Kind }} before deploy." },
    });
    await waitFor(() => {
      expect(screen.getByTestId("submit-trigger-form")).toBeEnabled();
    });

    fireEvent.click(screen.getByTestId("submit-trigger-form"));

    await waitFor(() => {
      expect(createTriggerRequests).toHaveLength(1);
    });
    expect(createTriggerRequests[0]).toEqual(
      expect.objectContaining({
        agent_name: "release",
        endpoint_slug: "ci-deploys",
        event: "webhook",
        name: "ci-webhook",
        prompt: "Inspect {{ .Kind }} before deploy.",
        scope: "global",
        webhook_id: "wbh_ci_deploys",
        webhook_secret_value: "whsec_ci_deploys",
      })
    );
    expect(createTriggerRequests[0]).not.toHaveProperty("workspace_id");
    await waitFor(() => {
      expect(screen.queryByTestId("automation-editor-dialog")).not.toBeInTheDocument();
    });
    expect(await screen.findByTestId("automation-item-trg_ci_webhook")).toBeInTheDocument();
    expect(routerState.navigateMock).toHaveBeenCalledWith({
      to: "/triggers/$triggerId",
      params: { triggerId: "trg_ci_webhook" },
    });
    expect(toast.success).toHaveBeenCalledWith("Created trigger ci-webhook.");
  });

  it("Should expose the webhook endpoint identity and copy control on the trigger detail", async () => {
    triggerStore.reset([
      makeTrigger({
        agent_name: "release",
        endpoint_slug: "ci-deploys",
        event: "webhook",
        id: "trg_ci_webhook",
        name: "ci-webhook",
        scope: "global",
        webhook_id: "wbh_ci_deploys",
        webhook_secret_present: true,
        workspace_id: undefined,
      }),
    ]);
    renderAutomationDetail("triggers", "trg_ci_webhook");

    expect(await screen.findByTestId("automation-detail-panel")).toBeInTheDocument();
    expect(screen.getByText("Webhook endpoint")).toBeInTheDocument();
    expect(screen.getByText("/api/webhooks/global/ci-deploys--wbh_ci_deploys")).toBeInTheDocument();
    expect(screen.getByText("curl")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Copy webhook URL" }));
    expect(clipboardWriteText).toHaveBeenCalledWith(
      "/api/webhooks/global/ci-deploys--wbh_ci_deploys"
    );
  });

  it("Should keep the trigger draft open on server failure and allow retry", async () => {
    createTriggerResponseOverride = () =>
      HttpResponse.json({ error: "automation trigger store unavailable" }, { status: 500 });
    renderAutomationPage("triggers");

    fireEvent.click(await screen.findByTestId("create-trigger-btn"));
    fireEvent.change(screen.getByTestId("trigger-name-input"), {
      target: { value: "store-retry" },
    });
    fireEvent.click(screen.getByTestId("trigger-event-ext"));
    fireEvent.change(screen.getByTestId("trigger-ext-ext-input"), { target: { value: "github" } });
    fireEvent.change(screen.getByTestId("trigger-ext-event-input"), { target: { value: "push" } });
    fireEvent.change(screen.getByTestId("trigger-agent-input"), { target: { value: "reviewer" } });
    fireEvent.change(screen.getByTestId("trigger-prompt-input"), {
      target: { value: "Review failed create retry." },
    });

    fireEvent.click(screen.getByTestId("submit-trigger-form"));

    await waitFor(() => {
      expect(createTriggerRequests).toHaveLength(1);
    });
    await waitFor(() => {
      expect(toast.error).toHaveBeenCalledWith("automation trigger store unavailable");
    });
    expect(screen.getByTestId("automation-editor-dialog")).toBeInTheDocument();
    expect(screen.getByTestId("trigger-name-input")).toHaveValue("store-retry");
    expect(screen.getByTestId("trigger-ext-ext-input")).toHaveValue("github");
    expect(screen.getByTestId("trigger-ext-event-input")).toHaveValue("push");
    expect(screen.getByTestId("trigger-agent-input")).toHaveValue("reviewer");
    expect(screen.getByTestId("trigger-prompt-input")).toHaveValue("Review failed create retry.");
    expect(screen.queryByTestId("automation-item-trg_store_retry")).not.toBeInTheDocument();

    createTriggerResponseOverride = null;
    fireEvent.click(screen.getByTestId("submit-trigger-form"));

    await waitFor(() => {
      expect(createTriggerRequests).toHaveLength(2);
    });
    await waitFor(() => {
      expect(screen.queryByTestId("automation-editor-dialog")).not.toBeInTheDocument();
    });
    expect(await screen.findByTestId("automation-item-trg_store_retry")).toBeInTheDocument();
    expect(toast.success).toHaveBeenCalledWith("Created trigger store-retry.");
  });

  it("Should keep the trigger draft open on conflict without creating an item", async () => {
    createTriggerResponseOverride = () =>
      HttpResponse.json({ error: "automation trigger name already exists" }, { status: 409 });
    renderAutomationPage("triggers");

    fireEvent.click(await screen.findByTestId("create-trigger-btn"));
    fireEvent.change(screen.getByTestId("trigger-name-input"), {
      target: { value: "duplicate-trigger" },
    });
    fireEvent.change(screen.getByTestId("trigger-agent-input"), { target: { value: "reviewer" } });
    fireEvent.change(screen.getByTestId("trigger-prompt-input"), {
      target: { value: "Review duplicate guard." },
    });

    fireEvent.click(screen.getByTestId("submit-trigger-form"));

    await waitFor(() => {
      expect(createTriggerRequests).toHaveLength(1);
    });
    expect(toast.error).toHaveBeenCalledWith("automation trigger name already exists");
    expect(screen.getByTestId("automation-editor-dialog")).toBeInTheDocument();
    expect(screen.getByTestId("trigger-name-input")).toHaveValue("duplicate-trigger");
    expect(screen.queryByTestId("automation-item-trg_duplicate_trigger")).not.toBeInTheDocument();
    expect(toast.success).not.toHaveBeenCalled();
  });

  it("Should ignore a rapid second trigger submit while the first create is pending", async () => {
    createTriggerResponseOverride = async body => {
      await delay(50);
      const trigger = createdTriggerFromBody(body);
      triggerStore.prepend(trigger);
      return HttpResponse.json({ trigger }, { status: 201 });
    };
    renderAutomationPage("triggers");

    fireEvent.click(await screen.findByTestId("create-trigger-btn"));
    fireEvent.change(screen.getByTestId("trigger-name-input"), {
      target: { value: "single-submit" },
    });
    fireEvent.click(screen.getByTestId("trigger-event-ext"));
    fireEvent.change(screen.getByTestId("trigger-ext-ext-input"), { target: { value: "github" } });
    fireEvent.change(screen.getByTestId("trigger-ext-event-input"), { target: { value: "push" } });
    fireEvent.change(screen.getByTestId("trigger-agent-input"), { target: { value: "reviewer" } });
    fireEvent.change(screen.getByTestId("trigger-prompt-input"), {
      target: { value: "Review one create only." },
    });

    fireEvent.click(screen.getByTestId("submit-trigger-form"));
    fireEvent.click(screen.getByTestId("submit-trigger-form"));

    await waitFor(() => {
      expect(screen.getByTestId("submit-trigger-form")).toBeDisabled();
    });
    await waitFor(() => {
      expect(createTriggerRequests).toHaveLength(1);
    });
    await waitFor(() => {
      expect(screen.queryByTestId("automation-editor-dialog")).not.toBeInTheDocument();
    });
    expect(await screen.findByTestId("automation-item-trg_single_submit")).toBeInTheDocument();
  });

  it("Should isolate workspace-scoped triggers when the active workspace changes", async () => {
    routerState.searchParams = { scope: "workspace" };
    renderAutomationPage("triggers");

    fireEvent.click(await screen.findByTestId("create-trigger-btn"));
    fillTriggerFields("alpha-only-trigger", "reviewer", "Review alpha events.");
    fireEvent.click(screen.getByTestId("trigger-event-ext"));
    fireEvent.change(screen.getByTestId("trigger-ext-ext-input"), { target: { value: "github" } });
    fireEvent.change(screen.getByTestId("trigger-ext-event-input"), { target: { value: "push" } });
    fireEvent.click(screen.getByTestId("submit-trigger-form"));

    await waitFor(() => {
      expect(createTriggerRequests).toHaveLength(1);
    });
    expect(createTriggerRequests[0]).toEqual(
      expect.objectContaining({ scope: "workspace", workspace_id: "ws_alpha" })
    );
    expect(await screen.findByTestId("automation-item-trg_alpha_only_trigger")).toBeInTheDocument();

    useActiveWorkspaceStore.setState({ selectedWorkspaceId: "ws_beta" });

    await waitFor(() => {
      expect(
        screen.queryByTestId("automation-item-trg_alpha_only_trigger")
      ).not.toBeInTheDocument();
    });

    useActiveWorkspaceStore.setState({ selectedWorkspaceId: "ws_alpha" });

    expect(await screen.findByTestId("automation-item-trg_alpha_only_trigger")).toBeInTheDocument();
  });

  it("Should edit a dynamic webhook trigger without requiring the signing secret", async () => {
    triggerStore.reset([
      makeTrigger({
        agent_name: "old-agent",
        endpoint_slug: "edit-hook",
        event: "webhook",
        id: "trg_webhook_edit",
        name: "webhook-edit",
        prompt: "Old webhook prompt.",
        scope: "global",
        webhook_id: "wbh_edit",
        webhook_secret_present: true,
        workspace_id: undefined,
      }),
    ]);
    renderAutomationDetail("triggers", "trg_webhook_edit");

    expect(await screen.findByTestId("automation-detail-panel")).toBeInTheDocument();
    fireEvent.click(await screen.findByTestId("edit-automation-btn"));

    expect(screen.getByRole("heading", { name: "Edit trigger" })).toBeInTheDocument();
    expect(screen.getByTestId("trigger-name-input")).toHaveValue("webhook-edit");
    expect(screen.getByTestId("trigger-endpoint-slug-input")).toHaveValue("edit-hook");
    expect(screen.getByTestId("trigger-webhook-id-input")).toHaveValue("wbh_edit");
    expect(screen.getByTestId("trigger-webhook-secret-value-input")).toHaveValue("");
    expect(screen.getByTestId("trigger-agent-input")).toHaveValue("old-agent");
    expect(screen.getByTestId("trigger-agent-input")).toBeDisabled();
    expect(screen.getByTestId("submit-trigger-form")).toBeEnabled();

    fireEvent.change(screen.getByTestId("trigger-prompt-input"), {
      target: { value: "Updated webhook prompt." },
    });
    fireEvent.click(screen.getByTestId("submit-trigger-form"));

    await waitFor(() => {
      expect(updateTriggerRequests).toHaveLength(1);
    });
    expect(updateTriggerRequests[0]).toEqual(
      expect.objectContaining({
        endpoint_slug: "edit-hook",
        event: "webhook",
        prompt: "Updated webhook prompt.",
        webhook_id: "wbh_edit",
      })
    );
    expect(updateTriggerRequests[0]).not.toHaveProperty("agent_name");
    expect(updateTriggerRequests[0]).not.toHaveProperty("scope");
    expect(updateTriggerRequests[0]).not.toHaveProperty("target_kind");
    expect(updateTriggerRequests[0]).not.toHaveProperty("workspace_id");
    expect(updateTriggerRequests[0]).not.toHaveProperty("webhook_secret_value");
    await waitFor(() => {
      expect(screen.queryByTestId("automation-editor-dialog")).not.toBeInTheDocument();
    });
    expect(screen.getByTestId("automation-detail-panel")).toHaveTextContent(
      "Updated webhook prompt."
    );
    expect(toast.success).toHaveBeenCalledWith("Updated trigger webhook-edit.");
  });

  it("Should preserve and escape long HTML pasted into a trigger prompt", async () => {
    const unsafePrompt = `${"<script>window.__agh_trigger_xss = true</script>"}${"x".repeat(5_000)}`;
    renderAutomationPage("triggers");

    fireEvent.click(await screen.findByTestId("create-trigger-btn"));
    fillTriggerFields("html-paste-trigger", "reviewer", unsafePrompt);
    fireEvent.click(screen.getByTestId("submit-trigger-form"));

    await waitFor(() => {
      expect(createTriggerRequests).toHaveLength(1);
    });
    expect(createTriggerRequests[0]?.prompt).toBe(unsafePrompt);
    const triggerRow = await screen.findByTestId("automation-item-trg_html_paste_trigger");
    // The markup must render as escaped text in the row preview, never as DOM.
    expect(triggerRow).toHaveTextContent("<script>window.__agh_trigger_xss = true</script>");
    expect(document.querySelector("script")).toBeNull();
    expect((window as { __agh_trigger_xss?: boolean }).__agh_trigger_xss).toBeUndefined();
  });

  it("Should keep a created trigger visible after remount with a fresh query client", async () => {
    const firstRender = renderAutomationPage("triggers");

    fireEvent.click(await screen.findByTestId("create-trigger-btn"));
    fillTriggerFields("refresh-trigger");
    fireEvent.click(screen.getByTestId("submit-trigger-form"));

    expect(await screen.findByTestId("automation-item-trg_refresh_trigger")).toBeInTheDocument();
    firstRender.unmount();

    renderAutomationPage("triggers");

    expect(await screen.findByTestId("automation-item-trg_refresh_trigger")).toBeInTheDocument();
  });

  it("Should create a trigger on the catalog then navigate to its detail", async () => {
    renderAutomationPage("triggers");

    fireEvent.click(await screen.findByTestId("create-trigger-btn"));
    fillTriggerFields("continuity-trigger", "reviewer", "Original trigger prompt.");
    fireEvent.click(screen.getByTestId("submit-trigger-form"));

    expect(await screen.findByTestId("automation-item-trg_continuity_trigger")).toBeInTheDocument();
    expect(routerState.navigateMock).toHaveBeenCalledWith({
      to: "/triggers/$triggerId",
      params: { triggerId: "trg_continuity_trigger" },
    });
    expect(triggerStore.get("trg_continuity_trigger")).toBeDefined();
  });

  it("Should edit and delete a trigger through detail-route stateful reads", async () => {
    triggerStore.reset([
      makeTrigger({
        agent_name: "reviewer",
        event: "session.stopped",
        id: "trg_continuity_trigger",
        name: "continuity-trigger",
        prompt: "Original trigger prompt.",
      }),
    ]);
    renderAutomationDetail("triggers", "trg_continuity_trigger");

    expect(await screen.findByTestId("automation-detail-panel")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("edit-automation-btn"));
    expect(screen.getByTestId("trigger-name-input")).toHaveValue("continuity-trigger");
    expect(screen.getByTestId("trigger-prompt-input")).toHaveValue("Original trigger prompt.");

    fireEvent.change(screen.getByTestId("trigger-prompt-input"), {
      target: { value: "Edited trigger prompt." },
    });
    fireEvent.click(screen.getByTestId("submit-trigger-form"));

    await waitFor(() => {
      expect(updateTriggerRequests).toHaveLength(1);
    });
    expect(screen.getByTestId("automation-detail-panel")).toHaveTextContent(
      "Edited trigger prompt."
    );

    fireEvent.click(screen.getByTestId("automation-detail-overflow"));
    fireEvent.click(screen.getByTestId("delete-automation-btn"));
    expect(triggerStore.get("trg_continuity_trigger")).toBeDefined();
    fireEvent.change(screen.getByTestId("automation-delete-confirm-typing"), {
      target: { value: "continuity-trigger" },
    });
    fireEvent.click(screen.getByTestId("confirm-delete-automation-btn"));

    await waitFor(() => {
      expect(triggerStore.get("trg_continuity_trigger")).toBeUndefined();
    });
    expect(routerState.navigateMock).toHaveBeenCalledWith({ to: "/triggers", replace: true });
  });

  it("Should submit the last-selected trigger workspace after filling the form first", async () => {
    renderAutomationPage("triggers");

    fireEvent.click(await screen.findByTestId("create-trigger-btn"));
    fillTriggerFields("late-workspace-trigger");
    selectTriggerWorkspace("ws_beta");
    fireEvent.click(screen.getByTestId("submit-trigger-form"));

    await waitFor(() => {
      expect(createTriggerRequests).toHaveLength(1);
    });
    expect(createTriggerRequests[0]).toEqual(
      expect.objectContaining({
        name: "late-workspace-trigger",
        scope: "workspace",
        workspace_id: "ws_beta",
      })
    );
  });

  it("Should create a workspace Trigger Loop target after leaving webhook scope", async () => {
    renderAutomationPage("triggers");

    fireEvent.click(await screen.findByTestId("create-trigger-btn"));
    fireEvent.change(screen.getByTestId("trigger-name-input"), {
      target: { value: "review-on-stop" },
    });
    fireEvent.click(screen.getByTestId("trigger-event-webhook"));
    fireEvent.click(screen.getByTestId("trigger-event-session.stopped"));
    fireEvent.click(screen.getByTestId("target-mode-loop"));
    fireEvent.change(await screen.findByTestId("loop-target-select"), {
      target: { value: "reviews-watch" },
    });
    fireEvent.change(screen.getByTestId("loop-input-field-pr"), {
      target: { value: "2" },
    });
    fireEvent.click(screen.getByTestId("trigger-scope-workspace"));
    selectTriggerWorkspace("ws_beta");

    expect(screen.getByTestId("automation-target-details")).toHaveTextContent("ws_beta");
    expect(screen.getByTestId("automation-request-payload")).toHaveTextContent(
      '"workspace_id": "ws_beta"'
    );
    await waitFor(() => {
      expect(screen.getByTestId("submit-trigger-form")).toBeEnabled();
    });
    fireEvent.click(screen.getByTestId("submit-trigger-form"));

    await waitFor(() => {
      expect(createTriggerRequests).toHaveLength(1);
    });
    expect(createTriggerRequests[0]).toEqual(
      expect.objectContaining({
        loop_target: expect.objectContaining({
          input_mapping: {},
          inputs: { pr: 2 },
          loop_name: "reviews-watch",
          network_participation: { mode: "local" },
          workspace_id: "ws_beta",
        }),
        name: "review-on-stop",
        scope: "workspace",
        target_kind: "loop",
        workspace_id: "ws_beta",
      })
    );
    expect(await screen.findByTestId("automation-item-trg_review_on_stop")).toBeInTheDocument();
    expect(screen.queryByTestId("automation-editor-dialog")).not.toBeInTheDocument();
  });

  it("Should keep a trigger draft open through a network drop and retry without a phantom row", async () => {
    createTriggerResponseOverride = () => HttpResponse.error();
    renderAutomationPage("triggers");

    fireEvent.click(await screen.findByTestId("create-trigger-btn"));
    fillTriggerFields("network-drop-trigger");
    fireEvent.click(screen.getByTestId("submit-trigger-form"));

    await waitFor(() => {
      expect(toast.error).toHaveBeenCalled();
    });
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Failed to save automation trigger. Check your connection and try again."
    );
    expect(screen.getByTestId("automation-editor-dialog")).toBeInTheDocument();
    expect(
      screen.queryByTestId("automation-item-trg_network_drop_trigger")
    ).not.toBeInTheDocument();

    createTriggerResponseOverride = null;
    fireEvent.click(screen.getByTestId("submit-trigger-form"));

    expect(
      await screen.findByTestId("automation-item-trg_network_drop_trigger")
    ).toBeInTheDocument();
  });

  it("Should support keyboard-only trigger creation and Escape cancellation", async () => {
    const user = userEvent.setup();
    renderAutomationPage("triggers");

    screen.getByTestId("create-trigger-btn").focus();
    await user.keyboard("{Enter}");
    expect(await screen.findByRole("heading", { name: "Create trigger" })).toBeInTheDocument();
    await user.keyboard("{Escape}");
    await waitFor(() => {
      expect(screen.queryByTestId("automation-editor-dialog")).not.toBeInTheDocument();
    });
    expect(screen.getByTestId("create-trigger-btn")).toBeEnabled();

    screen.getByTestId("create-trigger-btn").focus();
    await user.keyboard("{Enter}");
    expect(await screen.findByRole("heading", { name: "Create trigger" })).toBeInTheDocument();
    screen.getByTestId("trigger-name-input").focus();
    await user.keyboard("keyboard-trigger");
    screen.getByTestId("trigger-agent-input").focus();
    await user.keyboard("reviewer");
    screen.getByTestId("trigger-prompt-input").focus();
    await user.keyboard("Created from keyboard.");
    screen.getByTestId("submit-trigger-form").focus();
    await user.keyboard("{Enter}");

    await waitFor(() => {
      expect(createTriggerRequests).toHaveLength(1);
    });
    expect(await screen.findByTestId("automation-item-trg_keyboard_trigger")).toBeInTheDocument();
  });

  it("Should reopen trigger creation with a blank draft after cancel", async () => {
    renderAutomationPage("triggers");

    fireEvent.click(await screen.findByTestId("create-trigger-btn"));
    fireEvent.change(screen.getByTestId("trigger-name-input"), {
      target: { value: "cancelled-trigger" },
    });
    fireEvent.click(screen.getByTestId("trigger-event-ext"));
    fireEvent.change(screen.getByTestId("trigger-ext-ext-input"), { target: { value: "github" } });
    fireEvent.change(screen.getByTestId("trigger-ext-event-input"), { target: { value: "push" } });
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    await waitFor(() => {
      expect(screen.queryByTestId("automation-editor-dialog")).not.toBeInTheDocument();
    });

    fireEvent.click(screen.getByTestId("create-trigger-btn"));

    expect(await screen.findByRole("heading", { name: "Create trigger" })).toBeInTheDocument();
    expect(screen.getByTestId("trigger-name-input")).toHaveValue("");
    expect(screen.queryByTestId("trigger-ext-ext-input")).not.toBeInTheDocument();
    expect(screen.getByTestId("trigger-event-session.stopped")).toHaveAttribute(
      "aria-pressed",
      "true"
    );
    expect(createTriggerRequests).toEqual([]);
  });

  it("Should block incomplete event selection and missing workspace scope", async () => {
    const { unmount } = renderAutomationPage("triggers");

    fireEvent.click(await screen.findByTestId("create-trigger-btn"));
    fireEvent.change(screen.getByTestId("trigger-name-input"), {
      target: { value: "incomplete-ext" },
    });
    fireEvent.change(screen.getByTestId("trigger-agent-input"), { target: { value: "reviewer" } });
    fireEvent.change(screen.getByTestId("trigger-prompt-input"), {
      target: { value: "Review {{ .Kind }}." },
    });
    expect(screen.getByTestId("submit-trigger-form")).toBeEnabled();

    fireEvent.click(screen.getByTestId("trigger-event-ext"));
    expect(screen.getByTestId("submit-trigger-form")).toBeDisabled();
    expect(createTriggerRequests).toEqual([]);
    unmount();

    workspaces = [];
    handlers = automationCreateHandlers();
    renderAutomationPage("triggers");

    fireEvent.click(await screen.findByTestId("create-trigger-btn"));
    fireEvent.change(screen.getByTestId("trigger-name-input"), {
      target: { value: "missing-workspace" },
    });
    fireEvent.change(screen.getByTestId("trigger-agent-input"), { target: { value: "reviewer" } });
    fireEvent.change(screen.getByTestId("trigger-prompt-input"), {
      target: { value: "Review {{ .Kind }}." },
    });
    fireEvent.click(screen.getByTestId("trigger-scope-workspace"));

    expect(screen.getByTestId("trigger-workspace-select")).toBeDisabled();
    expect(screen.getByTestId("submit-trigger-form")).toBeDisabled();
    expect(createTriggerRequests).toEqual([]);
  });
});

describe("Create modal cancellation", () => {
  it("Should cancel job and trigger creation without posting", async () => {
    const jobsRender = renderAutomationPage("jobs");

    fireEvent.click(await screen.findByTestId("create-job-btn"));
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    await waitFor(() => {
      expect(screen.queryByTestId("automation-editor-dialog")).not.toBeInTheDocument();
    });
    jobsRender.unmount();

    const triggersRender = renderAutomationPage("triggers");
    fireEvent.click(await screen.findByTestId("create-trigger-btn"));
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    await waitFor(() => {
      expect(screen.queryByTestId("automation-editor-dialog")).not.toBeInTheDocument();
    });
    triggersRender.unmount();

    expect(createJobRequests).toEqual([]);
    expect(createTriggerRequests).toEqual([]);
  });
});
