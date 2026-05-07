// @vitest-environment jsdom

import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { ReactNode } from "react";

vi.mock("@tanstack/react-router", () => ({
  Link: ({
    to,
    params,
    children,
    ...rest
  }: {
    to: string;
    params?: Record<string, string>;
    children: ReactNode;
    [key: string]: unknown;
  }) => {
    const path = Object.entries(params ?? {}).reduce(
      (acc, [key, value]) => acc.replace(`$${key}`, String(value)),
      to
    );
    return (
      <a href={path} {...(rest as Record<string, unknown>)}>
        {children}
      </a>
    );
  },
}));

import { ChannelTabs, type ChannelTab } from "../channel-tabs";

function renderTabs({ activeTab }: { activeTab: ChannelTab }) {
  return render(
    <ChannelTabs activeTab={activeTab} channel="builders" directCount={4} threadCount={12} />
  );
}

describe("ChannelTabs", () => {
  it("renders three tabs as real anchor Links pointing at the canonical routes", () => {
    renderTabs({ activeTab: "threads" });

    const threads = screen.getByTestId("network-tab-threads");
    const directs = screen.getByTestId("network-tab-directs");
    const activity = screen.getByTestId("network-tab-activity");

    expect(threads.tagName).toBe("A");
    expect(directs.tagName).toBe("A");
    expect(activity.tagName).toBe("A");

    expect(threads.getAttribute("href")).toBe("/network/builders/threads");
    expect(directs.getAttribute("href")).toBe("/network/builders/directs");
    expect(activity.getAttribute("href")).toBe("/network/builders/activity");
  });

  it("marks only the matching tab with aria-current=page", () => {
    renderTabs({ activeTab: "directs" });

    expect(screen.getByTestId("network-tab-threads")).not.toHaveAttribute("aria-current");
    expect(screen.getByTestId("network-tab-directs")).toHaveAttribute("aria-current", "page");
    expect(screen.getByTestId("network-tab-activity")).not.toHaveAttribute("aria-current");
  });
});
