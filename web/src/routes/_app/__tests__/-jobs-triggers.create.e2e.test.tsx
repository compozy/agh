// Suite: automation create modal route E2E
// Invariant: jobs and triggers create modals submit the visible draft through real page hooks, query hooks, adapters, and the selected workspace scope.
// Boundary IN: jobs/triggers routes, AutomationOperationsPage, editor dialog/forms, TanStack Query hooks, and openapi-fetch.
// Boundary OUT: AGH daemon HTTP implementation, replaced by MSW handlers at the fetch boundary.

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { http, HttpResponse, type HttpHandler } from "msw";
import type { AnchorHTMLAttributes, ReactNode } from "react";
import { afterAll, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

import type {
  AutomationJob,
  AutomationRun,
  AutomationTrigger,
  CreateAutomationJobRequest,
  CreateAutomationTriggerRequest,
} from "@/systems/automation";
import {
  automationRunFixtures,
  primaryAutomationJobFixture,
  primaryAutomationTriggerFixture,
} from "@/systems/automation/mocks/fixtures";
import { settingsAutomationSectionFixture } from "@/systems/settings/mocks/fixtures";
import { workspaceKeys } from "@/systems/workspace";
import { useActiveWorkspaceStore } from "@/systems/workspace/hooks/use-active-workspace-store";
import { useUserHomeDirStore } from "@/systems/workspace/hooks/use-user-home-dir-store";
import type { WorkspacePayload } from "@/systems/workspace/types";
import { createMswFetch } from "@/test/msw-fetch";
import { renderWithTopbar } from "@/test/render-with-topbar";
import { routeComponent } from "@/test/route-options";

const { toast } = vi.hoisted(() => ({
  toast: {
    error: vi.fn(),
    success: vi.fn(),
  },
}));

interface MockLinkParams {
  id?: string;
}

interface MockLinkProps extends AnchorHTMLAttributes<HTMLAnchorElement> {
  params?: MockLinkParams;
}

vi.mock("@tanstack/react-router", () => ({
  createFileRoute: () => (opts: { component: () => ReactNode }) => ({
    component: opts.component,
  }),
  Link: ({ children, params, ...props }: MockLinkProps) => (
    <a href={`/session/${params?.id ?? ""}`} {...props}>
      {children}
    </a>
  ),
}));

vi.mock("sonner", () => ({
  toast,
}));

import { Route as JobsRoute } from "../jobs";
import { Route as TriggersRoute } from "../triggers";

const JobsPage = routeComponent(JobsRoute);
const TriggersPage = routeComponent(TriggersRoute);
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
let jobs: AutomationJob[] = [];
let triggers: AutomationTrigger[] = [];
let runs: AutomationRun[] = [];
let createJobRequests: CreateAutomationJobRequest[] = [];
let createTriggerRequests: CreateAutomationTriggerRequest[] = [];

function createQueryClient() {
  return new QueryClient({
    defaultOptions: {
      mutations: { retry: false },
      queries: { retry: false },
    },
  });
}

function renderAutomationPage(page: "jobs" | "triggers") {
  const Page = page === "jobs" ? JobsPage : TriggersPage;
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

function automationCreateHandlers(): HttpHandler[] {
  return [
    http.get("/api/workspaces", () => HttpResponse.json({ workspaces })),
    http.get("/api/settings/automation", () => HttpResponse.json(settingsAutomationSectionFixture)),
    http.get("/api/automation/jobs", () => HttpResponse.json({ jobs })),
    http.get("/api/automation/jobs/:id", ({ params }) => {
      const id = String(params.id);
      const job = jobs.find(item => item.id === id);
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
      const job = makeJob({
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
      jobs = [job, ...jobs];

      return HttpResponse.json({ job }, { status: 201 });
    }),
    http.get("/api/automation/triggers", () => HttpResponse.json({ triggers })),
    http.get("/api/automation/triggers/:id", ({ params }) => {
      const id = String(params.id);
      const trigger = triggers.find(item => item.id === id);
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
      const trigger = makeTrigger({
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
        webhook_id: body.webhook_id,
        webhook_secret_present: Boolean(body.webhook_secret_value),
        workspace_id: body.workspace_id,
      });
      triggers = [trigger, ...triggers];

      return HttpResponse.json({ trigger }, { status: 201 });
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
  jobs = [makeJob()];
  triggers = [makeTrigger()];
  runs = [...automationRunFixtures];
  createJobRequests = [];
  createTriggerRequests = [];
  handlers = automationCreateHandlers();
  toast.error.mockReset();
  toast.success.mockReset();
  useActiveWorkspaceStore.setState({ selectedWorkspaceId: "ws_alpha" });
  useUserHomeDirStore.getState().setUserHomeDir("/home/operator");
});

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

    fireEvent.click(screen.getByTestId("job-output-task"));
    fireEvent.change(screen.getByTestId("job-task-title"), {
      target: { value: "Audit payout drift" },
    });
    fireEvent.change(screen.getByTestId("job-task-desc"), {
      target: { value: "Open a task for payout drift review." },
    });
    fireEvent.change(screen.getByTestId("job-task-channel"), { target: { value: "ops-review" } });
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
          network_channel: "ops-review",
          title: "Audit payout drift",
        }),
        workspace_id: "ws_beta",
      })
    );
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
