import { useCallback, useEffect, useMemo, useState } from "react";

import { useDeleteWorkspace, useResolveWorkspace, useWorkspaces } from "@/systems/workspace";
import type { WorkspacePayload } from "@/systems/workspace";

import { useDirectoryBrowser } from "./use-directory-browser";
import {
  useOnboardingDraftStore,
  type OnboardingWorkspaceDraft,
} from "../stores/use-onboarding-draft-store";
import type { FSEntry } from "../types";

export interface OnboardingWorkspacesApi {
  currentPath: string;
  parent: string | null;
  home: string | null;
  entries: FSEntry[];
  isBrowsing: boolean;
  browseError: string | null;
  workspaces: OnboardingWorkspaceDraft[];
  isResolving: boolean;
  resolveError: string | null;
  navigateTo: (path: string) => void;
  goToParent: () => void;
  goHome: () => void;
  addWorkspace: (path: string) => Promise<void>;
  removeWorkspace: (path: string) => Promise<void>;
  isAdded: (path: string) => boolean;
  isRemoving: boolean;
}

function basename(path: string): string {
  const trimmed = path.replace(/\/+$/, "");
  const segment = trimmed.split("/").pop();
  return segment && segment.length > 0 ? segment : trimmed;
}

function draftFromWorkspace(
  workspace: Pick<WorkspacePayload, "id" | "name" | "root_dir">,
  fallbackPath: string
): OnboardingWorkspaceDraft {
  const path = workspace.root_dir.trim() || fallbackPath;
  return {
    path,
    name: workspace.name || basename(path),
    workspaceId: workspace.id,
  };
}

export function useOnboardingWorkspaces(): OnboardingWorkspacesApi {
  const workspaces = useOnboardingDraftStore(state => state.workspaces);
  const addToDraft = useOnboardingDraftStore(state => state.addWorkspace);
  const removeFromDraft = useOnboardingDraftStore(state => state.removeWorkspace);
  const resolveWorkspace = useResolveWorkspace();
  const deleteWorkspace = useDeleteWorkspace();
  const registeredWorkspaces = useWorkspaces();
  const [currentPath, setCurrentPath] = useState<string>("");
  const [resolveError, setResolveError] = useState<string | null>(null);

  const browse = useDirectoryBrowser({ path: currentPath || undefined, dirsOnly: true });
  const data = browse.data;

  useEffect(() => {
    if (workspaces.length > 0) {
      return;
    }
    for (const workspace of registeredWorkspaces.data ?? []) {
      const path = workspace.root_dir.trim();
      if (path.length === 0) {
        continue;
      }
      addToDraft(draftFromWorkspace(workspace, path));
    }
  }, [addToDraft, registeredWorkspaces.data, workspaces.length]);

  const navigateTo = useCallback((path: string) => {
    setCurrentPath(path);
  }, []);

  const goToParent = useCallback(() => {
    if (data?.parent) {
      setCurrentPath(data.parent);
    }
  }, [data?.parent]);

  const goHome = useCallback(() => {
    if (data?.home) {
      setCurrentPath(data.home);
    }
  }, [data?.home]);

  const addWorkspace = useCallback(
    async (path: string) => {
      const trimmed = path.trim();
      if (trimmed.length === 0 || workspaces.some(item => item.path === trimmed)) {
        return;
      }
      setResolveError(null);
      try {
        const workspace = await resolveWorkspace.mutateAsync({ path: trimmed });
        addToDraft(draftFromWorkspace(workspace, trimmed));
      } catch (error) {
        setResolveError(
          error instanceof Error ? error.message : "Failed to register that folder as a workspace."
        );
      }
    },
    [addToDraft, resolveWorkspace, workspaces]
  );

  const removeWorkspace = useCallback(
    async (path: string) => {
      const draft = workspaces.find(item => item.path === path);
      const workspaceId =
        draft?.workspaceId ??
        registeredWorkspaces.data?.find(workspace => workspace.root_dir === path)?.id ??
        "";
      if (workspaceId.length === 0) {
        removeFromDraft(path);
        return;
      }

      setResolveError(null);
      try {
        await deleteWorkspace.mutateAsync(workspaceId);
      } catch (error) {
        setResolveError(error instanceof Error ? error.message : "Failed to remove workspace.");
        return;
      }
      removeFromDraft(path);
    },
    [deleteWorkspace, registeredWorkspaces.data, removeFromDraft, workspaces]
  );

  const isAdded = useCallback(
    (path: string) => workspaces.some(item => item.path === path),
    [workspaces]
  );

  return {
    currentPath: data?.path ?? currentPath,
    parent: data?.parent ?? null,
    home: data?.home ?? null,
    entries: useMemo(() => data?.entries ?? [], [data?.entries]),
    isBrowsing: browse.isLoading || browse.isFetching,
    browseError: browse.error
      ? browse.error instanceof Error
        ? browse.error.message
        : "Failed to browse directory."
      : null,
    workspaces,
    isResolving: resolveWorkspace.isPending,
    isRemoving: deleteWorkspace.isPending,
    resolveError,
    navigateTo,
    goToParent,
    goHome,
    addWorkspace,
    removeWorkspace,
    isAdded,
  };
}
