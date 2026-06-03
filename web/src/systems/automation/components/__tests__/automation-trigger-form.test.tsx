import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
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
  workspaces?: Array<{ id: string; name: string }>;
}

const WORKSPACES = [
  { id: "ws_alpha", name: "alpha" },
  { id: "ws_beta", name: "beta" },
];

function renderTriggerForm({
  activeWorkspaceId = "ws_alpha",
  draft = createAutomationTriggerDraft(activeWorkspaceId),
  isPending = false,
  mode = "create" as "create" | "edit",
  workspaces = WORKSPACES,
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
        workspaces={workspaces}
      />
    );
  }

  render(<Harness />);

  return { onCancel, onChange, onSubmit };
}

function fillIdentity() {
  fireEvent.change(screen.getByTestId("trigger-name-input"), {
    target: { value: "push-review" },
  });
  fireEvent.change(screen.getByTestId("trigger-agent-input"), {
    target: { value: "reviewer" },
  });
  fireEvent.change(screen.getByTestId("trigger-prompt-input"), {
    target: { value: "Review {{ .Kind }} for session {{ .Data.session_id }}." },
  });
}

describe("AutomationTriggerForm", () => {
  it("selects a webhook event, forces global scope, and submits the webhook payload", () => {
    const { onCancel, onChange, onSubmit } = renderTriggerForm();

    const footer = document.body.querySelector('[data-slot="dialog-footer"]');
    expect(footer).not.toBeNull();
    expect(footer).toHaveAttribute("data-variant", "ruled");
    expect(screen.getByTestId("submit-trigger-form")).toBeDisabled();

    fillIdentity();
    // session.stopped (the default) in a workspace is already submittable.
    expect(screen.getByTestId("submit-trigger-form")).toBeEnabled();

    fireEvent.click(screen.getByTestId("trigger-event-webhook"));
    expect(onChange).toHaveBeenLastCalledWith(
      expect.objectContaining({ event: "webhook", scope: "global", workspace_id: undefined })
    );
    // webhook needs its signed-endpoint fields before it can be created.
    expect(screen.getByTestId("submit-trigger-form")).toBeDisabled();

    fireEvent.change(screen.getByTestId("trigger-endpoint-slug-input"), {
      target: { value: "repo-push" },
    });
    fireEvent.change(screen.getByTestId("trigger-webhook-id-input"), {
      target: { value: "wbh_repo_push" },
    });
    fireEvent.change(screen.getByTestId("trigger-webhook-secret-value-input"), {
      target: { value: "shared-secret" },
    });

    expect(screen.getByTestId("submit-trigger-form")).toBeEnabled();

    fireEvent.click(screen.getByTestId("submit-trigger-form"));
    fireEvent.click(screen.getByText("Cancel"));

    expect(onSubmit).toHaveBeenCalledOnce();
    expect(onCancel).toHaveBeenCalledOnce();
    expect(onChange).toHaveBeenLastCalledWith(
      expect.objectContaining({
        event: "webhook",
        scope: "global",
        workspace_id: undefined,
        endpoint_slug: "repo-push",
        webhook_id: "wbh_repo_push",
        webhook_secret_value: "shared-secret",
      })
    );
  });

  it("composes a hook event id from the inline sub-config", () => {
    const { onChange } = renderTriggerForm();

    fireEvent.click(screen.getByTestId("trigger-event-hook.completed"));
    fireEvent.change(screen.getByTestId("trigger-hook-name-input"), {
      target: { value: "transform" },
    });

    expect(onChange).toHaveBeenLastCalledWith(
      expect.objectContaining({ event: "hook.transform.completed" })
    );
  });

  it("toggles scope between workspace and global for non-webhook events", () => {
    const { onChange } = renderTriggerForm();

    fireEvent.click(screen.getByTestId("trigger-scope-global"));
    expect(onChange).toHaveBeenLastCalledWith(
      expect.objectContaining({ scope: "global", workspace_id: undefined })
    );

    fireEvent.click(screen.getByTestId("trigger-scope-workspace"));
    expect(onChange).toHaveBeenLastCalledWith(
      expect.objectContaining({ scope: "workspace", workspace_id: "ws_alpha" })
    );
  });

  it("selects a different workspace from the command selector", async () => {
    const user = userEvent.setup();
    const { onChange } = renderTriggerForm();

    await user.click(screen.getByTestId("trigger-workspace-select"));
    await user.click(screen.getByTestId("trigger-workspace-item-ws_beta"));

    expect(onChange).toHaveBeenLastCalledWith(
      expect.objectContaining({ scope: "workspace", workspace_id: "ws_beta" })
    );
    expect(screen.getByTestId("workspace-switcher-name")).toHaveTextContent("beta");
  });

  it("adds and edits structured filter conditions as an AND map", () => {
    const { onChange } = renderTriggerForm();

    fireEvent.click(screen.getByRole("button", { name: "Add condition" }));
    expect(onChange).toHaveBeenLastCalledWith(expect.objectContaining({ filter: { kind: "" } }));

    fireEvent.change(screen.getByTestId("trigger-filter-value-0"), {
      target: { value: "session.stopped" },
    });
    fireEvent.change(screen.getByTestId("trigger-filter-key-0"), {
      target: { value: "data.stop_reason" },
    });

    expect(onChange).toHaveBeenLastCalledWith(
      expect.objectContaining({ filter: { "data.stop_reason": "session.stopped" } })
    );

    fireEvent.click(screen.getByTestId("trigger-filter-remove-0"));
    expect(onChange).toHaveBeenLastCalledWith(expect.objectContaining({ filter: {} }));
  });

  it("hides webhook sub-config fields for non-webhook events", () => {
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
        event: "session.stopped",
      },
      isPending: true,
      mode: "edit",
    });

    expect(screen.getByTestId("submit-trigger-form")).toHaveTextContent("Saving...");

    fireEvent.submit(screen.getByTestId("automation-trigger-form"));

    expect(onSubmit).not.toHaveBeenCalled();
  });
});
