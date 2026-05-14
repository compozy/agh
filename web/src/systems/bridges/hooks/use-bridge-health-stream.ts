import { useEffect } from "react";
import { useQueryClient } from "@tanstack/react-query";

import { bridgeKeys } from "../lib/query-keys";
import type {
  BridgeDetailResponse,
  BridgeListFilter,
  BridgeRoute,
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
  filters?: BridgeListFilter;
}

const BRIDGE_HEALTH_STREAM_URL = "/api/bridges/health/stream";

function defaultEventSourceFactory(url: string): BridgeHealthEventSource {
  return new EventSource(url);
}

function normalizeOptionalText(value?: string | null): string | undefined {
  if (typeof value !== "string") {
    return undefined;
  }

  const normalized = value.trim();
  return normalized === "" ? undefined : normalized;
}

function buildBridgeHealthStreamUrl(filters: BridgeListFilter = {}) {
  const params = new URLSearchParams();
  const scope = normalizeOptionalText(filters.scope);
  const workspaceId = normalizeOptionalText(filters.workspace_id);
  const workspace = normalizeOptionalText(filters.workspace);

  if (scope) {
    params.set("scope", scope);
  }
  if (workspaceId) {
    params.set("workspace_id", workspaceId);
  }
  if (workspace) {
    params.set("workspace", workspace);
  }

  const query = params.toString();
  return query ? `${BRIDGE_HEALTH_STREAM_URL}?${query}` : BRIDGE_HEALTH_STREAM_URL;
}

function invalidateBridgeRoutesWhenCountChanges(
  queryClient: ReturnType<typeof useQueryClient>,
  bridgeID: string,
  nextRouteCount: number | undefined
) {
  if (nextRouteCount === undefined) {
    return;
  }

  const cachedRoutes = queryClient.getQueryData<BridgeRoute[] | undefined>(
    bridgeKeys.routes(bridgeID)
  );
  if (cachedRoutes !== undefined && cachedRoutes.length === nextRouteCount) {
    return;
  }

  void queryClient.invalidateQueries({ queryKey: bridgeKeys.routes(bridgeID) });
}

function mergeBridgeHealthSnapshot(
  current: BridgesListResponse | undefined,
  snapshot: BridgeHealthStreamSnapshot
): BridgesListResponse | undefined {
  if (!current) {
    return current;
  }

  const visibleBridgeIds = new Set(current.bridges.map(bridge => bridge.id));
  const bridge_health = Object.fromEntries(
    Object.entries(snapshot.bridge_health).filter(([bridgeID]) => visibleBridgeIds.has(bridgeID))
  );
  return {
    ...current,
    bridge_health,
  };
}

export function applyBridgeHealthSnapshot(
  queryClient: ReturnType<typeof useQueryClient>,
  snapshot: BridgeHealthStreamSnapshot
) {
  for (const [queryKey] of queryClient.getQueriesData<BridgesListResponse | undefined>({
    queryKey: bridgeKeys.lists(),
  })) {
    queryClient.setQueryData<BridgesListResponse | undefined>(queryKey, current =>
      mergeBridgeHealthSnapshot(current, snapshot)
    );
  }

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
    invalidateBridgeRoutesWhenCountChanges(queryClient, bridgeID, health.route_count);
  }
}

export function useBridgeHealthStream(options?: UseBridgeHealthStreamOptions) {
  const enabled = options?.enabled ?? true;
  const eventSourceFactory = options?.eventSourceFactory ?? defaultEventSourceFactory;
  const hasCustomFactory = Boolean(options?.eventSourceFactory);
  const scope = options?.filters?.scope;
  const workspaceId = options?.filters?.workspace_id;
  const workspace = options?.filters?.workspace;
  const queryClient = useQueryClient();

  useEffect(() => {
    if (
      !enabled ||
      typeof window === "undefined" ||
      (!hasCustomFactory && typeof EventSource === "undefined")
    ) {
      return undefined;
    }

    const source = eventSourceFactory(
      buildBridgeHealthStreamUrl({ scope, workspace_id: workspaceId, workspace })
    );
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
  }, [enabled, eventSourceFactory, hasCustomFactory, queryClient, scope, workspaceId, workspace]);
}
