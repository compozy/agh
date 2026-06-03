import { http, HttpResponse, type HttpHandler } from "msw";

import { onboardingCompletedFixture, onboardingIncompleteFixture } from "./fixtures";

export const handlers: HttpHandler[] = [
  http.get("/api/onboarding", () => HttpResponse.json({ onboarding: onboardingCompletedFixture })),
  http.post("/api/onboarding/complete", () =>
    HttpResponse.json({ onboarding: onboardingCompletedFixture })
  ),
  http.delete("/api/onboarding", () =>
    HttpResponse.json({ onboarding: onboardingIncompleteFixture })
  ),
];
