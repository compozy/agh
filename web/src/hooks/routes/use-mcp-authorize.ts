import { useRef, useState } from "react";

import {
  type SettingsMCPAuthBeginResponse,
  type SettingsMCPAuthExchangeRequest,
  type SettingsMCPAuthFilter,
  useBeginMCPAuth,
  useExchangeMCPAuth,
} from "@/systems/settings";

/**
 * Authorize/repair state machine honoring the PRD authorization floor:
 *   idle -> beginning -> waiting (live URL visible + copyable, browser optional)
 *        -> manual (paste code OR full redirect URL) or status poll
 *        -> confirmed (ONLY on authenticated && token_present) | failed.
 * The live authorization URL comes exclusively from `auth/begin`. Success is
 * declared only from a confirmed credential — an exchange HTTP 200 alone never
 * flips the UI. Failure/cancel preserves the prior public status and token.
 */

export type MCPAuthorizePhase =
  | "idle"
  | "beginning"
  | "waiting"
  | "manual"
  | "confirmed"
  | "failed";

export interface MCPAuthorizePriorStatus {
  status: string;
  tokenPresent: boolean;
}

export interface MCPAuthorizeState {
  phase: MCPAuthorizePhase;
  server: string | null;
  begin: SettingsMCPAuthBeginResponse | null;
  error: string | null;
  prior: MCPAuthorizePriorStatus | null;
}

const IDLE: MCPAuthorizeState = {
  phase: "idle",
  server: null,
  begin: null,
  error: null,
  prior: null,
};

function isConfirmed(status: string, tokenPresent: boolean): boolean {
  return status === "authenticated" && tokenPresent;
}

/** A full redirect URL is pasted back; a bare code otherwise. */
function toExchangeBody(value: string): SettingsMCPAuthExchangeRequest {
  const trimmed = value.trim();
  return /^https?:\/\//i.test(trimmed) ? { redirect_url: trimmed } : { code: trimmed };
}

function errorMessage(error: unknown, fallback: string): string {
  if (error instanceof Error && error.message.trim() !== "") return error.message;
  return fallback;
}

export function useMCPAuthorize(filter: SettingsMCPAuthFilter | null) {
  const [state, setState] = useState<MCPAuthorizeState>(IDLE);
  const beginAttempt = useRef(0);
  const beginMutation = useBeginMCPAuth();
  const exchangeMutation = useExchangeMCPAuth();

  const beginAuthorize = async (server: string, prior: MCPAuthorizePriorStatus) => {
    if (!filter) return;
    const attempt = ++beginAttempt.current;
    setState({ phase: "beginning", server, begin: null, error: null, prior });
    try {
      const begin = await beginMutation.mutateAsync({ name: server, filter });
      if (beginAttempt.current !== attempt) return;
      setState(current =>
        current.server === server && current.phase === "beginning"
          ? { ...current, phase: "waiting", begin }
          : current
      );
    } catch (error) {
      if (beginAttempt.current !== attempt) return;
      setState(current =>
        current.server === server
          ? {
              ...current,
              phase: "failed",
              error: errorMessage(error, "Authorization could not be started."),
            }
          : current
      );
    }
  };

  const enterManual = () => {
    setState(current =>
      current.phase === "waiting" || (current.phase === "failed" && current.begin !== null)
        ? { ...current, phase: "manual", error: null }
        : current
    );
  };

  const submitManual = async (value: string) => {
    if (!filter) return;
    const server = state.server;
    if (!server || !state.begin || value.trim() === "") return;
    try {
      const status = await exchangeMutation.mutateAsync({
        name: server,
        filter,
        body: toExchangeBody(value),
      });
      // A 200 is not success; require a confirmed credential from the payload.
      setState(current => {
        if (current.server !== server) return current;
        if (isConfirmed(status.status, Boolean(status.token_present))) {
          return { ...current, phase: "confirmed", error: null };
        }
        return {
          ...current,
          phase: "failed",
          error: "The provider did not return a confirmed credential.",
        };
      });
    } catch (error) {
      setState(current =>
        current.server === server
          ? {
              ...current,
              phase: "failed",
              error: errorMessage(error, "Authorization could not be completed."),
            }
          : current
      );
    }
  };

  /**
   * Auto-completion path: the browser step finishes server-side via the daemon
   * callback, so the page feeds the polled status of the authorizing server here.
   * Confirms only on authenticated && token_present.
   */
  const acknowledgeStatus = (status: string, tokenPresent: boolean) => {
    setState(current => {
      if (current.phase !== "waiting" && current.phase !== "manual") return current;
      if (!isConfirmed(status, tokenPresent)) return current;
      return { ...current, phase: "confirmed", error: null };
    });
  };

  const cancel = () => {
    beginAttempt.current += 1;
    setState(IDLE);
  };

  const retryBegin = async () => {
    if (!state.server || !state.prior) return;
    await beginAuthorize(state.server, state.prior);
  };

  return {
    ...state,
    isBeginning: beginMutation.isPending,
    isExchanging: exchangeMutation.isPending,
    /** True while the dialog is open and awaiting a confirmed credential. */
    isAwaiting: state.phase === "waiting" || state.phase === "manual",
    isOpen: state.phase !== "idle",
    beginAuthorize,
    retryBegin,
    enterManual,
    submitManual,
    acknowledgeStatus,
    cancel,
  };
}

export type UseMCPAuthorizeReturn = ReturnType<typeof useMCPAuthorize>;
