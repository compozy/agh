import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";

import {
  createSession,
  stopSession,
  resumeSession,
  type CreateSessionParams,
} from "../adapters/session-api";
import { sessionKeys } from "../lib/query-keys";

export function useCreateSession() {
  const queryClient = useQueryClient();
  const navigate = useNavigate();

  return useMutation({
    mutationFn: (params: CreateSessionParams) => createSession(params),
    onSuccess: session => {
      queryClient.invalidateQueries({ queryKey: sessionKeys.lists() });
      navigate({ to: "/session/$id", params: { id: session.id } });
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
