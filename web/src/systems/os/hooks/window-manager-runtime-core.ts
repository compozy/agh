import type { QueryClient } from "@tanstack/react-query";
import { createStore, type StoreApi } from "zustand/vanilla";

import {
  executeWindowManagerCommand,
  fetchWindowManagerSnapshot,
  WindowManagerApiError,
} from "../adapters/window-manager-api";
import type { OsDesktopRuntimeStore, OsWallpaper } from "../lib/os-types";
import { reconcileWindowManagerSnapshot, windowManagerKeys } from "../lib/window-manager-query";
import type {
  PixelRect,
  WindowManagerClientView,
  WindowManagerCommandInput,
  WindowManagerConfig,
  WindowManagerConnectionStatus,
  WindowManagerDiagnosticPayload,
  WindowManagerSnapshot,
} from "../lib/window-manager-types";
import { DEFAULT_WINDOW_MANAGER_WORK_AREA } from "../lib/window-manager-view";
import { windowManagerStore } from "../stores/window-manager-store";

export interface WindowManagerRuntimeBinding {
  workspaceId: string;
  clientId: string;
}

export function randomWindowManagerId(prefix: string): string {
  if (typeof globalThis.crypto?.randomUUID === "function") {
    return `${prefix}-${globalThis.crypto.randomUUID()}`;
  }
  return `${prefix}-${Date.now()}-${Math.random().toString(36).slice(2)}`;
}

function commandDiagnostic(error: unknown): WindowManagerDiagnosticPayload {
  if (error instanceof WindowManagerApiError && error.payload?.diagnostics[0]) {
    return error.payload.diagnostics[0];
  }
  return {
    code:
      error instanceof WindowManagerApiError
        ? (error.payload?.code ?? "command_failed")
        : "command_failed",
    path: null,
    message: error instanceof Error ? error.message : "The window command failed.",
  };
}

/** Query/client lifecycle and semantic command transport shared by the OS runtime. */
export abstract class WindowManagerRuntimeCore {
  private readonly unsubscribeQuery: () => void;
  private readonly unsubscribePresentation: () => void;
  private runtimeStore: StoreApi<OsDesktopRuntimeStore> | null = null;
  protected readonly queryClient: QueryClient;
  protected binding: WindowManagerRuntimeBinding | null = null;
  protected client: WindowManagerClientView | null = null;
  protected wallpaper: OsWallpaper = "ember";
  protected reduceMotion = false;
  protected dockMagnify = true;
  protected railCollapsedAgentIds: readonly string[] = [];
  protected loadError: Error | null = null;

  constructor(queryClient: QueryClient) {
    this.queryClient = queryClient;
    this.unsubscribeQuery = queryClient.getQueryCache().subscribe(event => {
      const key = event.query.queryKey;
      const configKey = windowManagerKeys.config();
      if (
        key.length === configKey.length &&
        key.every((part: unknown, index: number) => part === configKey[index])
      ) {
        this.publish();
        return;
      }
      if (!this.binding) return;
      if (
        key[0] === windowManagerKeys.all[0] &&
        key[1] === windowManagerKeys.all[1] &&
        key[2] === "snapshot" &&
        key[3] === this.binding.workspaceId
      ) {
        this.publish();
      }
    });
    this.unsubscribePresentation = windowManagerStore.subscribe((state, previous) => {
      if (
        state.connectionStatus !== previous.connectionStatus ||
        state.workArea !== previous.workArea
      ) {
        this.publish();
      }
    });
  }

  protected abstract buildView(): OsDesktopRuntimeStore;

  protected initializeView(): void {
    this.runtimeStore = createStore<OsDesktopRuntimeStore>()(() => this.buildView());
  }

  private store(): StoreApi<OsDesktopRuntimeStore> {
    if (this.runtimeStore === null) {
      throw new Error("Window-manager runtime store is not initialized.");
    }
    return this.runtimeStore;
  }

  protected get view(): OsDesktopRuntimeStore {
    return this.store().getState();
  }

  getState = (): OsDesktopRuntimeStore => this.store().getState();

  getInitialState = (): OsDesktopRuntimeStore => this.store().getInitialState();

  subscribe = (listener: (state: OsDesktopRuntimeStore, previous: OsDesktopRuntimeStore) => void) =>
    this.store().subscribe(listener);

  destroy(): void {
    this.unsubscribeQuery();
    this.unsubscribePresentation();
  }

  bind(binding: WindowManagerRuntimeBinding): void {
    if (
      this.binding?.workspaceId === binding.workspaceId &&
      this.binding.clientId === binding.clientId
    ) {
      return;
    }
    this.binding = { ...binding };
    this.client = null;
    this.loadError = null;
    windowManagerStore.getState().actions.bindClient(binding);
    this.publish();
  }

  unbind(): void {
    this.binding = null;
    this.client = null;
    this.loadError = null;
    windowManagerStore.getState().actions.unbindClient();
    this.publish();
  }

  setClient(client: WindowManagerClientView | null): void {
    if (client === null) {
      if (this.client === null) return;
      this.client = null;
      this.publish();
      return;
    }
    const binding = this.binding;
    if (
      binding === null ||
      client.workspaceId !== binding.workspaceId ||
      client.clientId !== binding.clientId ||
      (this.client !== null && client.presentationRevision <= this.client.presentationRevision)
    ) {
      return;
    }
    this.client = client;
    const transition = windowManagerStore.getState().transitionIntent;
    if (transition?.mode === "instant" && client.activeDesktopId === transition.toDesktopId) {
      windowManagerStore.getState().actions.setTransitionIntent(null);
    }
    this.publish();
  }

  setConnectionStatus(status: WindowManagerConnectionStatus): void {
    windowManagerStore.getState().actions.setConnectionStatus(status);
  }

  setLoadError(error: Error | null): void {
    this.loadError = error;
    this.publish();
  }

  clearConflict(): void {
    windowManagerStore.getState().actions.clearConflict();
  }

  refreshSnapshot(): void {
    const binding = this.binding;
    if (binding === null) return;
    void fetchWindowManagerSnapshot(binding.workspaceId)
      .then(snapshot => {
        this.queryClient.setQueryData<WindowManagerSnapshot>(
          windowManagerKeys.snapshot(binding.workspaceId),
          current => reconcileWindowManagerSnapshot(current, snapshot)
        );
        this.setLoadError(null);
      })
      .catch(error => {
        this.setLoadError(
          error instanceof Error ? error : new Error("Unable to reload the window layout.")
        );
      });
  }

  protected publish(): void {
    this.runtimeStore?.setState(this.buildView(), true);
  }

  protected snapshot(): WindowManagerSnapshot | null {
    if (!this.binding) return null;
    return (
      this.queryClient.getQueryData<WindowManagerSnapshot>(
        windowManagerKeys.snapshot(this.binding.workspaceId)
      ) ?? null
    );
  }

  protected config(): WindowManagerConfig | null {
    return this.queryClient.getQueryData<WindowManagerConfig>(windowManagerKeys.config()) ?? null;
  }

  protected currentLoadError(): Error | null {
    const snapshotError = this.binding
      ? this.queryClient.getQueryState(windowManagerKeys.snapshot(this.binding.workspaceId))?.error
      : null;
    const configError = this.queryClient.getQueryState(windowManagerKeys.config())?.error;
    if (snapshotError instanceof Error) return snapshotError;
    if (configError instanceof Error) return configError;
    return this.loadError;
  }

  protected workArea(): PixelRect {
    return windowManagerStore.getState().workArea?.rect ?? DEFAULT_WINDOW_MANAGER_WORK_AREA;
  }

  private startDispatch(command: WindowManagerCommandInput): Promise<boolean> | null {
    const binding = this.binding;
    const snapshot = this.snapshot();
    if (binding === null || snapshot === null || this.client === null) {
      windowManagerStore.getState().actions.reportDiagnostic({
        code: "client_unavailable",
        message: "Window commands are unavailable until this browser reconnects.",
        severity: "warning",
        field: null,
      });
      this.publish();
      return null;
    }

    const requestId = randomWindowManagerId("wm-command");
    const actions = windowManagerStore.getState().actions;
    if (
      !actions.beginCommand({
        id: requestId,
        kind: command.commandId,
        expectedRevision: snapshot.revision,
      })
    ) {
      return null;
    }

    return executeWindowManagerCommand(
      binding.workspaceId,
      binding.clientId,
      snapshot.revision,
      command
    )
      .then(result => {
        this.queryClient.setQueryData<WindowManagerSnapshot>(
          windowManagerKeys.snapshot(binding.workspaceId),
          current => reconcileWindowManagerSnapshot(current, result.snapshot)
        );
        if (result.client !== null) this.setClient(result.client);
        const firstDiagnostic = result.diagnostics[0];
        actions.completeCommand(
          requestId,
          firstDiagnostic
            ? {
                code: firstDiagnostic.code,
                message: firstDiagnostic.message,
                severity: "warning",
                field: firstDiagnostic.path,
              }
            : undefined
        );
        this.publish();
        return result.applied;
      })
      .catch(error => {
        if (command.commandId === "desktop.switch") {
          windowManagerStore.getState().actions.setTransitionIntent(null);
        }
        const diagnostic = commandDiagnostic(error);
        const storeDiagnostic = {
          code: diagnostic.code,
          message: diagnostic.message,
          severity: "error" as const,
          field: diagnostic.path,
        };
        if (
          error instanceof WindowManagerApiError &&
          error.status === 409 &&
          error.payload?.currentRevision !== null
        ) {
          actions.recordConflict(
            {
              commandId: requestId,
              expectedRevision: snapshot.revision,
              currentRevision: error.payload?.currentRevision ?? snapshot.revision,
            },
            storeDiagnostic
          );
        } else {
          actions.failCommand(requestId, storeDiagnostic);
        }
        this.publish();
        return false;
      });
  }

  protected dispatch(command: WindowManagerCommandInput): boolean {
    const pending = this.startDispatch(command);
    if (pending === null) return false;
    void pending;
    return true;
  }

  protected dispatchConfirmed(command: WindowManagerCommandInput): Promise<boolean> {
    return this.startDispatch(command) ?? Promise.resolve(false);
  }
}
