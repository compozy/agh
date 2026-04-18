import { render, screen, waitFor } from "@testing-library/react";
import { useReducedMotionConfig } from "motion/react";
import type { ReactNode } from "react";
import { describe, expect, it } from "vitest";

import { UIProvider, type UIProviderProps } from "./ui-provider";

function Probe() {
  const reduced = useReducedMotionConfig();
  return <span data-testid="probe">{String(reduced ?? "pending")}</span>;
}

function renderWithProvider(props?: Partial<UIProviderProps>, child: ReactNode = <Probe />) {
  return render(<UIProvider {...props}>{child}</UIProvider>);
}

describe("UIProvider", () => {
  it("Should render children without crashing under the default config", () => {
    renderWithProvider({}, <span data-testid="child">content</span>);
    expect(screen.getByTestId("child")).toHaveTextContent("content");
  });

  it("Should forward reducedMotion='always' to MotionConfig consumers", async () => {
    renderWithProvider({ reducedMotion: "always" });
    await waitFor(() => expect(screen.getByTestId("probe")).toHaveTextContent("true"));
  });

  it("Should forward reducedMotion='never' to MotionConfig consumers", async () => {
    renderWithProvider({ reducedMotion: "never" });
    await waitFor(() => expect(screen.getByTestId("probe")).toHaveTextContent("false"));
  });

  it("Should default to reducedMotion='user' which defers to OS preference", async () => {
    // test-setup.ts matchMedia returns `matches: true` for prefers-reduced-motion: reduce,
    // so 'user' mode should resolve to reduced=true in this environment.
    renderWithProvider();
    await waitFor(() => expect(screen.getByTestId("probe")).toHaveTextContent("true"));
  });
});
