import { describe, expect, it } from "vitest";

import { workspacePayloadSchema, workspaceResponseSchema, workspacesResponseSchema } from "./types";

describe("workspacePayloadSchema", () => {
  const validWorkspace = {
    id: "ws_alpha",
    root_dir: "/workspace/alpha",
    add_dirs: ["/workspace/shared"],
    name: "alpha",
    created_at: "2026-04-06T10:00:00Z",
    updated_at: "2026-04-06T10:00:00Z",
  };

  it("validates a minimal workspace payload", () => {
    const result = workspacePayloadSchema.safeParse(validWorkspace);
    expect(result.success).toBe(true);
  });

  it("validates a workspace with a default agent", () => {
    const result = workspacePayloadSchema.safeParse({
      ...validWorkspace,
      default_agent: "coder",
    });
    expect(result.success).toBe(true);
  });

  it("rejects incomplete payloads", () => {
    const { id: _, ...missingID } = validWorkspace;
    const result = workspacePayloadSchema.safeParse(missingID);
    expect(result.success).toBe(false);
  });
});

describe("workspace response schemas", () => {
  const validWorkspace = {
    id: "ws_alpha",
    root_dir: "/workspace/alpha",
    add_dirs: [],
    name: "alpha",
    created_at: "2026-04-06T10:00:00Z",
    updated_at: "2026-04-06T10:00:00Z",
  };

  it("validates workspaces list responses", () => {
    const result = workspacesResponseSchema.safeParse({
      workspaces: [validWorkspace],
    });
    expect(result.success).toBe(true);
  });

  it("validates single workspace responses", () => {
    const result = workspaceResponseSchema.safeParse({
      workspace: validWorkspace,
    });
    expect(result.success).toBe(true);
  });
});
