import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { AppHeader } from "./app-header";

describe("AppHeader", () => {
  it("renders wordmark, ALPHA chip, and placeholder nav", () => {
    render(<AppHeader />);
    expect(screen.getByTestId("app-header-wordmark")).toHaveTextContent("agh");
    expect(screen.getByTestId("app-header-alpha-chip")).toHaveTextContent(/alpha/i);
    const nav = screen.getByTestId("app-header-nav");
    expect(nav).toBeInTheDocument();
    expect(nav).toHaveAttribute("aria-label", "Primary");
  });

  it("uses the sticky blurred shell defined by DESIGN.md §4", () => {
    render(<AppHeader />);
    const header = screen.getByTestId("app-header");
    expect(header.className).toContain("sticky");
    expect(header.className).toContain("top-0");
    expect(header.className).toContain("bg-[rgba(20,19,18,0.92)]");
    expect(header.className).toContain("backdrop-blur-xl");
  });

  it("uses the NuixyberNext wordmark typography", () => {
    render(<AppHeader />);
    const wordmark = screen.getByTestId("app-header-wordmark");
    expect(wordmark.className).toContain("font-wordmark");
  });

  it("uses the mono ALPHA chip treatment", () => {
    render(<AppHeader />);
    const chip = screen.getByTestId("app-header-alpha-chip");
    expect(chip.className).toContain("font-mono");
    expect(chip.className).toContain("uppercase");
  });
});
