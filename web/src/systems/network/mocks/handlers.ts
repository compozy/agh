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
];
