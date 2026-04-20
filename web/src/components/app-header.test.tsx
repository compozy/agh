import { render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it } from "vitest";
import { vi } from "vitest";

import { AppHeader } from "./app-header";

let matchedRoute: Record<string, boolean> = {};

vi.mock("@tanstack/react-router", () => ({
  Link: ({
    children,
    to,
    ...props
  }: {
    children: ReactNode;
    to: string;
    [key: string]: unknown;
  }) => (
    <a href={to} {...props}>
      {children}
    </a>
  ),
  useMatchRoute: () => (opts: { to: string }) => matchedRoute[opts.to] ?? false,
}));

describe("AppHeader", () => {
  it("renders wordmark, ALPHA chip, and dashboard navigation", () => {
    matchedRoute = {};
    render(<AppHeader />);
    expect(screen.getByTestId("app-header-wordmark")).toHaveTextContent("agh");
    expect(screen.getByTestId("app-header-alpha-chip")).toHaveTextContent("ALPHA");
    expect(screen.getByTestId("app-header-home")).toHaveAttribute("href", "/");
    expect(screen.getByTestId("app-header-nav-dashboard")).toHaveAttribute("href", "/");
    const nav = screen.getByTestId("app-header-nav");
    expect(nav).toBeInTheDocument();
    expect(nav).toHaveAttribute("aria-label", "Primary");
  });

  it("uses the sticky blurred shell defined by DESIGN.md §4", () => {
    matchedRoute = {};
    render(<AppHeader />);
    const header = screen.getByTestId("app-header");
    expect(header.className).toContain("sticky");
    expect(header.className).toContain("top-0");
    expect(header.className).toContain("bg-[rgba(20,19,18,0.92)]");
    expect(header.className).toContain("backdrop-blur-xl");
  });

  it("uses the NuixyberNext wordmark typography", () => {
    matchedRoute = {};
    render(<AppHeader />);
    const wordmark = screen.getByTestId("app-header-wordmark");
    expect(wordmark.className).toContain("font-wordmark");
  });

  it("uses the mono ALPHA chip treatment", () => {
    matchedRoute = {};
    render(<AppHeader />);
    const chip = screen.getByTestId("app-header-alpha-chip");
    expect(chip.className).toContain("font-mono");
    expect(chip.className).toContain("uppercase");
  });

  it("marks the dashboard nav link active on the home route", () => {
    matchedRoute = { "/": true };
    render(<AppHeader />);
    expect(screen.getByTestId("app-header-nav-dashboard")).toHaveAttribute("data-active", "true");
  });
});
