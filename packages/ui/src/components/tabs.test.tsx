import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";

import { Tabs, TabsContent, TabsList, TabsTrigger } from "./tabs";

function TabsExample({ orientation }: { orientation?: "horizontal" | "vertical" }) {
  return (
    <Tabs defaultValue="one" orientation={orientation}>
      <TabsList>
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

  it("Should forward the line variant data attribute to TabsList", () => {
    const { container } = render(
      <Tabs defaultValue="one">
        <TabsList variant="line">
          <TabsTrigger value="one">One</TabsTrigger>
        </TabsList>
        <TabsContent value="one">Panel</TabsContent>
      </Tabs>
    );
    const list = container.querySelector("[data-slot=tabs-list]") as HTMLElement | null;
    expect(list).toHaveAttribute("data-variant", "line");
  });
});
