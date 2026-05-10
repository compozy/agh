import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { AgentInfoPanel } from "../agent-info-panel";
import type { AgentPayload } from "../../types";

function makeAgent(overrides: Partial<AgentPayload> = {}): AgentPayload {
  return {
    name: "codex-agent",
    provider: "codex",
    prompt: "Coordinate implementation tasks.",
    ...overrides,
  } as AgentPayload;
}

describe("AgentInfoPanel", () => {
  it("Should render MCP servers with the shared Item row contract", () => {
    render(
      <AgentInfoPanel
        agent={makeAgent({
          mcp_servers: [
            {
              name: "github",
              command: "github-mcp",
              transport: "stdio",
            },
          ],
        })}
      />
    );

    const row = screen.getByTestId("agent-info-mcp-row-github");
    expect(row).toHaveAttribute("data-slot", "item");
    expect(row).toHaveAttribute("role", "listitem");
    expect(row).toHaveTextContent("github");
    expect(row).toHaveTextContent("github-mcp");
    expect(screen.getByTestId("agent-info-mcp-kind-github")).toHaveTextContent("stdio");
  });

  it("Should keep the existing MCP empty-state copy", () => {
    render(<AgentInfoPanel agent={makeAgent({ mcp_servers: [] })} />);

    expect(screen.getByTestId("agent-info-mcp-empty")).toHaveTextContent("No MCP servers");
    expect(screen.getByTestId("agent-info-mcp-empty")).toHaveAttribute("data-fill", "false");
  });

  it("Should drive the panel width from the --rail-inspector-w CSS custom property", () => {
    render(<AgentInfoPanel agent={makeAgent()} />);
    const panel = screen.getByTestId("agent-info-panel") as HTMLElement;
    expect(panel.style.width).toContain("var(--rail-inspector-w");
  });
});
