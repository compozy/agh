import { describe, expect, it } from "vitest";

import type { BridgeCreateDraft } from "@/systems/bridges/types";

import {
  buildBridgeCreateRequest,
  createBridgeCreateDraft,
  parseBridgeDmPolicy,
  parseBridgeProviderConfig,
} from "./bridge-drafts";

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
