import {
  type QueryClient,
  type QueryKey,
  useMutation,
  useQueryClient,
} from "@tanstack/react-query";

import {
  cancelQueuedSessionPrompt,
  clearSessionConversation,
  createSession,
  type CreateSessionParams,
  deleteSession,
  repairSession,
  resumeSession,
  SessionApiError,
  sendSessionPrompt,
  steerSessionPrompt,
  stopSession,
} from "../adapters/session-api";
import { useActiveWorkspace } from "@/systems/workspace";
import { useSessionStore } from "./use-session-store";
import { sessionKeys } from "../lib/query-keys";
import type {
  SessionPayload,
  SessionPromptPayload,
  SessionPromptRequest,
  SessionRepairQuery,
} from "../types";

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

function requireWorkspace(workspaceId: string | null | undefined): string {
  if (!workspaceId) {
    throw new SessionApiError("No active workspace selected", 400);
  }
  return workspaceId;
}

interface UseSessionWorkspaceOptions {
  workspaceId?: string | null;
}

function resolveWorkspaceId(
  workspaceId: string | null | undefined,
  activeWorkspaceId: string | null | undefined
): string | null {
  return workspaceId ?? activeWorkspaceId ?? null;
}

function invalidateSessionPromptSurfaces(
  queryClient: QueryClient,
  workspaceId: string,
  id: string
) {
  void queryClient.invalidateQueries({ queryKey: sessionKeys.detail(workspaceId, id) });
  void queryClient.invalidateQueries({ queryKey: sessionKeys.events(workspaceId, id) });
  void queryClient.invalidateQueries({ queryKey: sessionKeys.history(workspaceId, id) });
  void queryClient.invalidateQueries({ queryKey: sessionKeys.transcript(workspaceId, id) });
  void queryClient.invalidateQueries({ queryKey: sessionKeys.lists() });
}

export function useCreateSession() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (params: CreateSessionParams) => createSession(params),
    onSuccess: session => {
      const workspaceId = requireWorkspace(session.workspace_id);
      queryClient.setQueryData(sessionKeys.detail(workspaceId, session.id), session);

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

      void queryClient.invalidateQueries({ queryKey: sessionKeys.detail(workspaceId, session.id) });
      void queryClient.invalidateQueries({ queryKey: sessionKeys.lists() });
    },
  });
}

export function useStopSession(options: UseSessionWorkspaceOptions = {}) {
  const queryClient = useQueryClient();
  const { activeWorkspaceId } = useActiveWorkspace();
  const workspaceId = resolveWorkspaceId(options.workspaceId, activeWorkspaceId);

  return useMutation({
    mutationFn: (id: string) => stopSession(requireWorkspace(workspaceId), id),
    onSettled: (_data, _error, id) => {
      const settledWorkspaceId = workspaceId ?? "";
      queryClient.invalidateQueries({ queryKey: sessionKeys.detail(settledWorkspaceId, id) });
      queryClient.invalidateQueries({ queryKey: sessionKeys.lists() });
    },
  });
}

export function useDeleteSession(options: UseSessionWorkspaceOptions = {}) {
  const queryClient = useQueryClient();
  const { activeWorkspaceId } = useActiveWorkspace();
  const workspaceId = resolveWorkspaceId(options.workspaceId, activeWorkspaceId);

  return useMutation({
    mutationFn: (id: string) => deleteSession(requireWorkspace(workspaceId), id),
    onSuccess: (_data, id) => {
      const successWorkspaceId = workspaceId ?? "";
      useSessionStore.getState().clearDraft(id);
      queryClient.removeQueries({ queryKey: sessionKeys.detail(successWorkspaceId, id) });
      queryClient.removeQueries({ queryKey: sessionKeys.history(successWorkspaceId, id) });
      queryClient.removeQueries({ queryKey: sessionKeys.transcript(successWorkspaceId, id) });
      queryClient.removeQueries({ queryKey: sessionKeys.events(successWorkspaceId, id) });

      return queryClient.invalidateQueries({ queryKey: sessionKeys.lists() });
    },
  });
}

export function useResumeSession(options: UseSessionWorkspaceOptions = {}) {
  const queryClient = useQueryClient();
  const { activeWorkspaceId } = useActiveWorkspace();
  const workspaceId = resolveWorkspaceId(options.workspaceId, activeWorkspaceId);

  return useMutation({
    mutationFn: (id: string) => resumeSession(requireWorkspace(workspaceId), id),
    onSettled: (_data, _error, id) => {
      const settledWorkspaceId = workspaceId ?? "";
      queryClient.invalidateQueries({ queryKey: sessionKeys.detail(settledWorkspaceId, id) });
      queryClient.invalidateQueries({ queryKey: sessionKeys.lists() });
    },
  });
}

export interface RepairSessionParams extends SessionRepairQuery {
  id: string;
}

export function useRepairSession(options: UseSessionWorkspaceOptions = {}) {
  const queryClient = useQueryClient();
  const { activeWorkspaceId } = useActiveWorkspace();
  const workspaceId = resolveWorkspaceId(options.workspaceId, activeWorkspaceId);

  return useMutation({
    mutationFn: ({ id, ...query }: RepairSessionParams) =>
      repairSession(requireWorkspace(workspaceId), id, query),
    onSettled: (_data, _error, params) => {
      const settledWorkspaceId = workspaceId ?? "";
      queryClient.invalidateQueries({
        queryKey: sessionKeys.detail(settledWorkspaceId, params.id),
      });
      queryClient.invalidateQueries({
        queryKey: sessionKeys.history(settledWorkspaceId, params.id),
      });
      queryClient.invalidateQueries({
        queryKey: sessionKeys.transcript(settledWorkspaceId, params.id),
      });
      queryClient.invalidateQueries({
        queryKey: sessionKeys.events(settledWorkspaceId, params.id),
      });
      queryClient.invalidateQueries({ queryKey: sessionKeys.lists() });
    },
  });
}

interface ClearConversationSnapshot {
  session: SessionPayload | undefined;
  transcript: unknown;
  history: unknown;
}

export function useClearSessionConversation(options: UseSessionWorkspaceOptions = {}) {
  const queryClient = useQueryClient();
  const { activeWorkspaceId } = useActiveWorkspace();
  const workspaceId = resolveWorkspaceId(options.workspaceId, activeWorkspaceId);

  return useMutation({
    mutationFn: (id: string) => clearSessionConversation(requireWorkspace(workspaceId), id),
    onMutate: async (id): Promise<ClearConversationSnapshot> => {
      const mutateWorkspaceId = requireWorkspace(workspaceId);
      await queryClient.cancelQueries({ queryKey: sessionKeys.detail(mutateWorkspaceId, id) });
      await queryClient.cancelQueries({ queryKey: sessionKeys.history(mutateWorkspaceId, id) });
      await queryClient.cancelQueries({ queryKey: sessionKeys.transcript(mutateWorkspaceId, id) });

      const snapshot: ClearConversationSnapshot = {
        session: queryClient.getQueryData<SessionPayload>(
          sessionKeys.detail(mutateWorkspaceId, id)
        ),
        transcript: queryClient.getQueryData(sessionKeys.transcript(mutateWorkspaceId, id)),
        history: queryClient.getQueryData(sessionKeys.history(mutateWorkspaceId, id)),
      };

      queryClient.setQueryData(sessionKeys.transcript(mutateWorkspaceId, id), []);
      queryClient.setQueryData(sessionKeys.history(mutateWorkspaceId, id), []);

      return snapshot;
    },
    onError: (_error, id, snapshot) => {
      if (!snapshot) {
        return;
      }

      if (snapshot.session) {
        const snapshotWorkspaceId = requireWorkspace(snapshot.session.workspace_id);
        queryClient.setQueryData(
          sessionKeys.detail(snapshotWorkspaceId, snapshot.session.id),
          snapshot.session
        );
      }

      const errorWorkspaceId = workspaceId ?? "";
      queryClient.setQueryData(sessionKeys.transcript(errorWorkspaceId, id), snapshot.transcript);
      queryClient.setQueryData(sessionKeys.history(errorWorkspaceId, id), snapshot.history);
    },
    onSuccess: (session, id) => {
      const workspaceId = requireWorkspace(session.workspace_id);
      queryClient.setQueryData(sessionKeys.detail(workspaceId, id), session);
      queryClient.setQueryData(sessionKeys.transcript(workspaceId, id), []);
      queryClient.setQueryData(sessionKeys.history(workspaceId, id), []);
    },
    onSettled: (_data, _error, id) => {
      const settledWorkspaceId = workspaceId ?? "";
      queryClient.invalidateQueries({ queryKey: sessionKeys.detail(settledWorkspaceId, id) });
      queryClient.invalidateQueries({ queryKey: sessionKeys.history(settledWorkspaceId, id) });
      queryClient.invalidateQueries({ queryKey: sessionKeys.transcript(settledWorkspaceId, id) });
      queryClient.invalidateQueries({ queryKey: sessionKeys.lists() });
    },
  });
}

export interface SessionPromptActionParams {
  id: string;
  message: string;
}

export interface SendSessionPromptParams extends SessionPromptActionParams {
  mode?: SessionPromptRequest["mode"];
}

export interface CancelQueuedSessionPromptParams {
  id: string;
  queueEntryId: string;
}

export function useSendSessionPrompt(options: UseSessionWorkspaceOptions = {}) {
  const queryClient = useQueryClient();
  const { activeWorkspaceId } = useActiveWorkspace();
  const workspaceId = resolveWorkspaceId(options.workspaceId, activeWorkspaceId);

  return useMutation<SessionPromptPayload, Error, SendSessionPromptParams>({
    mutationFn: ({ id, message, mode }) =>
      sendSessionPrompt(requireWorkspace(workspaceId), id, { message, mode }),
    onSettled: (_data, _error, params) => {
      invalidateSessionPromptSurfaces(queryClient, workspaceId ?? "", params.id);
    },
  });
}

export function useQueueSessionPrompt(options: UseSessionWorkspaceOptions = {}) {
  const queryClient = useQueryClient();
  const { activeWorkspaceId } = useActiveWorkspace();
  const workspaceId = resolveWorkspaceId(options.workspaceId, activeWorkspaceId);

  return useMutation<SessionPromptPayload, Error, SessionPromptActionParams>({
    mutationFn: ({ id, message }) =>
      sendSessionPrompt(requireWorkspace(workspaceId), id, { message, mode: "queue" }),
    onSettled: (_data, _error, params) => {
      invalidateSessionPromptSurfaces(queryClient, workspaceId ?? "", params.id);
    },
  });
}

export function useInterruptSessionPrompt(options: UseSessionWorkspaceOptions = {}) {
  const queryClient = useQueryClient();
  const { activeWorkspaceId } = useActiveWorkspace();
  const workspaceId = resolveWorkspaceId(options.workspaceId, activeWorkspaceId);

  return useMutation<SessionPromptPayload, Error, SessionPromptActionParams>({
    mutationFn: ({ id, message }) =>
      sendSessionPrompt(requireWorkspace(workspaceId), id, { message, mode: "interrupt" }),
    onSettled: (_data, _error, params) => {
      invalidateSessionPromptSurfaces(queryClient, workspaceId ?? "", params.id);
    },
  });
}

export function useSteerSessionPrompt(options: UseSessionWorkspaceOptions = {}) {
  const queryClient = useQueryClient();
  const { activeWorkspaceId } = useActiveWorkspace();
  const workspaceId = resolveWorkspaceId(options.workspaceId, activeWorkspaceId);

  return useMutation<SessionPromptPayload, Error, SessionPromptActionParams>({
    mutationFn: ({ id, message }) => steerSessionPrompt(requireWorkspace(workspaceId), id, message),
    onSettled: (_data, _error, params) => {
      invalidateSessionPromptSurfaces(queryClient, workspaceId ?? "", params.id);
    },
  });
}

export function useCancelQueuedSessionPrompt(options: UseSessionWorkspaceOptions = {}) {
  const queryClient = useQueryClient();
  const { activeWorkspaceId } = useActiveWorkspace();
  const workspaceId = resolveWorkspaceId(options.workspaceId, activeWorkspaceId);

  return useMutation<SessionPromptPayload, Error, CancelQueuedSessionPromptParams>({
    mutationFn: ({ id, queueEntryId }) =>
      cancelQueuedSessionPrompt(requireWorkspace(workspaceId), id, queueEntryId),
    onSettled: (_data, _error, params) => {
      invalidateSessionPromptSurfaces(queryClient, workspaceId ?? "", params.id);
    },
  });
}
