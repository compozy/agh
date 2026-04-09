import { describe, expect, it } from "vitest";

import {
  provenancePayloadSchema,
  skillActionResponseSchema,
  skillPayloadSchema,
  skillResponseSchema,
  skillsResponseSchema,
} from "./types";

describe("skillPayloadSchema", () => {
  const validSkill = {
    name: "test-skill",
    description: "A test skill",
    source: "bundled",
    enabled: true,
    dir: "/path/to/skill",
  };

  it("validates a minimal valid skill", () => {
    const result = skillPayloadSchema.safeParse(validSkill);
    expect(result.success).toBe(true);
  });

  it("validates a skill with all optional fields", () => {
    const full = {
      ...validSkill,
      version: "1.0.0",
      content: "skill content here",
      metadata: { key: "value", count: 42 },
      provenance: {
        slug: "test-skill",
        registry: "agentskills.io",
        version: "1.0.0",
        installed_at: "2026-01-01T00:00:00Z",
      },
    };
    const result = skillPayloadSchema.safeParse(full);
    expect(result.success).toBe(true);
  });

  it("validates a skill with null provenance", () => {
    const result = skillPayloadSchema.safeParse({ ...validSkill, provenance: null });
    expect(result.success).toBe(true);
  });

  it("rejects missing required field: name", () => {
    const { name: _, ...noName } = validSkill;
    const result = skillPayloadSchema.safeParse(noName);
    expect(result.success).toBe(false);
  });

  it("rejects missing required field: description", () => {
    const { description: _, ...noDesc } = validSkill;
    const result = skillPayloadSchema.safeParse(noDesc);
    expect(result.success).toBe(false);
  });

  it("rejects missing required field: source", () => {
    const { source: _, ...noSource } = validSkill;
    const result = skillPayloadSchema.safeParse(noSource);
    expect(result.success).toBe(false);
  });

  it("rejects missing required field: enabled", () => {
    const { enabled: _, ...noEnabled } = validSkill;
    const result = skillPayloadSchema.safeParse(noEnabled);
    expect(result.success).toBe(false);
  });

  it("rejects missing required field: dir", () => {
    const { dir: _, ...noDir } = validSkill;
    const result = skillPayloadSchema.safeParse(noDir);
    expect(result.success).toBe(false);
  });
});

describe("provenancePayloadSchema", () => {
  it("validates a valid provenance", () => {
    const result = provenancePayloadSchema.safeParse({
      slug: "test-skill",
      registry: "agentskills.io",
      version: "1.0.0",
      installed_at: "2026-01-01T00:00:00Z",
    });
    expect(result.success).toBe(true);
  });

  it("rejects missing slug", () => {
    const result = provenancePayloadSchema.safeParse({
      registry: "agentskills.io",
      version: "1.0.0",
      installed_at: "2026-01-01T00:00:00Z",
    });
    expect(result.success).toBe(false);
  });
});

describe("API response envelopes", () => {
  const validSkill = {
    name: "test-skill",
    description: "A test skill",
    source: "bundled",
    enabled: true,
    dir: "/path/to/skill",
  };

  it("skillsResponseSchema validates skills list", () => {
    const result = skillsResponseSchema.safeParse({ skills: [validSkill] });
    expect(result.success).toBe(true);
  });

  it("skillsResponseSchema validates empty list", () => {
    const result = skillsResponseSchema.safeParse({ skills: [] });
    expect(result.success).toBe(true);
  });

  it("skillResponseSchema validates single skill", () => {
    const result = skillResponseSchema.safeParse({ skill: validSkill });
    expect(result.success).toBe(true);
  });

  it("skillActionResponseSchema validates ok response", () => {
    const result = skillActionResponseSchema.safeParse({ ok: true });
    expect(result.success).toBe(true);
  });

  it("skillActionResponseSchema rejects missing ok", () => {
    const result = skillActionResponseSchema.safeParse({});
    expect(result.success).toBe(false);
  });
});
