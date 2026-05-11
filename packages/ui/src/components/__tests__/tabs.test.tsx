import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";

import { Tabs, TabsContent, TabsList, TabsTrigger } from "../tabs";

function TabsExample({
  orientation,
  variant,
}: {
  orientation?: "horizontal" | "vertical";
  variant?: "line" | "lane";
}) {
  return (
    <Tabs defaultValue="one" orientation={orientation}>
      <TabsList variant={variant}>
        <TabsTrigger value="one">One</TabsTrigger>
        <TabsTrigger value="two">Two</TabsTrigger>
        <TabsTrigger value="three">Three</TabsTrigger>
      </TabsList>
      <TabsContent value="one">Panel one</TabsContent>
      <TabsContent value="two">Panel two</TabsContent>
      <TabsContent value="three">Panel three</TabsContent>
    </Tabs>
  );
}

describe("Tabs", () => {
  it("Should render horizontally by default and activate the initial panel", () => {
    const { container } = render(<TabsExample />);
    const root = container.querySelector("[data-slot=tabs]") as HTMLElement | null;
    expect(root).not.toBeNull();
    expect(root).toHaveAttribute("data-orientation", "horizontal");
    expect(screen.getByText("Panel one")).toBeInTheDocument();
  });

  it("Should honor orientation='vertical'", () => {
    const { container } = render(<TabsExample orientation="vertical" />);
    const root = container.querySelector("[data-slot=tabs]") as HTMLElement | null;
    expect(root).toHaveAttribute("data-orientation", "vertical");
  });

  it("Should swap the active panel when a trigger is clicked", async () => {
    const user = userEvent.setup();
    render(<TabsExample />);
    await user.click(screen.getByRole("tab", { name: "Two" }));
    expect(screen.getByRole("tabpanel")).toHaveTextContent("Panel two");
  });

  it("Should default the TabsList variant to line", () => {
    const { container } = render(<TabsExample />);
    const list = container.querySelector("[data-slot=tabs-list]") as HTMLElement | null;
    expect(list).toHaveAttribute("data-variant", "line");
  });

  it("Should forward the lane variant data attribute to TabsList", () => {
    const { container } = render(<TabsExample variant="lane" />);
    const list = container.querySelector("[data-slot=tabs-list]") as HTMLElement | null;
    expect(list).toHaveAttribute("data-variant", "lane");
  });

  it("Should render the active-tab underline at 1.5px tall using the fg-strong token", () => {
    const { container } = render(<TabsExample variant="line" />);
    const trigger = container.querySelector('[data-slot="tabs-trigger"]') as HTMLElement | null;
    expect(trigger?.className).toContain("group-data-horizontal/tabs:after:bottom-[-1.5px]");
    expect(trigger?.className).toContain("group-data-horizontal/tabs:after:h-[1.5px]");
    expect(trigger?.className).toContain("after:bg-(--fg-strong)");
    expect(trigger?.className).not.toContain("after:bg-(--accent)");
  });

  it("Should render the active count chip with the neutral 0.07 glaze (no accent) for variant=line", () => {
    const { container } = render(
      <Tabs defaultValue="runs">
        <TabsList variant="line">
          <TabsTrigger count={3} value="runs">
            Runs
          </TabsTrigger>
        </TabsList>
        <TabsContent value="runs">Panel</TabsContent>
      </Tabs>
    );
    const count = container.querySelector('[data-slot="tabs-trigger-count"]') as HTMLElement | null;
    expect(count?.className).toContain(
      "group-data-[variant=line]/tabs-list:group-data-[active=true]:bg-(--btn-default-hover)"
    );
    expect(count?.className).toContain(
      "group-data-[variant=line]/tabs-list:group-data-[active=true]:text-(--fg)"
    );
    expect(count?.className).not.toContain("bg-(--accent)");
    expect(count?.className).not.toContain("text-(--accent-ink)");
  });

  it("Should render count and live label slots inside a trigger", () => {
    const { container } = render(
      <Tabs defaultValue="runs">
        <TabsList variant="line">
          <TabsTrigger count={3} liveLabel="Live" value="runs">
            Runs
          </TabsTrigger>
        </TabsList>
        <TabsContent value="runs">Panel</TabsContent>
      </Tabs>
    );

    expect(screen.getByRole("tab", { name: /runs3live/i })).toBeInTheDocument();
    expect(container.querySelector('[data-slot="tabs-trigger-count"]')).toHaveTextContent("3");
    expect(container.querySelector('[data-slot="tabs-trigger-live"]')).toHaveTextContent("Live");
  });

  it("Should render lane-variant separators on every trigger except the first", () => {
    const { container } = render(<TabsExample variant="lane" />);
    const triggers = container.querySelectorAll<HTMLElement>('[data-slot="tabs-trigger"]');
    expect(triggers).toHaveLength(3);
    const separatorClass =
      "group-data-[variant=lane]/tabs-list:[&:not(:first-child)]:before:content-['·']";
    for (const trigger of triggers) {
      expect(trigger.className).toContain(separatorClass);
    }
  });

  it("Should render lane-variant counts as bare mono with --faint, no chip background", () => {
    const { container } = render(
      <Tabs defaultValue="lane-a">
        <TabsList variant="lane">
          <TabsTrigger count={4} value="lane-a">
            Lane A
          </TabsTrigger>
          <TabsTrigger count={2} value="lane-b">
            Lane B
          </TabsTrigger>
        </TabsList>
        <TabsContent value="lane-a">Panel</TabsContent>
        <TabsContent value="lane-b">Panel</TabsContent>
      </Tabs>
    );
    const count = container.querySelector('[data-slot="tabs-trigger-count"]') as HTMLElement | null;
    expect(count?.textContent).toBe("4");
    expect(count?.className).toContain("group-data-[variant=lane]/tabs-list:bg-transparent");
    expect(count?.className).toContain("group-data-[variant=lane]/tabs-list:text-[10.5px]");
    expect(count?.className).toContain("group-data-[variant=lane]/tabs-list:text-(--faint)");
    expect(count?.className).toContain("font-mono");
    expect(count?.className).not.toContain("group-data-[variant=lane]/tabs-list:rounded-full");
    expect(count?.className).not.toContain(
      "group-data-[variant=lane]/tabs-list:bg-(--canvas-tint)"
    );
  });

  it("Should reject the chipped 'default' variant at the type level", () => {
    // @ts-expect-error — `variant="default"` is no longer part of the union.
    const guard = <TabsList variant="default" />;
    expect(guard).toBeTruthy();
  });
});
