import { useMutation, useQueryClient } from "@tanstack/react-query";

import { sessionKeys } from "@/systems/session";
import { createNetworkChannel, sendNetworkMessage } from "@/systems/network/adapters/network-api";
import { networkKeys } from "@/systems/network/lib/query-keys";
import type { CreateNetworkChannelRequest, NetworkSendRequest } from "@/systems/network/types";

function invalidateNetworkQueries(queryClient: ReturnType<typeof useQueryClient>) {
  return Promise.all([
    queryClient.invalidateQueries({ queryKey: networkKeys.all }),
    queryClient.invalidateQueries({ queryKey: sessionKeys.lists() }),
  ]);
}

export function useCreateNetworkChannel() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateNetworkChannelRequest) => createNetworkChannel(data),
    onSettled: () => invalidateNetworkQueries(queryClient),
  });
}

export function useSendNetworkMessage() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: NetworkSendRequest) => sendNetworkMessage(data),
    onSettled: () => invalidateNetworkQueries(queryClient),
  });
}
