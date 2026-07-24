import { render, waitFor } from "@testing-library/react";
import { cloneElement, isValidElement, type ReactElement, type ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";

// jsdom reports zero-sized layout boxes; pin a deterministic chart size so
// recharts actually renders bars and paths.
vi.mock("recharts", async () => {
  const actual = await vi.importActual<typeof import("recharts")>("recharts");
  function MockResponsiveContainer({ children }: { children: ReactNode }) {
    if (isValidElement(children)) {
      const element = children as ReactElement<{
        width?: number | string;
        height?: number | string;
      }>;
      return cloneElement(element, { width: 320, height: 120 });
    }
    return <div style={{ width: 320, height: 120 }}>{children}</div>;
  }
  return { ...actual, ResponsiveContainer: MockResponsiveContainer };
});

import { DayAreaChart } from "../day-area-chart";
import { DayStackedBars } from "../day-stacked-bars";

describe("DayStackedBars", () => {
  it("Should render caller-supplied series fills with aria metadata", async () => {
    const { container } = render(
      <DayStackedBars
        ariaLabel="Run outcomes per day"
        data={[
          { date: "Jul 16", completed: 3, failed: 1 },
          { date: "Jul 17", completed: 5, failed: 0 },
        ]}
        height={90}
        series={[
          { key: "completed", fill: "var(--color-success)" },
          { key: "failed", fill: "var(--color-danger)" },
        ]}
      />
    );

    const root = container.querySelector<HTMLElement>('[data-slot="day-stacked-bars"]');
    expect(root?.getAttribute("role")).toBe("img");
    expect(root?.getAttribute("aria-label")).toBe("Run outcomes per day");
    expect(root?.style.height).toBe("90px");

    await waitFor(() => {
      const fills = Array.from(container.querySelectorAll("path, rect")).map(node =>
        node.getAttribute("fill")
      );
      expect(fills).toContain("var(--color-success)");
      expect(fills).toContain("var(--color-danger)");
    });
  });
});

describe("DayAreaChart", () => {
  it("Should render the neutral-ink area with aria metadata", async () => {
    const { container } = render(
      <DayAreaChart
        ariaLabel="Tokens per day"
        data={[
          { date: "Jul 16", value: 12_000 },
          { date: "Jul 17", value: 48_000 },
        ]}
      />
    );

    const root = container.querySelector<HTMLElement>('[data-slot="day-area-chart"]');
    expect(root?.getAttribute("role")).toBe("img");
    expect(root?.getAttribute("aria-label")).toBe("Tokens per day");

    await waitFor(() => {
      const strokes = Array.from(container.querySelectorAll("path")).map(node =>
        node.getAttribute("stroke")
      );
      expect(strokes).toContain("var(--color-viz-line)");
    });
  });
});
