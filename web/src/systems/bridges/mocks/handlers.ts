import { http, HttpResponse, type HttpHandler } from "msw";

import {
  bridgeDetailFixture,
  bridgeProvidersFixture,
  bridgeRoutesFixture,
  bridgeSecretBindingsFixture,
  bridgesListFixture,
  createBridgeFixture,
  testBridgeDeliveryFixture,
  updateBridgeFixture,
} from "./fixtures";

export const handlers: HttpHandler[] = [
  http.get("/api/bridges", () => HttpResponse.json(bridgesListFixture)),
  http.get("/api/bridges/providers", () =>
    HttpResponse.json({ providers: bridgeProvidersFixture })
  ),
  http.get("/api/bridges/:id", ({ params }) => {
    const id = String(params.id);

    if (id !== bridgeDetailFixture.bridge.id) {
      return HttpResponse.json({ error: `Bridge not found: ${id}` }, { status: 404 });
    }

    return HttpResponse.json(bridgeDetailFixture);
  }),
  http.get("/api/bridges/:id/routes", ({ params }) => {
    const id = String(params.id);

    if (id !== bridgeDetailFixture.bridge.id) {
      return HttpResponse.json({ error: `Bridge not found: ${id}` }, { status: 404 });
    }

    return HttpResponse.json({ routes: bridgeRoutesFixture });
  }),
  http.get("/api/bridges/:id/secret-bindings", ({ params }) => {
    const id = String(params.id);

    if (id !== bridgeDetailFixture.bridge.id) {
      return HttpResponse.json({ error: `Bridge not found: ${id}` }, { status: 404 });
    }

    return HttpResponse.json({ bindings: bridgeSecretBindingsFixture });
  }),
  http.post("/api/bridges", async ({ request }) => {
    const body = (await request.json()) as {
      display_name?: string;
      scope?: "global" | "workspace";
      workspace_id?: string;
    };

    return HttpResponse.json(
      {
        ...createBridgeFixture,
        bridge: {
          ...createBridgeFixture.bridge,
          display_name: body.display_name?.trim() || createBridgeFixture.bridge.display_name,
          scope: body.scope ?? createBridgeFixture.bridge.scope,
          workspace_id: body.workspace_id ?? createBridgeFixture.bridge.workspace_id,
        },
      },
      { status: 201 }
    );
  }),
  http.patch("/api/bridges/:id", async ({ params, request }) => {
    const id = String(params.id);

    if (id !== bridgeDetailFixture.bridge.id) {
      return HttpResponse.json({ error: `Bridge not found: ${id}` }, { status: 404 });
    }

    const body = (await request.json()) as {
      display_name?: string | null;
      dm_policy?: "open" | "allowlist" | "pairing";
      provider_config?: Record<string, unknown> | null;
      routing_policy?: {
        include_group: boolean;
        include_peer: boolean;
        include_thread: boolean;
      } | null;
    };

    return HttpResponse.json({
      ...updateBridgeFixture,
      bridge: {
        ...updateBridgeFixture.bridge,
        display_name: body.display_name ?? updateBridgeFixture.bridge.display_name,
        dm_policy: body.dm_policy ?? updateBridgeFixture.bridge.dm_policy,
        provider_config: body.provider_config ?? updateBridgeFixture.bridge.provider_config,
        routing_policy: body.routing_policy ?? updateBridgeFixture.bridge.routing_policy,
      },
    });
  }),
  http.put("/api/bridges/:id/secret-bindings/:binding_name", ({ params }) => {
    const id = String(params.id);
    const bindingName = String(params.binding_name);

    if (id !== bridgeDetailFixture.bridge.id) {
      return HttpResponse.json({ error: `Bridge not found: ${id}` }, { status: 404 });
    }

    return HttpResponse.json({
      binding: {
        ...bridgeSecretBindingsFixture[0],
        binding_name: bindingName,
      },
    });
  }),
  http.delete("/api/bridges/:id/secret-bindings/:binding_name", ({ params }) => {
    const id = String(params.id);

    if (id !== bridgeDetailFixture.bridge.id) {
      return HttpResponse.json({ error: `Bridge not found: ${id}` }, { status: 404 });
    }

    return new HttpResponse(null, { status: 204 });
  }),
  http.post("/api/bridges/:id/enable", ({ params }) => {
    const id = String(params.id);

    if (id !== bridgeDetailFixture.bridge.id) {
      return HttpResponse.json({ error: `Bridge not found: ${id}` }, { status: 404 });
    }

    return HttpResponse.json({
      ...bridgeDetailFixture,
      bridge: {
        ...bridgeDetailFixture.bridge,
        enabled: true,
        status: "ready",
      },
    });
  }),
  http.post("/api/bridges/:id/disable", ({ params }) => {
    const id = String(params.id);

    if (id !== bridgeDetailFixture.bridge.id) {
      return HttpResponse.json({ error: `Bridge not found: ${id}` }, { status: 404 });
    }

    return HttpResponse.json({
      ...bridgeDetailFixture,
      bridge: {
        ...bridgeDetailFixture.bridge,
        enabled: false,
        status: "disabled",
      },
    });
  }),
  http.post("/api/bridges/:id/restart", ({ params }) => {
    const id = String(params.id);

    if (id !== bridgeDetailFixture.bridge.id) {
      return HttpResponse.json({ error: `Bridge not found: ${id}` }, { status: 404 });
    }

    return HttpResponse.json(bridgeDetailFixture);
  }),
  http.post("/api/bridges/:id/test-delivery", ({ params }) => {
    const id = String(params.id);

    if (id !== bridgeDetailFixture.bridge.id) {
      return HttpResponse.json({ error: `Bridge not found: ${id}` }, { status: 404 });
    }

    return HttpResponse.json(testBridgeDeliveryFixture);
  }),
];
