import { z } from "zod";

// --- AgentMCPServer ---

export const agentMCPServerSchema = z.object({
  name: z.string(),
  command: z.string(),
  args: z.array(z.string()).optional(),
  env: z.record(z.string(), z.string()).optional(),
});

export type AgentMCPServer = z.infer<typeof agentMCPServerSchema>;

// --- AgentPayload ---

export const agentPayloadSchema = z.object({
  name: z.string(),
  provider: z.string(),
  command: z.string().optional(),
  model: z.string().optional(),
  tools: z.array(z.string()).optional(),
  permissions: z.string().optional(),
  mcp_servers: z.array(agentMCPServerSchema).optional(),
  prompt: z.string(),
});

export type AgentPayload = z.infer<typeof agentPayloadSchema>;

// --- API Response Envelopes ---

export const agentsResponseSchema = z.object({
  agents: z.array(agentPayloadSchema),
});

export type AgentsResponse = z.infer<typeof agentsResponseSchema>;

export const agentResponseSchema = z.object({
  agent: agentPayloadSchema,
});

export type AgentResponse = z.infer<typeof agentResponseSchema>;
