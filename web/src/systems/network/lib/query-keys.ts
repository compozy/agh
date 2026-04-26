function normalizeText(value?: string | null) {
  return value ?? "";
}

function normalizeLimit(value?: number | null) {
  return value ?? 0;
}

function normalizeBool(value?: boolean | null) {
  return value === true ? 1 : 0;
}

export const networkKeys = {
  all: ["network"] as const,
  status: () => [...networkKeys.all, "status"] as const,
  channelsRoot: () => [...networkKeys.all, "channels"] as const,
  channels: () => [...networkKeys.channelsRoot(), "list"] as const,
  channelDetails: () => [...networkKeys.channelsRoot(), "detail"] as const,
  channelDetail: (channel: string) =>
    [...networkKeys.channelDetails(), normalizeText(channel)] as const,
  channelMessagesRoot: () => [...networkKeys.channelsRoot(), "messages"] as const,
  channelMessages: (
    channel: string,
    query?: {
      after?: string | null;
      before?: string | null;
      include_presence?: boolean | null;
      limit?: number | null;
    }
  ) =>
    [
      ...networkKeys.channelMessagesRoot(),
      normalizeText(channel),
      normalizeText(query?.before),
      normalizeText(query?.after),
      normalizeBool(query?.include_presence),
      normalizeLimit(query?.limit),
    ] as const,
  peersRoot: () => [...networkKeys.all, "peers"] as const,
  peers: (channel?: string | null) => [...networkKeys.peersRoot(), normalizeText(channel)] as const,
  peerDetails: () => [...networkKeys.peersRoot(), "detail"] as const,
  peerDetail: (peerId: string) => [...networkKeys.peerDetails(), normalizeText(peerId)] as const,
  peerMessagesRoot: () => [...networkKeys.peersRoot(), "messages"] as const,
  peerMessages: (
    peerId: string,
    query?: {
      after?: string | null;
      before?: string | null;
      include_presence?: boolean | null;
      limit?: number | null;
    }
  ) =>
    [
      ...networkKeys.peerMessagesRoot(),
      normalizeText(peerId),
      normalizeText(query?.before),
      normalizeText(query?.after),
      normalizeBool(query?.include_presence),
      normalizeLimit(query?.limit),
    ] as const,
};
