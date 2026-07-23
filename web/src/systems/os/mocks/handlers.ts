import { HttpResponse, type HttpHandler } from "msw";

import { aghApiMock } from "@/storybook/openapi-msw";

import { windowManagerClientFixture, windowManagerSnapshotFixture } from "./fixtures";

function requestClientId(value: unknown): string {
  if (value === null || typeof value !== "object") return "storybook-client";
  const clientId = Reflect.get(value, "client_id");
  return typeof clientId === "string" && clientId.trim() !== "" ? clientId : "storybook-client";
}

export const handlers: HttpHandler[] = [
  aghApiMock.get("/api/workspaces/{workspace_id}/window-manager", () =>
    HttpResponse.json(windowManagerSnapshotFixture)
  ),
  aghApiMock.post("/api/workspaces/{workspace_id}/window-manager/clients", async ({ request }) => {
    const clientId = requestClientId(await request.json());
    return HttpResponse.json(windowManagerClientFixture(clientId), {
      status: 201,
    });
  }),
  aghApiMock.post("/api/workspaces/{workspace_id}/window-manager/commands", async ({ request }) => {
    const clientId = requestClientId(await request.json());
    return HttpResponse.json({
      snapshot: windowManagerSnapshotFixture,
      applied: false,
      changes: {},
      diagnostics: [],
      client: windowManagerClientFixture(clientId),
    });
  }),
];
