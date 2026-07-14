import { describe, expect, it } from "vitest";

import {
  DEFAULT_NETWORK_PARTICIPATION_DRAFT,
  serializeNetworkParticipation,
} from "../network-participation";

describe("serializeNetworkParticipation", () => {
  it("Should default to local without legacy participation fields", () => {
    const payload = serializeNetworkParticipation(DEFAULT_NETWORK_PARTICIPATION_DRAFT);
    expect(payload).toEqual({ mode: "local" });
  });

  it("Should serialize live mode with optional channel addressing", () => {
    const payload = serializeNetworkParticipation({
      mode: "live",
      channelId: "builders",
      channelStrategy: "explicit",
    });
    expect(payload).toEqual({
      mode: "live",
      channel_id: "builders",
      channel_strategy: "explicit",
    });
  });
});
