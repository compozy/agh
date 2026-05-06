import { describe, expectTypeOf, it } from "vitest";

import type {
  ProvenancePayload,
  SkillActionResponse,
  SkillContentResponse,
  SkillPayload,
  SkillResponse,
  SkillsResponse,
} from "../types";

describe("skill contract types", () => {
  it("keeps skill payload fields aligned with the generated OpenAPI contract", () => {
    expectTypeOf<SkillPayload>().toMatchTypeOf<{
      name: string;
      description: string;
      source: string;
      enabled: boolean;
      dir: string;
      version?: string;
      metadata?: Record<string, unknown>;
      provenance?: ProvenancePayload | null;
    }>();

    expectTypeOf<ProvenancePayload>().toMatchTypeOf<{
      slug: string;
      registry: string;
      version: string;
      installed_at: string;
    }>();

    expectTypeOf<SkillsResponse>().toMatchTypeOf<{ skills: SkillPayload[] }>();
    expectTypeOf<SkillResponse>().toMatchTypeOf<{ skill: SkillPayload }>();
    expectTypeOf<SkillContentResponse>().toMatchTypeOf<{ content: string }>();
    expectTypeOf<SkillActionResponse>().toEqualTypeOf<{ ok: boolean }>();
  });
});
