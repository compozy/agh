import { type QueryKey, useMutation, useQueryClient } from "@tanstack/react-query";

import {
  clearSessionConversation,
  createSession,
  type CreateSessionParams,
  deleteSession,
  resumeSession,
  stopSession,
} from "../adapters/session-api";
import { useSessionStore } from "./use-session-store";
import { sessionKeys } from "../lib/query-keys";
import type { SessionPayload } from "../types";

function mergeSessionList(
  current: SessionPayload[] | undefined,
  session: SessionPayload
): SessionPayload[] | undefined {
  if (!current) {
    return current;
  }

  const withoutDuplicate = current.filter(item => item.id !== session.id);
  return [session, ...withoutDuplicate];
}

function shouldSeedList(queryKey: QueryKey, workspaceId?: string): boolean {
  if (!Array.isArray(queryKey)) {
    return false;
  }

  const scope = queryKey[2];
  return scope === "all" || (typeof workspaceId === "string" && scope === workspaceId);
}

export function useCreateSession() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (params: CreateSessionParams) => createSession(params),
    onSuccess: session => {
      queryClient.setQueryData(sessionKeys.detail(session.id), session);

      for (const [queryKey] of queryClient.getQueriesData<SessionPayload[]>({
        queryKey: sessionKeys.lists(),
      })) {
        if (!shouldSeedList(queryKey, session.workspace_id)) {
          continue;
        }

        queryClient.setQueryData<SessionPayload[]>(queryKey, current =>
          mergeSessionList(current, session)
        );
      }

      void queryClient.invalidateQueries({ queryKey: sessionKeys.detail(session.id) });
      void queryClient.invalidateQueries({ queryKey: sessionKeys.lists() });
    },
  });
}

export function useStopSession() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => stopSession(id),
    onSettled: (_data, _error, id) => {
      queryClient.invalidateQueries({ queryKey: sessionKeys.detail(id) });
      queryClient.invalidateQueries({ queryKey: sessionKeys.lists() });
    },
  });
}

export function useDeleteSession() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => deleteSession(id),
    onSuccess: (_data, id) => {
      useSessionStore.getState().clearDraft(id);
    },
    onSettled: (_data, _error, id) => {
      queryClient.removeQueries({ queryKey: sessionKeys.detail(id) });
      queryClient.removeQueries({ queryKey: sessionKeys.history(id) });
      queryClient.removeQueries({ queryKey: sessionKeys.transcript(id) });
      queryClient.invalidateQueries({ queryKey: sessionKeys.lists() });
    },
  });
}

export function useResumeSession() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => resumeSession(id),
    onSettled: (_data, _error, id) => {
      queryClient.invalidateQueries({ queryKey: sessionKeys.detail(id) });
      queryClient.invalidateQueries({ queryKey: sessionKeys.lists() });
    },
  });
}

interface ClearConversationSnapshot {
  session: SessionPayload | undefined;
  transcript: unknown;
  history: unknown;
}

export function useClearSessionConversation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => clearSessionConversation(id),
    onMutate: async (id): Promise<ClearConversationSnapshot> => {
      await queryClient.cancelQueries({ queryKey: sessionKeys.detail(id) });
      await queryClient.cancelQueries({ queryKey: sessionKeys.history(id) });
      await queryClient.cancelQueries({ queryKey: sessionKeys.transcript(id) });

      const snapshot: ClearConversationSnapshot = {
        session: queryClient.getQueryData<SessionPayload>(sessionKeys.detail(id)),
        transcript: queryClient.getQueryData(sessionKeys.transcript(id)),
        history: queryClient.getQueryData(sessionKeys.history(id)),
      };

      queryClient.setQueryData(sessionKeys.transcript(id), []);
      queryClient.setQueryData(sessionKeys.history(id), []);

      return snapshot;
    },
    onError: (_error, id, snapshot) => {
      if (!snapshot) {
        return;
      }

      if (snapshot.session) {
        queryClient.setQueryData(sessionKeys.detail(snapshot.session.id), snapshot.session);
      }

      queryClient.setQueryData(sessionKeys.transcript(id), snapshot.transcript);
      queryClient.setQueryData(sessionKeys.history(id), snapshot.history);
    },
    onSuccess: (session, id) => {
      queryClient.setQueryData(sessionKeys.detail(id), session);
      queryClient.setQueryData(sessionKeys.transcript(id), []);
      queryClient.setQueryData(sessionKeys.history(id), []);
    },
    onSettled: (_data, _error, id) => {
      queryClient.invalidateQueries({ queryKey: sessionKeys.detail(id) });
      queryClient.invalidateQueries({ queryKey: sessionKeys.history(id) });
      queryClient.invalidateQueries({ queryKey: sessionKeys.transcript(id) });
      queryClient.invalidateQueries({ queryKey: sessionKeys.lists() });
    },
  });
}
