import { useEffect, useEffectEvent, useRef, useState } from "react";

import { registerWindowManagerClient } from "../adapters/window-manager-api";
import type { WindowManagerClientView } from "../lib/window-manager-types";

const CLIENT_ID_STORAGE_KEY = "agh.window-manager.client-id";

function randomClientId(): string {
  const cryptoApi = globalThis.crypto;
  if (typeof cryptoApi?.randomUUID === "function") {
    return `web-${cryptoApi.randomUUID()}`;
  }
  if (typeof cryptoApi?.getRandomValues === "function") {
    const bytes = cryptoApi.getRandomValues(new Uint8Array(16));
    return `web-${Array.from(bytes, value => value.toString(16).padStart(2, "0")).join("")}`;
  }
  return `web-${Date.now()}-${Math.random().toString(36).slice(2)}`;
}

function scheduleRetry(callback: () => void, delay: number): () => void {
  let active = true;
  const timer = setTimeout(() => {
    if (!active) return;
    active = false;
    callback();
  }, delay);
  return () => {
    if (!active) return;
    active = false;
    clearTimeout(timer);
  };
}

export function stableWindowManagerClientId(): string {
  if (typeof window === "undefined") return "web-server-render";
  const created = randomClientId();
  try {
    const existing = window.localStorage.getItem(CLIENT_ID_STORAGE_KEY)?.trim();
    if (existing) return existing;
    window.localStorage.setItem(CLIENT_ID_STORAGE_KEY, created);
  } catch {
    return created;
  }
  return created;
}

export interface WindowManagerClientRegistrationState {
  clientId: string;
  registrationEpoch: number;
  client: WindowManagerClientView | null;
  status: "idle" | "registering" | "registered" | "error";
  error: Error | null;
  reregister: () => void;
}

export function useWindowManagerClient(
  workspaceId: string | null,
  onClientChange: (client: WindowManagerClientView | null) => void,
  onErrorChange: (error: Error | null) => void = () => {}
): WindowManagerClientRegistrationState {
  const [clientId] = useState(stableWindowManagerClientId);
  const [client, setClient] = useState<WindowManagerClientView | null>(null);
  const [error, setError] = useState<Error | null>(null);
  const [status, setStatus] = useState<WindowManagerClientRegistrationState["status"]>("idle");
  const [attempt, setAttempt] = useState(0);
  const publishedWorkspace = useRef<string | null>(null);
  const retryCount = useRef(0);
  const cancelRetry = useRef<(() => void) | null>(null);
  const publishClient = useEffectEvent(onClientChange);
  const publishError = useEffectEvent(onErrorChange);

  useEffect(() => {
    if (workspaceId === null) {
      setClient(null);
      setError(null);
      publishError(null);
      setStatus("idle");
      publishClient(null);
      publishedWorkspace.current = null;
      retryCount.current = 0;
      return undefined;
    }

    const controller = new AbortController();
    let ownedCancelRetry: (() => void) | null = null;
    if (publishedWorkspace.current !== workspaceId) {
      setClient(null);
      publishClient(null);
      publishedWorkspace.current = workspaceId;
      retryCount.current = 0;
    }
    setError(null);
    publishError(null);
    setStatus("registering");

    void registerWindowManagerClient(workspaceId, clientId, undefined, controller.signal)
      .then(view => {
        if (controller.signal.aborted) return;
        if (view.workspaceId !== workspaceId || view.clientId !== clientId) {
          throw new Error("The daemon registered a different window-manager client.");
        }
        setClient(view);
        setStatus("registered");
        publishError(null);
        retryCount.current = 0;
        publishClient(view);
      })
      .catch(cause => {
        if (controller.signal.aborted) return;
        const nextError =
          cause instanceof Error ? cause : new Error("Unable to register this browser client.");
        setError(nextError);
        publishError(nextError);
        setClient(null);
        setStatus("error");
        publishClient(null);
        const delay = Math.min(8_000, 500 * 2 ** Math.min(retryCount.current, 4));
        const cancel = scheduleRetry(() => {
          if (cancelRetry.current === cancel) cancelRetry.current = null;
          retryCount.current += 1;
          setAttempt(current => current + 1);
        }, delay);
        ownedCancelRetry = cancel;
        cancelRetry.current = cancel;
      });

    return () => {
      controller.abort();
      ownedCancelRetry?.();
      if (cancelRetry.current === ownedCancelRetry) {
        cancelRetry.current = null;
      }
    };
  }, [attempt, clientId, workspaceId]);

  return {
    clientId,
    registrationEpoch: attempt,
    client,
    status,
    error,
    reregister: () => {
      cancelRetry.current?.();
      cancelRetry.current = null;
      retryCount.current = 0;
      setClient(null);
      setError(null);
      setStatus("registering");
      publishClient(null);
      setAttempt(current => current + 1);
    },
  };
}
