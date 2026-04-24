import { getResponse } from "msw";
import { describe, expect, it } from "vitest";

import { networkChannelMessagesFixture, networkPeerMessagesFixture } from "./fixtures";
import { handlers } from "./handlers";

describe("network mock contracts", () => {
  it("keeps message fixtures aligned with the server mapper contract", () => {
    expect(networkChannelMessagesFixture[0]).toMatchObject({
      direction: "sent",
      display_name: "Storybook rollout",
      local: true,
      session_id: "sess-storybook",
    });
    expect(networkPeerMessagesFixture[0]).toMatchObject({
      direction: "sent",
      display_name: "Storybook rollout",
      local: true,
      session_id: "sess-storybook",
    });

    for (const message of [
      networkChannelMessagesFixture[1],
      networkChannelMessagesFixture[2],
      networkPeerMessagesFixture[1],
    ]) {
      expect(message).toBeDefined();
      expect(message?.session_id).toBeUndefined();
      expect(message?.direction).toBe("received");
    }
  });

  it("returns an internally consistent remote-peer conversation without rewriting ids", async () => {
    const response = await getResponse(
      handlers,
      new Request("http://localhost/api/network/peers/peer_storybook_remote/messages"),
      { baseUrl: "http://localhost" }
    );

    expect(response).not.toBeUndefined();
    expect(response?.ok).toBe(true);

    const payload = (await response?.json()) as { messages: typeof networkPeerMessagesFixture };

    expect(payload.messages).toEqual(networkPeerMessagesFixture);
  });
});
