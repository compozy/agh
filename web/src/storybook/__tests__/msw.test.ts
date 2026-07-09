import { HttpResponse } from "msw";
import { aghApiMock } from "@/storybook/openapi-msw";
import { describe, expect, it } from "vitest";

import {
  composeStorybookHandlerGroup,
  storybookMswParameters,
  storybookSystemHandlerGroups,
  storybookSystemHandlers,
} from "../msw";

function handlerSignature(handler: { info: { method: unknown; path: unknown } }) {
  const method = String(handler.info.method);
  const path = String(handler.info.path).replace(/^\*/, "");
  return `${method} ${path}`;
}

describe("storybook msw helpers", () => {
  it("creates grouped story overrides without requiring untouched domains to be repeated", () => {
    const bridgesOverride = [
      aghApiMock.get("/api/bridges", () => HttpResponse.json({ bridges: [], bridge_health: {} })),
    ];
    const parameters = storybookMswParameters({ bridges: bridgesOverride });
    const mergedGroups = {
      ...storybookSystemHandlerGroups,
      ...parameters.msw.handlers,
    };

    expect(parameters).toEqual({
      msw: {
        handlers: {
          bridges: composeStorybookHandlerGroup("bridges", bridgesOverride),
        },
      },
    });
    expect(mergedGroups.bridges).toEqual(composeStorybookHandlerGroup("bridges", bridgesOverride));
    expect(mergedGroups.network).toBe(storybookSystemHandlerGroups.network);
    expect(mergedGroups.settings).toBe(storybookSystemHandlerGroups.settings);
    expect(mergedGroups.tasks).toBe(storybookSystemHandlerGroups.tasks);
  });

  it("preserves untouched handlers inside an overridden group while replacing matching endpoints", () => {
    const bridgesOverride = [
      aghApiMock.get("/api/bridges", () => HttpResponse.json({ bridges: [], bridge_health: {} })),
    ];
    const composedGroup = composeStorybookHandlerGroup("bridges", bridgesOverride);
    const signatures = composedGroup.map(handlerSignature);

    expect(composedGroup[0]).toBe(bridgesOverride[0]);
    expect(signatures).toContain("GET /api/bridges/providers");
    expect(signatures.filter(signature => signature === "GET /api/bridges")).toHaveLength(1);
  });

  it("does not register duplicate local API method/path pairs after normalizing path params", () => {
    const signatures = storybookSystemHandlers
      .map(handlerSignature)
      .filter(signature => signature.includes(" /api/"))
      .map(signature => signature.replace(/:[^/]+/g, "{param}").replace(/\{[^/]+\}/g, "{param}"));

    expect(signatures).toHaveLength(new Set(signatures).size);
  });

  it("includes the route-owning vault handler group", () => {
    expect(storybookSystemHandlerGroups.vault.length).toBeGreaterThan(0);
  });

  it("includes the runtime handler group used by the shared app shell", () => {
    expect(storybookSystemHandlerGroups.runtime.length).toBeGreaterThan(0);
  });
});
