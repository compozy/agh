import { describe, expectTypeOf, it } from "vitest";

import type { AgentMCPServer, AgentPayload, AgentResponse, AgentsResponse } from "./types";

describe("agent contract types", () => {
  it("keeps required and optional agent fields aligned with the generated contract", () => {
    expectTypeOf<AgentPayload>().toMatchTypeOf<{
      name: string;
      provider: string;
      prompt: string;
      command?: string;
      model?: string;
      permissions?: string;
      tools?: string[];
    }>();

    expectTypeOf<AgentMCPServer>().toMatchTypeOf<{
      name: string;
      command: string;
      args?: string[];
      env?: Record<string, string>;
    }>();

    expectTypeOf<AgentsResponse>().toMatchTypeOf<{ agents: AgentPayload[] }>();
    expectTypeOf<AgentResponse>().toMatchTypeOf<{ agent: AgentPayload }>();
  });
});
