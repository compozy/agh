// @vitest-environment jsdom

import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { ListFilterBar, type NetworkListFilterCounts } from "./list-filter-bar";

const COUNTS: NetworkListFilterCounts = {
  all: 5,
  hasWork: 3,
  me: 0,
  pinned: 1,
  unread: 2,
};

describe("ListFilterBar", () => {
  it("Should render all filter chips with their counts", () => {
    render(
      <ListFilterBar
        counts={COUNTS}
        filter="all"
        onFilterChange={() => undefined}
        onSortChange={() => undefined}
        sort="recent_activity"
      />
    );

    expect(screen.getByTestId("network-filter-all")).toHaveTextContent("All");
    expect(screen.getByTestId("network-filter-all")).toHaveTextContent("5");
    expect(screen.getByTestId("network-filter-has-work")).toHaveTextContent("Has work");
    expect(screen.getByTestId("network-filter-has-work")).toHaveTextContent("3");
    expect(screen.getByTestId("network-filter-pinned")).toHaveTextContent("Pinned");
    expect(screen.getByTestId("network-filter-unread")).toHaveTextContent("Unread");
  });

  it("Should mark the active filter chip as pressed", () => {
    render(
      <ListFilterBar
        counts={COUNTS}
        filter="has_work"
        onFilterChange={() => undefined}
        onSortChange={() => undefined}
        sort="recent_activity"
      />
    );

    expect(screen.getByTestId("network-filter-all")).toHaveAttribute("aria-pressed", "false");
    expect(screen.getByTestId("network-filter-has-work")).toHaveAttribute("aria-pressed", "true");
  });

  it("Should fire onFilterChange when a chip is clicked", () => {
    const onFilterChange = vi.fn();
    render(
      <ListFilterBar
        counts={COUNTS}
        filter="all"
        onFilterChange={onFilterChange}
        onSortChange={() => undefined}
        sort="recent_activity"
      />
    );

    fireEvent.click(screen.getByTestId("network-filter-has-work"));
    expect(onFilterChange).toHaveBeenCalledWith("has_work");
  });

  it("Should disable mark-all-read when there are zero unread items", () => {
    render(
      <ListFilterBar
        counts={{ ...COUNTS, unread: 0 }}
        filter="all"
        isMarkAllReadDisabled
        onFilterChange={() => undefined}
        onMarkAllRead={() => undefined}
        onSortChange={() => undefined}
        sort="recent_activity"
      />
    );

    expect(screen.getByTestId("network-list-mark-all-read")).toBeDisabled();
  });

  it("Should fire onMarkAllRead when the action button is clicked", () => {
    const onMarkAllRead = vi.fn();
    render(
      <ListFilterBar
        counts={COUNTS}
        filter="all"
        onFilterChange={() => undefined}
        onMarkAllRead={onMarkAllRead}
        onSortChange={() => undefined}
        sort="recent_activity"
      />
    );

    fireEvent.click(screen.getByTestId("network-list-mark-all-read"));
    expect(onMarkAllRead).toHaveBeenCalledTimes(1);
  });
});
