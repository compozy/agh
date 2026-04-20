import { render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";

vi.mock("@tanstack/react-router", () => ({
  createFileRoute: () => (opts: { component: () => ReactNode }) => ({
    component: opts.component,
  }),
}));

import { Route } from "./index";

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const SettingsIndexPage = (Route as any).component as () => ReactNode;

describe("SettingsIndexPage", () => {
  it("renders the placeholder empty state for the default settings route", () => {
    render(<SettingsIndexPage />);

    expect(screen.getByTestId("settings-index-placeholder")).toBeInTheDocument();
    expect(screen.getByTestId("settings-index-empty")).toBeInTheDocument();
    expect(screen.getByText("Select a settings section")).toBeInTheDocument();
    expect(screen.getByText("Choose a section from the left to configure AGH")).toBeInTheDocument();
  });
});
