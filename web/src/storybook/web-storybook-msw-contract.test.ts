import type { HttpHandler } from "msw";
import { describe, expect, it } from "vitest";

import { handlers as agentHandlers } from "@/systems/agent/mocks";
import { handlers as automationHandlers } from "@/systems/automation/mocks";
import { handlers as bridgeHandlers } from "@/systems/bridges/mocks";
import { handlers as daemonHandlers } from "@/systems/daemon/mocks";
import { handlers as knowledgeHandlers } from "@/systems/knowledge/mocks";
import { handlers as networkHandlers } from "@/systems/network/mocks";
import { handlers as sessionHandlers } from "@/systems/session/mocks";
import { handlers as settingsHandlers } from "@/systems/settings/mocks";
import { handlers as skillHandlers } from "@/systems/skill/mocks";
import { handlers as workspaceHandlers } from "@/systems/workspace/mocks";

const { storybookSystemHandlerGroups, storybookSystemHandlers } =
  await import("../../.storybook/preview");
const { flattenStorybookHandlerGroups } = await import("./msw");

function handlerSignature(handler: HttpHandler) {
  const method = String(handler.info.method);
  const path = String(handler.info.path);
  return `${method} ${path}`;
}

describe("web Storybook MSW contract", () => {
  it("composes grouped default Storybook handlers in preview from every system mock barrel", () => {
    expect(storybookSystemHandlerGroups).toEqual({
      agent: agentHandlers,
      automation: automationHandlers,
      bridges: bridgeHandlers,
      daemon: daemonHandlers,
      knowledge: knowledgeHandlers,
      network: networkHandlers,
      session: sessionHandlers,
      settings: settingsHandlers,
      skill: skillHandlers,
      workspace: workspaceHandlers,
    });
    expect(storybookSystemHandlers).toEqual(
      flattenStorybookHandlerGroups(storybookSystemHandlerGroups)
    );
    expect(agentHandlers.length).toBeGreaterThan(0);
    expect(automationHandlers.length).toBeGreaterThan(0);
    expect(bridgeHandlers.length).toBeGreaterThan(0);
    expect(daemonHandlers.length).toBeGreaterThan(0);
    expect(knowledgeHandlers.length).toBeGreaterThan(0);
    expect(networkHandlers.length).toBeGreaterThan(0);
    expect(sessionHandlers.length).toBeGreaterThan(0);
    expect(settingsHandlers.length).toBeGreaterThan(0);
    expect(skillHandlers.length).toBeGreaterThan(0);
    expect(workspaceHandlers.length).toBeGreaterThan(0);
  });

  it("does not register duplicate method/path handler pairs across the combined system set", () => {
    const signatures = storybookSystemHandlers.map(handlerSignature);
    const uniqueSignatures = new Set(signatures);

    expect(uniqueSignatures.size).toBe(signatures.length);
  });
});
