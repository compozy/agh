import { describe, expect, it } from "vitest";

import type { NetworkCapabilityBrief, NetworkCapabilityCatalog } from "../types";
import {
  buildPeerCapabilityViews,
  formatNetworkKindLabel,
  getChannelRecencyAt,
  getPeerRecencyAt,
  hasCapabilityDetail,
  isHistoricalChannel,
  isPresenceOnlyChannel,
  sortNetworkChannels,
  sortNetworkPeers,
  summarizeChannelMeta,
  summarizeChannelPreview,
  summarizeChannelSubtitle,
} from "./network-formatters";

describe("buildPeerCapabilityViews", () => {
  it("Should merge brief entries with their catalog counterparts by id", () => {
    const brief: NetworkCapabilityBrief[] = [
      { id: "chat", summary: "Brief summary" },
      { id: "tools", summary: "Runs tools" },
    ];
    const catalog: NetworkCapabilityCatalog = {
      capabilities: [
        {
          id: "chat",
          summary: "Rich summary",
          outcome: "Peers align.",
          requirements: ["introspection"],
        },
      ],
    };

    const views = buildPeerCapabilityViews(brief, catalog);

    expect(views).toHaveLength(2);
    expect(views[0]?.id).toBe("chat");
    expect(views[0]?.summary).toBe("Brief summary");
    expect(views[0]?.detail?.outcome).toBe("Peers align.");
    expect(views[1]?.id).toBe("tools");
    expect(views[1]?.detail).toBeNull();
  });

  it("Should surface catalog-only capabilities that lack a brief entry", () => {
    const catalog: NetworkCapabilityCatalog = {
      capabilities: [{ id: "orphan", summary: "Only rich.", outcome: "No brief entry." }],
    };

    const views = buildPeerCapabilityViews(undefined, catalog);

    expect(views).toHaveLength(1);
    expect(views[0]?.id).toBe("orphan");
    expect(views[0]?.summary).toBe("Only rich.");
    expect(views[0]?.detail?.outcome).toBe("No brief entry.");
  });

  it("Should return an empty list when neither brief nor catalog has entries", () => {
    expect(buildPeerCapabilityViews(undefined, undefined)).toEqual([]);
    expect(buildPeerCapabilityViews([], { capabilities: [] })).toEqual([]);
  });
});

describe("hasCapabilityDetail", () => {
  it("Should report false when no catalog detail is attached", () => {
    expect(hasCapabilityDetail({ id: "chat", summary: "Brief only", detail: null })).toBe(false);
  });

  it("Should report true when any rich field carries content", () => {
    expect(
      hasCapabilityDetail({
        id: "chat",
        summary: "Brief",
        detail: {
          id: "chat",
          summary: "Brief",
          outcome: "Peers align.",
        },
      })
    ).toBe(true);

    expect(
      hasCapabilityDetail({
        id: "chat",
        summary: "Brief",
        detail: {
          id: "chat",
          summary: "Brief",
          outcome: "",
          requirements: ["introspection"],
        },
      })
    ).toBe(true);
  });

  it("Should report false for an empty catalog detail stub", () => {
    expect(
      hasCapabilityDetail({
        id: "chat",
        summary: "Brief",
        detail: { id: "chat", summary: "Brief", outcome: "" },
      })
    ).toBe(false);
  });
});

describe("formatNetworkKindLabel", () => {
  it.each(["capability", "say", "direct", "trace", "receipt", "greet", "whois"] as const)(
    "Should keep the %s timeline event aligned with the API kind name",
    kind => {
      expect(formatNetworkKindLabel(kind)).toBe(kind);
    }
  );

  it("Should preserve unknown kind strings as-is", () => {
    expect(formatNetworkKindLabel("custom-signal")).toBe("custom-signal");
  });
});

describe("presence-aware channel summaries", () => {
  it("Should mark presence-only rooms without pretending they have conversation", () => {
    const channel = {
      historical_participant_count: 2,
      message_count: 0,
      peer_count: 0,
      presence_count: 24,
      session_count: 0,
    };

    expect(isPresenceOnlyChannel(channel)).toBe(true);
    expect(summarizeChannelPreview(channel)).toBe("Presence only");
    expect(summarizeChannelSubtitle(channel)).toBe("2 participants · 24 presence");
  });

  it("Should label historical rooms once runtime peers are gone", () => {
    const channel = {
      historical_participant_count: 3,
      message_count: 4,
      peer_count: 0,
      presence_count: 0,
      session_count: 0,
    };

    expect(isHistoricalChannel(channel)).toBe(true);
    expect(summarizeChannelSubtitle(channel)).toBe("3 participants · historical");
  });

  it("Should treat omitted message_count as zero for history-only direct rooms", () => {
    const channel = {
      historical_participant_count: 2,
      peer_count: 0,
      presence_count: 2,
      session_count: 0,
    };

    expect(isPresenceOnlyChannel(channel)).toBe(true);
    expect(summarizeChannelPreview(channel)).toBe("Presence only");
    expect(summarizeChannelSubtitle(channel)).toBe("2 participants · 2 presence");
  });
});

describe("sortNetworkChannels", () => {
  it("Should sort presence-only rooms by last_presence_at when last_activity_at is missing", () => {
    const sorted = sortNetworkChannels([
      {
        channel: "alpha-room",
        created_at: "2026-04-28T04:12:31.630954Z",
        historical_participant_count: 2,
        last_presence_at: "2026-04-28T04:12:31.621681Z",
        peer_count: 0,
        presence_count: 2,
        purpose: "Older presence-only room",
        workspace_id: "ws_test",
      },
      {
        channel: "zulu-room",
        created_at: "2026-04-28T04:54:52.738585Z",
        historical_participant_count: 2,
        last_presence_at: "2026-04-28T04:54:52.730068Z",
        peer_count: 0,
        presence_count: 2,
        purpose: "Newer presence-only room",
        workspace_id: "ws_test",
      },
    ]);

    expect(sorted.map(channel => channel.channel)).toEqual(["zulu-room", "alpha-room"]);
  });

  it("Should sort reactivated rooms by fresher presence even when older activity exists", () => {
    const sorted = sortNetworkChannels([
      {
        channel: "coord.core",
        created_at: "2026-04-28T07:30:00Z",
        historical_participant_count: 2,
        last_activity_at: "2026-04-28T07:40:00Z",
        last_presence_at: "2026-04-28T07:40:00Z",
        message_count: 6,
        peer_count: 2,
        purpose: "Ongoing coordination room",
        workspace_id: "ws_test",
      },
      {
        channel: "launch-room",
        created_at: "2026-04-28T07:00:00Z",
        historical_participant_count: 7,
        last_activity_at: "2026-04-28T07:20:00Z",
        last_presence_at: "2026-04-28T07:50:00Z",
        message_count: 4,
        peer_count: 2,
        purpose: "Reactivated historical room",
        workspace_id: "ws_test",
      },
    ]);

    expect(sorted.map(channel => channel.channel)).toEqual(["launch-room", "coord.core"]);
  });
});

describe("channel recency helpers", () => {
  it("Should use fresher presence for effective channel recency when a historical room is reactivated", () => {
    expect(
      getChannelRecencyAt({
        last_activity_at: "2026-04-28T07:20:00Z",
        last_presence_at: "2026-04-28T07:50:00Z",
      })
    ).toBe("2026-04-28T07:50:00Z");
  });

  it("Should label fresher presence as presence metadata instead of stale activity", () => {
    expect(
      summarizeChannelMeta({
        last_activity_at: "2026-04-28T07:20:00Z",
        last_presence_at: "2026-04-28T07:50:00Z",
        message_count: 4,
        presence_count: 8,
      })
    ).toMatch(/^presence /);
  });
});

describe("peer recency helpers", () => {
  it("Should fall back to joined_at when local peers do not expose last_seen", () => {
    const peer = {
      joined_at: "2026-04-28T06:59:29.95469Z",
      last_seen: undefined,
    };

    expect(getPeerRecencyAt(peer)).toBe("2026-04-28T06:59:29.95469Z");
  });

  it("Should sort local peers by effective recency before display name", () => {
    const sorted = sortNetworkPeers([
      {
        channel: "builders",
        display_name: "Reviewer",
        joined_at: "2026-04-28T06:59:26.991192Z",
        local: true,
        peer_card: {
          artifacts_supported: ["capability"],
          capabilities: [],
          display_name: "Reviewer",
          peer_id: "peer-reviewer",
          profiles_supported: ["agh-network/v0"],
          trust_modes_supported: [],
        },
        peer_id: "peer-reviewer",
        session_id: "sess-reviewer",
      },
      {
        channel: "builders",
        display_name: "Coder",
        joined_at: "2026-04-28T06:59:29.95469Z",
        local: true,
        peer_card: {
          artifacts_supported: ["capability"],
          capabilities: [],
          display_name: "Coder",
          peer_id: "peer-coder",
          profiles_supported: ["agh-network/v0"],
          trust_modes_supported: [],
        },
        peer_id: "peer-coder",
        session_id: "sess-coder",
      },
    ]);

    expect(sorted.map(peer => peer.peer_id)).toEqual(["peer-coder", "peer-reviewer"]);
  });
});
