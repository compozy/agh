import { fireEvent, render, screen } from "@testing-library/react";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";

import { AutomationJobForm } from "../automation-job-form";
import { createAutomationJobDraft } from "../../lib/automation-drafts";
import type { CreateAutomationJobRequest } from "../../types";

interface RenderJobFormOptions {
  activeWorkspaceId?: string | null;
  draft?: CreateAutomationJobRequest;
  isPending?: boolean;
  mode?: "create" | "edit";
}

function renderJobForm({
  activeWorkspaceId = "ws_alpha",
  draft = createAutomationJobDraft(activeWorkspaceId),
  isPending = false,
  mode = "create" as "create" | "edit",
}: RenderJobFormOptions = {}) {
  const onCancel = vi.fn();
  const onChange = vi.fn();
  const onSubmit = vi.fn();

  function Harness() {
    const [currentDraft, setCurrentDraft] = useState<CreateAutomationJobRequest>(draft);

    return (
      <AutomationJobForm
        activeWorkspaceId={activeWorkspaceId}
        draft={currentDraft}
        isPending={isPending}
        mode={mode}
        onCancel={onCancel}
        onChange={nextDraft => {
          onChange(nextDraft);
          setCurrentDraft(nextDraft);
        }}
        onSubmit={onSubmit}
      />
    );
  }

  render(<Harness />);

  return { onCancel, onChange, onSubmit };
}

describe("AutomationJobForm", () => {
  it("updates core, scope, schedule, governance, and submit state for a create flow", () => {
    const { onCancel, onChange, onSubmit } = renderJobForm();

    expect(screen.getByTestId("submit-job-form")).toBeDisabled();
    expect(screen.getByTestId("submit-job-form")).toHaveTextContent("Create Job");
    expect(screen.getByText("Enabled on create")).toBeInTheDocument();

    fireEvent.change(screen.getByTestId("job-name-input"), {
      target: { value: "nightly-docs" },
    });
    fireEvent.change(screen.getByTestId("job-agent-input"), {
      target: { value: "writer" },
    });
    fireEvent.change(screen.getByTestId("job-prompt-input"), {
      target: { value: "Summarize documentation changes." },
    });

    fireEvent.click(screen.getByTestId("job-scope-global"));
    expect(onChange).toHaveBeenLastCalledWith(
      expect.objectContaining({ scope: "global", workspace_id: undefined })
    );

    fireEvent.click(screen.getByTestId("job-scope-workspace"));
    expect(onChange).toHaveBeenLastCalledWith(
      expect.objectContaining({ scope: "workspace", workspace_id: "ws_alpha" })
    );

    fireEvent.click(screen.getByTestId("job-schedule-mode-every"));
    fireEvent.change(screen.getByTestId("job-schedule-interval"), {
      target: { value: "45m" },
    });

    fireEvent.click(screen.getByTestId("job-schedule-mode-at"));
    fireEvent.change(screen.getByTestId("job-schedule-time"), {
      target: { value: "2026-04-15T15:00:00Z" },
    });

    fireEvent.click(screen.getByTestId("job-retry-strategy-backoff"));
    fireEvent.change(screen.getByTestId("job-retry-max"), {
      target: { value: "5" },
    });
    fireEvent.change(screen.getByTestId("job-retry-delay"), {
      target: { value: "10s" },
    });
    fireEvent.change(screen.getByTestId("job-fire-limit-max"), {
      target: { value: "7" },
    });
    fireEvent.change(screen.getByTestId("job-fire-limit-window"), {
      target: { value: "2h" },
    });
    fireEvent.click(screen.getByTestId("job-enabled-toggle"));

    expect(screen.getByTestId("submit-job-form")).toBeEnabled();

    fireEvent.click(screen.getByTestId("submit-job-form"));
    fireEvent.click(screen.getByText("Cancel"));

    expect(onSubmit).toHaveBeenCalledOnce();
    expect(onCancel).toHaveBeenCalledOnce();
    expect(onChange).toHaveBeenLastCalledWith(
      expect.objectContaining({
        enabled: false,
        fire_limit: { max: 7, window: "2h" },
        retry: { strategy: "backoff", max_retries: 5, base_delay: "10s" },
        schedule: { mode: "at", time: "2026-04-15T15:00:00Z" },
      })
    );
  });

  it("shows workspace guidance when no active workspace is bound", () => {
    const { onChange } = renderJobForm({
      activeWorkspaceId: null,
      draft: createAutomationJobDraft(undefined),
    });

    fireEvent.click(screen.getByTestId("job-scope-workspace"));

    expect(onChange).toHaveBeenLastCalledWith(
      expect.objectContaining({ scope: "workspace", workspace_id: undefined })
    );
  });

  it("resets retry values when switching back to none", () => {
    const { onChange } = renderJobForm();

    expect(screen.getByTestId("job-retry-max")).toBeDisabled();
    expect(screen.getByTestId("job-retry-max")).toHaveValue(0);
    expect(screen.getByTestId("job-retry-delay")).toHaveValue("");

    fireEvent.click(screen.getByTestId("job-retry-strategy-backoff"));
    expect(screen.getByTestId("job-retry-max")).toBeEnabled();
    expect(screen.getByTestId("job-retry-delay")).toHaveValue("2s");

    fireEvent.change(screen.getByTestId("job-retry-max"), {
      target: { value: "4" },
    });
    fireEvent.change(screen.getByTestId("job-retry-delay"), {
      target: { value: "8s" },
    });
    fireEvent.click(screen.getByTestId("job-retry-strategy-none"));

    expect(onChange).toHaveBeenLastCalledWith(
      expect.objectContaining({
        retry: { strategy: "none", max_retries: 0, base_delay: "" },
      })
    );
    expect(screen.getByTestId("job-retry-max")).toBeDisabled();
    expect(screen.getByTestId("job-retry-max")).toHaveValue(0);
    expect(screen.getByTestId("job-retry-delay")).toHaveValue("");
  });

  it("renders edit and pending labels without submitting", () => {
    const { onSubmit } = renderJobForm({
      draft: {
        ...createAutomationJobDraft("ws_alpha"),
        name: "daily-review",
        agent_name: "reviewer",
        prompt: "Review recent changes.",
      },
      isPending: true,
      mode: "edit",
    });

    expect(screen.getByText("Enabled")).toBeInTheDocument();
    expect(screen.getByTestId("submit-job-form")).toHaveTextContent("Saving...");

    fireEvent.submit(screen.getByTestId("automation-job-form"));

    expect(onSubmit).not.toHaveBeenCalled();
  });
});
