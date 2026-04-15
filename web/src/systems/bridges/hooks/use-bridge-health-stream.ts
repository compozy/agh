import { useEffect } from "react";
import { useQueryClient } from "@tanstack/react-query";

import { bridgeKeys } from "../lib/query-keys";
import type {
  BridgeDetailResponse,
  BridgesListResponse,
  BridgeHealthStreamSnapshot,
} from "../types";

interface BridgeHealthEventSource {
  addEventListener: (type: string, listener: EventListenerOrEventListenerObject) => void;
  close: () => void;
  onerror: ((event: Event) => void) | null;
  removeEventListener?: (type: string, listener: EventListenerOrEventListenerObject) => void;
}

interface UseBridgeHealthStreamOptions {
  enabled?: boolean;
  eventSourceFactory?: (url: string) => BridgeHealthEventSource;
}

const BRIDGE_HEALTH_STREAM_URL = "/api/bridges/health/stream";

function defaultEventSourceFactory(url: string): BridgeHealthEventSource {
  return new EventSource(url);
}

export function applyBridgeHealthSnapshot(
  queryClient: ReturnType<typeof useQueryClient>,
  snapshot: BridgeHealthStreamSnapshot
) {
  queryClient.setQueryData<BridgesListResponse | undefined>(bridgeKeys.list(), current =>
    current
      ? {
          ...current,
          bridge_health: snapshot.bridge_health,
        }
      : current
  );

  for (const [bridgeID, health] of Object.entries(snapshot.bridge_health)) {
    queryClient.setQueryData<BridgeDetailResponse | undefined>(
      bridgeKeys.detail(bridgeID),
      current =>
        current
          ? {
              ...current,
              health,
            }
          : current
    );
  }
}

export function useBridgeHealthStream(options?: UseBridgeHealthStreamOptions) {
  const enabled = options?.enabled ?? true;
  const eventSourceFactory = options?.eventSourceFactory ?? defaultEventSourceFactory;
  const hasCustomFactory = Boolean(options?.eventSourceFactory);
  const queryClient = useQueryClient();

  useEffect(() => {
    if (
      !enabled ||
      typeof window === "undefined" ||
      (!hasCustomFactory && typeof EventSource === "undefined")
    ) {
      return undefined;
    }

    const source = eventSourceFactory(BRIDGE_HEALTH_STREAM_URL);
    const handleSnapshot = (event: Event) => {
      if (!(event instanceof MessageEvent) || typeof event.data !== "string") {
        return;
      }

      try {
        const snapshot = JSON.parse(event.data) as BridgeHealthStreamSnapshot;
        applyBridgeHealthSnapshot(queryClient, snapshot);
      } catch (error) {
        console.error("Failed to parse bridge health snapshot", error);
      }
    };

    const handleError = (event: Event) => {
      console.error("Bridge health stream failed", event);
    };

    source.addEventListener("snapshot", handleSnapshot);
    source.onerror = handleError;

    return () => {
      source.removeEventListener?.("snapshot", handleSnapshot);
      source.onerror = null;
      source.close();
    };
  }, [enabled, eventSourceFactory, hasCustomFactory, queryClient]);
}
