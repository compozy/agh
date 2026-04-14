import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { AutomationJob, AutomationRun, AutomationTrigger } from "@/systems/automation";

const { toast } = vi.hoisted(() => ({
  toast: {
    error: vi.fn(),
    success: vi.fn(),
  },
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

vi.mock("@tanstack/react-router", () => ({
  createFileRoute: () => (opts: { component: () => React.ReactNode }) => ({
    component: opts.component,
  }),
}));

vi.mock("sonner", () => ({
  toast,
}));

vi.mock("@/systems/workspace", () => ({
  useActiveWorkspace: () => ({
    workspaces: [
      {
        add_dirs: [],
        created_at: "2026-04-03T12:00:00Z",
        id: "ws_test",
        name: "test-workspace",
        root_dir: "/workspace",
        updated_at: "2026-04-03T12:00:00Z",
      },
    ],
    hasWorkspaces: true,
    activeWorkspace: {
      add_dirs: [],
      created_at: "2026-04-03T12:00:00Z",
      id: "ws_test",
      name: "test-workspace",
      root_dir: "/workspace",
      updated_at: "2026-04-03T12:00:00Z",
    },
    activeWorkspaceId: "ws_test",
    clearActiveWorkspaceSelection: vi.fn(),
    isError: false,
    isLoading: false,
    setActiveWorkspaceId: vi.fn(),
  }),
}));

vi.mock("@/systems/automation", async () => {
  const actual = await vi.importActual("@/systems/automation");
  return {
    ...actual,
    useAutomationJobs: () => ({
      data: mockJobs,
      error: mockJobsError,
      isLoading: mockJobsLoading,
    }),
    useAutomationTriggers: () => ({
      data: mockTriggers,
      error: mockTriggersError,
      isLoading: mockTriggersLoading,
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
  };
});

import { Route } from "./automation";

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
    updated_at: "2026-04-11T09:05:00Z",
    workspace_id: "ws_test",
    ...overrides,
  };
}

function makeTrigger(overrides: Partial<AutomationTrigger> = {}): AutomationTrigger {
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
    updated_at: "2026-04-11T08:10:00Z",
    webhook_id: "wbh_push_review",
    workspace_id: "ws_test",
    ...overrides,
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

const AutomationPage = (Route as unknown as { component: () => React.ReactNode }).component;

describe("Automation route integration", () => {
  beforeEach(() => {
    vi.useRealTimers();
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

    mockCreateJobMutateAsync.mockResolvedValue(
      makeJob({ id: "job_created", name: "nightly-docs" })
    );
    mockTriggerJobMutateAsync.mockResolvedValue(
      makeRun({
        ended_at: undefined,
        id: "run_queued",
        started_at: "2026-04-11T11:00:00Z",
        status: "running",
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

  it("renders the jobs list, schedule detail, and run history from mocked hooks", () => {
    render(<AutomationPage />);

    const detailPanel = screen.getByTestId("automation-detail-panel");

    expect(screen.getByText("Automation")).toBeInTheDocument();
    expect(screen.getByTestId("automation-list-panel")).toBeInTheDocument();
    expect(screen.getByTestId("automation-item-job_daily_review")).toBeInTheDocument();
    expect(within(detailPanel).getByText("daily-review")).toBeInTheDocument();
    expect(within(detailPanel).getByText("Review recent changes.")).toBeInTheDocument();
    expect(within(detailPanel).getByText("0 9 * * *")).toBeInTheDocument();
    expect(screen.getByTestId("automation-run-run_001")).toBeInTheDocument();
  });

  it("switches to trigger management and shows trigger activation content", async () => {
    const user = userEvent.setup();
    render(<AutomationPage />);

    await user.click(screen.getByTestId("automation-kind-triggers"));

    const detailPanel = screen.getByTestId("automation-detail-panel");

    expect(screen.getByTestId("automation-item-trg_push_review")).toBeInTheDocument();
    expect(within(detailPanel).getByRole("heading", { name: "push-review" })).toBeInTheDocument();
    expect(within(detailPanel).getAllByText("ext.github.push")).toHaveLength(2);
    expect(within(detailPanel).getByText("Dispatches to")).toBeInTheDocument();
  });

  it("opens a create job modal and submits a workspace-scoped payload", async () => {
    const user = userEvent.setup();
    render(<AutomationPage />);

    await user.click(screen.getByTestId("create-automation-btn"));

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
  });

  it("uses the original job id when the visible selection changes during edit", async () => {
    const user = userEvent.setup();
    mockUpdateJobMutateAsync.mockResolvedValue(
      makeJob({ id: "job_daily_review", name: "daily-review-updated" })
    );

    const { rerender } = render(<AutomationPage />);

    await user.click(screen.getByTestId("edit-automation-btn"));
    fireEvent.change(screen.getByTestId("job-name-input"), {
      target: { value: "daily-review-updated" },
    });

    mockJobs = [
      makeJob({
        id: "job_release_notes",
        name: "release-notes",
        prompt: "Review the release notes.",
      }),
    ];
    rerender(<AutomationPage />);

    await user.click(screen.getByTestId("submit-job-form"));

    await waitFor(() => {
      expect(mockUpdateJobMutateAsync).toHaveBeenCalledWith({
        data: expect.objectContaining({ name: "daily-review-updated" }),
        id: "job_daily_review",
      });
    });
  });

  it("renders the no-runs state when the selected job has not executed yet", () => {
    mockJobRuns = [];

    render(<AutomationPage />);

    expect(screen.getByText("No runs recorded yet")).toBeInTheDocument();
    expect(
      screen.getByText("Runs will appear here after the first scheduled or manual execution.")
    ).toBeInTheDocument();
  });

  it("renders jobs and triggers empty states when no automation exists", async () => {
    const user = userEvent.setup();
    mockJobs = [];
    mockJobDetail = undefined;
    mockJobRuns = [];
    mockTriggers = [];
    mockTriggerDetail = undefined;
    mockTriggerRuns = [];

    render(<AutomationPage />);

    expect(screen.getByText("No jobs configured")).toBeInTheDocument();

    await user.click(screen.getByTestId("automation-kind-triggers"));

    expect(screen.getByText("No triggers configured")).toBeInTheDocument();
  });

  it("queues a manual run and prepends it to run history", async () => {
    const user = userEvent.setup();
    render(<AutomationPage />);

    await user.click(screen.getByTestId("trigger-job-btn"));

    await waitFor(() => {
      expect(mockTriggerJobMutateAsync).toHaveBeenCalledWith({ id: "job_daily_review" });
      expect(toast.success).toHaveBeenCalledWith("Queued run run_queued.");
      expect(screen.getByTestId("automation-run-run_queued")).toBeInTheDocument();
    });
  });
});
