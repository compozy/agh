import { render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { DETAIL_INSPECTOR_INLINE_BREAKPOINT } from "@agh/ui";

import { AgentInfoInspector } from "../agent-info-inspector";
import type { AgentPayload } from "../../types";

const ORIGINAL_MATCH_MEDIA = window.matchMedia;

function installMatchMedia(matches: boolean): void {
  Object.defineProperty(window, "matchMedia", {
    writable: true,
    configurable: true,
    value: (query: string) => ({
      matches,
      media: query,
      onchange: null,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      addListener: vi.fn(),
      removeListener: vi.fn(),
      dispatchEvent: () => false,
    }),
  });
}

beforeEach(() => {
  installMatchMedia(true);
});

afterEach(() => {
  Object.defineProperty(window, "matchMedia", {
    writable: true,
    configurable: true,
    value: ORIGINAL_MATCH_MEDIA,
  });
});

function makeAgent(overrides: Partial<AgentPayload> = {}): AgentPayload {
  return {
    name: "codex-agent",
    provider: "codex",
    prompt: "Coordinate implementation tasks.",
    ...overrides,
  } as AgentPayload;
}

describe("AgentInfoInspector", () => {
  it("Should render MCP servers with the shared Item row contract", () => {
    render(
      <AgentInfoInspector
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
  });

  it("Should render the transport label as an <Eyebrow> emitting uppercase STDIO/HTTP", () => {
    render(
      <AgentInfoInspector
        agent={makeAgent({
          mcp_servers: [
            { name: "github", command: "github-mcp", transport: "stdio" },
            { name: "fly", url: "https://fly.io", transport: "http" },
          ],
        })}
      />
    );

    const stdio = screen.getByTestId("agent-info-mcp-kind-github");
    expect(stdio).toHaveTextContent("STDIO");
    expect(stdio.tagName.toLowerCase()).toBe("span");
    expect(stdio.className).toContain("eyebrow");

    const http = screen.getByTestId("agent-info-mcp-kind-fly");
    expect(http).toHaveTextContent("HTTP");
  });

  it("Should never paint the transport as <Pill mono tone=info>", () => {
    render(
      <AgentInfoInspector
        agent={makeAgent({
          mcp_servers: [{ name: "github", command: "github-mcp", transport: "stdio" }],
        })}
      />
    );

    const kind = screen.getByTestId("agent-info-mcp-kind-github");
    expect(kind.getAttribute("data-slot")).not.toBe("pill");
    expect(kind.getAttribute("data-tone")).not.toBe("info");
  });

  it("Should keep the existing MCP empty-state copy", () => {
    render(<AgentInfoInspector agent={makeAgent({ mcp_servers: [] })} />);

    expect(screen.getByTestId("agent-info-mcp-empty")).toHaveTextContent("No MCP servers");
    expect(screen.getByTestId("agent-info-mcp-empty")).toHaveAttribute("data-fill", "false");
  });

  it("Should consume <DetailInspector> without a tab strip", () => {
    const { container } = render(<AgentInfoInspector agent={makeAgent()} />);
    const root = container.querySelector<HTMLElement>(
      '[data-slot="detail-inspector"][data-mode="inline"]'
    );
    expect(root).not.toBeNull();
    expect(root?.style.width).toBe("320px");
    expect(container.querySelector('[data-slot="detail-inspector-tabs"]')).toBeNull();
    expect(screen.queryByRole("tab")).toBeNull();
  });

  it("Should expose the canonical 1440 px breakpoint constant", () => {
    expect(DETAIL_INSPECTOR_INLINE_BREAKPOINT).toBe(1440);
  });
});
