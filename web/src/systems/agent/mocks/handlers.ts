import { HttpResponse, type HttpHandler } from "msw";
import { aghApiMock } from "@/storybook/openapi-msw";

import { agentFixtures } from "./fixtures";

const agentByName = new Map(agentFixtures.map(agent => [agent.name, agent]));

export const handlers: HttpHandler[] = [
  aghApiMock.get("/api/agents", () => HttpResponse.json({ agents: agentFixtures })),
  aghApiMock.get("/api/agents/{name}", ({ params }) => {
    const name = String(params.name);
    const agent = agentByName.get(name);

    if (!agent) {
      return HttpResponse.json({ error: `Agent not found: ${name}` }, { status: 404 });
    }

    return HttpResponse.json({ agent });
  }),
];
