import { describe, expect, it } from "vitest";

import type { BridgeCreateDraft } from "@/systems/bridges/types";

import {
  bridgeSecretBindingVaultRef,
  buildBridgeCreateRequest,
  buildBridgeSecretBindingRequest,
  buildBridgeUpdateRequest,
  createBridgeCreateDraft,
  createBridgeUpdateDraft,
  parseBridgeDmPolicy,
  parseBridgeProviderConfig,
} from "../bridge-drafts";

function makeDraft(overrides: Partial<BridgeCreateDraft> = {}): BridgeCreateDraft {
  return {
    deliveryDefaults: {
      mode: "reply",
      peer_id: "peer_123",
    },
    dmPolicy: "",
    displayName: "Telegram",
    providerConfigText: "",
    routingPolicy: {
      include_group: true,
      include_peer: true,
      include_thread: true,
    },
    scope: "workspace",
    selectedProviderKey: "ext-telegram::telegram",
    ...overrides,
  };
}

describe("createBridgeCreateDraft", () => {
  it("seeds the expanded bridge draft with provider config and dm policy defaults", () => {
    const draft = createBridgeCreateDraft(
      [
        {
          display_name: "Telegram",
          enabled: true,
          extension_name: "ext-telegram",
          health: "healthy",
          platform: "telegram",
          state: "active",
        },
      ],
      "ws_test"
    );

    expect(draft).toMatchObject({
      deliveryDefaults: {},
      dmPolicy: "",
      displayName: "Telegram",
      providerConfigText: "",
      scope: "workspace",
    });
  });
});

describe("parseBridgeProviderConfig", () => {
  it("accepts only JSON objects for provider config", () => {
    expect(parseBridgeProviderConfig("")).toEqual({});
    expect(parseBridgeProviderConfig('{"mode":"bot"}')).toEqual({
      value: { mode: "bot" },
    });
    expect(parseBridgeProviderConfig('["bot"]')).toEqual({
      error: "Provider configuration must be a JSON object.",
    });
  });
});

describe("buildBridgeCreateRequest", () => {
  it("preserves provider_config separately from delivery_defaults", () => {
    const result = buildBridgeCreateRequest(
      makeDraft({
        dmPolicy: "pairing",
        providerConfigText: '{\n  "mode": "bot",\n  "webhook_url": "https://example.test/hook"\n}',
      }),
      {
        extension_name: "ext-telegram",
        platform: "telegram",
      },
      "ws_test"
    );

    expect(result).toEqual({
      data: {
        delivery_defaults: {
          mode: "reply",
          peer_id: "peer_123",
        },
        display_name: "Telegram",
        dm_policy: "pairing",
        enabled: true,
        extension_name: "ext-telegram",
        platform: "telegram",
        provider_config: {
          mode: "bot",
          webhook_url: "https://example.test/hook",
        },
        routing_policy: {
          include_group: true,
          include_peer: true,
          include_thread: true,
        },
        scope: "workspace",
        status: "starting",
        workspace_id: "ws_test",
      },
      ok: true,
    });
  });

  it("serializes only supported dm_policy values into the payload", () => {
    const invalidDraft = makeDraft({
      dmPolicy: "unsupported" as BridgeCreateDraft["dmPolicy"],
    });

    const result = buildBridgeCreateRequest(
      invalidDraft,
      {
        extension_name: "ext-telegram",
        platform: "telegram",
      },
      "ws_test"
    );

    expect(result).toMatchObject({
      data: expect.objectContaining({
        dm_policy: undefined,
      }),
      ok: true,
    });
    expect(parseBridgeDmPolicy("open")).toBe("open");
    expect(parseBridgeDmPolicy("unsupported")).toBeUndefined();
  });
});

describe("createBridgeUpdateDraft", () => {
  it("hydrates mutable fields from an existing bridge", () => {
    const draft = createBridgeUpdateDraft({
      delivery_defaults: {
        mode: "reply",
        peer_id: "peer_123",
      },
      display_name: "Support",
      dm_policy: "allowlist",
      provider_config: {
        mode: "bot",
      },
      routing_policy: {
        include_group: true,
        include_peer: false,
        include_thread: true,
      },
    });

    expect(draft).toEqual({
      deliveryDefaults: {
        mode: "reply",
        peer_id: "peer_123",
      },
      displayName: "Support",
      dmPolicy: "allowlist",
      providerConfigText: '{\n  "mode": "bot"\n}',
      routingPolicy: {
        include_group: true,
        include_peer: false,
        include_thread: true,
      },
    });
  });
});

describe("buildBridgeUpdateRequest", () => {
  it("preserves nullable fields for clearing provider config and delivery defaults", () => {
    const result = buildBridgeUpdateRequest({
      deliveryDefaults: {},
      displayName: "Support Ops",
      dmPolicy: "",
      providerConfigText: "",
      routingPolicy: {
        include_group: true,
        include_peer: false,
        include_thread: true,
      },
    });

    expect(result).toEqual({
      data: {
        delivery_defaults: null,
        display_name: "Support Ops",
        dm_policy: undefined,
        provider_config: null,
        routing_policy: {
          include_group: true,
          include_peer: false,
          include_thread: true,
        },
      },
      ok: true,
    });
  });
});

describe("bridge secret binding helpers", () => {
  it("normalizes vault refs and builds vault-backed secret binding payloads", () => {
    expect(
      bridgeSecretBindingVaultRef({
        secret_ref: "vault:bridges/brg_support/bot_token",
      } as never)
    ).toBe("vault:bridges/brg_support/bot_token");

    expect(
      buildBridgeSecretBindingRequest("brg_support", "bot_token", "telegram-token", "bot_token")
    ).toEqual({
      data: {
        kind: "bot_token",
        secret_ref: "vault:bridges/brg_support/bot_token",
        secret_value: "telegram-token",
      },
      ok: true,
    });
  });

  it("rejects invalid vault binding inputs", () => {
    expect(
      buildBridgeSecretBindingRequest("not valid", "bot_token", "telegram-token", "bot_token")
    ).toEqual({
      error: "Secret binding must use a bridge vault reference.",
      ok: false,
    });
    expect(buildBridgeSecretBindingRequest("brg_support", "bot_token", " ", "bot_token")).toEqual({
      error: "Secret binding value is required.",
      ok: false,
    });
  });
});
