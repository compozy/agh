import { getResponse } from "msw";
import { describe, expect, it } from "vitest";

import {
  storyHeroNetworkChannel,
  storyPeerIds,
  storySessionIds,
} from "@/storybook/fintech-scenario";
import {
  networkDirectRoomMessagesFixture,
  networkDirectRoomsFixture,
  networkPeerFixture,
  networkRemotePeerFixture,
  networkThreadMessagesFixture,
  networkThreadsFixture,
} from "./fixtures";
import { handlers } from "./handlers";

describe("network mock contracts", () => {
  it("keeps thread message fixtures aligned with the conversation contract", () => {
    expect(networkThreadMessagesFixture[0]).toMatchObject({
      direction: "sent",
      display_name: "Northstar Launch Control",
      local: true,
      session_id: storySessionIds.product,
    });
    expect(networkDirectRoomMessagesFixture[0]).toMatchObject({
      direction: "sent",
      display_name: "Launch room command brief",
      local: true,
      session_id: storySessionIds.product,
    });

    for (const message of [
      networkThreadMessagesFixture[1],
      networkThreadMessagesFixture[5],
      networkDirectRoomMessagesFixture[1],
    ]) {
      expect(message).toBeDefined();
      expect(message?.session_id).toBeUndefined();
      expect(message?.direction).toBe("received");
    }

    expect(networkThreadMessagesFixture.length).toBeGreaterThanOrEqual(20);
  });

  it("returns the thread list with surface-aligned summary rows", async () => {
    const response = await getResponse(
      handlers,
      new Request(`http://localhost/api/network/channels/${storyHeroNetworkChannel}/threads`),
      { baseUrl: "http://localhost" }
    );

    expect(response).not.toBeUndefined();
    expect(response?.ok).toBe(true);

    const payload = (await response?.json()) as { threads: typeof networkThreadsFixture };
    expect(payload.threads.map(thread => thread.thread_id)).toEqual(
      networkThreadsFixture.map(thread => thread.thread_id)
    );
  });

  it("returns the direct room list with two-party membership", async () => {
    const response = await getResponse(
      handlers,
      new Request(`http://localhost/api/network/channels/${storyHeroNetworkChannel}/directs`),
      { baseUrl: "http://localhost" }
    );

    expect(response).not.toBeUndefined();
    expect(response?.ok).toBe(true);

    const payload = (await response?.json()) as { directs: typeof networkDirectRoomsFixture };
    expect(payload.directs[0]?.peer_a < (payload.directs[0]?.peer_b ?? "")).toBe(true);
  });

  it("returns thread messages without leaking direct surface fields", async () => {
    const thread = networkThreadsFixture[0];
    expect(thread).toBeDefined();
    if (!thread) {
      return;
    }
    const response = await getResponse(
      handlers,
      new Request(
        `http://localhost/api/network/channels/${storyHeroNetworkChannel}/threads/${thread.thread_id}/messages`
      ),
      { baseUrl: "http://localhost" }
    );
    expect(response?.ok).toBe(true);
    const payload = (await response?.json()) as {
      messages: { surface: string; thread_id: string; direct_id?: string }[];
    };
    for (const message of payload.messages) {
      expect(message.surface).toBe("thread");
      expect(message.thread_id).toBe(thread.thread_id);
    }
  });

  it("keeps peer detail mocks aligned with the truthful local-vs-remote payload shape", async () => {
    const localResponse = await getResponse(
      handlers,
      new Request(`http://localhost/api/network/peers/${storyPeerIds.local}`),
      { baseUrl: "http://localhost" }
    );
    const remoteResponse = await getResponse(
      handlers,
      new Request(`http://localhost/api/network/peers/${storyPeerIds.remote}`),
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
    expect(localPayload.peer.joined_at).toBe("2026-04-17T14:00:00Z");
    expect(localPayload.peer.last_seen).toBeUndefined();
    expect(localPayload.peer.peer_card.peer_id).toBe(storyPeerIds.local);

    expect(remotePayload.peer.local).toBe(false);
    expect(remotePayload.peer.joined_at).toBe("2026-04-17T14:08:00Z");
    expect(remotePayload.peer.last_seen).toBe("2026-04-17T18:15:00Z");
    expect(remotePayload.peer.peer_card.peer_id).toBe(storyPeerIds.remote);
  });

  it("rejects legacy direct kind sends and returns the canonical error", async () => {
    const response = await getResponse(
      handlers,
      new Request("http://localhost/api/network/send", {
        body: JSON.stringify({
          channel: storyHeroNetworkChannel,
          kind: "direct",
          session_id: storySessionIds.product,
          to: storyPeerIds.remote,
        }),
        method: "POST",
      }),
      { baseUrl: "http://localhost" }
    );
    expect(response?.status).toBe(400);
    const payload = (await response?.json()) as { error: string };
    expect(payload.error).toContain("legacy direct kind is not accepted");
  });

  it("handles network send requests without falling through to the real daemon", async () => {
    const response = await getResponse(
      handlers,
      new Request("http://localhost/api/network/send", {
        body: JSON.stringify({
          body: { text: "Please confirm whether the BR timeout copy is still blocked." },
          channel: storyHeroNetworkChannel,
          kind: "say",
          session_id: storySessionIds.product,
          surface: "thread",
          thread_id: "thread_launch_command",
        }),
        method: "POST",
      }),
      { baseUrl: "http://localhost" }
    );

    expect(response).not.toBeUndefined();
    expect(response?.ok).toBe(true);

    const payload = (await response?.json()) as {
      message: { channel: string; id: string; kind: string; session_id: string; surface?: string };
    };

    expect(payload.message).toMatchObject({
      channel: storyHeroNetworkChannel,
      kind: "say",
      session_id: storySessionIds.product,
      surface: "thread",
    });
  });

  it("returns a validation response for malformed network send requests", async () => {
    const response = await getResponse(
      handlers,
      new Request("http://localhost/api/network/send", {
        body: JSON.stringify({
          channel: 123,
          kind: "say",
          session_id: storySessionIds.product,
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
