import type { QueryClient } from "@tanstack/react-query";
import { useRemoteThreadListRuntime } from "@assistant-ui/react";

import type { SessionPayload } from "../types";
import { sessionDetailOptions, sessionsListOptions } from "./query-options";

type SessionThreadListAdapter = Parameters<typeof useRemoteThreadListRuntime>[0]["adapter"];

function toThreadMetadata(session: SessionPayload) {
  return {
    remoteId: session.id,
    externalId: session.workspace_id,
    title: session.name ?? session.agent_name,
    status: "regular" as const,
  };
}

function unsupportedOperation(name: string): never {
  throw new Error(`AGH session threads do not support ${name}`);
}

export function createSessionThreadListAdapter({
  queryClient,
  workspaceId,
}: {
  queryClient: QueryClient;
  workspaceId?: string;
}) {
  const adapter = {
    async list() {
      const sessions = await queryClient.ensureQueryData(sessionsListOptions(workspaceId ?? null));

      return {
        threads: sessions.map(toThreadMetadata),
      };
    },
    async fetch(threadId: string) {
      if (!workspaceId) {
        throw new Error("AGH session thread fetch requires a workspace id");
      }
      const session = await queryClient.ensureQueryData(
        sessionDetailOptions(workspaceId, threadId)
      );
      return toThreadMetadata(session);
    },
    async initialize(threadId: string) {
      if (!workspaceId) {
        throw new Error("AGH session thread initialization requires a workspace id");
      }
      const session = await queryClient.ensureQueryData(
        sessionDetailOptions(workspaceId, threadId)
      );

      return {
        remoteId: session.id,
        externalId: session.workspace_id,
      };
    },
    async rename() {
      unsupportedOperation("renaming");
    },
    async archive() {
      unsupportedOperation("archiving");
    },
    async unarchive() {
      unsupportedOperation("unarchiving");
    },
    async delete() {
      unsupportedOperation("deleting");
    },
    async generateTitle() {
      unsupportedOperation("title generation");
    },
  } satisfies SessionThreadListAdapter;

  return adapter;
}
