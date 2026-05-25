import { ChevronUp, Folder, FolderPlus, House, Plus, Spline } from "lucide-react";

import { Button, Spinner, cn } from "@agh/ui";

import type { OnboardingWorkspacesApi } from "../hooks/use-onboarding-workspaces";

interface DirectoryBrowserProps {
  workspaces: OnboardingWorkspacesApi;
}

export function DirectoryBrowser({ workspaces }: DirectoryBrowserProps) {
  const {
    currentPath,
    parent,
    home,
    entries,
    isBrowsing,
    browseError,
    isResolving,
    navigateTo,
    goToParent,
    goHome,
    addWorkspace,
    isAdded,
  } = workspaces;

  return (
    <div
      className="overflow-hidden rounded-md bg-canvas-soft ring-1 ring-inset ring-line"
      data-testid="onboarding-directory-browser"
    >
      <div className="flex items-center gap-2 border-b border-line px-3 py-2">
        <Button
          variant="ghost"
          size="icon-sm"
          onClick={goHome}
          disabled={!home}
          aria-label="Go to home directory"
        >
          <House className="size-3.5" />
        </Button>
        <Button
          variant="ghost"
          size="icon-sm"
          onClick={goToParent}
          disabled={!parent}
          aria-label="Go to parent directory"
        >
          <ChevronUp className="size-3.5" />
        </Button>
        <span className="truncate font-mono text-xs text-subtle" title={currentPath}>
          {currentPath || "~"}
        </span>
        <span className="ml-auto">
          <Button
            variant="outline"
            size="sm"
            onClick={() => void addWorkspace(currentPath)}
            disabled={!currentPath || isAdded(currentPath) || isResolving}
            data-testid="onboarding-add-current-dir"
          >
            {isResolving ? <Spinner /> : <Plus className="size-3.5" />}
            Use this folder
          </Button>
        </span>
      </div>

      <div className="max-h-64 overflow-y-auto p-1.5">
        {isBrowsing ? (
          <div className="flex items-center gap-2 px-2.5 py-6 text-sm text-muted">
            <Spinner /> Reading directory…
          </div>
        ) : browseError ? (
          <p className="px-2.5 py-6 text-sm text-danger" role="alert">
            {browseError}
          </p>
        ) : entries.length === 0 ? (
          <p className="px-2.5 py-6 text-sm text-faint">No sub-folders here.</p>
        ) : (
          entries.map(entry => (
            <div
              key={entry.path}
              className="group flex items-center gap-2.5 rounded px-2.5 py-1.5 hover:bg-hover"
            >
              <button
                type="button"
                onClick={() => navigateTo(entry.path)}
                className="flex min-w-0 flex-1 items-center gap-2.5 text-left"
                data-testid="onboarding-dir-entry"
              >
                {entry.is_dir ? (
                  <Folder className="size-4 flex-none text-warning" />
                ) : (
                  <Spline className="size-4 flex-none text-faint" />
                )}
                <span className="truncate text-sm text-fg">{entry.name}</span>
              </button>
              <Button
                variant="ghost"
                size="icon-sm"
                onClick={() => void addWorkspace(entry.path)}
                disabled={isAdded(entry.path) || isResolving}
                aria-label={`Add ${entry.name} as a workspace`}
                className={cn(
                  "opacity-0 transition-opacity group-hover:opacity-100",
                  isAdded(entry.path) && "opacity-40"
                )}
              >
                <FolderPlus className="size-3.5" />
              </Button>
            </div>
          ))
        )}
      </div>
    </div>
  );
}
