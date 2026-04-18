import { http, HttpResponse, type HttpHandler } from "msw";

import {
  automationJobFixtures,
  automationRunFixtures,
  automationTriggerFixtures,
  primaryAutomationJobFixture,
  primaryAutomationTriggerFixture,
} from "./fixtures";

const jobById = new Map(automationJobFixtures.map(job => [job.id, job]));
const triggerById = new Map(automationTriggerFixtures.map(trigger => [trigger.id, trigger]));

export const handlers: HttpHandler[] = [
  http.get("/api/automation/jobs", () => HttpResponse.json({ jobs: automationJobFixtures })),
  http.get("/api/automation/jobs/:id", ({ params }) => {
    const id = String(params.id);
    const job = jobById.get(id);

    if (!job) {
      return HttpResponse.json({ error: `Automation job not found: ${id}` }, { status: 404 });
    }

    return HttpResponse.json({ job });
  }),
  http.post("/api/automation/jobs", async ({ request }) => {
    const body = (await request.json()) as Partial<typeof primaryAutomationJobFixture>;
    return HttpResponse.json(
      {
        job: {
          ...primaryAutomationJobFixture,
          ...body,
          id: body.name
            ? `job_${String(body.name).replace(/[^a-zA-Z0-9]+/g, "_")}`
            : primaryAutomationJobFixture.id,
        },
      },
      { status: 201 }
    );
  }),
  http.patch("/api/automation/jobs/:id", async ({ params, request }) => {
    const id = String(params.id);
    const job = jobById.get(id);

    if (!job) {
      return HttpResponse.json({ error: `Automation job not found: ${id}` }, { status: 404 });
    }

    const body = (await request.json()) as Partial<typeof primaryAutomationJobFixture>;
    return HttpResponse.json({ job: { ...job, ...body, id } });
  }),
  http.delete("/api/automation/jobs/:id", ({ params }) => {
    const id = String(params.id);

    if (!jobById.has(id)) {
      return HttpResponse.json({ error: `Automation job not found: ${id}` }, { status: 404 });
    }

    return new HttpResponse(null, { status: 204 });
  }),
  http.post("/api/automation/jobs/:id/trigger", ({ params }) => {
    const id = String(params.id);

    if (!jobById.has(id)) {
      return HttpResponse.json({ error: `Automation job not found: ${id}` }, { status: 404 });
    }

    return HttpResponse.json({
      run: {
        ...automationRunFixtures[0],
        id: `run_${id}_manual`,
        job_id: id,
        status: "queued",
      },
    });
  }),
  http.get("/api/automation/jobs/:id/runs", ({ params }) => {
    const id = String(params.id);

    if (!jobById.has(id)) {
      return HttpResponse.json({ error: `Automation job not found: ${id}` }, { status: 404 });
    }

    return HttpResponse.json({
      runs: automationRunFixtures.filter(run => run.job_id === id),
    });
  }),
  http.get("/api/automation/triggers", () =>
    HttpResponse.json({ triggers: automationTriggerFixtures })
  ),
  http.get("/api/automation/triggers/:id", ({ params }) => {
    const id = String(params.id);
    const trigger = triggerById.get(id);

    if (!trigger) {
      return HttpResponse.json({ error: `Automation trigger not found: ${id}` }, { status: 404 });
    }

    return HttpResponse.json({ trigger });
  }),
  http.post("/api/automation/triggers", async ({ request }) => {
    const body = (await request.json()) as Partial<typeof primaryAutomationTriggerFixture>;
    return HttpResponse.json(
      {
        trigger: {
          ...primaryAutomationTriggerFixture,
          ...body,
          id: body.name
            ? `trg_${String(body.name).replace(/[^a-zA-Z0-9]+/g, "_")}`
            : primaryAutomationTriggerFixture.id,
        },
      },
      { status: 201 }
    );
  }),
  http.patch("/api/automation/triggers/:id", async ({ params, request }) => {
    const id = String(params.id);
    const trigger = triggerById.get(id);

    if (!trigger) {
      return HttpResponse.json({ error: `Automation trigger not found: ${id}` }, { status: 404 });
    }

    const body = (await request.json()) as Partial<typeof primaryAutomationTriggerFixture>;
    return HttpResponse.json({ trigger: { ...trigger, ...body, id } });
  }),
  http.delete("/api/automation/triggers/:id", ({ params }) => {
    const id = String(params.id);

    if (!triggerById.has(id)) {
      return HttpResponse.json({ error: `Automation trigger not found: ${id}` }, { status: 404 });
    }

    return new HttpResponse(null, { status: 204 });
  }),
  http.get("/api/automation/triggers/:id/runs", ({ params }) => {
    const id = String(params.id);

    if (!triggerById.has(id)) {
      return HttpResponse.json({ error: `Automation trigger not found: ${id}` }, { status: 404 });
    }

    return HttpResponse.json({
      runs: automationRunFixtures.filter(run => run.trigger_id === id),
    });
  }),
  http.get("/api/automation/runs", () => HttpResponse.json({ runs: automationRunFixtures })),
];
