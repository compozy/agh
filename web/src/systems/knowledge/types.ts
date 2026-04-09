import { z } from "zod";

// --- MemoryType ---

export const memoryTypeSchema = z.enum(["user", "feedback", "project", "reference"]);

export type MemoryType = z.infer<typeof memoryTypeSchema>;

// --- MemoryScope ---

export const memoryScopeSchema = z.enum(["global", "workspace"]);

export type MemoryScope = z.infer<typeof memoryScopeSchema>;

// --- MemoryHeader ---

export const memoryHeaderSchema = z.object({
  filename: z.string(),
  mod_time: z.string(),
  name: z.string(),
  description: z.string().optional(),
  type: memoryTypeSchema,
  agent_name: z.string().optional(),
});

export type MemoryHeader = z.infer<typeof memoryHeaderSchema>;

// --- API Response / Request Schemas ---

export const memoryReadResponseSchema = z.object({
  content: z.string(),
});

export type MemoryReadResponse = z.infer<typeof memoryReadResponseSchema>;

export const memoryMutationResponseSchema = z.object({
  ok: z.boolean(),
});

export type MemoryMutationResponse = z.infer<typeof memoryMutationResponseSchema>;

export const memoryConsolidateResponseSchema = z.object({
  triggered: z.boolean(),
  reason: z.string().optional(),
});

export type MemoryConsolidateResponse = z.infer<typeof memoryConsolidateResponseSchema>;

// --- Filter Types ---

export type KnowledgeFilter = {
  scope?: MemoryScope;
  workspace?: string;
  type?: MemoryType;
  search?: string;
};
