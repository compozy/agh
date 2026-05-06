// @vitest-environment jsdom

import { act, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { WorkBanner } from "../work-banner";

describe("WorkBanner auto-hide and escalation (`_design.md` §5.8.2)", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  it("Should render nothing when openCount is 0 from the start", () => {
    render(<WorkBanner hasNeedsInput={false} openCount={0} />);
    expect(screen.queryByTestId("network-work-banner")).toBeNull();
  });

  it("Should render the default tint for working state", () => {
    render(<WorkBanner hasNeedsInput={false} openCount={2} />);
    const banner = screen.getByTestId("network-work-banner");
    expect(banner).toHaveAttribute("data-escalate", "false");
    expect(banner).toHaveTextContent("2 active work in flight");
    expect(banner.className).toContain("color-warning-tint");
  });

  it("Should escalate to solid warning when any work needs input", () => {
    render(<WorkBanner hasNeedsInput openCount={3} />);
    const banner = screen.getByTestId("network-work-banner");
    expect(banner).toHaveAttribute("data-escalate", "true");
    expect(banner).toHaveTextContent("1 needs input · 2 working");
    // Solid warning bg-color, not the tint.
    expect(banner.className).toContain("bg-[color:var(--color-warning)]");
  });

  it("Should auto-hide within 400ms when openCount returns to 0", () => {
    const { rerender } = render(<WorkBanner hasNeedsInput={false} openCount={2} />);
    expect(screen.getByTestId("network-work-banner")).toBeInTheDocument();

    rerender(<WorkBanner hasNeedsInput={false} openCount={0} />);
    const fading = screen.getByTestId("network-work-banner");
    expect(fading).toHaveAttribute("data-state", "fading");

    act(() => {
      vi.advanceTimersByTime(400);
    });
    expect(screen.queryByTestId("network-work-banner")).toBeNull();
  });

  it("Should render the explicit breakdown when needsInputCount + workingCount are provided", () => {
    render(<WorkBanner hasNeedsInput needsInputCount={2} openCount={3} workingCount={1} />);
    const banner = screen.getByTestId("network-work-banner");
    expect(banner).toHaveAttribute("data-escalate", "true");
    expect(banner).toHaveTextContent("2 needs input · 1 working");
  });

  it("Should omit zero buckets in the breakdown", () => {
    render(<WorkBanner hasNeedsInput={false} needsInputCount={0} openCount={2} workingCount={2} />);
    expect(screen.getByTestId("network-work-banner")).toHaveTextContent("2 working");
  });
});
