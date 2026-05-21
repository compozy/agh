import type { OperationQuery, OperationRequestBody, OperationResponse } from "@/lib/api-contract";

export type BridgeListFilter = OperationQuery<"listBridges">;
export type BridgesListResponse = OperationResponse<"listBridges", 200>;
export type BridgeSummary = BridgesListResponse["bridges"][number];
export type BridgeHealthMap = NonNullable<BridgesListResponse["bridge_health"]>;
export type BridgeDetailResponse = OperationResponse<"getBridge", 200>;
export type BridgeHealth = BridgeDetailResponse["health"];
export type BridgeRoute = OperationResponse<"listBridgeRoutes", 200>["routes"][number];
export type BridgeTargetsQuery = OperationQuery<"listBridgeTargets">;
export type BridgeTargetsResponse = OperationResponse<"listBridgeTargets", 200>;
export type BridgeTarget = BridgeTargetsResponse["targets"][number];
export type BridgeResolveTargetRequest = OperationRequestBody<"resolveBridgeTarget">;
export type BridgeResolveTargetResponse = OperationResponse<"resolveBridgeTarget", 200 | 404 | 422>;
export type BridgeProvider = OperationResponse<"listBridgeProviders", 200>["providers"][number];
export type CreateBridgeRequest = OperationRequestBody<"createBridge">;
export type CreateBridgeResponse = OperationResponse<"createBridge", 201>;
export type UpdateBridgeRequest = OperationRequestBody<"updateBridge">;
export type UpdateBridgeResponse = OperationResponse<"updateBridge", 200>;
export type BridgeSecretBindingsResponse = OperationResponse<"listBridgeSecretBindings", 200>;
export type BridgeSecretBinding = BridgeSecretBindingsResponse["bindings"][number];
export type PutBridgeSecretBindingRequest = OperationRequestBody<"putBridgeSecretBinding">;
export type PutBridgeSecretBindingResponse = OperationResponse<"putBridgeSecretBinding", 200>;
export type TestBridgeDeliveryRequest = OperationRequestBody<"testBridgeDelivery">;
export type TestBridgeDeliveryResponse = OperationResponse<"testBridgeDelivery", 200>;
export type EnableBridgeResponse = OperationResponse<"enableBridge", 200>;
export type DisableBridgeResponse = OperationResponse<"disableBridge", 200>;
export type RestartBridgeResponse = OperationResponse<"restartBridge", 200>;

export type BridgeScope = BridgeSummary["scope"];
export type BridgeScopeFilter = "all" | BridgeScope;
export type BridgeStatus = BridgeSummary["status"];
export type BridgeRoutingPolicy = BridgeSummary["routing_policy"];
export type BridgeDeliveryMode = NonNullable<TestBridgeDeliveryRequest["target"]["mode"]>;
export type BridgeDmPolicy = NonNullable<CreateBridgeRequest["dm_policy"]>;
export type BridgeProviderConfig = NonNullable<CreateBridgeRequest["provider_config"]>;
export type BridgeProviderSecretSlot = NonNullable<BridgeProvider["secret_slots"]>[number];
export type BridgeProviderConfigSchemaHint = NonNullable<BridgeProvider["config_schema"]>;

export interface BridgeDeliveryDefaults {
  group_id?: string;
  mode?: BridgeDeliveryMode;
  peer_id?: string;
  thread_id?: string;
}

export interface BridgeCreateDraft {
  deliveryDefaults: BridgeDeliveryDefaults;
  dmPolicy: BridgeDmPolicy | "";
  displayName: string;
  providerConfigText: string;
  routingPolicy: BridgeRoutingPolicy;
  scope: BridgeScope;
  selectedProviderKey: string;
}

export interface BridgeTestDeliveryDraft {
  message: string;
  target: BridgeDeliveryDefaults;
}

export interface BridgeUpdateDraft {
  deliveryDefaults: BridgeDeliveryDefaults;
  dmPolicy: BridgeDmPolicy | "";
  displayName: string;
  providerConfigText: string;
  routingPolicy: BridgeRoutingPolicy;
}

export interface BridgeHealthStreamSnapshot {
  bridge_health: BridgeHealthMap;
  generated_at: string;
}
