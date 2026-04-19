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
  it("renders all four slices populated from a fixture session", () => {
    render(
      <SessionInspector messages={messages} usage={usage} memoryDocs={memoryDocs} files={files} />
    );
    const stacked = screen.getByTestId("session-inspector-stacked");
    expect(within(stacked).getAllByTestId("session-inspector-trace-row").length).toBeGreaterThan(0);
    expect(within(stacked).getByTestId("session-inspector-usage-grid")).toBeInTheDocument();
    expect(within(stacked).getByTestId("session-inspector-memory-list")).toBeInTheDocument();
    expect(within(stacked).getByTestId("session-inspector-files-list")).toBeInTheDocument();
  });

  it("keeps tabbed panels clickable and preserves content when switching", async () => {
    const user = userEvent.setup();
    render(
      <SessionInspector messages={messages} usage={usage} memoryDocs={memoryDocs} files={files} />
    );
    const tabbed = screen.getByTestId("session-inspector-tabbed");

    await user.click(within(tabbed).getByTestId("session-inspector-tab-files"));
    const panel = within(tabbed).getByTestId("session-inspector-tab-panel");
    expect(panel.getAttribute("data-active-tab")).toBe("files");
    expect(within(tabbed).getByTestId("session-inspector-files-list")).toBeInTheDocument();

    await user.click(within(tabbed).getByTestId("session-inspector-tab-memory"));
    expect(
      within(tabbed).getByTestId("session-inspector-tab-panel").getAttribute("data-active-tab")
    ).toBe("memory");

    await user.click(within(tabbed).getByTestId("session-inspector-tab-usage"));
    expect(
      within(tabbed).getByTestId("session-inspector-tab-panel").getAttribute("data-active-tab")
    ).toBe("usage");

    // Tab triggers expose `aria-selected` via Base UI Tabs for a11y consumers.
    const filesTrigger = within(tabbed).getByTestId("session-inspector-tab-files");
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
