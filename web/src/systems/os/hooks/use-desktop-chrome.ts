import { useRouter } from "@tanstack/react-router";
import { useEffect, useRef, useState } from "react";
import { toast } from "sonner";
import { useStore } from "zustand";

import { useActiveWorkspaceStore } from "@/systems/workspace";

import type { OsShellHandle } from "../contexts/os-shell-context";
import { createDesktopPersistence } from "../lib/desktop-persistence";
import { OsStateClient, type OsSocket } from "../lib/os-state-client";
import { RoutingCoordinator, type OsRouterPort } from "../lib/routing-coordinator";
import { OS_COMPACT_BREAKPOINT, type OsWallpaper, type OsWindowLocation } from "../lib/os-types";
import { desktopStore } from "../stores/desktop-store";

interface MutableShellHandle extends OsShellHandle {
  setFlush(flush: () => void): void;
}

export interface DesktopChromeModel {
  shell: OsShellHandle;
  wallpaper: OsWallpaper;
}

function browserSocketFactory(url: string): OsSocket {
  const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
  const nativeSocket = new WebSocket(`${protocol}//${window.location.host}${url}`);
  const socket: OsSocket = {
    send: data => nativeSocket.send(data),
    close: () => nativeSocket.close(),
    onopen: null,
    onmessage: null,
    onclose: null,
    onerror: null,
  };
  nativeSocket.onopen = () => socket.onopen?.();
  nativeSocket.onmessage = event => socket.onmessage?.({ data: event.data });
  nativeSocket.onclose = () => socket.onclose?.();
  nativeSocket.onerror = () => socket.onerror?.();
  return socket;
}

function navigateTo(
  router: ReturnType<typeof useRouter>,
  location: OsWindowLocation,
  replace: boolean
): void {
  const href = `${location.pathname}${router.options.stringifySearch(location.search)}`;
  if (replace) router.history.replace(href);
  else router.history.push(href);
}

/**
 * Desktop chrome lifecycle: the shell handle (store + coordinator + flush),
 * the per-workspace desktop-state client (start/stop/rebind on switch — rule
 * 8), hydration completion (rule 4), viewport-derived presentation, and the
 * one-shot soft-cap guidance toast (US-001.EC-3).
 */
export function useDesktopChrome(activeWorkspaceId: string | null): DesktopChromeModel {
  const router = useRouter();
  const [shell] = useState<MutableShellHandle>(() => {
    let flush: () => void = () => {};
    const port: OsRouterPort = {
      navigate: location => navigateTo(router, location, false),
      replace: location => navigateTo(router, location, true),
    };
    return {
      store: desktopStore,
      coordinator: new RoutingCoordinator(desktopStore, port),
      flushPersistence: () => flush(),
      setFlush: next => {
        flush = next;
      },
    };
  });

  const previousWorkspaceRef = useRef<string | null>(null);

  // A workspace-selection change flips the coordinator into its switch cycle
  // SYNCHRONOUSLY — before the router can commit a cross-workspace deep link.
  // A route match landing between the selection change and the lifecycle
  // effect below is then held as the switch's focus intent instead of being
  // reconciled into the outgoing space's store (US-016.EC-2, no leak).
  useEffect(() => {
    return useActiveWorkspaceStore.subscribe(state => {
      const next = state.selectedWorkspaceId;
      const current = previousWorkspaceRef.current;
      if (current !== null && next !== null && next !== current) {
        shell.coordinator.beginWorkspaceSwitch();
      }
    });
  }, [shell]);

  // Desktop-state client lifecycle: one client per active workspace; a switch
  // tears the old one down (flushing the outgoing space's pending writes as
  // one batch), resets the desktop, and rehydrates (rule 8).
  useEffect(() => {
    if (activeWorkspaceId === null) return;
    if (
      previousWorkspaceRef.current !== null &&
      previousWorkspaceRef.current !== activeWorkspaceId
    ) {
      shell.coordinator.beginWorkspaceSwitch();
      desktopStore.getState().resetForWorkspace();
    }
    previousWorkspaceRef.current = activeWorkspaceId;

    const persistence = createDesktopPersistence(desktopStore);
    const client = new OsStateClient({
      workspaceId: activeWorkspaceId,
      socketFactory: browserSocketFactory,
      callbacks: persistence.callbacks,
    });
    const unbind = persistence.bind(client);
    shell.setFlush(() => client.flush());
    client.start();
    const handleBeforeUnload = () => client.flush();
    window.addEventListener("beforeunload", handleBeforeUnload);
    return () => {
      window.removeEventListener("beforeunload", handleBeforeUnload);
      client.flush();
      shell.setFlush(() => {});
      unbind();
      client.stop();
    };
  }, [activeWorkspaceId, shell]);

  // Hydration completion drives the URL truth-up / deep-link intent (rule 4).
  const hydration = useStore(desktopStore, state => state.hydration);
  useEffect(() => {
    if (hydration !== "pending") shell.coordinator.completeHydration();
  }, [hydration, shell]);

  // Presentation mode is derived from the viewport, never persisted.
  useEffect(() => {
    const query = window.matchMedia(`(max-width: ${OS_COMPACT_BREAKPOINT - 1}px)`);
    const apply = () =>
      desktopStore.getState().setPresentation(query.matches ? "compact" : "floating");
    apply();
    query.addEventListener("change", apply);
    return () => query.removeEventListener("change", apply);
  }, []);

  // Soft-cap guidance (US-001.EC-3): one toast when the 13th window opens.
  const softCapNotice = useStore(desktopStore, state => state.softCapNotice);
  useEffect(() => {
    if (!softCapNotice) return;
    toast("More than 12 windows open", {
      description: "Everything keeps working — minimizing idle windows keeps the desktop fluid.",
    });
    desktopStore.getState().clearSoftCapNotice();
  }, [softCapNotice]);

  const wallpaper = useStore(desktopStore, state => state.wallpaper);

  return { shell, wallpaper };
}
