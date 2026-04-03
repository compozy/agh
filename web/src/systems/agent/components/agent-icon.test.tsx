import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { AgentIcon, providerIconMap } from "./agent-icon";

describe("AgentIcon", () => {
  it('maps "claude" provider to BrainCircuit icon', () => {
    render(<AgentIcon provider="claude" data-testid="icon" />);
    const icon = screen.getByTestId("icon");
    expect(icon).toBeInTheDocument();
    expect(icon.tagName.toLowerCase()).toBe("svg");
  });

  it('maps "codex" provider to Code icon', () => {
    render(<AgentIcon provider="codex" data-testid="icon" />);
    expect(screen.getByTestId("icon")).toBeInTheDocument();
  });

  it('maps "gemini" provider to Sparkles icon', () => {
    render(<AgentIcon provider="gemini" data-testid="icon" />);
    expect(screen.getByTestId("icon")).toBeInTheDocument();
  });

  it("returns fallback icon for unknown provider", () => {
    render(<AgentIcon provider="unknown-provider" data-testid="icon" />);
    expect(screen.getByTestId("icon")).toBeInTheDocument();
  });

  it("is case-insensitive for provider matching", () => {
    render(<AgentIcon provider="Claude" data-testid="icon" />);
    expect(screen.getByTestId("icon")).toBeInTheDocument();
  });

  it("has known providers in providerIconMap", () => {
    expect(providerIconMap).toHaveProperty("claude");
    expect(providerIconMap).toHaveProperty("codex");
    expect(providerIconMap).toHaveProperty("gemini");
    expect(providerIconMap).toHaveProperty("openai");
    expect(providerIconMap).toHaveProperty("ollama");
  });

  it("applies custom className", () => {
    render(<AgentIcon provider="claude" data-testid="icon" className="size-8" />);
    const icon = screen.getByTestId("icon");
    expect(icon.getAttribute("class")).toContain("size-8");
  });
});
