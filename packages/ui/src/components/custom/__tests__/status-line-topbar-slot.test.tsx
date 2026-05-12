import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { StatusLineTopbarSlot } from "../status-line-topbar-slot";

describe("StatusLineTopbarSlot", () => {
  it("Should render the ConnectionIndicator plus typed items with tone data attribute", () => {
    const { container } = render(
      <StatusLineTopbarSlot
        status="connected"
        items={[
          { label: "sessions", value: "12", tone: "neutral" },
          { label: "agents", value: "3", tone: "info" },
          { value: "workspace · launch", tone: "success" },
        ]}
      />
    );

    const root = container.querySelector<HTMLElement>('[data-slot="status-line-topbar-slot"]');
    expect(root?.dataset.status).toBe("connected");
    expect(root?.querySelector('[data-slot="connection-indicator"]')).not.toBeNull();

    const items = container.querySelectorAll<HTMLElement>(
      '[data-slot="status-line-topbar-slot-item"]'
    );
    expect(items).toHaveLength(3);
    expect(items[0]?.dataset.tone).toBe("neutral");
    expect(items[1]?.dataset.tone).toBe("info");
    expect(items[2]?.dataset.tone).toBe("success");

    expect(screen.getByText("12")).toBeInTheDocument();
    expect(screen.getByText("3")).toBeInTheDocument();
    expect(screen.getByText("workspace · launch")).toBeInTheDocument();
  });

  it("Should render labels only when present", () => {
    const { container } = render(
      <StatusLineTopbarSlot
        status="connected"
        items={[
          { label: "sessions", value: "12", tone: "neutral" },
          { value: "no-label", tone: "info" },
        ]}
      />
    );

    const labels = container.querySelectorAll<HTMLElement>(
      '[data-slot="status-line-topbar-slot-item-label"]'
    );
    expect(labels).toHaveLength(1);
    expect(labels[0]?.textContent).toBe("sessions");
  });

  it("Should default to neutral tone when an item omits the tone field", () => {
    const { container } = render(
      <StatusLineTopbarSlot status="connecting" items={[{ value: "resyncing…" }]} />
    );
    const item = container.querySelector<HTMLElement>('[data-slot="status-line-topbar-slot-item"]');
    expect(item?.dataset.tone).toBe("neutral");
  });
});
