import { getResponse } from "msw";
import { describe, expect, it } from "vitest";

import {
  networkChannelMessagesFixture,
  networkPeerMessagesFixture,
  networkPeerFixture,
  networkRemotePeerFixture,
} from "./fixtures";
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

  it("keeps peer detail mocks aligned with the truthful local-vs-remote payload shape", async () => {
    const localResponse = await getResponse(
      handlers,
      new Request("http://localhost/api/network/peers/peer_storybook_local"),
      { baseUrl: "http://localhost" }
    );
    const remoteResponse = await getResponse(
      handlers,
      new Request("http://localhost/api/network/peers/peer_storybook_remote"),
      { baseUrl: "http://localhost" }
    );

    expect(localResponse).not.toBeUndefined();
    expect(remoteResponse).not.toBeUndefined();
    expect(localResponse?.ok).toBe(true);
    expect(remoteResponse?.ok).toBe(true);

    const localPayload = (await localResponse?.json()) as { peer: typeof networkPeerFixture };
    const remotePayload = (await remoteResponse?.json()) as {
      peer: typeof networkRemotePeerFixture;
    };

    expect(localPayload.peer.local).toBe(true);
    expect(localPayload.peer.joined_at).toBe("2026-04-17T17:40:00Z");
    expect(localPayload.peer.last_seen).toBeUndefined();
    expect(localPayload.peer.peer_card.peer_id).toBe("peer_storybook_local");

    expect(remotePayload.peer.local).toBe(false);
    expect(remotePayload.peer.joined_at).toBe("2026-04-17T17:42:00Z");
    expect(remotePayload.peer.last_seen).toBe("2026-04-17T18:09:00Z");
    expect(remotePayload.peer.peer_card.peer_id).toBe("peer_storybook_remote");
  });

  it("handles Storybook send requests without falling through to the real daemon", async () => {
    const response = await getResponse(
      handlers,
      new Request("http://localhost/api/network/send", {
        body: JSON.stringify({
          body: { text: "Ship the Storybook route." },
          channel: "storybook",
          kind: "say",
          session_id: "sess-storybook",
        }),
        method: "POST",
      }),
      { baseUrl: "http://localhost" }
    );

    expect(response).not.toBeUndefined();
    expect(response?.ok).toBe(true);

    const payload = (await response?.json()) as {
      message: { channel: string; id: string; kind: string; session_id: string };
    };

    expect(payload.message).toEqual({
      channel: "storybook",
      id: "msg_storybook_sent",
      kind: "say",
      session_id: "sess-storybook",
    });
  });

  it("returns a validation response for malformed Storybook send requests", async () => {
    const response = await getResponse(
      handlers,
      new Request("http://localhost/api/network/send", {
        body: JSON.stringify({
          channel: 123,
          kind: "say",
          session_id: "sess-storybook",
        }),
        method: "POST",
      }),
      { baseUrl: "http://localhost" }
    );

    expect(response).not.toBeUndefined();
    expect(response?.status).toBe(400);

    const payload = (await response?.json()) as { error: string };
    expect(payload.error).toBe("Session, channel, and kind are required.");
  });
});
