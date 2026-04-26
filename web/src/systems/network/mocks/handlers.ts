import { http, HttpResponse, type HttpHandler } from "msw";

import {
  createNetworkChannelFixture,
  networkChannelFixture,
  networkChannelMessagesFixture,
  networkChannelsFixture,
  networkPeerFixture,
  networkPeerMessagesFixture,
  networkPeersFixture,
  networkStatusFixture,
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
  http.get("/api/network/channels/:channel/messages", ({ params }) => {
    const channel = String(params.channel);

    if (!networkChannelsFixture.channels.some(candidate => candidate.channel === channel)) {
      return HttpResponse.json({ error: `Channel not found: ${channel}` }, { status: 404 });
    }

    return HttpResponse.json({
      messages: networkChannelMessagesFixture.map(message => ({
        ...message,
        channel,
      })),
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

    if (!networkPeersFixture.some(peer => peer.peer_id === peerId)) {
      return HttpResponse.json({ error: `Peer not found: ${peerId}` }, { status: 404 });
    }

    return HttpResponse.json({
      peer: {
        ...networkPeerFixture,
        peer_id: peerId,
        display_name:
          networkPeersFixture.find(peer => peer.peer_id === peerId)?.display_name ??
          networkPeerFixture.display_name,
      },
    });
  }),
  http.get("/api/network/peers/:peer_id/messages", ({ params }) => {
    const peerId = String(params.peer_id);

    if (!networkPeersFixture.some(peer => peer.peer_id === peerId)) {
      return HttpResponse.json({ error: `Peer not found: ${peerId}` }, { status: 404 });
    }

    return HttpResponse.json({
      messages: networkPeerMessagesFixture,
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

    if (!sessionId || !channel || !kind) {
      return HttpResponse.json(
        { error: "Session, channel, and kind are required." },
        { status: 400 }
      );
    }

    return HttpResponse.json({
      message: {
        id: readOptionalString(body, "id") ?? "msg_storybook_sent",
        session_id: sessionId,
        channel,
        kind,
        to: readOptionalString(body, "to"),
        interaction_id: readOptionalString(body, "interaction_id"),
        reply_to: readOptionalString(body, "reply_to"),
        trace_id: readOptionalString(body, "trace_id"),
        causation_id: readOptionalString(body, "causation_id"),
        expires_at: typeof body?.expires_at === "number" ? body.expires_at : undefined,
      },
    });
  }),
];
