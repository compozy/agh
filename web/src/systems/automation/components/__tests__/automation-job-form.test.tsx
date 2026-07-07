import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";

vi.mock("@/systems/loops/hooks/use-loops", async () => {
  const { loopCatalogFixtures } = await import("@/systems/loops/mocks/fixtures");
  return { useLoops: () => ({ data: loopCatalogFixtures, isLoading: false }) };
});

import { AutomationJobForm } from "../automation-job-form";
import { createAutomationJobDraft } from "../../lib/automation-drafts";
import type { CreateAutomationJobRequest } from "../../types";

const WORKSPACE_ID = "ws_alpha";
const STORY_AGENTS = ["reviewer", "writer", "auditor"];
const WORKSPACES = [
  { id: WORKSPACE_ID, name: "alpha" },
  { id: "ws_beta", name: "beta" },
];

interface RenderJobFormOptions {
  activeWorkspaceId?: string | null;
  draft?: CreateAutomationJobRequest;
  isPending?: boolean;
  mode?: "create" | "edit";
  agents?: string[];
  workspaces?: typeof WORKSPACES;
}

/**
 * Controlled harness: holds the draft via useState and syncs every `onChange`
 * back into state so multi-step interactions (output-mode and schedule-mode
 * switches) re-render against the updated draft.
 */
function renderJobForm({
  activeWorkspaceId = WORKSPACE_ID,
  draft = createAutomationJobDraft(activeWorkspaceId),
  isPending = false,
  mode = "create" as "create" | "edit",
  agents = STORY_AGENTS,
  workspaces = WORKSPACES,
}: RenderJobFormOptions = {}) {
  const onCancel = vi.fn();
  const onChange = vi.fn();
  const onSubmit = vi.fn();

  function Harness() {
    const [currentDraft, setCurrentDraft] = useState<CreateAutomationJobRequest>(draft);

    return (
      <AutomationJobForm
        activeWorkspaceId={activeWorkspaceId}
        agents={agents}
        draft={currentDraft}
        isPending={isPending}
        mode={mode}
        onCancel={onCancel}
        onChange={nextDraft => {
          onChange(nextDraft);
          setCurrentDraft(nextDraft);
        }}
        onSubmit={onSubmit}
        workspaces={workspaces}
      />
    );
  }

  render(<Harness />);

  return { onCancel, onChange, onSubmit };
}

describe("AutomationJobForm", () => {
  it("Should update the job name through onChange", () => {
    const { onChange } = renderJobForm();

    fireEvent.change(screen.getByTestId("job-name-input"), {
      target: { value: "nightly-docs" },
    });

    expect(screen.getByTestId("job-name-input")).toHaveValue("nightly-docs");
    expect(onChange).toHaveBeenLastCalledWith(expect.objectContaining({ name: "nightly-docs" }));
  });

  it("Should switch scope to global and clear workspace_id", () => {
    const { onChange } = renderJobForm();

    fireEvent.click(screen.getByTestId("job-scope-global"));

    expect(onChange).toHaveBeenLastCalledWith(
      expect.objectContaining({ scope: "global", workspace_id: undefined })
    );
    expect(screen.getByTestId("job-scope-global")).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByText(/Global jobs aren't bound to a workspace/i)).toBeInTheDocument();
  });

  it("Should choose a workspace from the command selector", async () => {
    const user = userEvent.setup();
    const { onChange } = renderJobForm();

    await user.click(screen.getByTestId("job-workspace-select"));
    await user.click(screen.getByTestId("job-workspace-item-ws_beta"));

    expect(onChange).toHaveBeenLastCalledWith(
      expect.objectContaining({ scope: "workspace", workspace_id: "ws_beta" })
    );
    expect(screen.getByTestId("workspace-switcher-name")).toHaveTextContent("beta");
  });

  it("Should switch output mode to task, reveal task fields, set draft.task, and hide the agent prompt", () => {
    const { onChange } = renderJobForm({
      draft: {
        ...createAutomationJobDraft(WORKSPACE_ID),
        name: "reconcile",
        agent_name: "auditor",
        prompt: "Reconcile the ledger.",
      },
    });

    expect(screen.getByTestId("job-prompt-input")).toBeInTheDocument();
    expect(screen.queryByTestId("job-task-title")).not.toBeInTheDocument();

    fireEvent.click(screen.getByTestId("job-target-task"));

    expect(onChange).toHaveBeenLastCalledWith(
      expect.objectContaining({
        task: expect.objectContaining({ title: "", description: "", owner: null }),
        retry: expect.objectContaining({ strategy: "none" }),
      })
    );
    expect(screen.getByTestId("job-task-title")).toBeInTheDocument();
    expect(screen.getByTestId("job-task-desc")).toBeInTheDocument();
    expect(screen.getByTestId("job-owner-kind")).toBeInTheDocument();
    expect(screen.queryByTestId("job-prompt-input")).not.toBeInTheDocument();
    expect(screen.getByTestId("job-target-task")).toHaveAttribute("aria-pressed", "true");
  });

  it("Should switch back to agent mode and clear draft.task", () => {
    const { onChange } = renderJobForm({
      draft: {
        ...createAutomationJobDraft(WORKSPACE_ID),
        name: "reconcile",
        task: { title: "Reconcile", description: "", owner: null },
        retry: { strategy: "none", max_retries: 0, base_delay: "" },
      },
    });

    expect(screen.getByTestId("job-task-title")).toBeInTheDocument();

    fireEvent.click(screen.getByTestId("job-target-agent"));

    expect(onChange).toHaveBeenLastCalledWith(expect.objectContaining({ task: undefined }));
    expect(screen.getByTestId("job-prompt-input")).toBeInTheDocument();
    expect(screen.queryByTestId("job-task-title")).not.toBeInTheDocument();
    expect(screen.getByTestId("job-target-agent")).toHaveAttribute("aria-pressed", "true");
  });

  it("Should switch the target to Run loop with a static input form and no payload mapping (§9.14)", () => {
    const { onChange } = renderJobForm();

    fireEvent.click(screen.getByTestId("job-target-loop"));
    expect(onChange).toHaveBeenLastCalledWith(
      expect.objectContaining({
        target_kind: "loop",
        loop_target: expect.objectContaining({ loop_name: "" }),
        task: undefined,
      })
    );
    expect(screen.getByTestId("loop-target-fields")).toBeInTheDocument();
    expect(screen.queryByTestId("job-prompt-input")).not.toBeInTheDocument();

    fireEvent.change(screen.getByTestId("loop-target-select"), {
      target: { value: "software-delivery" },
    });
    expect(onChange).toHaveBeenLastCalledWith(
      expect.objectContaining({
        loop_target: expect.objectContaining({ loop_name: "software-delivery" }),
      })
    );
    // Jobs fire on a schedule, not an event, so there is no payload mapping table.
    expect(screen.queryByTestId("loop-input-mapping")).not.toBeInTheDocument();
    expect(screen.getAllByTestId("loop-input-control").length).toBeGreaterThan(0);
  });

  it("Should switch the schedule mode to every and then to at", () => {
    const { onChange } = renderJobForm();

    fireEvent.click(screen.getByTestId("job-schedule-mode-every"));

    expect(onChange).toHaveBeenLastCalledWith(
      expect.objectContaining({ schedule: { mode: "every", interval: "30m" } })
    );
    expect(screen.getByLabelText("Interval")).toHaveValue("30m");
    expect(screen.getByTestId("job-schedule-mode-every")).toHaveAttribute("aria-pressed", "true");

    fireEvent.click(screen.getByTestId("job-schedule-mode-at"));

    const lastCall = onChange.mock.calls.at(-1)?.[0] as CreateAutomationJobRequest;
    expect(lastCall.schedule.mode).toBe("at");
    expect(typeof lastCall.schedule.time).toBe("string");
    expect(lastCall.schedule.time).not.toBe("");
    expect(screen.getByLabelText("Run date and time")).toBeInTheDocument();
    expect(screen.getByTestId("job-schedule-mode-at")).toHaveAttribute("aria-pressed", "true");
  });

  it("Should keep a decodable cron expression editable after selecting Custom", () => {
    const { onChange } = renderJobForm();
    const cronInput = screen.getByLabelText("Cron expression");

    expect(cronInput).toHaveAttribute("readonly");

    fireEvent.click(screen.getByRole("button", { name: "Custom" }));
    expect(cronInput).not.toHaveAttribute("readonly");

    fireEvent.change(cronInput, { target: { value: "*/15 * * * *" } });
    expect(onChange).toHaveBeenLastCalledWith(
      expect.objectContaining({ schedule: { mode: "cron", expr: "*/15 * * * *" } })
    );
    expect(cronInput).not.toHaveAttribute("readonly");

    fireEvent.click(screen.getByRole("button", { name: "Hourly" }));
    expect(cronInput).toHaveAttribute("readonly");
  });

  it("Should disable submit when name, agent, or prompt are empty and enable + call onSubmit when valid", () => {
    const { onChange, onSubmit } = renderJobForm({
      draft: {
        ...createAutomationJobDraft(WORKSPACE_ID),
        name: "",
        agent_name: "",
        prompt: "",
      },
    });

    expect(screen.getByTestId("submit-job-form")).toBeDisabled();

    fireEvent.change(screen.getByTestId("job-name-input"), {
      target: { value: "daily-review" },
    });
    expect(screen.getByTestId("submit-job-form")).toBeDisabled();

    fireEvent.change(screen.getByTestId("job-agent-input"), {
      target: { value: "reviewer" },
    });
    expect(screen.getByTestId("submit-job-form")).toBeDisabled();

    fireEvent.change(screen.getByTestId("job-prompt-input"), {
      target: { value: "Review recent changes." },
    });

    expect(onChange).toHaveBeenLastCalledWith(
      expect.objectContaining({
        name: "daily-review",
        agent_name: "reviewer",
        prompt: "Review recent changes.",
      })
    );
    expect(screen.getByTestId("submit-job-form")).toBeEnabled();

    fireEvent.click(screen.getByTestId("submit-job-form"));
    expect(onSubmit).toHaveBeenCalledOnce();
  });

  it("Should not submit while a pending save is in flight", () => {
    const { onSubmit } = renderJobForm({
      draft: {
        ...createAutomationJobDraft(WORKSPACE_ID),
        name: "daily-review",
        agent_name: "reviewer",
        prompt: "Review recent changes.",
      },
      isPending: true,
    });

    expect(screen.getByTestId("submit-job-form")).toHaveTextContent("Saving...");

    fireEvent.submit(screen.getByTestId("automation-job-form"));

    expect(onSubmit).not.toHaveBeenCalled();
  });
});
