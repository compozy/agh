import type {
  BridgeDetailResponse,
  BridgeProvider,
  BridgeRoute,
  BridgeSecretBinding,
  BridgesListResponse,
  CreateBridgeResponse,
  TestBridgeDeliveryResponse,
  UpdateBridgeResponse,
} from "../types";

export const bridgeProvidersFixture: BridgeProvider[] = [
  {
    config_schema: {
      schema: "provider-config",
      version: "2026-04-15",
    },
    description: "Provider-specific runtime settings",
    display_name: "Telegram",
    enabled: true,
    extension_name: "ext-telegram",
    health: "healthy",
    platform: "telegram",
    secret_slots: [
      {
        description: "Bot token",
        name: "bot_token",
        required: true,
      },
    ],
    state: "active",
  },
];

export const bridgesListFixture: BridgesListResponse = {
  bridge_health: {
    brg_support: {
      auth_failures_total: 0,
      bridge_instance_id: "brg_support",
      delivery_backlog: 1,
      delivery_dropped_total: 0,
      delivery_failures_total: 0,
      route_count: 2,
      status: "ready",
    },
  },
  bridges: [
    {
      created_at: "2026-04-13T12:00:00Z",
      dm_policy: "open",
      display_name: "Support",
      enabled: true,
      extension_name: "ext-telegram",
      id: "brg_support",
      platform: "telegram",
      provider_config: {
        mode: "bot",
      },
      routing_policy: {
        include_group: true,
        include_peer: true,
        include_thread: true,
      },
      scope: "workspace",
      source: "dynamic",
      status: "ready",
      updated_at: "2026-04-13T12:30:00Z",
      workspace_id: "ws_storybook",
      delivery_defaults: {
        mode: "reply",
        group_id: "grp_support",
      },
    },
  ],
};

export const bridgeRoutesFixture: BridgeRoute[] = [
  {
    agent_name: "support-agent",
    bridge_instance_id: "brg_support",
    created_at: "2026-04-13T12:00:00Z",
    group_id: "grp_support",
    last_activity_at: "2026-04-13T12:15:00Z",
    peer_id: "peer_customer_123",
    routing_key_hash: "abc123",
    scope: "workspace",
    session_id: "sess_support",
    thread_id: "thread_456",
    updated_at: "2026-04-13T12:15:00Z",
    workspace_id: "ws_storybook",
  },
];

export const bridgeSecretBindingsFixture: BridgeSecretBinding[] = [
  {
    binding_name: "bot_token",
    bridge_instance_id: "brg_support",
    created_at: "2026-04-13T12:05:00Z",
    kind: "env",
    updated_at: "2026-04-13T12:05:00Z",
    vault_ref: "env:AGH_BRIDGE_BOT_TOKEN",
  },
];

export const bridgeDetailFixture: BridgeDetailResponse = {
  bridge: bridgesListFixture.bridges[0],
  health: bridgesListFixture.bridge_health?.brg_support ?? {
    auth_failures_total: 0,
    bridge_instance_id: "brg_support",
    delivery_backlog: 0,
    delivery_dropped_total: 0,
    delivery_failures_total: 0,
    route_count: 0,
    status: "ready",
  },
};

export const createBridgeFixture: CreateBridgeResponse = bridgeDetailFixture;
export const updateBridgeFixture: UpdateBridgeResponse = bridgeDetailFixture;

export const testBridgeDeliveryFixture: TestBridgeDeliveryResponse = {
  delivery_target: {
    bridge_instance_id: "brg_support",
    group_id: "grp_support",
    mode: "reply",
    peer_id: "peer_customer_123",
    thread_id: "thread_456",
  },
  message: "Delivery target resolved for the selected bridge.",
  status: "ready",
};
