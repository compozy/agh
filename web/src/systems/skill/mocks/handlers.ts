import { http, HttpResponse, type HttpHandler } from "msw";

import { skillActionFixture, skillContentFixtures, skillFixtures } from "./fixtures";

const skillByName = new Map(skillFixtures.map(skill => [skill.name, skill]));

export const handlers: HttpHandler[] = [
  http.get("/api/skills", () => HttpResponse.json({ skills: skillFixtures })),
  http.get("/api/skills/:name", ({ params }) => {
    const name = String(params.name);
    const skill = skillByName.get(name);

    if (!skill) {
      return HttpResponse.json({ error: `Skill not found: ${name}` }, { status: 404 });
    }

    return HttpResponse.json({ skill });
  }),
  http.get("/api/skills/:name/content", ({ params }) => {
    const name = String(params.name);
    const content = skillContentFixtures[name];

    if (!content) {
      return HttpResponse.json({ error: `Skill not found: ${name}` }, { status: 404 });
    }

    return HttpResponse.json({ content });
  }),
  http.post("/api/skills/:name/enable", ({ params }) => {
    if (!skillByName.has(String(params.name))) {
      return HttpResponse.json(
        { error: `Skill not found: ${String(params.name)}` },
        { status: 404 }
      );
    }

    return HttpResponse.json(skillActionFixture);
  }),
  http.post("/api/skills/:name/disable", ({ params }) => {
    if (!skillByName.has(String(params.name))) {
      return HttpResponse.json(
        { error: `Skill not found: ${String(params.name)}` },
        { status: 404 }
      );
    }

    return HttpResponse.json(skillActionFixture);
  }),
];
