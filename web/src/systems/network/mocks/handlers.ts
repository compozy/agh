import { http, HttpResponse, type HttpHandler } from "msw";

import {
  createNetworkChannelFixture,
  networkChannelFixture,
  networkChannelsFixture,
  networkDirectRoomDetailFixture,
  networkDirectRoomMessagesFixture,
  networkDirectRoomsFixture,
  networkPeerFixture,
  networkPeersFixture,
  networkRemotePeerFixture,
  networkStatusFixture,
  networkThreadDetailFixture,
  networkThreadMessagesFixture,
  networkThreadsFixture,
  networkWorkFixture,
} from "./fixtures";

function readRecord(value: unknown): Record<string, unknown> | null {
  if (typeof value !== "object" || value === null || Array.isArray(value)) {
    return null;
  }

  return value as Record<string, unknown>;
}

function readRequiredString(record: Record<string, unknown> | null, key: string): string | null {
  const value = record?.[key];
  if (typeof value !== "string") {
    return null;
  }

  const trimmed = value.trim();
  return trimmed.length > 0 ? trimmed : null;
}

function readOptionalString(
  record: Record<string, unknown> | null,
  key: string
): string | undefined {
  const value = record?.[key];
  if (typeof value !== "string") {
    return undefined;
  }

  return value.trim() || undefined;
}

export const handlers: HttpHandler[] = [
  http.get("/api/network/status", () => HttpResponse.json({ network: networkStatusFixture })),
  http.get("/api/network/channels", () => HttpResponse.json(networkChannelsFixture)),
  http.get("/api/network/channels/:channel", ({ params }) => {
    const channel = String(params.channel);

    if (!networkChannelsFixture.channels.some(candidate => candidate.channel === channel)) {
      return HttpResponse.json({ error: `Channel not found: ${channel}` }, { status: 404 });
    }

    return HttpResponse.json({
      channel: {
        ...networkChannelFixture,
        channel,
      },
    });
  }),
  http.get("/api/network/channels/:channel/threads", ({ params }) => {
    const channel = String(params.channel);

    if (!networkChannelsFixture.channels.some(candidate => candidate.channel === channel)) {
      return HttpResponse.json({ error: `Channel not found: ${channel}` }, { status: 404 });
    }

    return HttpResponse.json({
      threads: networkThreadsFixture.map(thread => ({ ...thread, channel })),
    });
  }),
  http.get("/api/network/channels/:channel/threads/:thread_id", ({ params }) => {
    const channel = String(params.channel);
    const threadId = String(params.thread_id);

    if (!networkChannelsFixture.channels.some(candidate => candidate.channel === channel)) {
      return HttpResponse.json({ error: `Channel not found: ${channel}` }, { status: 404 });
    }

    return HttpResponse.json({
      thread: {
        ...networkThreadDetailFixture,
        channel,
        thread_id: threadId,
      },
    });
  }),
  http.get("/api/network/channels/:channel/threads/:thread_id/messages", ({ params }) => {
    const channel = String(params.channel);
    const threadId = String(params.thread_id);

    if (!networkChannelsFixture.channels.some(candidate => candidate.channel === channel)) {
      return HttpResponse.json({ error: `Channel not found: ${channel}` }, { status: 404 });
    }

    return HttpResponse.json({
      messages: networkThreadMessagesFixture.map(message => ({
        ...message,
        channel,
        surface: "thread",
        thread_id: threadId,
      })),
    });
  }),
  http.get("/api/network/channels/:channel/directs", ({ params }) => {
    const channel = String(params.channel);

    if (!networkChannelsFixture.channels.some(candidate => candidate.channel === channel)) {
      return HttpResponse.json({ error: `Channel not found: ${channel}` }, { status: 404 });
    }

    return HttpResponse.json({
      directs: networkDirectRoomsFixture.map(direct => ({ ...direct, channel })),
    });
  }),
  http.post("/api/network/channels/:channel/directs/resolve", async ({ params, request }) => {
    const channel = String(params.channel);
    const body = readRecord(await request.json());
    const peerId = readRequiredString(body, "peer_id");
    const sessionId = readRequiredString(body, "session_id");

    if (!networkChannelsFixture.channels.some(candidate => candidate.channel === channel)) {
      return HttpResponse.json({ error: `Channel not found: ${channel}` }, { status: 404 });
    }

    if (!peerId || !sessionId) {
      return HttpResponse.json(
        { error: "peer_id and session_id are required to resolve a direct room." },
        { status: 400 }
      );
    }

    return HttpResponse.json({
      direct: {
        ...networkDirectRoomDetailFixture,
        channel,
      },
    });
  }),
  http.get("/api/network/channels/:channel/directs/:direct_id", ({ params }) => {
    const channel = String(params.channel);
    const directId = String(params.direct_id);

    if (!networkChannelsFixture.channels.some(candidate => candidate.channel === channel)) {
      return HttpResponse.json({ error: `Channel not found: ${channel}` }, { status: 404 });
    }

    return HttpResponse.json({
      direct: {
        ...networkDirectRoomDetailFixture,
        channel,
        direct_id: directId,
      },
    });
  }),
  http.get("/api/network/channels/:channel/directs/:direct_id/messages", ({ params }) => {
    const channel = String(params.channel);
    const directId = String(params.direct_id);

    if (!networkChannelsFixture.channels.some(candidate => candidate.channel === channel)) {
      return HttpResponse.json({ error: `Channel not found: ${channel}` }, { status: 404 });
    }

    return HttpResponse.json({
      messages: networkDirectRoomMessagesFixture.map(message => ({
        ...message,
        channel,
        surface: "direct",
        direct_id: directId,
      })),
    });
  }),
  http.get("/api/network/work/:work_id", ({ params }) => {
    const workId = String(params.work_id);
    return HttpResponse.json({
      work: {
        ...networkWorkFixture,
        work_id: workId,
      },
    });
  }),
  http.get("/api/network/peers", ({ request }) => {
    const channel = new URL(request.url).searchParams.get("channel");
    const peers = channel
      ? networkPeersFixture.filter(peer => peer.channel === channel)
      : networkPeersFixture;

    return HttpResponse.json({ peers });
  }),
  http.get("/api/network/peers/:peer_id", ({ params }) => {
    const peerId = String(params.peer_id);
    const peerSummary = networkPeersFixture.find(peer => peer.peer_id === peerId);

    if (!peerSummary) {
      return HttpResponse.json({ error: `Peer not found: ${peerId}` }, { status: 404 });
    }

    const baseDetail = peerSummary.local ? networkPeerFixture : networkRemotePeerFixture;

    return HttpResponse.json({
      peer: {
        ...baseDetail,
        channel: peerSummary.channel,
        joined_at: peerSummary.joined_at,
        last_seen: peerSummary.last_seen,
        local: peerSummary.local,
        peer_id: peerId,
        display_name: peerSummary.display_name ?? baseDetail.display_name,
        peer_card: peerSummary.peer_card,
        session_id: peerSummary.session_id,
      },
    });
  }),
  http.post("/api/network/channels", async ({ request }) => {
    const body = (await request.json()) as {
      agent_names?: string[];
      channel?: string;
      purpose?: string;
      workspace_id?: string;
    };

    if (!body.channel?.trim() || !body.workspace_id?.trim() || !body.purpose?.trim()) {
      return HttpResponse.json(
        { error: "Channel, workspace, and purpose are required." },
        { status: 400 }
      );
    }

    return HttpResponse.json(
      {
        channel: {
          ...createNetworkChannelFixture.channel,
          channel: body.channel.trim(),
          purpose: body.purpose.trim(),
          sessions: createNetworkChannelFixture.channel.sessions?.map((session, index) => ({
            ...session,
            id: `sess-created-${index + 1}`,
            agent_name: body.agent_names?.[index] ?? session.agent_name,
            workspace_id: body.workspace_id,
          })),
        },
      },
      { status: 201 }
    );
  }),
  http.post("/api/network/send", async ({ request }) => {
    const body = readRecord(await request.json());
    const sessionId = readRequiredString(body, "session_id");
    const channel = readRequiredString(body, "channel");
    const kind = readRequiredString(body, "kind");

    if ((body != null && Object.hasOwn(body, "interaction_id")) || kind === "direct") {
      return HttpResponse.json(
        { error: "Use surface/direct_id/thread_id/work_id; legacy direct kind is not accepted." },
        { status: 400 }
      );
    }

    if (!sessionId || !channel || !kind) {
      return HttpResponse.json(
        { error: "Session, channel, and kind are required." },
        { status: 400 }
      );
    }

    return HttpResponse.json({
      message: {
        id: readOptionalString(body, "id") ?? "msg_risk_ops_sent",
        session_id: sessionId,
        channel,
        kind,
        surface: readOptionalString(body, "surface"),
        thread_id: readOptionalString(body, "thread_id"),
        direct_id: readOptionalString(body, "direct_id"),
        work_id: readOptionalString(body, "work_id"),
        to: readOptionalString(body, "to"),
        reply_to: readOptionalString(body, "reply_to"),
        trace_id: readOptionalString(body, "trace_id"),
        causation_id: readOptionalString(body, "causation_id"),
        expires_at: typeof body?.expires_at === "number" ? body.expires_at : undefined,
      },
    });
  }),
];
