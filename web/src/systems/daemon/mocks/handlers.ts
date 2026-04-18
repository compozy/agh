import { http, HttpResponse, type HttpHandler } from "msw";

import { daemonHealthFixture, daemonStatusFixture } from "./fixtures";

export const handlers: HttpHandler[] = [
  http.get("/api/observe/health", () => HttpResponse.json({ health: daemonHealthFixture })),
  http.get("/api/daemon/status", () => HttpResponse.json({ daemon: daemonStatusFixture })),
];
