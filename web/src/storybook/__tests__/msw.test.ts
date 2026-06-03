import { http, HttpResponse } from "msw";
import { describe, expect, it } from "vitest";

import {
  composeStorybookHandlerGroup,
  flattenStorybookHandlerGroups,
  storybookMswParameters,
  storybookSystemHandlerGroups,
} from "../msw";

describe("storybook msw helpers", () => {
  it("creates grouped story overrides without requiring untouched domains to be repeated", () => {
    const bridgesOverride = [
      http.get("/api/bridges", () => HttpResponse.json({ bridges: [], bridge_health: {} })),
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
      http.get("/api/bridges", () => HttpResponse.json({ bridges: [], bridge_health: {} })),
    ];
    const composedGroup = composeStorybookHandlerGroup("bridges", bridgesOverride);
    const signatures = composedGroup.map(
      handler => `${String(handler.info.method)} ${String(handler.info.path)}`
    );

    expect(composedGroup[0]).toBe(bridgesOverride[0]);
    expect(signatures).toContain("GET /api/bridges/providers");
    expect(signatures.filter(signature => signature === "GET /api/bridges")).toHaveLength(1);
  });

  it("flattens grouped handlers in insertion order for duplicate-signature checks", () => {
    const flattened = flattenStorybookHandlerGroups(storybookSystemHandlerGroups);

    expect(flattened).toEqual([
      ...storybookSystemHandlerGroups.agent,
      ...storybookSystemHandlerGroups.automation,
      ...storybookSystemHandlerGroups.bridges,
      ...storybookSystemHandlerGroups.daemon,
      ...storybookSystemHandlerGroups.knowledge,
      ...storybookSystemHandlerGroups.network,
      ...storybookSystemHandlerGroups.onboarding,
      ...storybookSystemHandlerGroups.session,
      ...storybookSystemHandlerGroups.settings,
      ...storybookSystemHandlerGroups.skill,
      ...storybookSystemHandlerGroups.scheduler,
      ...storybookSystemHandlerGroups.tasks,
      ...storybookSystemHandlerGroups.workspace,
    ]);
  });
});
