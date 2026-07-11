import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { AgentPageActions, AgentPageMeta, AgentPageStatusPill } from "../agent-page-header";

describe("AgentPageStatusPill", () => {
  it("Should show ACTIVE when any session is active", () => {
    render(<AgentPageStatusPill activeCount={1} />);
    expect(screen.getByTestId("agent-page-header-status")).toHaveTextContent("ACTIVE");
  });

  it("Should show IDLE when no sessions are active", () => {
    render(<AgentPageStatusPill activeCount={0} />);
    expect(screen.getByTestId("agent-page-header-status")).toHaveTextContent("IDLE");
  });
});

describe("AgentPageMeta", () => {
  it("Should render category and origin meta", () => {
    render(
      <AgentPageMeta
        agent={{
          name: "coder",
          provider: "claude",
          prompt: "x",
          definition_digest: "d".repeat(64),
          origin: "workspace",
          category_path: ["ops", "release"],
        }}
      />
    );
    expect(screen.getByTestId("agent-page-meta")).toHaveTextContent(/ops/);
    expect(screen.getByTestId("agent-page-meta")).toHaveTextContent(/Workspace|workspace/i);
  });
});

describe("AgentPageActions", () => {
  it("Should expose New session, Edit settings, Duplicate, and Delete only", async () => {
    const user = userEvent.setup();
    const onEditSettings = vi.fn();
    const onNewSession = vi.fn();
    const onDuplicate = vi.fn();
    const onDelete = vi.fn();

    render(
      <AgentPageActions
        onEditSettings={onEditSettings}
        onNewSession={onNewSession}
        onDuplicate={onDuplicate}
        onDelete={onDelete}
        isCreatingSession={false}
        newSessionDisabled={false}
      />
    );

    expect(screen.queryByTestId("agent-page-configure")).not.toBeInTheDocument();
    expect(screen.queryByTestId("agent-page-refresh")).not.toBeInTheDocument();
    expect(screen.queryByText("Pause")).not.toBeInTheDocument();
    expect(screen.queryByText("Reset")).not.toBeInTheDocument();
    expect(screen.queryByText("Rename")).not.toBeInTheDocument();

    await user.click(screen.getByTestId("agent-page-new-session"));
    await user.click(screen.getByTestId("agent-page-edit-settings"));
    expect(onNewSession).toHaveBeenCalledTimes(1);
    expect(onEditSettings).toHaveBeenCalledTimes(1);

    await user.click(screen.getByRole("button", { name: "More agent actions" }));
    await user.click(await screen.findByRole("menuitem", { name: "Duplicate" }));
    await user.click(screen.getByRole("button", { name: "More agent actions" }));
    await user.click(await screen.findByRole("menuitem", { name: "Delete…" }));
    expect(onDuplicate).toHaveBeenCalledTimes(1);
    expect(onDelete).toHaveBeenCalledTimes(1);
  });
});
