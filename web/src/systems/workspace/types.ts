import { z } from "zod";

export const workspacePayloadSchema = z.object({
  id: z.string(),
  root_dir: z.string(),
  add_dirs: z.array(z.string()),
  name: z.string(),
  default_agent: z.string().optional(),
  created_at: z.string(),
  updated_at: z.string(),
});

export type WorkspacePayload = z.infer<typeof workspacePayloadSchema>;

export const workspacesResponseSchema = z.object({
  workspaces: z.array(workspacePayloadSchema),
});

export type WorkspacesResponse = z.infer<typeof workspacesResponseSchema>;

export const workspaceResponseSchema = z.object({
  workspace: workspacePayloadSchema,
});

export type WorkspaceResponse = z.infer<typeof workspaceResponseSchema>;
