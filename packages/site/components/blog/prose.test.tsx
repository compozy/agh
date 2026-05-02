import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { Mono } from "./prose";

describe("blog prose code", () => {
  it("keeps inline code chrome on textual code spans", () => {
    render(<Mono>agh-network/v0</Mono>);

    const code = screen.getByText("agh-network/v0");

    expect(code.className).toContain("border-(--color-divider)");
    expect(code.className).toContain("bg-(--color-surface-elevated)");
  });

  it("does not apply inline code chrome inside highlighted fenced code blocks", () => {
    render(
      <Mono data-language="bash" data-theme="vitesse-dark" style={{ display: "grid" }}>
        <span>agh network send</span>
      </Mono>
    );

    const code = screen.getByText("agh network send").closest("code");

    expect(code).not.toBeNull();
    expect(code?.className).not.toContain("border-(--color-divider)");
    expect(code?.className).not.toContain("bg-(--color-surface-elevated)");
    expect(code?.className).toContain("text-inherit");
  });
});
