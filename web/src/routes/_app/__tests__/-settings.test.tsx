import { render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

let matchedRoutes: Record<string, boolean> = {};

vi.mock("@tanstack/react-router", () => ({
  createFileRoute: () => (opts: { component: () => ReactNode }) => ({
    component: opts.component,
  }),
  Outlet: () => <div data-testid="settings-outlet-marker" />,
  Link: ({
    children,
    to,
    ...rest
  }: {
    children: ReactNode;
    to: string;
    [key: string]: unknown;
  }) => (
    <a href={to} {...(rest as Record<string, string | undefined>)}>
      {children}
    </a>
  ),
  useMatchRoute: () => (opts: { to: string; fuzzy?: boolean }) => matchedRoutes[opts.to] ?? false,
}));

import { routeComponent } from "@/test/route-options";
import { Route, SETTINGS_SECTIONS } from "../settings";

const SettingsShell = routeComponent(Route);

describe("SettingsShell", () => {
  beforeEach(() => {
    matchedRoutes = {};
  });

  it("renders the shell frame with section nav and outlet container", () => {
    render(<SettingsShell />);
    expect(screen.getByTestId("settings-shell")).toBeInTheDocument();
    expect(screen.getByTestId("settings-section-nav")).toBeInTheDocument();
    expect(screen.getByTestId("settings-shell-outlet")).toBeInTheDocument();
    expect(screen.getByTestId("settings-outlet-marker")).toBeInTheDocument();
  });

  it("renders a nav link for every declared settings section", () => {
    render(<SettingsShell />);

    for (const section of SETTINGS_SECTIONS) {
      const link = screen.getByTestId(`settings-section-${section.slug}`);
      expect(link).toHaveAttribute("href", `/settings/${section.slug}`);
      expect(link).toHaveTextContent(section.label);
    }
  });

  it("marks the matching section as active when the current route fuzzy-matches its path", () => {
    matchedRoutes["/settings/general"] = true;
    render(<SettingsShell />);

    const active = screen.getByTestId("settings-section-general");
    expect(active).toHaveAttribute("data-active", "true");
    expect(active).toHaveAttribute("aria-current", "page");
    expect(screen.getByTestId("settings-section-active-general")).toBeInTheDocument();
  });

  it("only activates one section at a time when the active route changes", () => {
    matchedRoutes["/settings/skills"] = true;
    render(<SettingsShell />);

    expect(screen.getByTestId("settings-section-skills")).toHaveAttribute("data-active", "true");

    for (const section of SETTINGS_SECTIONS) {
      if (section.slug === "skills") continue;
      expect(screen.getByTestId(`settings-section-${section.slug}`)).toHaveAttribute(
        "data-active",
        "false"
      );
      expect(
        screen.queryByTestId(`settings-section-active-${section.slug}`)
      ).not.toBeInTheDocument();
    }
  });

  it("renders no active section when the settings root is loaded without a child section", () => {
    render(<SettingsShell />);

    for (const section of SETTINGS_SECTIONS) {
      expect(screen.getByTestId(`settings-section-${section.slug}`)).toHaveAttribute(
        "data-active",
        "false"
      );
    }
    expect(screen.queryByTestId(/settings-section-active-/)).not.toBeInTheDocument();
  });

  it("exposes section slugs matching the Paper design reference set", () => {
    expect(SETTINGS_SECTIONS.map(section => section.slug)).toEqual([
      "general",
      "providers",
      "vault",
      "mcp-servers",
      "memory",
      "skills",
      "automation",
      "network",
      "observability",
      "hooks-extensions",
    ]);
  });
});
