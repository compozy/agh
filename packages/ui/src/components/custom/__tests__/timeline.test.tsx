import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Timeline } from "../timeline";
import { TimelineEvent } from "../timeline-event";

describe("Timeline", () => {
  it("Should render events in order with markers", () => {
    const { container } = render(
      <Timeline ariaLabel="Run history">
        <TimelineEvent title="Started" tone="info" time="03:00" />
        <TimelineEvent title="Completed" tone="success" time="03:05" />
      </Timeline>
    );
    expect(screen.getByLabelText("Run history")).toBeInTheDocument();
    const markers = container.querySelectorAll('[data-slot="timeline-event-marker"]');
    expect(markers).toHaveLength(2);
    const items = Array.from(container.querySelectorAll('[data-slot="timeline-event"]'));
    expect(items).toHaveLength(2);
    expect(items[0].textContent).toContain("Started");
    expect(items[1].textContent).toContain("Completed");
  });
});
