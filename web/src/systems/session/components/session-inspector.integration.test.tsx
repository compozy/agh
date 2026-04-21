import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

vi.mock("@/lib/utils", async importActual => {
  const actual = await importActual<typeof import("@/lib/utils")>();
  return {
    ...actual,
    cn: (...args: unknown[]) => args.filter(Boolean).join(" "),
  };
});

import {
  SessionInspector,
  SessionInspectorDrawer,
  type InspectorFileEntry,
  type InspectorMemoryDoc,
  type InspectorUsage,
} from "./session-inspector";
import type { UIMessage } from "../types";

function ts(offsetMin: number): number {
  return Date.parse("2026-04-18T14:00:00Z") + offsetMin * 60 * 1000;
}

const messages: UIMessage[] = [
  { id: "m-0", role: "system", content: "Session resumed.", timestamp: ts(0) },
  { id: "m-1", role: "user", content: "Refactor the event mapper.", timestamp: ts(1) },
  {
    id: "m-2",
    role: "assistant",
    content: "Starting work on it.",
    timestamp: ts(2),
  },
  {
    id: "m-3",
    role: "tool_call",
    content: "",
    toolName: "Read",
    toolInput: { file_path: "src/stream.ts" },
    toolResult: { filePath: "src/stream.ts" },
    timestamp: ts(3),
  },
  {
    id: "m-4",
    role: "tool_call",
    content: "",
    toolName: "Bash",
    toolInput: { command: "rg stream" },
    toolResult: { stdout: "match" },
    timestamp: ts(4),
  },
  {
    id: "m-5",
    role: "assistant",
    content: "Proposed patch below.",
    timestamp: ts(5),
  },
  {
    id: "m-6",
    role: "diff",
    content: "",
    diff: { path: "src/stream.ts", additions: 4, removals: 38, content: "" },
    timestamp: ts(6),
  },
];

const usage: InspectorUsage = {
  tokensIn: 12_500,
  tokensOut: 2_100,
  costUsd: 0.05,
  ratePerSecond: 120,
  tokensInDelta: 120,
};

const memoryDocs: InspectorMemoryDoc[] = [
  { id: "doc-1", kind: "ws", title: "agh.md", bytes: 4_820 },
];

const files: InspectorFileEntry[] = [{ path: "src/stream.ts", readCount: 1 }];

describe("SessionInspector — integration", () => {
  it("renders all four slices populated from a fixture session", async () => {
    const user = userEvent.setup();
    render(
      <SessionInspector messages={messages} usage={usage} memoryDocs={memoryDocs} files={files} />
    );
    const body = screen.getByTestId("session-inspector-body");

    // Trace (top) and memory (bottom) are the default active tabs.
    expect(within(body).getAllByTestId("session-inspector-trace-row").length).toBeGreaterThan(0);
    expect(within(body).getByTestId("session-inspector-memory-list")).toBeInTheDocument();

    await user.click(within(body).getByTestId("session-inspector-tab-usage"));
    expect(within(body).getByTestId("session-inspector-usage-grid")).toBeInTheDocument();

    await user.click(within(body).getByTestId("session-inspector-tab-files"));
    expect(within(body).getByTestId("session-inspector-files-list")).toBeInTheDocument();
  });

  it("switches top and bottom tab groups independently", async () => {
    const user = userEvent.setup();
    render(
      <SessionInspector messages={messages} usage={usage} memoryDocs={memoryDocs} files={files} />
    );
    const body = screen.getByTestId("session-inspector-body");
    const top = () => within(body).getByTestId("session-inspector-top-panel");
    const bottom = () => within(body).getByTestId("session-inspector-bottom-panel");

    await user.click(within(body).getByTestId("session-inspector-tab-files"));
    expect(bottom().getAttribute("data-active-tab")).toBe("files");
    expect(within(body).getByTestId("session-inspector-files-list")).toBeInTheDocument();
    // Top group unchanged.
    expect(top().getAttribute("data-active-tab")).toBe("trace");

    await user.click(within(body).getByTestId("session-inspector-tab-usage"));
    expect(top().getAttribute("data-active-tab")).toBe("usage");
    expect(bottom().getAttribute("data-active-tab")).toBe("files");

    await user.click(within(body).getByTestId("session-inspector-tab-memory"));
    expect(bottom().getAttribute("data-active-tab")).toBe("memory");
    expect(top().getAttribute("data-active-tab")).toBe("usage");

    // Tab triggers expose `role="tab"` via Base UI Tabs for a11y consumers.
    const filesTrigger = within(body).getByTestId("session-inspector-tab-files");
    expect(filesTrigger.getAttribute("role")).toBe("tab");
  });

  it("fires onViewAllTrace when the trace has more events than the render limit", async () => {
    const onViewAll = vi.fn();
    const thread = Array.from({ length: 12 }, (_, i) => ({
      id: `msg-${i}`,
      role: i % 2 === 0 ? ("assistant" as const) : ("user" as const),
      content: `#${i}`,
      timestamp: ts(i),
    }));
    render(<SessionInspector messages={thread} onViewAllTrace={onViewAll} />);
    const viewAll = screen.getAllByTestId("session-inspector-trace-view-all")[0];
    await userEvent.setup().click(viewAll);
    expect(onViewAll).toHaveBeenCalledTimes(1);
  });

  it("opens the drawer with the inspector body inside a Sheet", async () => {
    const user = userEvent.setup();
    render(
      <SessionInspectorDrawer
        messages={messages}
        usage={usage}
        memoryDocs={memoryDocs}
        files={files}
      />
    );
    const trigger = screen.getByTestId("session-inspector-drawer-trigger");
    await user.click(trigger);
    const drawer = await screen.findByTestId("session-inspector-drawer");
    expect(within(drawer).getByTestId("session-inspector-body")).toBeInTheDocument();
  });
});
