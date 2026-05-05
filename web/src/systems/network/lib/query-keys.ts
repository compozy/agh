import type { NetworkSurface } from "../types";

function normalizeText(value?: string | null) {
  return value ?? "";
}

function normalizeLimit(value?: number | null) {
  return value ?? 0;
}

interface ConversationMessagesQuery {
  after?: string | null;
  before?: string | null;
  kind?: string | null;
  limit?: number | null;
  work_id?: string | null;
}

function messagesQuerySegments(query?: ConversationMessagesQuery) {
  return [
    normalizeText(query?.before),
    normalizeText(query?.after),
    normalizeText(query?.kind),
    normalizeText(query?.work_id),
    normalizeLimit(query?.limit),
  ] as const;
}

export const networkKeys = {
  all: ["network"] as const,
  status: () => [...networkKeys.all, "status"] as const,

  channelsRoot: () => [...networkKeys.all, "channels"] as const,
  channels: () => [...networkKeys.channelsRoot(), "list"] as const,
  channelDetails: () => [...networkKeys.channelsRoot(), "detail"] as const,
  channelDetail: (channel: string) =>
    [...networkKeys.channelDetails(), normalizeText(channel)] as const,

  channelScope: (channel: string) =>
    [...networkKeys.all, "channel", normalizeText(channel)] as const,

  threadsList: (channel: string, query?: { after?: string | null; limit?: number | null }) =>
    [
      ...networkKeys.channelScope(channel),
      "thread" satisfies NetworkSurface,
      "list",
      normalizeText(query?.after),
      normalizeLimit(query?.limit),
    ] as const,
  threadDetail: (channel: string, threadId: string) =>
    [
      ...networkKeys.channelScope(channel),
      "thread" satisfies NetworkSurface,
      "detail",
      normalizeText(threadId),
    ] as const,
  threadMessages: (channel: string, threadId: string, query?: ConversationMessagesQuery) =>
    [
      ...networkKeys.channelScope(channel),
      "thread" satisfies NetworkSurface,
      "messages",
      normalizeText(threadId),
      ...messagesQuerySegments(query),
    ] as const,

  directsList: (
    channel: string,
    query?: { after?: string | null; limit?: number | null; peer_id?: string | null }
  ) =>
    [
      ...networkKeys.channelScope(channel),
      "direct" satisfies NetworkSurface,
      "list",
      normalizeText(query?.after),
      normalizeText(query?.peer_id),
      normalizeLimit(query?.limit),
    ] as const,
  directDetail: (channel: string, directId: string) =>
    [
      ...networkKeys.channelScope(channel),
      "direct" satisfies NetworkSurface,
      "detail",
      normalizeText(directId),
    ] as const,
  directMessages: (channel: string, directId: string, query?: ConversationMessagesQuery) =>
    [
      ...networkKeys.channelScope(channel),
      "direct" satisfies NetworkSurface,
      "messages",
      normalizeText(directId),
      ...messagesQuerySegments(query),
    ] as const,

  workRoot: () => [...networkKeys.all, "work"] as const,
  work: (workId: string) => [...networkKeys.workRoot(), normalizeText(workId)] as const,

  peersRoot: () => [...networkKeys.all, "peers"] as const,
  peers: (channel?: string | null) => [...networkKeys.peersRoot(), normalizeText(channel)] as const,
  peerDetails: () => [...networkKeys.peersRoot(), "detail"] as const,
  peerDetail: (peerId: string) => [...networkKeys.peerDetails(), normalizeText(peerId)] as const,
};
