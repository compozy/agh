import { fireEvent, render, screen, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { DETAIL_INSPECTOR_INLINE_BREAKPOINT } from "@agh/ui";

import { SessionLedgerUnavailableError } from "../../adapters/session-api";
import type { SessionLedgerResponse } from "../../types";
import { SessionInspector } from "../session-inspector";

const ORIGINAL_MATCH_MEDIA = window.matchMedia;

function installMatchMedia(matches: boolean): void {
  Object.defineProperty(window, "matchMedia", {
    writable: true,
    configurable: true,
    value: (query: string) => ({
      matches,
      media: query,
      onchange: null,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      addListener: vi.fn(),
      removeListener: vi.fn(),
      dispatchEvent: () => false,
    }),
  });
}

beforeEach(() => {
  installMatchMedia(true);
});

afterEach(() => {
  Object.defineProperty(window, "matchMedia", {
    writable: true,
    configurable: true,
    value: ORIGINAL_MATCH_MEDIA,
  });
});

function makeLedger(overrides?: Partial<SessionLedgerResponse>): SessionLedgerResponse {
  return {
    meta: {
      version: 1,
      session_id: "sess_123",
      workspace_id: "ws_alpha",
      root_session_id: "sess_root",
      parent_session_id: "sess_parent",
      spawn_depth: 2,
      path: "/sessions/ws_alpha/sess_123/ledger.jsonl",
      checksum: "sha256:abc123",
      created_at: "2026-04-20T10:00:00Z",
      stopped_at: "2026-04-20T11:00:00Z",
      ...overrides?.meta,
    },
    events: overrides?.events ?? [
      { sequence: 1, event_type: "session.started", emitted_at: "2026-04-20T10:00:00Z" },
      { sequence: 2, event_type: "memory.recall", emitted_at: "2026-04-20T10:01:00Z" },
      { sequence: 3, event_type: "memory.event", emitted_at: "2026-04-20T10:02:00Z" },
    ],
  };
}

function openMemoryTab() {
  fireEvent.click(screen.getByTestId("session-inspector-tab-memory"));
}

describe("SessionInspector — DetailInspector chrome (ADR-014 §2 / §3)", () => {
  it("Should consume <DetailInspector> with 5 tabs in a single flat tab strip", () => {
    const ledger = makeLedger();
    render(<SessionInspector messages={[]} sessionId="sess_123" memory={{ ledger }} />);

    expect(screen.getByTestId("session-inspector-tab-trace")).toBeInTheDocument();
    expect(screen.getByTestId("session-inspector-tab-usage")).toBeInTheDocument();
    expect(screen.getByTestId("session-inspector-tab-memory")).toBeInTheDocument();
    expect(screen.getByTestId("session-inspector-tab-files")).toBeInTheDocument();
    expect(screen.getByTestId("session-inspector-tab-vault")).toBeInTheDocument();
  });

  it("Should render inline at >= 1440 px viewport (data-mode=inline) at 320 px width", () => {
    installMatchMedia(true);
    const { container } = render(
      <SessionInspector messages={[]} sessionId="sess_123" memory={{ ledger: null }} />
    );
    const root = container.querySelector<HTMLElement>(
      '[data-slot="detail-inspector"][data-mode="inline"]'
    );
    expect(root).not.toBeNull();
    expect(root?.style.width).toBe("320px");
  });

  it("Should collapse into the right-anchored sheet drawer below 1440 px", () => {
    installMatchMedia(false);
    render(
      <SessionInspector
        messages={[]}
        sessionId="sess_123"
        memory={{ ledger: null }}
        drawerOpen
        onDrawerOpenChange={() => {}}
      />
    );
    const drawer = document.querySelector('[data-slot="detail-inspector"][data-mode="drawer"]');
    expect(drawer).not.toBeNull();
  });

  it("Should expose DETAIL_INSPECTOR_INLINE_BREAKPOINT as the canonical 1440 px constant", () => {
    expect(DETAIL_INSPECTOR_INLINE_BREAKPOINT).toBe(1440);
  });
});

describe("SessionInspector — Memory v2 forensic ledger surface", () => {
  it("Should render lineage meta and ledger events when the ledger is materialized", () => {
    const ledger = makeLedger();

    render(<SessionInspector messages={[]} sessionId="sess_123" memory={{ ledger }} />);

    openMemoryTab();

    const memorySurface = screen.getByTestId("session-inspector-memory");
    expect(memorySurface).toHaveAttribute("data-state", "ready");

    const meta = screen.getByTestId("session-inspector-memory-meta");
    expect(
      within(meta).getByTestId("session-inspector-memory-meta-workspace-value")
    ).toHaveTextContent("ws_alpha");
    expect(
      within(meta).getByTestId("session-inspector-memory-meta-root-session-value")
    ).toHaveTextContent("sess_root");
    expect(
      within(meta).getByTestId("session-inspector-memory-meta-parent-session-value")
    ).toHaveTextContent("sess_parent");
    expect(
      within(meta).getByTestId("session-inspector-memory-meta-spawn-depth-value")
    ).toHaveTextContent("2");
    expect(within(meta).getByTestId("session-inspector-memory-meta-path-value")).toHaveTextContent(
      "/sessions/ws_alpha/sess_123/ledger.jsonl"
    );
    expect(
      within(meta).getByTestId("session-inspector-memory-meta-checksum-value")
    ).toHaveTextContent("sha256:abc123");
    expect(
      within(meta).getByTestId("session-inspector-memory-meta-version-value")
    ).toHaveTextContent("v1");

    const eventsPanel = screen.getByTestId("session-inspector-memory-events");
    expect(within(eventsPanel).getByText("Ledger events")).toBeInTheDocument();
    expect(screen.getByTestId("session-inspector-memory-events-count")).toHaveTextContent("3");
    const rows = screen.getAllByTestId("session-inspector-memory-event-row");
    expect(rows).toHaveLength(3);
    expect(
      within(rows[0]!).getByTestId("session-inspector-memory-event-sequence")
    ).toHaveTextContent("#1");
    expect(within(rows[0]!).getByTestId("session-inspector-memory-event-type")).toHaveTextContent(
      "session.started"
    );
    expect(within(rows[1]!).getByTestId("session-inspector-memory-event-type")).toHaveTextContent(
      "memory.recall"
    );
    expect(within(rows[2]!).getByTestId("session-inspector-memory-event-type")).toHaveTextContent(
      "memory.event"
    );
  });

  it("Should label the events panel as ledger events even when no memory.* events are present", () => {
    const ledger = makeLedger({
      events: [
        { sequence: 1, event_type: "session.started", emitted_at: "2026-04-20T10:00:00Z" },
        { sequence: 2, event_type: "transcript.user", emitted_at: "2026-04-20T10:01:00Z" },
        { sequence: 3, event_type: "session.stopped", emitted_at: "2026-04-20T10:05:00Z" },
      ],
    });

    render(<SessionInspector messages={[]} sessionId="sess_123" memory={{ ledger }} />);

    openMemoryTab();

    const eventsPanel = screen.getByTestId("session-inspector-memory-events");
    expect(within(eventsPanel).getByText("Ledger events")).toBeInTheDocument();
    expect(within(eventsPanel).queryByText("Memory events")).not.toBeInTheDocument();
    expect(screen.getByTestId("session-inspector-memory-events-count")).toHaveTextContent("3");
    const rows = screen.getAllByTestId("session-inspector-memory-event-row");
    expect(rows).toHaveLength(3);
    expect(within(rows[0]!).getByTestId("session-inspector-memory-event-type")).toHaveTextContent(
      "session.started"
    );
    expect(within(rows[1]!).getByTestId("session-inspector-memory-event-type")).toHaveTextContent(
      "transcript.user"
    );
    expect(within(rows[2]!).getByTestId("session-inspector-memory-event-type")).toHaveTextContent(
      "session.stopped"
    );
  });

  it("Should render a forensic-empty state when no ledger has materialized yet", () => {
    render(<SessionInspector messages={[]} sessionId="sess_123" memory={{ ledger: null }} />);

    openMemoryTab();

    const memorySurface = screen.getByTestId("session-inspector-memory");
    expect(memorySurface).toHaveAttribute("data-state", "unavailable");
    expect(screen.getByTestId("session-inspector-memory-empty")).toBeInTheDocument();
    expect(screen.getByTestId("session-inspector-memory-empty")).toHaveTextContent(
      "No session ledger yet"
    );
  });

  it("Should treat 404 ledger errors as the truthful empty/unavailable state", () => {
    render(
      <SessionInspector
        messages={[]}
        sessionId="sess_123"
        memory={{ ledger: null, error: new SessionLedgerUnavailableError("sess_123") }}
      />
    );

    openMemoryTab();

    const memorySurface = screen.getByTestId("session-inspector-memory");
    expect(memorySurface).toHaveAttribute("data-state", "unavailable");
    expect(screen.queryByTestId("session-inspector-memory-error")).not.toBeInTheDocument();
    expect(screen.getByTestId("session-inspector-memory-empty")).toBeInTheDocument();
  });

  it("Should render a loading state while the ledger query resolves", () => {
    render(<SessionInspector messages={[]} sessionId="sess_123" memory={{ isLoading: true }} />);

    openMemoryTab();

    expect(screen.getByTestId("session-inspector-memory")).toHaveAttribute("data-state", "loading");
    expect(screen.getByTestId("session-inspector-memory-loading")).toBeInTheDocument();
  });

  it("Should render a forensic error state for non-404 ledger failures", () => {
    render(
      <SessionInspector
        messages={[]}
        sessionId="sess_123"
        memory={{ error: new Error("ledger materializer crashed") }}
      />
    );

    openMemoryTab();

    expect(screen.getByTestId("session-inspector-memory")).toHaveAttribute("data-state", "error");
    expect(screen.getByTestId("session-inspector-memory-error")).toHaveTextContent(
      "ledger materializer crashed"
    );
  });

  it("Should remain read-only and never expose editor, promote, or replay controls", () => {
    const ledger = makeLedger();

    render(<SessionInspector messages={[]} sessionId="sess_123" memory={{ ledger }} />);

    openMemoryTab();

    const memorySurface = screen.getByTestId("session-inspector-memory");
    expect(within(memorySurface).queryAllByRole("button")).toHaveLength(0);
    expect(within(memorySurface).queryAllByRole("textbox")).toHaveLength(0);
    expect(within(memorySurface).queryByText(/promote/i)).not.toBeInTheDocument();
    expect(within(memorySurface).queryByText(/replay/i)).not.toBeInTheDocument();
    expect(within(memorySurface).queryByText(/edit/i)).not.toBeInTheDocument();
  });

  it("Should render an event-empty state when the ledger has zero events", () => {
    const ledger = makeLedger({
      meta: {
        version: 1,
        session_id: "sess_x",
        spawn_depth: 0,
        path: "/p",
        checksum: "sha256:x",
        created_at: "2026-04-20T10:00:00Z",
      },
      events: [],
    });

    render(<SessionInspector messages={[]} sessionId="sess_x" memory={{ ledger }} />);

    openMemoryTab();

    const empty = screen.getByTestId("session-inspector-memory-events-empty");
    expect(empty).toBeInTheDocument();
    expect(empty).toHaveTextContent("No ledger events");
    expect(screen.queryByTestId("session-inspector-memory-events-list")).not.toBeInTheDocument();
  });
});
