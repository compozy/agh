import type { HttpHandler } from "msw";

import { handlers as agentHandlers } from "@/systems/agent/mocks";
import { handlers as automationHandlers } from "@/systems/automation/mocks";
import { handlers as bridgeHandlers } from "@/systems/bridges/mocks";
import { handlers as daemonHandlers } from "@/systems/status/mocks";
import { handlers as knowledgeHandlers } from "@/systems/knowledge/mocks";
import { handlers as networkHandlers } from "@/systems/network/mocks";
import { handlers as onboardingHandlers } from "@/systems/onboarding/mocks";
import { handlers as sessionHandlers } from "@/systems/session/mocks";
import { handlers as settingsHandlers } from "@/systems/settings/mocks";
import { handlers as skillHandlers } from "@/systems/skill/mocks";
import { handlers as schedulerHandlers } from "@/systems/scheduler/mocks";
import { handlers as tasksHandlers } from "@/systems/tasks/mocks";
import { handlers as workspaceHandlers } from "@/systems/workspace/mocks";

export type StorybookHandlerGroupName =
  | "agent"
  | "automation"
  | "bridges"
  | "daemon"
  | "knowledge"
  | "network"
  | "onboarding"
  | "session"
  | "settings"
  | "skill"
  | "scheduler"
  | "tasks"
  | "workspace";

export type StorybookHandlerGroups = Record<StorybookHandlerGroupName, HttpHandler[]>;
export type StorybookHandlerOverrides = Partial<StorybookHandlerGroups>;

export const storybookSystemHandlerGroups: StorybookHandlerGroups = {
  agent: agentHandlers,
  automation: automationHandlers,
  bridges: bridgeHandlers,
  daemon: daemonHandlers,
  knowledge: knowledgeHandlers,
  network: networkHandlers,
  onboarding: onboardingHandlers,
  session: sessionHandlers,
  settings: settingsHandlers,
  skill: skillHandlers,
  scheduler: schedulerHandlers,
  tasks: tasksHandlers,
  workspace: workspaceHandlers,
};

export function flattenStorybookHandlerGroups(
  groups: StorybookHandlerGroups | StorybookHandlerOverrides
) {
  return Object.values(groups).flat();
}

export const storybookSystemHandlers = flattenStorybookHandlerGroups(storybookSystemHandlerGroups);

function handlerSignature(handler: HttpHandler) {
  const method = String(handler.info.method);
  const path = String(handler.info.path);

  return `${method} ${path}`;
}

export function composeStorybookHandlerGroup(
  groupName: StorybookHandlerGroupName,
  overrides: HttpHandler[]
) {
  const overrideSignatures = new Set(overrides.map(handlerSignature));

  return [
    ...overrides,
    ...storybookSystemHandlerGroups[groupName].filter(
      handler => !overrideSignatures.has(handlerSignature(handler))
    ),
  ];
}

export function composeStorybookHandlerOverrides(overrides: StorybookHandlerOverrides) {
  return Object.fromEntries(
    Object.entries(overrides).map(([groupName, handlers]) => [
      groupName,
      composeStorybookHandlerGroup(groupName as StorybookHandlerGroupName, handlers),
    ])
  ) as StorybookHandlerOverrides;
}

export function storybookMswParameters(overrides: StorybookHandlerOverrides) {
  return {
    msw: {
      handlers: composeStorybookHandlerOverrides(overrides),
    },
  } as const;
}
