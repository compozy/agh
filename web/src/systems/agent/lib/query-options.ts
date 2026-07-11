import { queryOptions } from "@tanstack/react-query";

import { fetchAgent, fetchAgents } from "../adapters/agent-api";
import {
  fetchAgentHeartbeat,
  fetchAgentHeartbeatHistory,
  fetchAgentHeartbeatStatus,
  type FetchAgentHeartbeatStatusParams,
} from "../adapters/agent-heartbeat-api";
import { fetchAgentSoul, fetchAgentSoulHistory } from "../adapters/agent-soul-api";
import { agentKeys } from "./query-keys";

export function agentsListOptions(workspace?: string | null) {
  return queryOptions({
    queryKey: agentKeys.list(workspace),
    queryFn: ({ signal }) => fetchAgents(workspace, signal),
    staleTime: 60_000,
  });
}

export function agentDetailOptions(name: string, workspace?: string | null) {
  return queryOptions({
    queryKey: agentKeys.detail(name, workspace),
    queryFn: ({ signal }) => fetchAgent(name, workspace, signal),
    staleTime: 60_000,
    enabled: !!name,
  });
}

export function agentSoulOptions(name: string, workspace?: string | null) {
  return queryOptions({
    queryKey: agentKeys.soul(name, workspace),
    queryFn: ({ signal }) => fetchAgentSoul(name, workspace, signal),
    staleTime: 30_000,
    enabled: !!name,
  });
}

export function agentSoulHistoryOptions(name: string, workspace?: string | null) {
  return queryOptions({
    queryKey: agentKeys.soulHistory(name, workspace),
    queryFn: ({ signal }) => fetchAgentSoulHistory(name, workspace, signal),
    staleTime: 30_000,
    enabled: !!name,
  });
}

export function agentHeartbeatOptions(name: string, workspace?: string | null) {
  return queryOptions({
    queryKey: agentKeys.heartbeat(name, workspace),
    queryFn: ({ signal }) => fetchAgentHeartbeat(name, workspace, signal),
    staleTime: 30_000,
    enabled: !!name,
  });
}

export function agentHeartbeatHistoryOptions(name: string, workspace?: string | null) {
  return queryOptions({
    queryKey: agentKeys.heartbeatHistory(name, workspace),
    queryFn: ({ signal }) => fetchAgentHeartbeatHistory(name, workspace, signal),
    staleTime: 30_000,
    enabled: !!name,
  });
}

export function agentHeartbeatStatusOptions(
  name: string,
  options: FetchAgentHeartbeatStatusParams = {}
) {
  return queryOptions({
    queryKey: agentKeys.heartbeatStatus(name, options),
    queryFn: ({ signal }) => fetchAgentHeartbeatStatus(name, options, signal),
    staleTime: 5_000,
    enabled: !!name,
  });
}
