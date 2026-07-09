import { HttpResponse, type HttpHandler } from "msw";
import { aghApiMock } from "@/storybook/openapi-msw";

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
  aghApiMock.get("/api/sessions", () => HttpResponse.json({ sessions: sessionFixtures })),
  aghApiMock.get("/api/sessions/{session_id}", ({ params }) => {
    const id = String(params.session_id);
    const session = sessionById.get(id);

    if (!session) {
      return HttpResponse.json({ error: `Session not found: ${id}` }, { status: 404 });
    }

    return HttpResponse.json({ session });
  }),
  aghApiMock.post("/api/sessions", async ({ request }) => {
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
  aghApiMock.get("/api/workspaces/{workspace_id}/sessions/{session_id}", ({ params }) => {
    const id = String(params.session_id);
    const session = sessionById.get(id);

    if (!session) {
      return HttpResponse.json({ error: `Session not found: ${id}` }, { status: 404 });
    }

    return HttpResponse.json({ session });
  }),
  aghApiMock.delete("/api/workspaces/{workspace_id}/sessions/{session_id}", ({ params }) => {
    const id = String(params.session_id);

    if (!sessionById.has(id)) {
      return HttpResponse.json({ error: `Session not found: ${id}` }, { status: 404 });
    }

    return new HttpResponse(null, { status: 204 });
  }),
  aghApiMock.post("/api/workspaces/{workspace_id}/sessions/{session_id}/attach", ({ params }) => {
    const id = String(params.session_id);
    const session = sessionById.get(id);

    if (!session) {
      return HttpResponse.json({ error: `Session not found: ${id}` }, { status: 404 });
    }

    return HttpResponse.json({
      session: {
        ...session,
        attached_to: "web:storybook",
        attach_expires_at: "2026-04-17T18:12:00Z",
      },
      attach: {
        session_id: id,
        attached_to: "web:storybook",
        attach_expires_at: "2026-04-17T18:12:00Z",
        attached_at: "2026-04-17T18:11:00Z",
      },
    });
  }),
  aghApiMock.get("/api/workspaces/{workspace_id}/sessions/{session_id}/recap", ({ params }) => {
    const id = String(params.session_id);
    const session = sessionById.get(id);

    if (!session) {
      return HttpResponse.json({ error: `Session not found: ${id}` }, { status: 404 });
    }

    return HttpResponse.json({
      recap: {
        session,
        recent_markers: [],
        recent_messages: sessionTranscriptFixture,
        pending_inputs: 0,
        pending_markers: 0,
        snapshot: {
          generated_at: "2026-04-17T18:11:00Z",
          event_cursor: sessionEventsFixture.length,
          transcript_cursor: sessionTranscriptFixture.length,
          queue_generation: 0,
          consistency: "read_snapshot",
        },
      },
    });
  }),
  aghApiMock.post(
    "/api/workspaces/{workspace_id}/sessions/{session_id}/repair",
    ({ params, request }) => {
      const id = String(params.session_id);

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
    }
  ),
  aghApiMock.post("/api/workspaces/{workspace_id}/sessions/{session_id}/approve", ({ params }) => {
    const id = String(params.session_id);

    if (!sessionById.has(id)) {
      return HttpResponse.json({ error: `Session not found: ${id}` }, { status: 404 });
    }

    return HttpResponse.json(sessionApprovalFixture);
  }),
  aghApiMock.get("/api/workspaces/{workspace_id}/sessions/{session_id}/events", ({ params }) => {
    const id = String(params.session_id);

    if (!sessionById.has(id)) {
      return HttpResponse.json({ error: `Session not found: ${id}` }, { status: 404 });
    }

    return HttpResponse.json({ events: sessionEventsFixture });
  }),
  aghApiMock.get("/api/workspaces/{workspace_id}/sessions/{session_id}/history", ({ params }) => {
    const id = String(params.session_id);

    if (!sessionById.has(id)) {
      return HttpResponse.json({ error: `Session not found: ${id}` }, { status: 404 });
    }

    return HttpResponse.json({ history: sessionHistoryFixture });
  }),
  aghApiMock.get(
    "/api/workspaces/{workspace_id}/sessions/{session_id}/transcript",
    ({ params }) => {
      const id = String(params.session_id);

      if (!sessionById.has(id)) {
        return HttpResponse.json({ error: `Session not found: ${id}` }, { status: 404 });
      }

      return HttpResponse.json({
        entries: sessionTranscriptFixture.map((message, index) => ({
          message,
          sequence: index + 1,
        })),
      });
    }
  ),
];
