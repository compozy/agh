import type { OperationResponse } from "@/lib/api-contract";

export type AgentsResponse = OperationResponse<"listAgents", 200>;
export type AgentPayload = AgentsResponse["agents"][number];
export type AgentResponse = OperationResponse<"getAgent", 200>;
export type AgentMCPServer = NonNullable<AgentPayload["mcp_servers"]>[number];
