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

  workspace: (workspaceId: string) =>
    [...networkKeys.all, "workspace", normalizeText(workspaceId)] as const,

  channelsRoot: (workspaceId: string) =>
    [...networkKeys.workspace(workspaceId), "channels"] as const,
  channels: (workspaceId: string) => [...networkKeys.channelsRoot(workspaceId), "list"] as const,
  channelDetails: (workspaceId: string) =>
    [...networkKeys.channelsRoot(workspaceId), "detail"] as const,
  channelDetail: (workspaceId: string, channel: string) =>
    [...networkKeys.channelDetails(workspaceId), normalizeText(channel)] as const,

  channelScope: (workspaceId: string, channel: string) =>
    [...networkKeys.workspace(workspaceId), "channel", normalizeText(channel)] as const,

  threadsList: (
    workspaceId: string,
    channel: string,
    query?: { after?: string | null; limit?: number | null }
  ) =>
    [
      ...networkKeys.channelScope(workspaceId, channel),
      "thread" satisfies NetworkSurface,
      "list",
      normalizeText(query?.after),
      normalizeLimit(query?.limit),
    ] as const,
  threadDetail: (workspaceId: string, channel: string, threadId: string) =>
    [
      ...networkKeys.channelScope(workspaceId, channel),
      "thread" satisfies NetworkSurface,
      "detail",
      normalizeText(threadId),
    ] as const,
  threadMessages: (
    workspaceId: string,
    channel: string,
    threadId: string,
    query?: ConversationMessagesQuery
  ) =>
    [
      ...networkKeys.channelScope(workspaceId, channel),
      "thread" satisfies NetworkSurface,
      "messages",
      normalizeText(threadId),
      ...messagesQuerySegments(query),
    ] as const,

  directsList: (
    workspaceId: string,
    channel: string,
    query?: { after?: string | null; limit?: number | null; peer_id?: string | null }
  ) =>
    [
      ...networkKeys.channelScope(workspaceId, channel),
      "direct" satisfies NetworkSurface,
      "list",
      normalizeText(query?.after),
      normalizeText(query?.peer_id),
      normalizeLimit(query?.limit),
    ] as const,
  directDetail: (workspaceId: string, channel: string, directId: string) =>
    [
      ...networkKeys.channelScope(workspaceId, channel),
      "direct" satisfies NetworkSurface,
      "detail",
      normalizeText(directId),
    ] as const,
  directMessages: (
    workspaceId: string,
    channel: string,
    directId: string,
    query?: ConversationMessagesQuery
  ) =>
    [
      ...networkKeys.channelScope(workspaceId, channel),
      "direct" satisfies NetworkSurface,
      "messages",
      normalizeText(directId),
      ...messagesQuerySegments(query),
    ] as const,

  workRoot: (workspaceId: string) => [...networkKeys.workspace(workspaceId), "work"] as const,
  work: (workspaceId: string, workId: string) =>
    [...networkKeys.workRoot(workspaceId), normalizeText(workId)] as const,

  peersRoot: (workspaceId: string) => [...networkKeys.workspace(workspaceId), "peers"] as const,
  peers: (workspaceId: string, channel?: string | null) =>
    [...networkKeys.peersRoot(workspaceId), normalizeText(channel)] as const,
  peerDetails: (workspaceId: string) => [...networkKeys.peersRoot(workspaceId), "detail"] as const,
  peerDetail: (workspaceId: string, peerId: string) =>
    [...networkKeys.peerDetails(workspaceId), normalizeText(peerId)] as const,
};
