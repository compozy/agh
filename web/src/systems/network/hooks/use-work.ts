import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";

import { networkWorkOptions } from "../lib/query-options";
import { isTerminalNetworkWorkState } from "../lib/network-formatters";
import type { NetworkConversationMessage, NetworkSurface, NetworkWorkDetail } from "../types";
import { useNetworkMessages } from "./use-messages";

export interface UseNetworkWorkArgs {
  workId: string | null | undefined;
  /** Inspector mounted? When false, polling drops back to the on-focus default. */
  inspectorOpen?: boolean;
  enabled?: boolean;
}

export interface UseNetworkWorkResult {
  work: NetworkWorkDetail | null;
  isLoading: boolean;
  error: Error | null;
}

/**
 * Detail fetch for a single `work_id`. The 3s polling interval is only active
 * when the inspector is open AND the state is non-terminal per `_design.md`
 * §9.1. When the inspector is closed, the query falls back to refetch-on-focus
 * (the staleTime + refetchOnWindowFocus from `networkWorkOptions`).
 */
export function useNetworkWork({
  workId,
  inspectorOpen = false,
  enabled = true,
}: UseNetworkWorkArgs): UseNetworkWorkResult {
  const isReady = enabled && Boolean(workId);
  const baseOptions = networkWorkOptions(workId ?? "", isReady);
  const query = useQuery({
    ...baseOptions,
    refetchInterval: query => {
      if (!inspectorOpen) {
        return false;
      }
      const data = query.state.data as NetworkWorkDetail | undefined;
      if (data && isTerminalNetworkWorkState(data.state)) {
        return false;
      }
      const fallback = baseOptions.refetchInterval;
      if (typeof fallback === "number") {
        return fallback;
      }
      return 3_000;
    },
  });

  return useMemo(
    () => ({
      work: query.data ?? null,
      isLoading: isReady && query.isLoading,
      error: query.error ?? null,
    }),
    [isReady, query.data, query.isLoading, query.error]
  );
}

export interface OpenWorkEntry {
  workId: string;
  state: string;
  /** Last message that surfaced this work. Used for the "jump to message" link. */
  messageId: string;
  targetPeerId: string | null;
  openedAt: string | null;
  lastActivityAt: string | null;
}

export interface UseOpenWorkArgs {
  channel: string | null | undefined;
  surface: NetworkSurface | null | undefined;
  containerId: string | null | undefined;
  enabled?: boolean;
}

export interface UseOpenWorkResult {
  /** Work entries that the inspector / chip can render — never includes silent states. */
  entries: OpenWorkEntry[];
  /** Open work count from the active container's loaded messages — drives banner visibility. */
  openCount: number;
  /** True when at least one entry is in `needs_input`, used by the banner escalation. */
  hasNeedsInput: boolean;
  /** Count of entries currently awaiting human input — feeds the banner breakdown. */
  needsInputCount: number;
  /** Count of entries actively working — feeds the banner breakdown. */
  workingCount: number;
  isLoading: boolean;
}

interface MutableOpenWorkState {
  workId: string;
  state: string;
  messageId: string;
  targetPeerId: string | null;
  timestamp: number;
  openedAt: number;
}

function readNumericTimestamp(message: NetworkConversationMessage): number {
  const parsed = new Date(message.timestamp).getTime();
  return Number.isNaN(parsed) ? 0 : parsed;
}

function readWorkState(message: NetworkConversationMessage): string | null {
  // Per `_techspec.md` Example direct-room work envelope, lifecycle messages
  // carry `body.state` (working / needs_input / completed / failed / canceled).
  const body = (message.body ?? null) as { state?: unknown } | null;
  if (!body || typeof body.state !== "string") {
    return null;
  }
  const trimmed = body.state.trim();
  return trimmed.length > 0 ? trimmed : null;
}

function shouldReplaceWorkState(
  existing: MutableOpenWorkState | undefined,
  nextState: string,
  timestamp: number
): boolean {
  if (!existing) {
    return true;
  }
  if (timestamp !== existing.timestamp) {
    return timestamp > existing.timestamp;
  }
  if (isTerminalNetworkWorkState(existing.state) && !isTerminalNetworkWorkState(nextState)) {
    return false;
  }
  return true;
}

/**
 * Aggregates open-work state from the messages already loaded for the active
 * container. The technique is intentional: the public API exposes
 * `open_work_count` on summaries (`_techspec.md`) and individual `getNetworkWork`
 * lookups, but no per-container "list works" endpoint. Scanning the loaded
 * messages avoids fanning out an N+1 of `getNetworkWork` calls.
 *
 * `_design.md` §13.3 forbids client-side aggregation of *full message lists*
 * for **channel-level** counters; the active container's already-loaded
 * timeline is fair game and is not a counter — it is forensic detail.
 */
export function useOpenWork({
  channel,
  surface,
  containerId,
  enabled = true,
}: UseOpenWorkArgs): UseOpenWorkResult {
  const messagesQuery = useNetworkMessages({
    channel,
    containerId,
    surface,
    enabled,
  });

  return useMemo(() => {
    if (!enabled) {
      return {
        entries: [],
        openCount: 0,
        hasNeedsInput: false,
        needsInputCount: 0,
        workingCount: 0,
        isLoading: false,
      };
    }

    const byWork = new Map<string, MutableOpenWorkState>();
    for (const message of messagesQuery.messages) {
      const workId = message.work_id;
      if (!workId) {
        continue;
      }
      const state = readWorkState(message);
      const timestamp = readNumericTimestamp(message);
      const existing = byWork.get(workId);
      const nextState = state ?? existing?.state ?? "submitted";
      if (shouldReplaceWorkState(existing, nextState, timestamp)) {
        byWork.set(workId, {
          workId,
          state: nextState,
          messageId: message.message_id,
          targetPeerId: message.peer_to ?? existing?.targetPeerId ?? null,
          timestamp,
          openedAt: existing?.openedAt ?? timestamp,
        });
      }
    }

    const entries: OpenWorkEntry[] = [];
    let hasNeedsInput = false;
    let needsInputCount = 0;
    let workingCount = 0;
    for (const candidate of byWork.values()) {
      if (isTerminalNetworkWorkState(candidate.state)) {
        continue;
      }
      if (candidate.state === "submitted") {
        // Silent state — counts toward openCount for forensic completeness but
        // not toward the banner/chip surfacing per chromatic rule 6.6.
      }
      entries.push({
        workId: candidate.workId,
        state: candidate.state,
        messageId: candidate.messageId,
        targetPeerId: candidate.targetPeerId,
        openedAt: candidate.openedAt > 0 ? new Date(candidate.openedAt).toISOString() : null,
        lastActivityAt:
          candidate.timestamp > 0 ? new Date(candidate.timestamp).toISOString() : null,
      });
      if (candidate.state === "needs_input") {
        hasNeedsInput = true;
        needsInputCount += 1;
      } else if (candidate.state === "working") {
        workingCount += 1;
      }
    }

    entries.sort((left, right) => left.workId.localeCompare(right.workId));
    return {
      entries,
      openCount: entries.length,
      hasNeedsInput,
      needsInputCount,
      workingCount,
      isLoading: messagesQuery.isLoading,
    };
  }, [enabled, messagesQuery.messages, messagesQuery.isLoading]);
}
