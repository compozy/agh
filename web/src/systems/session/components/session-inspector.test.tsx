import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import type { UIMessage } from "../types";

vi.mock("@/lib/utils", async importActual => {
  const actual = await importActual<typeof import("@/lib/utils")>();
  return {
    ...actual,
    cn: (...args: unknown[]) => args.filter(Boolean).join(" "),
  };
});

import {
  SessionInspector,
  deriveFileReads,
  deriveTraceEvents,
  type InspectorFileEntry,
  type InspectorMemoryDoc,
  type InspectorUsage,
} from "./session-inspector";

function ts(offsetMin: number): number {
  return Date.parse("2026-04-18T14:00:00Z") + offsetMin * 60 * 1000;
}

function makeMessage(overrides: Partial<UIMessage> & { id: string }): UIMessage {
  return {
    role: "assistant",
    content: "",
    timestamp: ts(0),
    ...overrides,
  };
}

function buildThread(count: number): UIMessage[] {
  return Array.from({ length: count }, (_, i) =>
    makeMessage({
      id: `msg-${i}`,
      role: i === 0 ? "system" : i % 2 === 0 ? "assistant" : "user",
      content: `message ${i}`,
      timestamp: ts(i),
    })
  );
}

describe("deriveTraceEvents", () => {
  it("returns the last N events with oldest tagged as start", () => {
    const events = deriveTraceEvents(buildThread(10), 6);
    expect(events).toHaveLength(6);
    expect(events[0].id).toBe("msg-4");
    // First slot in the truncated list still maps to its own role (not "start")
    expect(events[0].kind).not.toBe("start");
  });

  it("tags the absolute first message of a short thread as start", () => {
    const events = deriveTraceEvents(buildThread(3), 6);
    expect(events).toHaveLength(3);
    expect(events[0].kind).toBe("start");
    expect(events[0].label).toBe("Session started");
  });

  it("maps tool messages to kind=tool and pending when result is missing", () => {
    const thread: UIMessage[] = [
      makeMessage({ id: "a", role: "assistant" }),
      makeMessage({
        id: "b",
        role: "tool_call",
        toolName: "Bash",
        toolInput: { command: "ls" },
      }),
    ];
    const events = deriveTraceEvents(thread);
    expect(events[1].kind).toBe("tool");
    expect(events[1].status).toBe("pending");
    expect(events[1].label).toBe("Bash");
  });

  it("maps errored tools to danger status", () => {
    const thread: UIMessage[] = [
      makeMessage({ id: "a", role: "assistant" }),
      makeMessage({
        id: "b",
        role: "tool_call",
        toolName: "Bash",
        toolError: true,
        toolResult: { stderr: "nope" },
      }),
    ];
    const events = deriveTraceEvents(thread);
    expect(events[1].status).toBe("error");
  });
});

describe("deriveFileReads", () => {
  it("counts unique files by path and increments on repeat reads", () => {
    const thread: UIMessage[] = [
      makeMessage({
        id: "t1",
        role: "tool_call",
        toolName: "Read",
        toolInput: { file_path: "a.ts" },
        toolResult: { filePath: "a.ts" },
      }),
      makeMessage({
        id: "t2",
        role: "tool_call",
        toolName: "Read",
        toolInput: { file_path: "a.ts" },
        toolResult: { filePath: "a.ts" },
      }),
      makeMessage({
        id: "t3",
        role: "tool_call",
        toolName: "Read",
        toolInput: { file_path: "b.ts" },
        toolResult: { filePath: "b.ts" },
      }),
    ];
    expect(deriveFileReads(thread)).toEqual<InspectorFileEntry[]>([
      { path: "a.ts", readCount: 2 },
      { path: "b.ts", readCount: 1 },
    ]);
  });

  it("falls back to file_path in toolInput when no toolResult filePath is present", () => {
    const thread: UIMessage[] = [
      makeMessage({
        id: "t1",
        role: "tool_call",
        toolName: "Read",
        toolInput: { file_path: "only-input.ts" },
      }),
    ];
    expect(deriveFileReads(thread)).toEqual<InspectorFileEntry[]>([
      { path: "only-input.ts", readCount: 1 },
    ]);
  });
});

describe("<SessionInspector />", () => {
  const usage: InspectorUsage = {
    tokensIn: 12_481,
    tokensOut: 2_108,
    costUsd: 0.048,
    ratePerSecond: 128.4,
    tokensInDelta: 320,
    tokensOutDelta: -40,
    costDelta: 0,
  };
  const memoryDocs: InspectorMemoryDoc[] = [
    { id: "doc-1", kind: "ws", title: "agh.md", bytes: 4_820 },
    { id: "doc-2", kind: "repo", title: "big.md", bytes: 2_000_000 },
  ];
  const files: InspectorFileEntry[] = [
    { path: "a.ts", readCount: 2 },
    { path: "b.ts", readCount: 1 },
  ];

  it("renders only the 6 most recent trace rows plus a view-all link", () => {
    const onViewAll = vi.fn();
    render(
      <SessionInspector
        messages={buildThread(10)}
        usage={usage}
        memoryDocs={memoryDocs}
        files={files}
        onViewAllTrace={onViewAll}
      />
    );
    const panel = screen.getByTestId("session-inspector-top-panel");
    const rows = within(panel).getAllByTestId("session-inspector-trace-row");
    expect(rows).toHaveLength(6);
    const viewAll = within(panel).getByTestId("session-inspector-trace-view-all");
    expect(viewAll).toBeInTheDocument();
  });

  it("renders each trace row with mono timestamp, kind badge, and status dot", () => {
    render(
      <SessionInspector
        messages={[
          makeMessage({
            id: "m-1",
            role: "tool_call",
            toolName: "Bash",
            toolError: true,
            toolResult: { stderr: "nope" },
          }),
        ]}
        usage={null}
        memoryDocs={[]}
        files={[]}
      />
    );
    const panel = screen.getByTestId("session-inspector-top-panel");
    const kind = within(panel).getByTestId("session-inspector-trace-kind");
    expect(kind.textContent).toBe("START");
    const dot = within(panel).getByTestId("session-inspector-trace-dot");
    expect(dot.getAttribute("data-tone")).toBe("danger");
  });

  it("uses accent tone for pending trace rows", () => {
    render(
      <SessionInspector
        messages={[
          makeMessage({ id: "m-0", role: "user" }),
          makeMessage({
            id: "m-1",
            role: "tool_call",
            toolName: "Bash",
          }),
        ]}
        usage={null}
      />
    );
    const panel = screen.getByTestId("session-inspector-top-panel");
    const rows = within(panel).getAllByTestId("session-inspector-trace-row");
    const pending = rows[1];
    expect(pending.getAttribute("data-status")).toBe("pending");
    const dot = within(pending).getByTestId("session-inspector-trace-dot");
    expect(dot.getAttribute("data-tone")).toBe("accent");
    expect(dot.getAttribute("data-pulse")).toBe("true");
  });

  it("renders four Metric tiles and colors deltas via tone", async () => {
    const user = userEvent.setup();
    render(<SessionInspector messages={buildThread(2)} usage={usage} />);
    await user.click(screen.getByTestId("session-inspector-tab-usage"));
    const panel = screen.getByTestId("session-inspector-top-panel");
    const grid = within(panel).getByTestId("session-inspector-usage-grid");
    const tiles = within(grid).getAllByTestId(/session-inspector-usage-/);
    // tokens-in + tokens-out + cost + rate
    expect(tiles).toHaveLength(4);

    const tokensIn = within(panel).getByTestId("session-inspector-usage-tokens-in");
    expect(tokensIn.getAttribute("data-tone")).toBe("success");
    const tokensOut = within(panel).getByTestId("session-inspector-usage-tokens-out");
    expect(tokensOut.getAttribute("data-tone")).toBe("danger");
    const cost = within(panel).getByTestId("session-inspector-usage-cost");
    expect(cost.getAttribute("data-tone")).toBe("default");
  });

  it("renders the Usage Empty state when usage is null", async () => {
    const user = userEvent.setup();
    render(<SessionInspector messages={buildThread(2)} usage={null} />);
    await user.click(screen.getByTestId("session-inspector-tab-usage"));
    expect(screen.getByTestId("session-inspector-usage-empty")).toBeInTheDocument();
  });

  it("renders the Memory Empty state when no docs are attached", () => {
    render(<SessionInspector messages={buildThread(2)} memoryDocs={[]} />);
    expect(screen.getByTestId("session-inspector-memory-empty")).toBeInTheDocument();
  });

  it("renders each memory row with kind badge, title, and formatted byte size", () => {
    render(<SessionInspector messages={buildThread(1)} memoryDocs={memoryDocs} />);
    const panel = screen.getByTestId("session-inspector-bottom-panel");
    const rows = within(panel).getAllByTestId("session-inspector-memory-row");
    expect(rows).toHaveLength(2);
    expect(within(rows[0]).getByTestId("session-inspector-memory-kind").textContent).toBe("ws");
    expect(within(rows[0]).getByTestId("session-inspector-memory-title").textContent).toBe(
      "agh.md"
    );
    expect(within(rows[0]).getByTestId("session-inspector-memory-bytes").textContent).toBe(
      "4.7 kB"
    );
    expect(within(rows[1]).getByTestId("session-inspector-memory-bytes").textContent).toBe(
      "1.9 MB"
    );
  });

  it("wraps the files list in a ScrollArea and renders path + read count per row", async () => {
    const user = userEvent.setup();
    render(<SessionInspector messages={buildThread(1)} files={files} />);
    await user.click(screen.getByTestId("session-inspector-tab-files"));
    const panel = screen.getByTestId("session-inspector-bottom-panel");
    const scroll = within(panel).getByTestId("session-inspector-files-scroll");
    expect(scroll.getAttribute("data-slot")).toBe("scroll-area");
    const rows = within(scroll).getAllByTestId("session-inspector-files-row");
    expect(rows).toHaveLength(2);
    expect(within(rows[0]).getByTestId("session-inspector-files-path").textContent).toBe("a.ts");
    expect(within(rows[0]).getByTestId("session-inspector-files-count").textContent).toBe("×2");
  });

  it("derives the files list from messages when no explicit files prop is provided", async () => {
    const user = userEvent.setup();
    const thread: UIMessage[] = [
      makeMessage({
        id: "t1",
        role: "tool_call",
        toolName: "Read",
        toolInput: { file_path: "derived.ts" },
        toolResult: { filePath: "derived.ts" },
      }),
    ];
    render(<SessionInspector messages={thread} />);
    await user.click(screen.getByTestId("session-inspector-tab-files"));
    const row = screen.getAllByTestId("session-inspector-files-row")[0];
    expect(within(row).getByTestId("session-inspector-files-path").textContent).toBe("derived.ts");
  });

  it("renders two stacked tab groups with trace + memory active by default", () => {
    render(<SessionInspector messages={buildThread(3)} usage={usage} />);
    expect(screen.getByTestId("session-inspector-body")).toBeInTheDocument();
    expect(screen.queryByTestId("session-inspector-stacked")).not.toBeInTheDocument();
    expect(screen.queryByTestId("session-inspector-tabbed")).not.toBeInTheDocument();
    expect(screen.getByTestId("session-inspector-top-panel").getAttribute("data-active-tab")).toBe(
      "trace"
    );
    expect(
      screen.getByTestId("session-inspector-bottom-panel").getAttribute("data-active-tab")
    ).toBe("memory");
  });

  it("switches top and bottom groups independently", async () => {
    const user = userEvent.setup();
    render(<SessionInspector messages={buildThread(3)} usage={usage} />);
    await user.click(screen.getByTestId("session-inspector-tab-usage"));
    expect(screen.getByTestId("session-inspector-top-panel").getAttribute("data-active-tab")).toBe(
      "usage"
    );
    // Bottom group stays on memory.
    expect(
      screen.getByTestId("session-inspector-bottom-panel").getAttribute("data-active-tab")
    ).toBe("memory");

    await user.click(screen.getByTestId("session-inspector-tab-files"));
    expect(
      screen.getByTestId("session-inspector-bottom-panel").getAttribute("data-active-tab")
    ).toBe("files");
    // Top group stays on usage.
    expect(screen.getByTestId("session-inspector-top-panel").getAttribute("data-active-tab")).toBe(
      "usage"
    );
  });

  it("hides the inline aside under the xl breakpoint", () => {
    render(<SessionInspector messages={buildThread(2)} />);
    const aside = screen.getByTestId("session-inspector");
    expect(aside.className).toMatch(/xl:flex/);
    expect(aside.className).toMatch(/\bhidden\b/);
  });
});
