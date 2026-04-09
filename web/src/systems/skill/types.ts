import { z } from "zod";

// --- ProvenancePayload ---

export const provenancePayloadSchema = z.object({
  slug: z.string(),
  registry: z.string(),
  version: z.string(),
  installed_at: z.string(),
});

export type ProvenancePayload = z.infer<typeof provenancePayloadSchema>;

// --- SkillPayload ---

export const skillPayloadSchema = z.object({
  name: z.string(),
  description: z.string(),
  version: z.string().optional(),
  source: z.string(),
  enabled: z.boolean(),
  dir: z.string(),
  content: z.string().optional(),
  metadata: z.record(z.string(), z.unknown()).optional(),
  provenance: provenancePayloadSchema.nullable().optional(),
});

export type SkillPayload = z.infer<typeof skillPayloadSchema>;

// --- API Response Envelopes ---

export const skillsResponseSchema = z.object({
  skills: z.array(skillPayloadSchema),
});

export type SkillsResponse = z.infer<typeof skillsResponseSchema>;

export const skillResponseSchema = z.object({
  skill: skillPayloadSchema,
});

export type SkillResponse = z.infer<typeof skillResponseSchema>;

export const skillActionResponseSchema = z.object({
  ok: z.boolean(),
});

export type SkillActionResponse = z.infer<typeof skillActionResponseSchema>;
