import { act, fireEvent, render, screen } from "@testing-library/react";
import { ListChecksIcon } from "lucide-react";
import { describe, expect, it, vi } from "vitest";

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
      {slot?.search ? "yes" : "no"} title:{slot?.title ? "yes" : "no"} meta:
      {slot?.meta ? "yes" : "no"} overflow:{slot?.overflow ? "yes" : "no"} back:
      {slot?.back ? "yes" : "no"}
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

  it("Should auto-resolve count from the navCount prop when slot and route omit it", () => {
    render(
      <TopbarSlotProvider>
        <Topbar navCount={7} route={{ title: "Tasks", navCountKey: "tasks" }} />
      </TopbarSlotProvider>
    );
    const count = screen.getByTestId("topbar-count");
    expect(count).toHaveTextContent("7");
  });

  it("Should prefer slot count over navCount when both are provided", () => {
    function Setup() {
      useTopbarSlot({ count: 99 });
      return null;
    }
    render(
      <TopbarSlotProvider>
        <Setup />
        <Topbar navCount={7} route={{ title: "Tasks", navCountKey: "tasks" }} />
      </TopbarSlotProvider>
    );
    expect(screen.getByTestId("topbar-count")).toHaveTextContent("99");
    expect(screen.queryByText("7")).toBeNull();
  });

  it("Should prefer route getCount over navCount when slot count is omitted", () => {
    render(
      <TopbarSlotProvider>
        <Topbar navCount={7} route={{ title: "Tasks", getCount: () => 42, navCountKey: "tasks" }} />
      </TopbarSlotProvider>
    );
    expect(screen.getByTestId("topbar-count")).toHaveTextContent("42");
  });

  it("Should not render the count chip when all sources are undefined", () => {
    render(
      <TopbarSlotProvider>
        <Topbar route={{ title: "Tasks" }} />
      </TopbarSlotProvider>
    );
    expect(screen.queryByTestId("topbar-count")).toBeNull();
  });

  it("Should render the back chevron when slot.back is provided", () => {
    const onBack = vi.fn();
    function Setup() {
      useTopbarSlot({ back: onBack, backLabel: "Back to tasks" });
      return null;
    }
    render(
      <TopbarSlotProvider>
        <Setup />
        <Topbar route={{ title: "Detail" }} />
      </TopbarSlotProvider>
    );
    const back = screen.getByTestId("topbar-back");
    expect(back).toHaveAttribute("aria-label", "Back to tasks");
    fireEvent.click(back);
    expect(onBack).toHaveBeenCalledOnce();
  });

  it("Should default backLabel to 'Go back' when not provided", () => {
    function Setup() {
      useTopbarSlot({ back: () => undefined });
      return null;
    }
    render(
      <TopbarSlotProvider>
        <Setup />
        <Topbar route={{ title: "Detail" }} />
      </TopbarSlotProvider>
    );
    expect(screen.getByTestId("topbar-back")).toHaveAttribute("aria-label", "Go back");
  });

  it("Should set data-mode='detail' when slot.back is present", () => {
    function Setup() {
      useTopbarSlot({ back: () => undefined });
      return null;
    }
    const { container } = render(
      <TopbarSlotProvider>
        <Setup />
        <Topbar route={{ title: "Detail" }} />
      </TopbarSlotProvider>
    );
    const header = container.querySelector('[data-slot="topbar"]');
    expect(header).toHaveAttribute("data-mode", "detail");
  });

  it("Should default data-mode='default' when slot.back is absent", () => {
    const { container } = render(
      <TopbarSlotProvider>
        <Topbar route={{ title: "Tasks" }} />
      </TopbarSlotProvider>
    );
    const header = container.querySelector('[data-slot="topbar"]');
    expect(header).toHaveAttribute("data-mode", "default");
  });

  it("Should render the meta slot adjacent to the title", () => {
    function Setup() {
      useTopbarSlot({ meta: <span data-testid="meta-chip">meta</span> });
      return null;
    }
    render(
      <TopbarSlotProvider>
        <Setup />
        <Topbar route={{ title: "Detail" }} />
      </TopbarSlotProvider>
    );
    expect(screen.getByTestId("topbar-meta")).toContainElement(screen.getByTestId("meta-chip"));
  });

  it("Should render the overflow slot at the trailing edge", () => {
    function Setup() {
      useTopbarSlot({ overflow: <span data-testid="overflow-trigger">…</span> });
      return null;
    }
    render(
      <TopbarSlotProvider>
        <Setup />
        <Topbar route={{ title: "Detail" }} />
      </TopbarSlotProvider>
    );
    expect(screen.getByTestId("topbar-overflow")).toContainElement(
      screen.getByTestId("overflow-trigger")
    );
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
          {mounted ? (
            <ProbeSlot
              slot={{
                actions: <span data-testid="a" />,
                back: () => undefined,
                meta: <span data-testid="m" />,
                overflow: <span data-testid="o" />,
              }}
              label="a"
            />
          ) : null}
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
    expect(screen.getByTestId("inspector")).toHaveTextContent("meta:yes");
    expect(screen.getByTestId("inspector")).toHaveTextContent("overflow:yes");
    expect(screen.getByTestId("inspector")).toHaveTextContent("back:yes");

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
    expect(screen.getByTestId("inspector")).toHaveTextContent("meta:no");
    expect(screen.getByTestId("inspector")).toHaveTextContent("overflow:no");
    expect(screen.getByTestId("inspector")).toHaveTextContent("back:no");
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
