import { act, render, screen } from "@testing-library/react";
import { ListChecksIcon } from "lucide-react";
import { describe, expect, it } from "vitest";

import {
  Topbar,
  TopbarSlotProvider,
  type TopbarSlotValue,
  useTopbarSlot,
  useTopbarSlotValue,
} from "../topbar";

function ProbeSlot({ slot, label }: { slot: TopbarSlotValue | null; label: string }) {
  useTopbarSlot(slot);
  return <span data-testid={`probe-${label}`} />;
}

function SlotInspector({ probeId }: { probeId: string }) {
  const slot = useTopbarSlotValue();
  return (
    <span data-testid={probeId}>
      tabs:{slot?.tabs ? "yes" : "no"} actions:{slot?.actions ? "yes" : "no"} search:
      {slot?.search ? "yes" : "no"} title:{slot?.title ? "yes" : "no"}
    </span>
  );
}

describe("Topbar", () => {
  it("Should render route icon, title, and count", () => {
    render(
      <TopbarSlotProvider>
        <Topbar
          route={{
            title: "Tasks",
            icon: ListChecksIcon,
            getCount: () => 12,
          }}
        />
      </TopbarSlotProvider>
    );
    expect(screen.getByText("Tasks")).toBeInTheDocument();
    expect(screen.getByText("12")).toBeInTheDocument();
  });

  it("Should render a fallback title when the route context is null", () => {
    render(
      <TopbarSlotProvider>
        <Topbar route={null} />
      </TopbarSlotProvider>
    );
    expect(screen.getByText("Untitled")).toBeInTheDocument();
  });

  it("Should expose tabs/search/actions slots from context", () => {
    function Setup() {
      useTopbarSlot({
        tabs: <span data-testid="lane-tabs">tabs</span>,
        search: <span data-testid="search">search</span>,
        actions: <span data-testid="action-btn">action</span>,
      });
      return null;
    }
    render(
      <TopbarSlotProvider>
        <Setup />
        <Topbar route={{ title: "Tasks" }} />
      </TopbarSlotProvider>
    );
    expect(screen.getByTestId("lane-tabs")).toBeInTheDocument();
    expect(screen.getByTestId("search")).toBeInTheDocument();
    expect(screen.getByTestId("action-btn")).toBeInTheDocument();
  });

  it("Should let the slot override route title and count for live data", () => {
    function Setup() {
      useTopbarSlot({ title: "Live title", count: 42 });
      return null;
    }
    render(
      <TopbarSlotProvider>
        <Setup />
        <Topbar route={{ title: "Static", getCount: () => 12 }} />
      </TopbarSlotProvider>
    );
    expect(screen.getByText("Live title")).toBeInTheDocument();
    expect(screen.queryByText("Static")).toBeNull();
    expect(screen.getByText("42")).toBeInTheDocument();
    expect(screen.queryByText("12")).toBeNull();
  });

  it("Should mark the title element focusable so the shell can move focus on route resolve", () => {
    render(
      <TopbarSlotProvider>
        <Topbar route={{ title: "Tasks" }} />
      </TopbarSlotProvider>
    );
    const title = screen.getByTestId("topbar-title-text");
    expect(title.tagName).toBe("H1");
    expect(title.getAttribute("tabindex")).toBe("-1");
  });

  it("Should re-push the slot when the consumer's slot reference changes", () => {
    function Setup({ count }: { count: number }) {
      useTopbarSlot({ count });
      return null;
    }
    const { rerender } = render(
      <TopbarSlotProvider>
        <Setup count={1} />
        <Topbar route={{ title: "Tasks" }} />
      </TopbarSlotProvider>
    );
    expect(screen.getByText("1")).toBeInTheDocument();
    act(() => {
      rerender(
        <TopbarSlotProvider>
          <Setup count={42} />
          <Topbar route={{ title: "Tasks" }} />
        </TopbarSlotProvider>
      );
    });
    expect(screen.getByText("42")).toBeInTheDocument();
  });

  it("Should clear slot subfields when the slot consumer unmounts", () => {
    function Harness({ mounted }: { mounted: boolean }) {
      return (
        <>
          {mounted ? <ProbeSlot slot={{ actions: <span data-testid="a" /> }} label="a" /> : null}
          <SlotInspector probeId="inspector" />
        </>
      );
    }
    const { rerender } = render(
      <TopbarSlotProvider>
        <Harness mounted />
      </TopbarSlotProvider>
    );
    expect(screen.getByTestId("inspector")).toHaveTextContent("actions:yes");

    act(() => {
      rerender(
        <TopbarSlotProvider>
          <Harness mounted={false} />
        </TopbarSlotProvider>
      );
    });

    expect(screen.getByTestId("inspector")).toHaveTextContent("actions:no");
    expect(screen.getByTestId("inspector")).toHaveTextContent("tabs:no");
    expect(screen.getByTestId("inspector")).toHaveTextContent("search:no");
    expect(screen.getByTestId("inspector")).toHaveTextContent("title:no");
  });

  it("Should be a no-op when used outside a TopbarSlotProvider (test ergonomics)", () => {
    function Harness() {
      useTopbarSlot({ actions: <span data-testid="a" /> });
      return <span data-testid="probe">ok</span>;
    }
    expect(() => render(<Harness />)).not.toThrow();
    expect(screen.getByTestId("probe")).toHaveTextContent("ok");
  });
});
