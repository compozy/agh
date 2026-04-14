import { useMutation, useQueryClient } from "@tanstack/react-query";

import { createBridge, testBridgeDelivery } from "../adapters/bridges-api";
import { bridgeKeys } from "../lib/query-keys";
import type { CreateBridgeRequest, TestBridgeDeliveryRequest } from "../types";

interface BridgeMutationIdParams {
  id: string;
}

interface TestBridgeDeliveryParams extends BridgeMutationIdParams {
  data: TestBridgeDeliveryRequest;
}

function invalidateBridgeQueries(queryClient: ReturnType<typeof useQueryClient>, id?: string) {
  return Promise.all([
    queryClient.invalidateQueries({ queryKey: bridgeKeys.all }),
    ...(id
      ? [
          queryClient.invalidateQueries({ queryKey: bridgeKeys.detail(id) }),
          queryClient.invalidateQueries({ queryKey: bridgeKeys.routes(id) }),
        ]
      : []),
  ]);
}

export function useCreateBridge() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateBridgeRequest) => createBridge(data),
    onSettled: () => invalidateBridgeQueries(queryClient),
  });
}

export function useTestBridgeDelivery() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: TestBridgeDeliveryParams) => testBridgeDelivery(id, data),
    onSettled: (_result, _error, { id }) => invalidateBridgeQueries(queryClient, id),
  });
}
