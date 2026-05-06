import { fireEvent, render, screen } from "@testing-library/react";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";

import { AutomationTriggerForm } from "../automation-trigger-form";
import { createAutomationTriggerDraft } from "../../lib/automation-drafts";
import type { CreateAutomationTriggerRequest } from "../../types";

interface RenderTriggerFormOptions {
  activeWorkspaceId?: string | null;
  draft?: CreateAutomationTriggerRequest;
  isPending?: boolean;
  mode?: "create" | "edit";
}

function renderTriggerForm({
  activeWorkspaceId = "ws_alpha",
  draft = createAutomationTriggerDraft(activeWorkspaceId),
  isPending = false,
  mode = "create" as "create" | "edit",
}: RenderTriggerFormOptions = {}) {
  const onCancel = vi.fn();
  const onChange = vi.fn();
  const onSubmit = vi.fn();

  function Harness() {
    const [currentDraft, setCurrentDraft] = useState<CreateAutomationTriggerRequest>(draft);

    return (
      <AutomationTriggerForm
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

describe("AutomationTriggerForm", () => {
  it("updates trigger fields, parses filters, and submits webhook triggers", () => {
    const { onCancel, onChange, onSubmit } = renderTriggerForm();

    expect(screen.getByTestId("submit-trigger-form")).toBeDisabled();

    fireEvent.change(screen.getByTestId("trigger-name-input"), {
      target: { value: "push-review" },
    });
    fireEvent.change(screen.getByTestId("trigger-agent-input"), {
      target: { value: "reviewer" },
    });
    fireEvent.change(screen.getByTestId("trigger-event-input"), {
      target: { value: "webhook" },
    });
    fireEvent.change(screen.getByTestId("trigger-prompt-input"), {
      target: { value: "Review push event {{ .Data.branch }}." },
    });
    fireEvent.change(screen.getByTestId("trigger-filter-input"), {
      target: { value: "data.branch=main\nraw-line\n=ignored" },
    });

    fireEvent.change(screen.getByTestId("trigger-endpoint-slug-input"), {
      target: { value: "repo-push" },
    });
    fireEvent.change(screen.getByTestId("trigger-webhook-id-input"), {
      target: { value: "wbh_repo_push" },
    });
    fireEvent.change(screen.getByTestId("trigger-webhook-secret-value-input"), {
      target: { value: "shared-secret" },
    });

    fireEvent.click(screen.getByTestId("trigger-scope-global"));
    expect(onChange).toHaveBeenLastCalledWith(
      expect.objectContaining({ scope: "global", workspace_id: undefined })
    );

    fireEvent.click(screen.getByTestId("trigger-scope-workspace"));
    fireEvent.click(screen.getByTestId("trigger-governance-toggle"));
    fireEvent.click(screen.getByTestId("trigger-retry-strategy-backoff"));
    fireEvent.change(screen.getByTestId("trigger-retry-max"), {
      target: { value: "6" },
    });
    fireEvent.change(screen.getByTestId("trigger-retry-delay"), {
      target: { value: "9s" },
    });
    fireEvent.change(screen.getByTestId("trigger-fire-limit-max"), {
      target: { value: "11" },
    });
    fireEvent.change(screen.getByTestId("trigger-fire-limit-window"), {
      target: { value: "3h" },
    });
    fireEvent.click(screen.getByTestId("trigger-enabled-toggle"));

    expect(screen.getByTestId("submit-trigger-form")).toBeEnabled();

    fireEvent.click(screen.getByTestId("submit-trigger-form"));
    fireEvent.click(screen.getByText("Cancel"));

    expect(onSubmit).toHaveBeenCalledOnce();
    expect(onCancel).toHaveBeenCalledOnce();
    expect(onChange).toHaveBeenLastCalledWith(
      expect.objectContaining({
        enabled: false,
        endpoint_slug: "repo-push",
        webhook_id: "wbh_repo_push",
        webhook_secret_value: "shared-secret",
        filter: { "data.branch": "main", "raw-line": "" },
        retry: { strategy: "backoff", max_retries: 6, base_delay: "9s" },
        fire_limit: { max: 11, window: "3h" },
      })
    );
  });

  it("hides webhook-only fields for non-webhook events", () => {
    renderTriggerForm({
      draft: {
        ...createAutomationTriggerDraft("ws_alpha"),
        name: "session-review",
        agent_name: "reviewer",
        prompt: "Review stopped session",
        event: "session.stopped",
      },
    });

    expect(screen.queryByTestId("trigger-endpoint-slug-input")).not.toBeInTheDocument();
    expect(screen.queryByTestId("trigger-webhook-id-input")).not.toBeInTheDocument();
    expect(screen.queryByTestId("trigger-webhook-secret-ref-input")).not.toBeInTheDocument();
    expect(screen.queryByTestId("trigger-webhook-secret-value-input")).not.toBeInTheDocument();
  });

  it("resets trigger retry values when switching back to none", () => {
    const { onChange } = renderTriggerForm();

    fireEvent.click(screen.getByTestId("trigger-governance-toggle"));

    expect(screen.getByTestId("trigger-retry-max")).toBeDisabled();
    expect(screen.getByTestId("trigger-retry-max")).toHaveValue(0);
    expect(screen.getByTestId("trigger-retry-delay")).toHaveValue("");

    fireEvent.click(screen.getByTestId("trigger-retry-strategy-backoff"));
    fireEvent.change(screen.getByTestId("trigger-retry-max"), {
      target: { value: "6" },
    });
    fireEvent.change(screen.getByTestId("trigger-retry-delay"), {
      target: { value: "9s" },
    });
    fireEvent.click(screen.getByTestId("trigger-retry-strategy-none"));

    expect(onChange).toHaveBeenLastCalledWith(
      expect.objectContaining({
        retry: { strategy: "none", max_retries: 0, base_delay: "" },
      })
    );
    expect(screen.getByTestId("trigger-retry-max")).toBeDisabled();
    expect(screen.getByTestId("trigger-retry-max")).toHaveValue(0);
    expect(screen.getByTestId("trigger-retry-delay")).toHaveValue("");
  });

  it("renders edit and pending labels without submitting", () => {
    const { onSubmit } = renderTriggerForm({
      draft: {
        ...createAutomationTriggerDraft("ws_alpha"),
        name: "push-review",
        agent_name: "reviewer",
        prompt: "Review push event.",
      },
      isPending: true,
      mode: "edit",
    });

    expect(screen.getByTestId("submit-trigger-form")).toHaveTextContent("Saving...");

    fireEvent.submit(screen.getByTestId("automation-trigger-form"));

    expect(onSubmit).not.toHaveBeenCalled();
  });
});
