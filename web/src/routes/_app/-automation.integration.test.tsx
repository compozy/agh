import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { AutomationJob, AutomationRun, AutomationTrigger } from "@/systems/automation";

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

vi.mock("@tanstack/react-router", () => ({
  createFileRoute: () => (opts: { component: () => React.ReactNode }) => ({
    component: opts.component,
  }),
}));

vi.mock("@/systems/workspace", () => ({
  useActiveWorkspace: () => ({
    workspaces: [
      {
        id: "ws_test",
        root_dir: "/workspace",
        add_dirs: [],
        name: "test-workspace",
        created_at: "2026-04-03T12:00:00Z",
        updated_at: "2026-04-03T12:00:00Z",
      },
    ],
    hasWorkspaces: true,
    activeWorkspace: {
      id: "ws_test",
      root_dir: "/workspace",
      add_dirs: [],
      name: "test-workspace",
      created_at: "2026-04-03T12:00:00Z",
      updated_at: "2026-04-03T12:00:00Z",
    },
    activeWorkspaceId: "ws_test",
    setActiveWorkspaceId: vi.fn(),
    clearActiveWorkspaceSelection: vi.fn(),
    isLoading: false,
    isError: false,
  }),
}));

vi.mock("@/systems/automation", async () => {
  const actual = await vi.importActual("@/systems/automation");
  return {
    ...actual,
    useAutomationJobs: () => ({
      data: mockJobs,
      isLoading: mockJobsLoading,
      error: mockJobsError,
    }),
    useAutomationTriggers: () => ({
      data: mockTriggers,
      isLoading: mockTriggersLoading,
      error: mockTriggersError,
    }),
    useAutomationJob: () => ({
      data: mockJobDetail,
      isLoading: mockJobDetailLoading,
      error: mockJobDetailError,
    }),
    useAutomationTrigger: () => ({
      data: mockTriggerDetail,
      isLoading: mockTriggerDetailLoading,
      error: mockTriggerDetailError,
    }),
    useAutomationJobRuns: () => ({
      data: mockJobRuns,
      isLoading: mockJobRunsLoading,
      error: mockJobRunsError,
    }),
    useAutomationTriggerRuns: () => ({
      data: mockTriggerRuns,
      isLoading: mockTriggerRunsLoading,
      error: mockTriggerRunsError,
    }),
    useCreateAutomationJob: () => ({
      mutateAsync: mockCreateJobMutateAsync,
      isPending: mockCreateJobPending,
    }),
    useUpdateAutomationJob: () => ({
      mutateAsync: mockUpdateJobMutateAsync,
      isPending: mockUpdateJobPending,
    }),
    useDeleteAutomationJob: () => ({
      mutateAsync: mockDeleteJobMutateAsync,
      isPending: mockDeleteJobPending,
    }),
    useTriggerAutomationJob: () => ({
      mutateAsync: mockTriggerJobMutateAsync,
      isPending: mockTriggerJobPending,
    }),
    useCreateAutomationTrigger: () => ({
      mutateAsync: mockCreateTriggerMutateAsync,
      isPending: mockCreateTriggerPending,
    }),
    useUpdateAutomationTrigger: () => ({
      mutateAsync: mockUpdateTriggerMutateAsync,
      isPending: mockUpdateTriggerPending,
    }),
    useDeleteAutomationTrigger: () => ({
      mutateAsync: mockDeleteTriggerMutateAsync,
      isPending: mockDeleteTriggerPending,
    }),
  };
});

import { Route } from "./automation";

function makeJob(overrides: Partial<AutomationJob> = {}): AutomationJob {
  return {
    id: "job_daily_review",
    name: "daily-review",
    agent_name: "reviewer",
    prompt: "Review recent changes.",
    scope: "workspace",
    workspace_id: "ws_test",
    source: "dynamic",
    enabled: true,
    schedule: { mode: "cron", expr: "0 9 * * *" },
    retry: { strategy: "none", max_retries: 3, base_delay: "2s" },
    fire_limit: { max: 12, window: "1h" },
    next_run: "2026-04-12T09:00:00Z",
    created_at: "2026-04-11T09:00:00Z",
    updated_at: "2026-04-11T09:05:00Z",
    ...overrides,
  };
}

function makeTrigger(overrides: Partial<AutomationTrigger> = {}): AutomationTrigger {
  return {
    id: "trg_push_review",
    name: "push-review",
    agent_name: "reviewer",
    prompt: "Review push event {{ .Data.branch }}.",
    event: "ext.github.push",
    filter: { "data.branch": "main" },
    scope: "workspace",
    workspace_id: "ws_test",
    source: "dynamic",
    enabled: true,
    retry: { strategy: "backoff", max_retries: 4, base_delay: "5s" },
    fire_limit: { max: 12, window: "1h" },
    endpoint_slug: "push-review",
    webhook_id: "wbh_push_review",
    created_at: "2026-04-11T08:00:00Z",
    updated_at: "2026-04-11T08:10:00Z",
    ...overrides,
  };
}

function makeRun(overrides: Partial<AutomationRun> = {}): AutomationRun {
  return {
    id: "run_001",
    status: "completed",
    attempt: 1,
    job_id: "job_daily_review",
    session_id: "sess_001",
    started_at: "2026-04-11T10:00:00Z",
    ended_at: "2026-04-11T10:05:00Z",
    ...overrides,
  };
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const AutomationPage = (Route as any).component as () => React.ReactNode;

describe("Automation route integration", () => {
  beforeEach(() => {
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
      makeRun({ id: "run_trigger", trigger_id: "trg_push_review", job_id: undefined }),
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

    mockCreateJobMutateAsync.mockResolvedValue(
      makeJob({ id: "job_created", name: "nightly-docs" })
    );
    mockTriggerJobMutateAsync.mockResolvedValue(
      makeRun({
        id: "run_queued",
        status: "running",
        started_at: "2026-04-11T11:00:00Z",
        ended_at: undefined,
      })
    );
  });

  it("renders loading and error states from the active list query", () => {
    mockJobsLoading = true;
    mockJobs = [];
    const { rerender } = render(<AutomationPage />);

    expect(screen.getByTestId("automation-loading")).toBeInTheDocument();

    mockJobsLoading = false;
    mockJobs = [];
    mockJobsError = new Error("boom");
    rerender(<AutomationPage />);

    expect(screen.getByTestId("automation-error")).toHaveTextContent("boom");
  });

  it("renders the jobs list, detail pane, and run history from mocked API-backed hooks", () => {
    render(<AutomationPage />);

    const detailPanel = screen.getByTestId("automation-detail-panel");

    expect(screen.getByText("Automation")).toBeInTheDocument();
    expect(screen.getByTestId("automation-list-panel")).toBeInTheDocument();
    expect(detailPanel).toBeInTheDocument();
    expect(screen.getByTestId("automation-item-job_daily_review")).toBeInTheDocument();
    expect(within(detailPanel).getByText("daily-review")).toBeInTheDocument();
    expect(within(detailPanel).getByText("Review recent changes.")).toBeInTheDocument();
    expect(screen.getByTestId("automation-run-run_001")).toBeInTheDocument();
  });

  it("switches to trigger management and shows trigger detail content", async () => {
    const user = userEvent.setup();
    render(<AutomationPage />);

    await user.click(screen.getByTestId("automation-kind-triggers"));

    const detailPanel = screen.getByTestId("automation-detail-panel");

    expect(screen.getByTestId("automation-item-trg_push_review")).toBeInTheDocument();
    expect(within(detailPanel).getByRole("heading", { name: "push-review" })).toBeInTheDocument();
    expect(within(detailPanel).getByText("ext.github.push", { selector: "p" })).toBeInTheDocument();
  });

  it("submits a workspace-scoped job create payload using the active workspace id", async () => {
    const user = userEvent.setup();
    render(<AutomationPage />);

    await user.click(screen.getByTestId("create-automation-btn"));
    await user.type(screen.getByTestId("job-name-input"), "nightly-docs");
    await user.type(screen.getByTestId("job-agent-input"), "writer");
    await user.type(
      screen.getByTestId("job-prompt-input"),
      "Summarize docs changes and publish a digest."
    );
    await user.click(screen.getByTestId("submit-job-form"));

    expect(mockCreateJobMutateAsync).toHaveBeenCalledWith(
      expect.objectContaining({
        scope: "workspace",
        workspace_id: "ws_test",
        name: "nightly-docs",
        agent_name: "writer",
      })
    );
    expect(await screen.findByText("Created job nightly-docs.")).toBeInTheDocument();
  });

  it("queues a manual run and prepends it to run history", async () => {
    const user = userEvent.setup();
    render(<AutomationPage />);

    await user.click(screen.getByTestId("trigger-job-btn"));

    expect(mockTriggerJobMutateAsync).toHaveBeenCalledWith({ id: "job_daily_review" });
    expect(await screen.findByText("Queued run run_queued.")).toBeInTheDocument();
    expect(screen.getByTestId("automation-run-run_queued")).toBeInTheDocument();
  });
});
