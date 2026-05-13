import { http, HttpResponse, type HttpHandler } from "msw";

import {
  primarySessionFixture,
  sessionApprovalFixture,
  sessionEventsFixture,
  sessionFixtures,
  sessionHistoryFixture,
  sessionRepairFixture,
  sessionTranscriptFixture,
} from "./fixtures";

const sessionById = new Map(sessionFixtures.map(session => [session.id, session]));

export const handlers: HttpHandler[] = [
  http.get("/api/sessions", () => HttpResponse.json({ sessions: sessionFixtures })),
  http.post("/api/sessions", async ({ request }) => {
    const body = (await request.json()) as {
      agent_name?: string;
      name?: string;
      workspace?: string;
      workspace_path?: string;
      channel?: string;
    };

    return HttpResponse.json(
      {
        session: {
          ...primarySessionFixture,
          id: `sess_${(body.name ?? body.agent_name ?? "story").replace(/[^a-zA-Z0-9]+/g, "_").toLowerCase()}`,
          name: body.name ?? primarySessionFixture.name,
          agent_name: body.agent_name ?? primarySessionFixture.agent_name,
          workspace_path:
            body.workspace_path ?? body.workspace ?? primarySessionFixture.workspace_path,
          channel: body.channel ?? primarySessionFixture.channel,
        },
      },
      { status: 201 }
    );
  }),
  http.get("/api/workspaces/:workspace_id/sessions/:id", ({ params }) => {
    const id = String(params.id);
    const session = sessionById.get(id);

    if (!session) {
      return HttpResponse.json({ error: `Session not found: ${id}` }, { status: 404 });
    }

    return HttpResponse.json({ session });
  }),
  http.delete("/api/workspaces/:workspace_id/sessions/:id", ({ params }) => {
    const id = String(params.id);

    if (!sessionById.has(id)) {
      return HttpResponse.json({ error: `Session not found: ${id}` }, { status: 404 });
    }

    return new HttpResponse(null, { status: 204 });
  }),
  http.post("/api/workspaces/:workspace_id/sessions/:id/resume", ({ params }) => {
    const id = String(params.id);
    const session = sessionById.get(id);

    if (!session) {
      return HttpResponse.json({ error: `Session not found: ${id}` }, { status: 404 });
    }

    return HttpResponse.json({
      session: {
        ...session,
        state: "active",
      },
    });
  }),
  http.post("/api/workspaces/:workspace_id/sessions/:id/repair", ({ params, request }) => {
    const id = String(params.id);

    if (!sessionById.has(id)) {
      return HttpResponse.json({ error: `Session not found: ${id}` }, { status: 404 });
    }

    const url = new URL(request.url);
    const dryRun = url.searchParams.get("dry_run") === "true";

    return HttpResponse.json({
      repair: {
        ...sessionRepairFixture,
        session_id: id,
        persisted: !dryRun,
        actions: sessionRepairFixture.actions.map(action => ({
          ...action,
          persisted: !dryRun,
        })),
      },
    });
  }),
  http.post("/api/workspaces/:workspace_id/sessions/:id/approve", ({ params }) => {
    const id = String(params.id);

    if (!sessionById.has(id)) {
      return HttpResponse.json({ error: `Session not found: ${id}` }, { status: 404 });
    }

    return HttpResponse.json(sessionApprovalFixture);
  }),
  http.get("/api/workspaces/:workspace_id/sessions/:id/events", ({ params }) => {
    const id = String(params.id);

    if (!sessionById.has(id)) {
      return HttpResponse.json({ error: `Session not found: ${id}` }, { status: 404 });
    }

    return HttpResponse.json({ events: sessionEventsFixture });
  }),
  http.get("/api/workspaces/:workspace_id/sessions/:id/history", ({ params }) => {
    const id = String(params.id);

    if (!sessionById.has(id)) {
      return HttpResponse.json({ error: `Session not found: ${id}` }, { status: 404 });
    }

    return HttpResponse.json({ history: sessionHistoryFixture });
  }),
  http.get("/api/workspaces/:workspace_id/sessions/:id/transcript", ({ params }) => {
    const id = String(params.id);

    if (!sessionById.has(id)) {
      return HttpResponse.json({ error: `Session not found: ${id}` }, { status: 404 });
    }

    return HttpResponse.json({ messages: sessionTranscriptFixture });
  }),
];
