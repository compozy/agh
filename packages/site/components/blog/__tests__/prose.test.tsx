import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { Mono, ProseH2, ProseH3 } from "../prose";

describe("blog prose code", () => {
  it("keeps inline code chrome on textual code spans", () => {
    render(<Mono>agh-network/v0</Mono>);

    const code = screen.getByText("agh-network/v0");

    expect(code.className).toContain("border-(--line)");
    expect(code.className).toContain("bg-(--elevated)");
  });

  it("does not apply inline code chrome inside highlighted fenced code blocks", () => {
    render(
      <Mono data-language="bash" data-theme="vitesse-dark" style={{ display: "grid" }}>
        <span>agh network send</span>
      </Mono>
    );

    const code = screen.getByText("agh network send").closest("code");

    expect(code).not.toBeNull();
    expect(code?.className).not.toContain("border-(--line)");
    expect(code?.className).not.toContain("bg-(--elevated)");
    expect(code?.className).toContain("text-inherit");
  });
});

describe("blog prose headings", () => {
  it("derives stable anchors when compiled blog headings do not pass an id", () => {
    render(
      <>
        <ProseH2>Why agents need a workplace</ProseH2>
        <ProseH3>
          Tools and <code>MCP</code>
        </ProseH3>
      </>
    );

    expect(screen.getByRole("heading", { name: "Why agents need a workplace" }).id).toBe(
      "why-agents-need-a-workplace"
    );
    expect(screen.getByRole("heading", { name: "Tools and MCP" }).id).toBe("tools-and-mcp");
  });

  it("preserves explicit heading ids from MDX when present", () => {
    render(<ProseH2 id="custom-anchor">Custom Anchor</ProseH2>);

    expect(screen.getByRole("heading", { name: "Custom Anchor" }).id).toBe("custom-anchor");
  });
});
