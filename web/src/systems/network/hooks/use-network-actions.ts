import { useMutation, useQueryClient } from "@tanstack/react-query";

import { sessionKeys } from "@/systems/session";

import { createNetworkChannel } from "../adapters/network-api";
import { networkKeys } from "../lib/query-keys";
import type { CreateNetworkChannelRequest } from "../types";

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
