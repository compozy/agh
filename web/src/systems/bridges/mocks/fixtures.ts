import type {
  BridgeDetailResponse,
  BridgeProvider,
  BridgeResolveTargetResponse,
  BridgeRoute,
  BridgeSecretBinding,
  BridgeTargetsResponse,
  BridgesListResponse,
  CreateBridgeResponse,
  TestBridgeDeliveryResponse,
  UpdateBridgeResponse,
} from "../types";
import { storyWorkspaceIds } from "@/storybook/fintech-scenario";

export const bridgeProvidersFixture: BridgeProvider[] = [
  {
    config_schema: {
      schema: "provider-config",
      version: "2026-04-17",
    },
    description: "Provider-specific runtime settings",
    display_name: "Slack",
    enabled: true,
    extension_name: "ext-slack",
    health: "healthy",
    platform: "slack",
    secret_slots: [
      {
        description: "Bot token",
        name: "bot_token",
        required: true,
      },
      {
        description: "Signing secret",
        name: "signing_secret",
        required: true,
      },
    ],
    state: "active",
  },
];

export const bridgesListFixture: BridgesListResponse = {
  bridge_health: {
    brg_launch_room: {
      auth_failures_total: 0,
      bridge_instance_id: "brg_launch_room",
      delivery_backlog: 2,
      delivery_dropped_total: 0,
      delivery_failures_total: 0,
      route_count: 3,
      status: "ready",
    },
  },
  bridges: [
    {
      created_at: "2026-04-17T14:00:00Z",
      dm_policy: "open",
      display_name: "Launch room dispatch",
      enabled: true,
      extension_name: "ext-slack",
      id: "brg_launch_room",
      notification_suppress: false,
      platform: "slack",
      provider_config: {
        workspace: "northstar-launch",
      },
      routing_policy: {
        include_group: true,
        include_peer: true,
        include_thread: true,
      },
      scope: "workspace",
      source: "dynamic",
      status: "ready",
      updated_at: "2026-04-17T18:00:00Z",
      workspace_id: storyWorkspaceIds.hq,
      delivery_defaults: {
        mode: "reply",
        group_id: "slack_launch_room",
      },
    },
  ],
};

export const bridgeRoutesFixture: BridgeRoute[] = [
  {
    agent_name: "support-lead-agent",
    bridge_instance_id: "brg_launch_room",
    created_at: "2026-04-17T14:05:00Z",
    group_id: "slack_launch_room",
    last_activity_at: "2026-04-17T18:08:00Z",
    peer_id: "merchant_nsp_2044",
    routing_key_hash: "launch-room-merchant-2044",
    scope: "workspace",
    session_id: "sess_support_swarm",
    thread_id: "thread_launch_2044",
    updated_at: "2026-04-17T18:08:00Z",
    workspace_id: storyWorkspaceIds.support,
  },
  {
    agent_name: "marketing-lead-agent",
    bridge_instance_id: "brg_launch_room",
    created_at: "2026-04-17T15:00:00Z",
    group_id: "slack_launch_room",
    last_activity_at: "2026-04-17T17:58:00Z",
    peer_id: "campaign_meta_launch",
    routing_key_hash: "launch-room-campaign-meta",
    scope: "workspace",
    session_id: "sess_marketing_launch_copy",
    thread_id: "thread_launch_meta",
    updated_at: "2026-04-17T17:58:00Z",
    workspace_id: storyWorkspaceIds.growth,
  },
];

export const bridgeTargetsFixture: BridgeTargetsResponse = {
  bridge_id: "brg_launch_room",
  cache_stale: false,
  generated_at: "2026-04-17T18:10:00Z",
  last_successful_refresh_at: "2026-04-17T18:09:00Z",
  targets: [
    {
      bridge_id: "brg_launch_room",
      canonical_route: "slack:channel:slack_launch_room",
      capabilities: ["direct-send", "reply"],
      display_name: "Launch room",
      last_seen_at: "2026-04-17T18:09:00Z",
      normalized: "launch room",
      qualifier: "slack",
      target_type: "channel",
      updated_at: "2026-04-17T18:09:00Z",
    },
    {
      bridge_id: "brg_launch_room",
      canonical_route: "slack:thread:thread_launch_2044",
      capabilities: ["reply"],
      display_name: "Merchant launch thread",
      last_seen_at: "2026-04-17T18:08:00Z",
      normalized: "merchant launch thread",
      qualifier: "launch-room",
      target_type: "thread",
      updated_at: "2026-04-17T18:08:00Z",
    },
  ],
  total: 2,
};

export const bridgeResolveTargetFixture: BridgeResolveTargetResponse = {
  result: {
    ambiguous: false,
    match: bridgeTargetsFixture.targets[0],
    step: 2,
  },
};

export const bridgeSecretBindingsFixture: BridgeSecretBinding[] = [
  {
    binding_name: "bot_token",
    bridge_instance_id: "brg_launch_room",
    created_at: "2026-04-17T14:02:00Z",
    kind: "env",
    updated_at: "2026-04-17T14:02:00Z",
    secret_ref: "vault:bridges/brg_launch_room/bot_token",
  },
  {
    binding_name: "signing_secret",
    bridge_instance_id: "brg_launch_room",
    created_at: "2026-04-17T14:02:00Z",
    kind: "env",
    updated_at: "2026-04-17T14:02:00Z",
    secret_ref: "vault:bridges/brg_launch_room/signing_secret",
  },
];

export const bridgeDetailFixture: BridgeDetailResponse = {
  bridge: bridgesListFixture.bridges[0],
  health: bridgesListFixture.bridge_health?.brg_launch_room ?? {
    auth_failures_total: 0,
    bridge_instance_id: "brg_launch_room",
    delivery_backlog: 0,
    delivery_dropped_total: 0,
    delivery_failures_total: 0,
    route_count: 0,
    status: "ready",
  },
};

export const createBridgeFixture: CreateBridgeResponse = { ...bridgeDetailFixture };
export const updateBridgeFixture: UpdateBridgeResponse = { ...bridgeDetailFixture };

export const testBridgeDeliveryFixture: TestBridgeDeliveryResponse = {
  delivery_target: {
    bridge_instance_id: "brg_launch_room",
    group_id: "slack_launch_room",
    mode: "reply",
    peer_id: "merchant_nsp_2044",
    thread_id: "thread_launch_2044",
  },
  message: "Delivery target resolved for the selected launch-room Slack bridge.",
  status: "ready",
};
