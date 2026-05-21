import { http, HttpResponse, type HttpHandler } from "msw";

import {
  schedulerBacklogFixture,
  schedulerDrainResultFixture,
  schedulerPausedStatusFixture,
  schedulerStatusFixture,
} from "./fixtures";

export const handlers: HttpHandler[] = [
  http.get("/api/scheduler", () => HttpResponse.json({ scheduler: schedulerStatusFixture })),
  http.post("/api/scheduler/pause", async ({ request }) => {
    const body = (await request.json().catch(() => ({}))) as { reason?: string };
    return HttpResponse.json({
      scheduler: {
        ...schedulerPausedStatusFixture,
        paused_reason: body.reason ?? schedulerPausedStatusFixture.paused_reason,
      },
    });
  }),
  http.post("/api/scheduler/resume", () =>
    HttpResponse.json({ scheduler: { ...schedulerStatusFixture, paused: false } })
  ),
  http.post("/api/scheduler/drain", () => HttpResponse.json(schedulerDrainResultFixture)),
  http.get("/api/scheduler/backlog", () => HttpResponse.json({ backlog: schedulerBacklogFixture })),
];
