import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderWithTopbar } from "@/test/render-with-topbar";
import { routeComponent } from "@/test/route-options";
import userEvent from "@testing-library/user-event";
import type { AnchorHTMLAttributes, ReactElement, ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { AutomationJob, AutomationRun, AutomationTrigger } from "@/systems/automation";

const { settingsAutomationQuery, toast } = vi.hoisted(() => ({
  settingsAutomationQuery: {
    current: {
      data: {
        runtime: {
          available: true,
          running: true,
          scheduler_running: true,
          job_enabled: 1,
          job_total: 1,
          trigger_enabled: 1,
          trigger_total: 1,
        },
      },
      error: null,
      isLoading: false,
    },
  },
  toast: {
    error: vi.fn(),
    success: vi.fn(),
  },
}));

const workspaceContext = vi.hoisted(() => ({
  activeWorkspaceId: "ws_test" as string | null,
  isLoading: false,
}));

let mockJobs: AutomationJob[] = [];
let mockJobsLoading = false;
let mockJobsError: Error | null = null;

let mockTriggers: AutomationTrigger[] = [];
let mockTriggersLoading = false;
let mockTriggersError: Error | null = null;

let mockJobDetail: AutomationJob | undefined;
let mockJobDetailLoading = false;
let mockJobDetailError: Error | null = null;

let mockTriggerDetail: AutomationTrigger | undefined;
let mockTriggerDetailLoading = false;
let mockTriggerDetailError: Error | null = null;

let mockJobRuns: AutomationRun[] = [];
let mockJobRunsLoading = false;
let mockJobRunsError: Error | null = null;

let mockTriggerRuns: AutomationRun[] = [];
let mockTriggerRunsLoading = false;
let mockTriggerRunsError: Error | null = null;

const mockCreateJobMutateAsync = vi.fn();
const mockUpdateJobMutateAsync = vi.fn();
const mockDeleteJobMutateAsync = vi.fn();
const mockTriggerJobMutateAsync = vi.fn();
const mockCreateTriggerMutateAsync = vi.fn();
const mockUpdateTriggerMutateAsync = vi.fn();
const mockDeleteTriggerMutateAsync = vi.fn();

let mockCreateJobPending = false;
let mockUpdateJobPending = false;
let mockDeleteJobPending = false;
let mockTriggerJobPending = false;
let mockCreateTriggerPending = false;
let mockUpdateTriggerPending = false;
let mockDeleteTriggerPending = false;

interface MockLinkParams {
  jobId?: string;
  triggerId?: string;
}

interface MockLinkProps extends AnchorHTMLAttributes<HTMLAnchorElement> {
  params?: MockLinkParams;
}

const routerState = vi.hoisted(() => ({
  childMatches: [] as unknown[],
  navigateMock: vi.fn(),
  params: {} as Record<string, string>,
}));

vi.mock("@tanstack/react-router", () => ({
  createFileRoute: () => (opts: { component: () => React.ReactNode }) => ({
    component: opts.component,
    useParams: () => routerState.params,
    useSearch: () => ({}),
  }),
  Link: ({ children, params, to, ...props }: MockLinkProps & { to?: string }) => (
    <a href={to ?? `/${params?.jobId ?? params?.triggerId ?? ""}`} {...props}>
      {children}
    </a>
  ),
  Outlet: () => <div data-testid="router-outlet" />,
  useChildMatches: () => routerState.childMatches,
  useNavigate: () => routerState.navigateMock,
}));

vi.mock("sonner", () => ({
  toast,
}));

vi.mock("@/systems/settings", () => ({
  useSettingsAutomation: () => settingsAutomationQuery.current,
}));

vi.mock("@/systems/workspace", async () => {
  const actual = await vi.importActual<typeof import("@/systems/workspace")>("@/systems/workspace");

  const workspace = {
    add_dirs: [],
    created_at: "2026-04-03T12:00:00Z",
    id: "ws_test",
    name: "test-workspace",
    root_dir: "/workspace",
    updated_at: "2026-04-03T12:00:00Z",
  };

  return {
    ...actual,
    useActiveWorkspace: () => ({
      workspaces: [workspace],
      hasWorkspaces: true,
      activeWorkspace: workspace,
      activeWorkspaceId: workspaceContext.activeWorkspaceId,
      clearActiveWorkspaceSelection: vi.fn(),
      isError: false,
      isLoading: workspaceContext.isLoading,
      setActiveWorkspaceId: vi.fn(),
    }),
  };
});

vi.mock("@/systems/automation/hooks/use-automation", () => ({
  useAutomationJobs: () => ({
    error: mockJobsError,
    fetchNextPage: vi.fn(),
    hasNextPage: false,
    isFetchingNextPage: false,
    isLoading: mockJobsLoading,
    jobs: mockJobs,
    total: mockJobs.length,
  }),
  useAutomationTriggers: () => ({
    error: mockTriggersError,
    fetchNextPage: vi.fn(),
    hasNextPage: false,
    isFetchingNextPage: false,
    isLoading: mockTriggersLoading,
    total: mockTriggers.length,
    triggers: mockTriggers,
  }),
  useAutomationJob: () => ({
    data: mockJobDetail,
    error: mockJobDetailError,
    isLoading: mockJobDetailLoading,
  }),
  useAutomationTrigger: () => ({
    data: mockTriggerDetail,
    error: mockTriggerDetailError,
    isLoading: mockTriggerDetailLoading,
  }),
  useAutomationJobRuns: () => ({
    data: mockJobRuns,
    error: mockJobRunsError,
    isLoading: mockJobRunsLoading,
  }),
  useAutomationTriggerRuns: () => ({
    data: mockTriggerRuns,
    error: mockTriggerRunsError,
    isLoading: mockTriggerRunsLoading,
  }),
  useAutomationRuns: () => ({ data: [], error: null, isLoading: false }),
}));

vi.mock("@/systems/automation/hooks/use-automation-actions", () => ({
  useCreateAutomationJob: () => ({
    isPending: mockCreateJobPending,
    mutateAsync: mockCreateJobMutateAsync,
  }),
  useUpdateAutomationJob: () => ({
    isPending: mockUpdateJobPending,
    mutateAsync: mockUpdateJobMutateAsync,
  }),
  useDeleteAutomationJob: () => ({
    isPending: mockDeleteJobPending,
    mutateAsync: mockDeleteJobMutateAsync,
  }),
  useTriggerAutomationJob: () => ({
    isPending: mockTriggerJobPending,
    mutateAsync: mockTriggerJobMutateAsync,
  }),
  useCreateAutomationTrigger: () => ({
    isPending: mockCreateTriggerPending,
    mutateAsync: mockCreateTriggerMutateAsync,
  }),
  useUpdateAutomationTrigger: () => ({
    isPending: mockUpdateTriggerPending,
    mutateAsync: mockUpdateTriggerMutateAsync,
  }),
  useDeleteAutomationTrigger: () => ({
    isPending: mockDeleteTriggerPending,
    mutateAsync: mockDeleteTriggerMutateAsync,
  }),
}));

import { Route as JobsRoute } from "../jobs";
import { Route as JobDetailRoute } from "../jobs.$jobId";
import { Route as TriggersRoute } from "../triggers";
import { Route as TriggerDetailRoute } from "../triggers.$triggerId";

function makeJob(overrides: Partial<AutomationJob> = {}): AutomationJob {
  return {
    agent_name: "reviewer",
    created_at: "2026-04-11T09:00:00Z",
    enabled: true,
    fire_limit: { max: 12, window: "1h" },
    id: "job_daily_review",
    name: "daily-review",
    next_run: "2026-04-12T09:00:00Z",
    prompt: "Review recent changes.",
    retry: { strategy: "none", max_retries: 3, base_delay: "2s" },
    schedule: { mode: "cron", expr: "0 9 * * *" },
    scope: "workspace",
    source: "dynamic",
    target_kind: "agent",
    updated_at: "2026-04-11T09:05:00Z",
    workspace_id: "ws_test",
    ...overrides,
  };
}

function makeTrigger(overrides: Partial<AutomationTrigger> = {}): AutomationTrigger {
  const webhookSecretPresent = overrides.webhook_secret_present ?? false;

  return {
    agent_name: "reviewer",
    created_at: "2026-04-11T08:00:00Z",
    enabled: true,
    endpoint_slug: "push-review",
    event: "ext.github.push",
    filter: { "data.branch": "main" },
    fire_limit: { max: 12, window: "1h" },
    id: "trg_push_review",
    name: "push-review",
    prompt: "Review push event {{ .Data.branch }}.",
    retry: { strategy: "backoff", max_retries: 4, base_delay: "5s" },
    scope: "workspace",
    source: "dynamic",
    target_kind: "agent",
    updated_at: "2026-04-11T08:10:00Z",
    webhook_id: "wbh_push_review",
    workspace_id: "ws_test",
    ...overrides,
    webhook_secret_present: webhookSecretPresent,
  };
}

function makeRun(overrides: Partial<AutomationRun> = {}): AutomationRun {
  return {
    attempt: 1,
    ended_at: "2026-04-11T10:05:00Z",
    id: "run_001",
    job_id: "job_daily_review",
    session_id: "sess_001",
    started_at: "2026-04-11T10:00:00Z",
    status: "completed",
    ...overrides,
  };
}

const JobsPage = routeComponent(JobsRoute);
const TriggersPage = routeComponent(TriggersRoute);
const JobDetailPage = routeComponent(JobDetailRoute);
const TriggerDetailPage = routeComponent(TriggerDetailRoute);

function render(ui: ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  const withQueryClient = (child: ReactNode) => (
    <QueryClientProvider client={queryClient}>{child}</QueryClientProvider>
  );
  const result = renderWithTopbar(withQueryClient(ui));
  return {
    ...result,
    rerender: (next: ReactNode) => result.rerender(withQueryClient(next)),
  };
}

beforeEach(() => {
  vi.useRealTimers();
  routerState.childMatches = [];
  routerState.params = {};
  routerState.navigateMock.mockReset();
  workspaceContext.activeWorkspaceId = "ws_test";
  workspaceContext.isLoading = false;
  mockJobs = [makeJob()];
  mockJobsLoading = false;
  mockJobsError = null;
  mockTriggers = [makeTrigger()];
  mockTriggersLoading = false;
  mockTriggersError = null;
  mockJobDetail = makeJob();
  mockJobDetailLoading = false;
  mockJobDetailError = null;
  mockTriggerDetail = makeTrigger();
  mockTriggerDetailLoading = false;
  mockTriggerDetailError = null;
  mockJobRuns = [makeRun()];
  mockJobRunsLoading = false;
  mockJobRunsError = null;
  mockTriggerRuns = [
    makeRun({ id: "run_trigger", job_id: undefined, trigger_id: "trg_push_review" }),
  ];
  mockTriggerRunsLoading = false;
  mockTriggerRunsError = null;

  mockCreateJobPending = false;
  mockUpdateJobPending = false;
  mockDeleteJobPending = false;
  mockTriggerJobPending = false;
  mockCreateTriggerPending = false;
  mockUpdateTriggerPending = false;
  mockDeleteTriggerPending = false;

  mockCreateJobMutateAsync.mockReset();
  mockUpdateJobMutateAsync.mockReset();
  mockDeleteJobMutateAsync.mockReset();
  mockTriggerJobMutateAsync.mockReset();
  mockCreateTriggerMutateAsync.mockReset();
  mockUpdateTriggerMutateAsync.mockReset();
  mockDeleteTriggerMutateAsync.mockReset();
  toast.success.mockReset();
  toast.error.mockReset();
  settingsAutomationQuery.current = {
    data: {
      runtime: {
        available: true,
        running: true,
        scheduler_running: true,
        job_enabled: 1,
        job_total: 1,
        trigger_enabled: 1,
        trigger_total: 1,
      },
    },
    error: null,
    isLoading: false,
  };

  mockCreateJobMutateAsync.mockResolvedValue(makeJob({ id: "job_created", name: "nightly-docs" }));
  mockUpdateJobMutateAsync.mockResolvedValue(
    makeJob({ id: "job_daily_review", name: "daily-review-updated" })
  );
  mockDeleteJobMutateAsync.mockResolvedValue(undefined);
  mockCreateTriggerMutateAsync.mockResolvedValue(
    makeTrigger({ id: "trg_created", name: "qa-trigger-browser", event: "ext.test.qa" })
  );
  mockDeleteTriggerMutateAsync.mockResolvedValue(undefined);
  mockTriggerJobMutateAsync.mockResolvedValue(
    makeRun({
      ended_at: undefined,
      id: "run_queued",
      started_at: "2026-04-11T11:00:00Z",
      status: "running",
    })
  );
});

describe("Jobs catalog route", () => {
  it("renders loading and error states from the jobs list query", () => {
    mockJobsLoading = true;
    mockJobs = [];
    const { rerender } = render(<JobsPage />);

    expect(screen.getByTestId("jobs-loading")).toBeInTheDocument();

    mockJobsLoading = false;
    mockJobs = [];
    mockJobsError = new Error("boom");
    rerender(<JobsPage />);

    expect(screen.getByTestId("jobs-error")).toHaveTextContent("boom");
  });

  it("renders the jobs catalog rows from mocked hooks", () => {
    render(<JobsPage />);

    expect(screen.getByTestId("jobs-shell")).toBeInTheDocument();
    expect(screen.getByTestId("jobs-list-rows")).toBeInTheDocument();
    const row = screen.getByTestId("automation-item-job_daily_review");
    expect(within(row).getByText("daily-review")).toBeInTheDocument();
    expect(within(row).getByText("ENABLED")).toBeInTheDocument();
  });

  it("renders the Outlet when a detail child route is active", () => {
    routerState.childMatches = [{ id: "detail" }];
    render(<JobsPage />);

    expect(screen.getByTestId("router-outlet")).toBeInTheDocument();
    expect(screen.queryByTestId("jobs-shell")).not.toBeInTheDocument();
    // The list route publishes a null slot while a child is mounted so the
    // detail route's own topbar publish wins (single-publisher store).
    expect(screen.queryByTestId("create-job-btn")).not.toBeInTheDocument();
  });

  it("shows a runtime-unavailable alert instead of treating cached jobs as healthy", () => {
    settingsAutomationQuery.current = {
      data: {
        runtime: {
          available: false,
          running: false,
          scheduler_running: false,
          job_enabled: 0,
          job_total: 0,
          trigger_enabled: 0,
          trigger_total: 0,
        },
      },
      error: null,
      isLoading: false,
    };

    render(<JobsPage />);

    expect(screen.getByTestId("jobs-runtime-alert")).toHaveTextContent(
      "automation runtime is disabled"
    );
  });

  it("opens a create job modal and submits a workspace-scoped payload", async () => {
    const user = userEvent.setup();
    render(<JobsPage />);

    await user.click(screen.getByTestId("create-job-btn"));

    expect(screen.getByTestId("automation-job-form")).toBeInTheDocument();

    fireEvent.change(screen.getByTestId("job-name-input"), {
      target: { value: "nightly-docs" },
    });
    fireEvent.change(screen.getByTestId("job-agent-input"), {
      target: { value: "writer" },
    });
    fireEvent.change(screen.getByTestId("job-prompt-input"), {
      target: { value: "Summarize docs changes and publish a digest." },
    });
    await user.click(screen.getByTestId("submit-job-form"));

    await waitFor(() => {
      expect(mockCreateJobMutateAsync).toHaveBeenCalledWith(
        expect.objectContaining({
          agent_name: "writer",
          name: "nightly-docs",
          scope: "workspace",
          workspace_id: "ws_test",
        })
      );
      expect(toast.success).toHaveBeenCalledWith("Created job nightly-docs.");
    });
    expect(routerState.navigateMock).toHaveBeenCalledWith({
      to: "/jobs/$jobId",
      params: { jobId: "job_created" },
    });
  });

  it("queues a manual run from a catalog row", async () => {
    const user = userEvent.setup();
    render(<JobsPage />);

    await user.click(screen.getByTestId("automation-run-now-job_daily_review"));

    await waitFor(() => {
      expect(mockTriggerJobMutateAsync).toHaveBeenCalledWith({ id: "job_daily_review" });
      expect(toast.success).toHaveBeenCalledWith("Queued run run_queued.");
    });
  });

  it("renders the empty jobs state when no jobs exist", () => {
    mockJobs = [];

    render(<JobsPage />);

    expect(screen.getByTestId("jobs-list-empty")).toBeInTheDocument();
  });
});

describe("Job detail route", () => {
  beforeEach(() => {
    routerState.params = { jobId: "job_daily_review" };
  });

  it("renders the job detail, schedule, and run history", () => {
    render(<JobDetailPage />);

    const detailPanel = screen.getByTestId("automation-detail-panel");
    expect(within(detailPanel).getByText("daily-review")).toBeInTheDocument();
    expect(within(detailPanel).getByText("Review recent changes.")).toBeInTheDocument();
    expect(within(detailPanel).getByText("0 9 * * *")).toBeInTheDocument();
    expect(screen.getByTestId("automation-run-run_001")).toBeInTheDocument();
  });

  it("renders the no-runs state when the job has not executed yet", () => {
    mockJobRuns = [];

    render(<JobDetailPage />);

    expect(screen.getByText("No runs recorded yet")).toBeInTheDocument();
  });

  it("renders the detail error state when the routed job id does not resolve", () => {
    routerState.params = { jobId: "job_missing" };
    mockJobDetail = undefined;
    mockJobDetailError = new Error("automation job not found");

    render(<JobDetailPage />);

    expect(screen.getByTestId("automation-detail-error")).toBeInTheDocument();
    expect(screen.getByText("automation job not found")).toBeInTheDocument();
  });

  it("preserves cached job detail when a background refetch fails", () => {
    mockJobDetailError = new Error("background refetch failed");

    render(<JobDetailPage />);

    expect(screen.getByTestId("automation-detail-panel")).toHaveTextContent("daily-review");
    expect(screen.queryByTestId("automation-detail-error")).not.toBeInTheDocument();
  });

  it("Should withhold a workspace job and its runs outside the active workspace", () => {
    mockJobDetail = makeJob({ workspace_id: "ws_other" });

    render(<JobDetailPage />);

    expect(screen.getByTestId("automation-detail-error")).toHaveTextContent(
      "This workspace-scoped job belongs to another workspace."
    );
    expect(screen.queryByTestId("automation-detail-panel")).not.toBeInTheDocument();
    expect(screen.queryByTestId("automation-run-run_001")).not.toBeInTheDocument();
    expect(screen.queryByTestId("edit-automation-btn")).not.toBeInTheDocument();
    expect(screen.queryByTestId("trigger-job-btn")).not.toBeInTheDocument();
  });

  it("Should close a job editor when the active workspace changes", async () => {
    const user = userEvent.setup();
    const { rerender } = render(<JobDetailPage />);
    await user.click(screen.getByTestId("edit-automation-btn"));
    expect(screen.getByTestId("automation-job-form")).toBeInTheDocument();

    workspaceContext.activeWorkspaceId = "ws_other";
    rerender(<JobDetailPage />);

    await waitFor(() =>
      expect(screen.queryByTestId("automation-job-form")).not.toBeInTheDocument()
    );

    workspaceContext.activeWorkspaceId = "ws_test";
    rerender(<JobDetailPage />);

    expect(screen.queryByTestId("automation-job-form")).not.toBeInTheDocument();
    expect(mockUpdateJobMutateAsync).not.toHaveBeenCalled();
  });

  it("edits the job using the route-scoped id", async () => {
    const user = userEvent.setup();
    render(<JobDetailPage />);

    await user.click(screen.getByTestId("edit-automation-btn"));
    fireEvent.change(screen.getByTestId("job-name-input"), {
      target: { value: "daily-review-updated" },
    });
    await user.click(screen.getByTestId("submit-job-form"));

    await waitFor(() => {
      expect(mockUpdateJobMutateAsync).toHaveBeenCalledWith({
        data: expect.objectContaining({ name: "daily-review-updated" }),
        id: "job_daily_review",
      });
    });
  });

  it("queues a manual run and prepends it to the run history", async () => {
    const user = userEvent.setup();
    render(<JobDetailPage />);

    await user.click(screen.getByTestId("trigger-job-btn"));

    await waitFor(() => {
      expect(mockTriggerJobMutateAsync).toHaveBeenCalledWith({ id: "job_daily_review" });
      expect(toast.success).toHaveBeenCalledWith("Queued run run_queued.");
      expect(screen.getByTestId("automation-run-run_queued")).toBeInTheDocument();
    });
  });

  it("deletes the job and navigates back to the list", async () => {
    const user = userEvent.setup();
    render(<JobDetailPage />);

    fireEvent.click(screen.getByTestId("automation-detail-overflow"));
    fireEvent.click(screen.getByTestId("delete-automation-btn"));
    fireEvent.change(screen.getByTestId("automation-delete-confirm-typing"), {
      target: { value: "daily-review" },
    });
    await user.click(screen.getByTestId("confirm-delete-automation-btn"));

    await waitFor(() => {
      expect(mockDeleteJobMutateAsync).toHaveBeenCalledWith({ id: "job_daily_review" });
      expect(routerState.navigateMock).toHaveBeenCalledWith({ to: "/jobs", replace: true });
    });
  });
});

describe("Triggers catalog route", () => {
  it("renders loading and error states from the triggers list query", () => {
    mockTriggersLoading = true;
    mockTriggers = [];
    const { rerender } = render(<TriggersPage />);

    expect(screen.getByTestId("triggers-loading")).toBeInTheDocument();

    mockTriggersLoading = false;
    mockTriggers = [];
    mockTriggersError = new Error("boom");
    rerender(<TriggersPage />);

    expect(screen.getByTestId("triggers-error")).toHaveTextContent("boom");
  });

  it("renders the triggers catalog rows from mocked hooks", () => {
    render(<TriggersPage />);

    expect(screen.getByTestId("triggers-shell")).toBeInTheDocument();
    const row = screen.getByTestId("automation-item-trg_push_review");
    expect(within(row).getByText("push-review")).toBeInTheDocument();
    expect(within(row).getByText("ext.github.push")).toBeInTheDocument();
  });

  it("opens a create trigger modal and submits a valid retry-none payload", async () => {
    const user = userEvent.setup();
    render(<TriggersPage />);

    await user.click(screen.getByTestId("create-trigger-btn"));

    fireEvent.change(screen.getByTestId("trigger-name-input"), {
      target: { value: "qa-trigger-browser" },
    });
    fireEvent.change(screen.getByTestId("trigger-agent-input"), {
      target: { value: "reviewer" },
    });
    fireEvent.click(screen.getByTestId("trigger-event-ext"));
    fireEvent.change(screen.getByTestId("trigger-ext-ext-input"), {
      target: { value: "test" },
    });
    fireEvent.change(screen.getByTestId("trigger-ext-event-input"), {
      target: { value: "qa" },
    });
    fireEvent.change(screen.getByTestId("trigger-prompt-input"), {
      target: { value: "Review {{ .Kind }}." },
    });

    await user.click(screen.getByTestId("submit-trigger-form"));

    await waitFor(() => {
      expect(mockCreateTriggerMutateAsync).toHaveBeenCalledWith(
        expect.objectContaining({
          agent_name: "reviewer",
          event: "ext.test.qa",
          name: "qa-trigger-browser",
          scope: "workspace",
          workspace_id: "ws_test",
        })
      );
      expect(toast.success).toHaveBeenCalledWith("Created trigger qa-trigger-browser.");
    });
    expect(routerState.navigateMock).toHaveBeenCalledWith({
      to: "/triggers/$triggerId",
      params: { triggerId: "trg_created" },
    });
  });

  it("renders the empty triggers state when no triggers exist", () => {
    mockTriggers = [];

    render(<TriggersPage />);

    expect(screen.getByTestId("triggers-list-empty")).toBeInTheDocument();
  });
});

describe("Trigger detail route", () => {
  beforeEach(() => {
    routerState.params = { triggerId: "trg_push_review" };
  });

  it("renders the trigger detail, event, and run history", () => {
    render(<TriggerDetailPage />);

    const detailPanel = screen.getByTestId("automation-detail-panel");
    expect(within(detailPanel).getByText("push-review")).toBeInTheDocument();
    expect(within(detailPanel).getAllByText("ext.github.push").length).toBeGreaterThan(0);
    expect(screen.getByTestId("automation-run-run_trigger")).toBeInTheDocument();
  });

  it("preserves cached trigger detail when a background refetch fails", () => {
    mockTriggerDetailError = new Error("background refetch failed");

    render(<TriggerDetailPage />);

    expect(screen.getByTestId("automation-detail-panel")).toHaveTextContent("push-review");
    expect(screen.queryByTestId("automation-detail-error")).not.toBeInTheDocument();
  });

  it("Should withhold a workspace trigger and its runs outside the active workspace", () => {
    mockTriggerDetail = makeTrigger({ workspace_id: "ws_other" });

    render(<TriggerDetailPage />);

    expect(screen.getByTestId("automation-detail-error")).toHaveTextContent(
      "This workspace-scoped trigger belongs to another workspace."
    );
    expect(screen.queryByTestId("automation-detail-panel")).not.toBeInTheDocument();
    expect(screen.queryByTestId("automation-run-run_trigger")).not.toBeInTheDocument();
    expect(screen.queryByTestId("edit-automation-btn")).not.toBeInTheDocument();
  });

  it("deletes the trigger and navigates back to the list", async () => {
    const user = userEvent.setup();
    render(<TriggerDetailPage />);

    fireEvent.click(screen.getByTestId("automation-detail-overflow"));
    fireEvent.click(screen.getByTestId("delete-automation-btn"));
    fireEvent.change(screen.getByTestId("automation-delete-confirm-typing"), {
      target: { value: "push-review" },
    });
    await user.click(screen.getByTestId("confirm-delete-automation-btn"));

    await waitFor(() => {
      expect(mockDeleteTriggerMutateAsync).toHaveBeenCalledWith({ id: "trg_push_review" });
      expect(routerState.navigateMock).toHaveBeenCalledWith({ to: "/triggers", replace: true });
    });
  });
});
