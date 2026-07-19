import { act, fireEvent, render, screen } from "@testing-library/react";
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
      routeNav:{slot?.routeNav ? "yes" : "no"} actions:{slot?.actions ? "yes" : "no"} overflow:
      {slot?.overflow ? "yes" : "no"} crumb:{slot?.crumb ? "yes" : "no"} crumb-value:
      {typeof slot?.crumb === "string" ? slot.crumb : "no"}
    </span>
  );
}

describe("Topbar", () => {
  it("Should own the route H1 beside the breadcrumb in the leading context zone", () => {
    render(
      <TopbarSlotProvider>
        <Topbar breadcrumb={<span data-testid="crumb-trail">Operate</span>} title="Tasks" />
      </TopbarSlotProvider>
    );
    const context = document.querySelector("[data-slot='topbar-context']");
    expect(context).toContainElement(screen.getByTestId("crumb-trail"));
    const title = screen.getByRole("heading", { level: 1, name: "Tasks" });
    expect(title).toHaveAttribute("tabindex", "-1");
    expect(title).toHaveAttribute("data-slot", "topbar-title");
  });

  it("Should render route identity without breadcrumb or slots", () => {
    const { container } = render(
      <TopbarSlotProvider>
        <Topbar title="Home" />
      </TopbarSlotProvider>
    );
    const header = container.querySelector("[data-slot='topbar']");
    expect(header).toBeInTheDocument();
    expect(screen.getByRole("heading", { level: 1, name: "Home" })).toBeInTheDocument();
    expect(container.querySelector("[data-slot='topbar-route-nav']")).toBeNull();
    expect(container.querySelector("[data-slot='topbar-actions']")).toBeNull();
  });

  it("Should expose routeNav/actions/overflow slots in their zones", () => {
    function Setup() {
      useTopbarSlot({
        routeNav: <span data-testid="route-nav-links">views</span>,
        actions: <span data-testid="action-btn">action</span>,
        overflow: <span data-testid="overflow-trigger">…</span>,
      });
      return null;
    }
    render(
      <TopbarSlotProvider>
        <Setup />
        <Topbar title="Tasks" />
      </TopbarSlotProvider>
    );
    expect(document.querySelector("[data-slot='topbar-route-nav']")).toContainElement(
      screen.getByTestId("route-nav-links")
    );
    expect(document.querySelector("[data-slot='topbar-actions']")).toContainElement(
      screen.getByTestId("action-btn")
    );
    expect(screen.getByTestId("topbar-overflow")).toContainElement(
      screen.getByTestId("overflow-trigger")
    );
  });

  it("Should publish a slot without rerendering its producer subtree", () => {
    let producerRenders = 0;
    function Setup() {
      producerRenders += 1;
      useTopbarSlot({ actions: <span data-testid="live-action" /> });
      return null;
    }

    render(
      <TopbarSlotProvider>
        <Setup />
        <Topbar title="Tasks" />
      </TopbarSlotProvider>
    );

    expect(screen.getByTestId("live-action")).toBeInTheDocument();
    expect(producerRenders).toBe(1);
  });

  it("Should re-push the slot when the consumer's slot reference changes", () => {
    function Setup({ label }: { label: string }) {
      useTopbarSlot({ actions: <span>{label}</span> });
      return null;
    }
    const { rerender } = render(
      <TopbarSlotProvider>
        <Setup label="first" />
        <Topbar title="Tasks" />
      </TopbarSlotProvider>
    );
    expect(screen.getByText("first")).toBeInTheDocument();
    act(() => {
      rerender(
        <TopbarSlotProvider>
          <Setup label="second" />
          <Topbar title="Tasks" />
        </TopbarSlotProvider>
      );
    });
    expect(screen.getByText("second")).toBeInTheDocument();
  });

  it("Should clear slot subfields when the slot consumer unmounts", () => {
    function Harness({ mounted }: { mounted: boolean }) {
      return (
        <>
          {mounted ? (
            <ProbeSlot
              slot={{
                crumb: "loop-a",
                routeNav: <span data-testid="rn" />,
                actions: <span data-testid="a" />,
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
    expect(screen.getByTestId("inspector")).toHaveTextContent("routeNav:yes");
    expect(screen.getByTestId("inspector")).toHaveTextContent("overflow:yes");
    expect(screen.getByTestId("inspector")).toHaveTextContent("crumb:yes");

    act(() => {
      rerender(
        <TopbarSlotProvider>
          <Harness mounted={false} />
        </TopbarSlotProvider>
      );
    });

    expect(screen.getByTestId("inspector")).toHaveTextContent("actions:no");
    expect(screen.getByTestId("inspector")).toHaveTextContent("routeNav:no");
    expect(screen.getByTestId("inspector")).toHaveTextContent("overflow:no");
    expect(screen.getByTestId("inspector")).toHaveTextContent("crumb:no");
  });

  it("Should preserve the active slot when an older consumer unmounts", () => {
    function Harness({ showOlder }: { showOlder: boolean }) {
      return (
        <>
          {showOlder ? <ProbeSlot slot={{ crumb: "Older" }} label="older" /> : null}
          <ProbeSlot slot={{ crumb: "Active" }} label="active" />
          <SlotInspector probeId="inspector" />
        </>
      );
    }

    const { rerender } = render(
      <TopbarSlotProvider>
        <Harness showOlder />
      </TopbarSlotProvider>
    );
    expect(screen.getByTestId("inspector")).toHaveTextContent("crumb-value:Active");

    act(() => {
      rerender(
        <TopbarSlotProvider>
          <Harness showOlder={false} />
        </TopbarSlotProvider>
      );
    });

    expect(screen.getByTestId("inspector")).toHaveTextContent("crumb-value:Active");
  });

  it("Should not let a non-owning null consumer erase the active slot", () => {
    render(
      <TopbarSlotProvider>
        <ProbeSlot slot={{ crumb: "Active" }} label="active" />
        <ProbeSlot slot={null} label="inactive" />
        <SlotInspector probeId="inspector" />
      </TopbarSlotProvider>
    );

    expect(screen.getByTestId("inspector")).toHaveTextContent("crumb:yes");
  });

  it("Should publish replaced action handlers when only the callback identity changes", () => {
    const firstAction = vi.fn();
    const secondAction = vi.fn();

    function Harness({ onRun }: { onRun: () => void }) {
      useTopbarSlot({
        actions: (
          <button onClick={onRun} type="button">
            Run
          </button>
        ),
      });
      return null;
    }

    const { rerender } = render(
      <TopbarSlotProvider>
        <Harness onRun={firstAction} />
        <Topbar title="Tasks" />
      </TopbarSlotProvider>
    );

    fireEvent.click(screen.getByRole("button", { name: "Run" }));
    expect(firstAction).toHaveBeenCalledTimes(1);
    expect(secondAction).not.toHaveBeenCalled();

    act(() => {
      rerender(
        <TopbarSlotProvider>
          <Harness onRun={secondAction} />
          <Topbar title="Tasks" />
        </TopbarSlotProvider>
      );
    });

    fireEvent.click(screen.getByRole("button", { name: "Run" }));
    expect(firstAction).toHaveBeenCalledTimes(1);
    expect(secondAction).toHaveBeenCalledTimes(1);
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
