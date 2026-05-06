import { describe, expect, it } from "vitest";

import {
  createNetworkChannelDraft,
  formatNetworkKindLabel,
  formatNetworkRelativeTime,
  getMessageAuthorInitial,
  getMostRecentTimestamp,
  getNetworkKindTone,
  getNetworkStatusTone,
  getPeerDisplayName,
  getPeerRecencyAt,
  isNetworkRunning,
  toggleDraftAgent,
  toNetworkKindFilter,
} from "../network-formatters";

describe("formatNetworkKindLabel", () => {
  it.each(["capability", "say", "trace", "receipt", "greet", "whois"] as const)(
    "Should keep the %s timeline event aligned with the API kind name",
    kind => {
      expect(formatNetworkKindLabel(kind)).toBe(kind);
    }
  );

  it("Should reject the legacy 'direct' kind", () => {
    expect(toNetworkKindFilter("direct")).toBeNull();
  });

  it("Should preserve unknown kind strings as-is", () => {
    expect(formatNetworkKindLabel("custom-signal")).toBe("custom-signal");
  });

  it("Should map known kinds to a chromatic tone, unknown to neutral", () => {
    expect(getNetworkKindTone("capability")).toBe("info");
    expect(getNetworkKindTone("trace")).toBe("info");
    expect(getNetworkKindTone("custom-signal")).toBe("neutral");
  });
});

describe("getMostRecentTimestamp", () => {
  it("returns the fresher of the two values", () => {
    expect(getMostRecentTimestamp("2026-04-13T10:00:00Z", "2026-04-13T11:00:00Z")).toBe(
      "2026-04-13T11:00:00Z"
    );
  });

  it("falls back when one side is missing", () => {
    expect(getMostRecentTimestamp(null, "2026-04-13T10:00:00Z")).toBe("2026-04-13T10:00:00Z");
    expect(getMostRecentTimestamp("2026-04-13T10:00:00Z", null)).toBe("2026-04-13T10:00:00Z");
    expect(getMostRecentTimestamp(null, null)).toBeNull();
  });
});

describe("formatNetworkRelativeTime", () => {
  it("falls back to a stable label for missing or invalid input", () => {
    expect(formatNetworkRelativeTime(undefined)).toBe("Unavailable");
    expect(formatNetworkRelativeTime("not-a-date")).toBe("Unavailable");
  });

  it("returns 'just now' for very recent timestamps", () => {
    const now = new Date().toISOString();
    expect(formatNetworkRelativeTime(now)).toBe("just now");
  });
});

describe("createNetworkChannelDraft", () => {
  it("creates an empty draft with no agents selected", () => {
    expect(createNetworkChannelDraft()).toEqual({
      channelName: "",
      purpose: "",
      selectedAgentNames: [],
    });
  });

  it("toggles an agent into and out of the draft selection", () => {
    const empty = createNetworkChannelDraft();
    const added = toggleDraftAgent(empty, "alpha");
    const removed = toggleDraftAgent(added, "alpha");

    expect(added.selectedAgentNames).toEqual(["alpha"]);
    expect(removed.selectedAgentNames).toEqual([]);
  });
});

describe("getNetworkStatusTone", () => {
  it.each([
    ["running", "success"],
    ["online", "success"],
    ["starting", "warning"],
    ["degraded", "warning"],
    ["stopped", "danger"],
    ["offline", "danger"],
    ["unknown-state", "neutral"],
    [null, "neutral"],
  ] as const)("maps %s to %s tone", (status, tone) => {
    expect(getNetworkStatusTone(status)).toBe(tone);
  });
});

describe("isNetworkRunning", () => {
  it("treats enabled+running as running", () => {
    expect(
      isNetworkRunning({
        channels: 1,
        delivery_workers: 1,
        enabled: true,
        local_peers: 1,
        messages_sent: 0,
        queued_messages: 0,
        remote_peers: 0,
        status: "running",
      })
    ).toBe(true);
  });

  it("treats disabled or stopped network as not running", () => {
    expect(
      isNetworkRunning({
        channels: 0,
        delivery_workers: 0,
        enabled: false,
        local_peers: 0,
        messages_sent: 0,
        queued_messages: 0,
        remote_peers: 0,
        status: "running",
      })
    ).toBe(false);
    expect(
      isNetworkRunning({
        channels: 0,
        delivery_workers: 0,
        enabled: true,
        local_peers: 0,
        messages_sent: 0,
        queued_messages: 0,
        remote_peers: 0,
        status: "stopped",
      })
    ).toBe(false);
  });
});

describe("getPeerDisplayName + getPeerRecencyAt", () => {
  it("falls through display_name → peer_card.display_name → peer_id", () => {
    expect(
      getPeerDisplayName({
        display_name: "Reviewer",
        peer_card: {
          artifacts_supported: [],
          capabilities: [],
          display_name: "Reviewer (card)",
          peer_id: "peer-reviewer",
          profiles_supported: [],
          trust_modes_supported: [],
        },
        peer_id: "peer-reviewer",
      })
    ).toBe("Reviewer");
  });

  it("returns the freshest of last_seen and joined_at", () => {
    expect(
      getPeerRecencyAt({
        joined_at: "2026-04-28T06:00:00Z",
        last_seen: "2026-04-28T07:00:00Z",
      })
    ).toBe("2026-04-28T07:00:00Z");
    expect(
      getPeerRecencyAt({
        joined_at: "2026-04-28T07:00:00Z",
        last_seen: undefined,
      })
    ).toBe("2026-04-28T07:00:00Z");
  });
});

describe("getMessageAuthorInitial", () => {
  it("uses display_name first letter, capitalized", () => {
    expect(
      getMessageAuthorInitial({
        display_name: "claude-opus",
        peer_from: "peer-x",
      })
    ).toBe("C");
  });

  it("falls back to peer_from when display_name is missing", () => {
    expect(getMessageAuthorInitial({ peer_from: "  reviewer" })).toBe("R");
  });

  it("returns a stable placeholder for empty author info", () => {
    expect(getMessageAuthorInitial({ peer_from: "" })).toBe("?");
  });
});
