import { describe, expectTypeOf, it } from "vitest";

import type {
  KnowledgeFilter,
  MemoryConsolidateResponse,
  MemoryHeader,
  MemoryMutationResponse,
  MemoryReadResponse,
  MemoryScope,
  MemoryType,
} from "./types";

describe("knowledge contract types", () => {
  it("keeps memory payloads aligned with the generated contract", () => {
    expectTypeOf<MemoryType>().toEqualTypeOf<"user" | "feedback" | "project" | "reference">();
    expectTypeOf<MemoryScope>().toEqualTypeOf<"global" | "workspace" | undefined>();

    expectTypeOf<MemoryHeader>().toMatchTypeOf<{
      filename: string;
      mod_time: string;
      name: string;
      type: Exclude<MemoryType, undefined>;
      description?: string;
      agent_name?: string;
    }>();

    expectTypeOf<MemoryReadResponse>().toEqualTypeOf<{ content: string }>();
    expectTypeOf<MemoryMutationResponse>().toEqualTypeOf<{ ok: boolean }>();
    expectTypeOf<MemoryConsolidateResponse>().toMatchTypeOf<{
      triggered: boolean;
      reason?: string;
    }>();

    expectTypeOf<KnowledgeFilter>().toMatchTypeOf<{
      scope?: MemoryScope;
      workspace?: string;
      type?: MemoryType;
      search?: string;
    }>();
  });
});
