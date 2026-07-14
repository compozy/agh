import { useMutation, useQueryClient } from "@tanstack/react-query";

import { registerBridgeWebhook, sendBridgeTest, verifyBridge } from "../adapters/bridge-setup-api";
import { bridgeKeys } from "../lib/query-keys";
import type { SendBridgeTestRequest } from "../types";

interface BridgeSetupMutationParams {
  id: string;
}

interface SendBridgeTestParams extends BridgeSetupMutationParams {
  data: SendBridgeTestRequest;
}

function invalidateBridgeDeliveryQueries(
  queryClient: ReturnType<typeof useQueryClient>,
  id: string
) {
  return Promise.all([
    queryClient.invalidateQueries({ queryKey: bridgeKeys.lists() }),
    queryClient.invalidateQueries({ queryKey: bridgeKeys.detail(id), exact: true }),
    queryClient.invalidateQueries({ queryKey: bridgeKeys.routes(id), exact: true }),
  ]);
}

export function useVerifyBridge() {
  return useMutation({
    mutationFn: ({ id }: BridgeSetupMutationParams) => verifyBridge(id),
  });
}

export function useRegisterBridgeWebhook() {
  return useMutation({
    mutationFn: ({ id }: BridgeSetupMutationParams) => registerBridgeWebhook(id),
  });
}

export function useSendBridgeTest() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ data, id }: SendBridgeTestParams) => sendBridgeTest(id, data),
    onSettled: (_result, _error, { id }) => invalidateBridgeDeliveryQueries(queryClient, id),
  });
}
