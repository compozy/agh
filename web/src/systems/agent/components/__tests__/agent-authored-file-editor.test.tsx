import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { AgentDigestConflictError } from "../../adapters/agent-api";
import type { AgentSoulHistoryResponse, AgentSoulPayload } from "../../types";
import { AgentAuthoredFileEditor } from "../agent-authored-file-editor";

vi.mock("../../hooks/use-unsaved-guard", () => ({
  useUnsavedGuard: () => ({ confirmDialog: null }),
}));

function soulPayload(overrides: Partial<AgentSoulPayload> = {}): AgentSoulPayload {
  return {
    active: true,
    present: true,
    enabled: true,
    valid: true,
    validation_status: "valid",
    digest: "soul-digest",
    source_path: "agents/coder/SOUL.md",
    frontmatter: {
      version: "1",
      role: "coder",
      tone: ["concise"],
      principles: ["Keep scope tight"],
    },
    body: "# Soul\n\nShip carefully.",
    limits: { max_body_bytes: 65536 },
    config_provenance: {
      digest: "config-digest",
      enabled: true,
      max_body_bytes: 65536,
      context_projection_bytes: 0,
    },
    ...overrides,
  };
}

const history = {
  revisions: [
    {
      id: "rev-1",
      action: "put",
      actor: { kind: "user" },
      agent_name: "coder",
      created_at: "2026-07-11T10:00:00Z",
      source_path: "agents/coder/SOUL.md",
    },
  ],
} as AgentSoulHistoryResponse;

describe("AgentAuthoredFileEditor", () => {
  it("Should edit the complete source, validate diagnostics, save with CAS, and restore history", async () => {
    const user = userEvent.setup();
    const onValidate = vi.fn().mockResolvedValue({
      validation_status: "invalid",
      diagnostics: [{ message: "Role is required", line: 2 }],
    });
    const onSave = vi.fn().mockResolvedValue(undefined);
    const onRestore = vi.fn().mockResolvedValue(undefined);
    render(
      <AgentAuthoredFileEditor
        kind="soul"
        payload={soulPayload()}
        isLoading={false}
        isError={false}
        history={history}
        onValidate={onValidate}
        onSave={onSave}
        onRestore={onRestore}
        onRetry={vi.fn()}
      />
    );

    const editor = screen.getByTestId("agent-soul-textarea");
    expect((editor as HTMLTextAreaElement).value).toContain('role: "coder"');
    await user.type(editor, "\nNew constraint.");
    await user.click(screen.getByTestId("agent-soul-validate"));
    expect(onValidate).toHaveBeenCalledWith(expect.stringContaining("New constraint."));
    expect(await screen.findByText("Role is required")).toBeVisible();

    await user.click(screen.getByTestId("agent-soul-save"));
    expect(onSave).toHaveBeenCalledWith(
      expect.stringContaining('principles:\n  - "Keep scope tight"'),
      "soul-digest"
    );

    await user.click(screen.getByTestId("agent-soul-history-toggle"));
    await user.click(screen.getByTestId("agent-soul-restore-rev-1"));
    expect(onRestore).toHaveBeenCalledWith("rev-1", "soul-digest");
  });

  it("Should render reload-and-retry recovery for a stale digest", async () => {
    const user = userEvent.setup();
    const retry = vi.fn();
    const onSave = vi.fn().mockRejectedValue(new AgentDigestConflictError("stale"));
    render(
      <AgentAuthoredFileEditor
        kind="soul"
        payload={soulPayload()}
        isLoading={false}
        isError={false}
        history={history}
        onValidate={vi.fn().mockResolvedValue({})}
        onSave={onSave}
        onRestore={vi.fn().mockResolvedValue(undefined)}
        onRetry={retry}
      />
    );

    await user.type(screen.getByTestId("agent-soul-textarea"), " changed");
    await user.click(screen.getByTestId("agent-soul-save"));
    expect(await screen.findByText("This file changed elsewhere. Reload and retry.")).toBeVisible();
    await user.click(screen.getByTestId("agent-soul-reload"));
    expect(retry).toHaveBeenCalledTimes(1);
  });

  it("Should create a missing file through PUT with the read digest", async () => {
    const user = userEvent.setup();
    const onSave = vi.fn().mockResolvedValue(undefined);
    render(
      <AgentAuthoredFileEditor
        kind="soul"
        payload={soulPayload({ present: false, active: false, validation_status: "missing" })}
        isLoading={false}
        isError={false}
        history={{ revisions: [] } as AgentSoulHistoryResponse}
        onValidate={vi.fn().mockResolvedValue({})}
        onSave={onSave}
        onRestore={vi.fn().mockResolvedValue(undefined)}
        onRetry={vi.fn()}
      />
    );

    await user.click(screen.getByTestId("agent-soul-create"));
    expect(onSave).toHaveBeenCalledWith(expect.stringContaining("# Soul"), "soul-digest");
  });
});
