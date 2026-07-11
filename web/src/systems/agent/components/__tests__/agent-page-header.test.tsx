import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { AgentPageActions, AgentPageStatusPill } from "../agent-page-header";

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

describe("AgentPageActions", () => {
  it("Should wire refresh/configure/new-session controls", async () => {
    const onRefresh = vi.fn();
    const onConfigure = vi.fn();
    const onNewSession = vi.fn();
    render(
      <AgentPageActions
        agent={{
          name: "coder",
          provider: "claude",
          prompt: "x",
          definition_digest: "d".repeat(64),
          origin: "workspace",
        }}
        isRefreshing={false}
        onRefresh={onRefresh}
        onConfigure={onConfigure}
        onNewSession={onNewSession}
        isCreatingSession={false}
        newSessionDisabled={false}
      />
    );
    screen.getByTestId("agent-page-refresh").click();
    screen.getByTestId("agent-page-configure").click();
    screen.getByTestId("agent-page-new-session").click();
    expect(onRefresh).toHaveBeenCalledTimes(1);
    expect(onConfigure).toHaveBeenCalledTimes(1);
    expect(onNewSession).toHaveBeenCalledTimes(1);
  });
});
