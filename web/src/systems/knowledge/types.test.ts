import { describe, expect, it } from "vitest";

import {
  memoryConsolidateResponseSchema,
  memoryHeaderSchema,
  memoryMutationResponseSchema,
  memoryReadResponseSchema,
  memoryScopeSchema,
  memoryTypeSchema,
} from "./types";

describe("memoryTypeSchema", () => {
  it("accepts valid memory types", () => {
    for (const t of ["user", "feedback", "project", "reference"]) {
      expect(memoryTypeSchema.safeParse(t).success).toBe(true);
    }
  });

  it("rejects invalid type", () => {
    expect(memoryTypeSchema.safeParse("invalid").success).toBe(false);
  });
});

describe("memoryScopeSchema", () => {
  it("accepts valid scopes", () => {
    for (const s of ["global", "workspace"]) {
      expect(memoryScopeSchema.safeParse(s).success).toBe(true);
    }
  });

  it("rejects invalid scope", () => {
    expect(memoryScopeSchema.safeParse("other").success).toBe(false);
  });
});

describe("memoryHeaderSchema", () => {
  const validHeader = {
    filename: "user_role.md",
    mod_time: "2026-04-01T12:00:00Z",
    name: "User Role",
    type: "user",
  };

  it("validates a minimal valid header", () => {
    const result = memoryHeaderSchema.safeParse(validHeader);
    expect(result.success).toBe(true);
  });

  it("validates a header with all optional fields", () => {
    const full = {
      ...validHeader,
      description: "Stores user role info",
      agent_name: "coder",
    };
    const result = memoryHeaderSchema.safeParse(full);
    expect(result.success).toBe(true);
  });

  it("rejects missing required field: name", () => {
    const { name: _, ...noName } = validHeader;
    expect(memoryHeaderSchema.safeParse(noName).success).toBe(false);
  });

  it("rejects missing required field: type", () => {
    const { type: _, ...noType } = validHeader;
    expect(memoryHeaderSchema.safeParse(noType).success).toBe(false);
  });

  it("rejects missing required field: filename", () => {
    const { filename: _, ...noFilename } = validHeader;
    expect(memoryHeaderSchema.safeParse(noFilename).success).toBe(false);
  });

  it("rejects missing required field: mod_time", () => {
    const { mod_time: _, ...noModTime } = validHeader;
    expect(memoryHeaderSchema.safeParse(noModTime).success).toBe(false);
  });

  it("rejects invalid type value", () => {
    expect(memoryHeaderSchema.safeParse({ ...validHeader, type: "invalid" }).success).toBe(false);
  });
});

describe("API response schemas", () => {
  it("memoryReadResponseSchema validates content", () => {
    const result = memoryReadResponseSchema.safeParse({ content: "hello world" });
    expect(result.success).toBe(true);
  });

  it("memoryReadResponseSchema rejects missing content", () => {
    expect(memoryReadResponseSchema.safeParse({}).success).toBe(false);
  });

  it("memoryMutationResponseSchema validates ok", () => {
    expect(memoryMutationResponseSchema.safeParse({ ok: true }).success).toBe(true);
  });

  it("memoryMutationResponseSchema rejects missing ok", () => {
    expect(memoryMutationResponseSchema.safeParse({}).success).toBe(false);
  });

  it("memoryConsolidateResponseSchema validates triggered", () => {
    expect(
      memoryConsolidateResponseSchema.safeParse({ triggered: true, reason: "success" }).success
    ).toBe(true);
  });

  it("memoryConsolidateResponseSchema allows missing reason", () => {
    expect(memoryConsolidateResponseSchema.safeParse({ triggered: false }).success).toBe(true);
  });

  it("memoryConsolidateResponseSchema rejects missing triggered", () => {
    expect(memoryConsolidateResponseSchema.safeParse({}).success).toBe(false);
  });
});
