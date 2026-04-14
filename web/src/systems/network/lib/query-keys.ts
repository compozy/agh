function normalizeText(value?: string | null) {
  return value ?? "";
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
  channelMessages: (channel: string, limit: number) =>
    [...networkKeys.channelMessagesRoot(), normalizeText(channel), limit] as const,
  peersRoot: () => [...networkKeys.all, "peers"] as const,
  peers: (channel?: string | null) => [...networkKeys.peersRoot(), normalizeText(channel)] as const,
  peerDetails: () => [...networkKeys.peersRoot(), "detail"] as const,
  peerDetail: (peerId: string) => [...networkKeys.peerDetails(), normalizeText(peerId)] as const,
};
