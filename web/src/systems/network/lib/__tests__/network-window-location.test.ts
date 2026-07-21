import { describe, expect, it } from "vitest";

import {
  networkThreadsLocation,
  networkWindowTrail,
  parseNetworkWindowLocation,
} from "../network-window-location";

describe("parseNetworkWindowLocation", () => {
  it("Should parse channel tab locations and scope detail ids to their tab", () => {
    const threads = parseNetworkWindowLocation({
      pathname: "/network/ws_1/general/threads/th_1",
      search: { view: "full" },
    });
    expect(threads.workspaceId).toBe("ws_1");
    expect(threads.channel).toBe("general");
    expect(threads.activeThreadId).toBe("th_1");
    expect(threads.activeDirectId).toBeNull();
    expect(threads.view).toBe("full");

    const directs = parseNetworkWindowLocation({
      pathname: "/network/ws_1/general/directs/dr_1",
      search: {},
    });
    expect(directs.activeDirectId).toBe("dr_1");
    expect(directs.activeThreadId).toBeNull();
    expect(directs.view).toBeUndefined();
  });

  it("Should fall back to the root location for unrecognized pathnames", () => {
    const parsed = parseNetworkWindowLocation({ pathname: "/network", search: {} });
    expect(parsed.workspaceId).toBeNull();
    expect(parsed.channel).toBeNull();
    expect(parsed.activeTab).toBe("threads");
  });
});

describe("networkWindowTrail", () => {
  it("Should keep the root head while only a channel is selected", () => {
    const parsed = parseNetworkWindowLocation({
      pathname: "/network/ws_1/general/threads",
      search: {},
    });
    expect(networkWindowTrail(parsed)).toEqual({
      parents: [],
      leaf: "Network",
      drilledIn: false,
    });
  });

  it("Should drill into a conversation with the conversation label as leaf", () => {
    const parsed = parseNetworkWindowLocation({
      pathname: "/network/ws_1/general/threads/th_1",
      search: {},
    });
    expect(networkWindowTrail(parsed, "Cut v0.16 release notes")).toEqual({
      parents: [
        { id: "root", label: "Network" },
        { id: "channel", label: "#general" },
      ],
      leaf: "Cut v0.16 release notes",
      drilledIn: true,
    });
  });

  it("Should fall back to generic conversation leaves when no label resolves", () => {
    const thread = parseNetworkWindowLocation({
      pathname: "/network/ws_1/general/threads/th_1",
      search: {},
    });
    expect(networkWindowTrail(thread).leaf).toBe("Thread");

    const direct = parseNetworkWindowLocation({
      pathname: "/network/ws_1/general/directs/dr_1",
      search: {},
    });
    expect(networkWindowTrail(direct, null).leaf).toBe("Direct");
  });
});

describe("networkThreadsLocation", () => {
  it("Should encode segments and carry the full view flag in search", () => {
    expect(networkThreadsLocation("ws 1", "rel/coord", "th 9", "full")).toEqual({
      pathname: "/network/ws%201/rel%2Fcoord/threads/th%209",
      search: { view: "full" },
    });
    expect(networkThreadsLocation("ws_1", "general").search).toEqual({});
  });
});
