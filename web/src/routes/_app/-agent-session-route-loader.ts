import type { QueryClient } from "@tanstack/react-query";

import {
  SessionNotFoundError,
  sessionByIdOptions,
  sessionDetailOptions,
  sessionKeys,
  sessionTranscriptOptions,
  type SessionPayload,
} from "@/systems/session";
import {
  adoptRouteWorkspaceWhenSelectionInvalid,
  selectRouteWorkspaceForNavigation,
} from "./-route-preload";

export interface AgentSessionRouteLoaderData {
  workspaceId: string | null;
}

export async function prefetchAgentSessionRoute({
  queryClient,
  sessionId,
  returnWorkspaceId,
  preload = false,
}: {
  queryClient: QueryClient;
  sessionId: string;
  returnWorkspaceId?: string;
  preload?: boolean;
}): Promise<AgentSessionRouteLoaderData> {
  const workspaceId = await resolveSessionRouteWorkspace(queryClient, sessionId);
  if (!workspaceId) {
    return { workspaceId: null };
  }

  if (!preload) {
    if (returnWorkspaceId === workspaceId) {
      await selectRouteWorkspaceForNavigation(queryClient, workspaceId);
    } else {
      await adoptRouteWorkspaceWhenSelectionInvalid(queryClient, workspaceId);
    }
  }
  await Promise.allSettled([
    queryClient.ensureQueryData(sessionDetailOptions(workspaceId, sessionId)),
    queryClient.ensureInfiniteQueryData(sessionTranscriptOptions(workspaceId, sessionId)),
  ]);

  return { workspaceId };
}

async function resolveSessionRouteWorkspace(
  queryClient: QueryClient,
  sessionId: string
): Promise<string | null> {
  const cachedWorkspaceId = normalizeWorkspaceId(
    queryClient.getQueryData<SessionPayload>(sessionKeys.byId(sessionId))?.workspace_id
  );
  if (cachedWorkspaceId) {
    return cachedWorkspaceId;
  }
  try {
    const session = await queryClient.ensureQueryData(sessionByIdOptions(sessionId));
    return normalizeWorkspaceId(session.workspace_id);
  } catch (error) {
    if (error instanceof SessionNotFoundError) {
      return null;
    }
    throw error;
  }
}

function normalizeWorkspaceId(value: string | null | undefined): string | null {
  const trimmed = value?.trim();
  return trimmed ? trimmed : null;
}
