import { useQuery } from "@tanstack/react-query";
import { AlertCircle } from "lucide-react";

import { Button, Spinner } from "@agh/ui";

import { windowManagerConfigOptions } from "@/systems/os";
import {
  SettingsGroup,
  SettingsPageFrame,
  useSettingsTopbar,
  WindowManagerConfigEditor,
  WindowManagerLayoutDocumentEditor,
  windowManagerLayoutOptions,
  windowManagerLayoutProfilesOptions,
} from "@/systems/settings";
import { useActiveWorkspace } from "@/systems/workspace";

function LoadingState() {
  return (
    <div
      className="flex flex-1 items-center justify-center"
      data-testid="settings-page-layouts-loading"
    >
      <Spinner className="size-5 text-subtle" />
    </div>
  );
}

function ErrorState({ error, onRetry }: { error: Error; onRetry: () => void }) {
  return (
    <div
      className="flex flex-1 items-center justify-center"
      data-testid="settings-page-layouts-error"
    >
      <div className="flex max-w-sm flex-col items-center gap-2 text-center">
        <AlertCircle aria-hidden="true" className="size-6 text-danger" />
        <p className="text-small-body text-subtle">{error.message}</p>
        <Button size="cta-lg" type="button" variant="outline" onClick={onRetry}>
          Retry
        </Button>
      </div>
    </div>
  );
}

/** Global window-manager defaults plus the active workspace's authoritative layout document. */
export function LayoutsSettingsPage() {
  useSettingsTopbar("layouts");
  const workspace = useActiveWorkspace();
  const workspaceId = workspace.activeWorkspaceId ?? "";
  const settings = useQuery(windowManagerConfigOptions());
  const profiles = useQuery(windowManagerLayoutProfilesOptions());
  const layout = useQuery(windowManagerLayoutOptions(workspaceId));
  const waitingForWorkspace = !workspace.hasHydrated || workspace.isLoading;
  const isPending =
    settings.isPending ||
    profiles.isPending ||
    waitingForWorkspace ||
    (workspaceId !== "" && layout.isPending);
  const error =
    settings.error instanceof Error
      ? settings.error
      : profiles.error instanceof Error
        ? profiles.error
        : workspace.error instanceof Error
          ? workspace.error
          : layout.error instanceof Error
            ? layout.error
            : null;

  if (isPending) return <LoadingState />;
  if (error !== null) {
    return (
      <ErrorState
        error={error}
        onRetry={() => {
          void Promise.all([
            settings.refetch(),
            profiles.refetch(),
            workspace.refetch(),
            ...(workspaceId === "" ? [] : [layout.refetch()]),
          ]);
        }}
      />
    );
  }
  if (!settings.data) return <LoadingState />;

  const profileRecords = profiles.data ?? [];
  const activeWorkspaceName = workspace.activeWorkspace?.name ?? null;
  const meta = [
    activeWorkspaceName ? { key: "workspace", content: <span>{activeWorkspaceName}</span> } : null,
    layout.data
      ? {
          key: "revision",
          content: <span>Revision {layout.data.revision}</span>,
        }
      : null,
    {
      key: "profiles",
      content: (
        <span>
          {profileRecords.length} profile{profileRecords.length === 1 ? "" : "s"}
        </span>
      ),
    },
  ].filter((entry): entry is NonNullable<typeof entry> => entry !== null);

  return (
    <SettingsPageFrame
      description="Control global window behavior and review the active workspace layout before applying it."
      meta={meta}
      slug="layouts"
      wide
    >
      <WindowManagerConfigEditor key={JSON.stringify(settings.data)} config={settings.data} />
      {workspaceId === "" || !layout.data ? (
        <SettingsGroup
          title="Workspace layout"
          description="Layout documents are workspace-scoped daemon state."
        >
          <p className="px-4 py-5 text-form-label text-subtle">
            Select a workspace to export, validate, preview, or apply its layout.
          </p>
        </SettingsGroup>
      ) : (
        <WindowManagerLayoutDocumentEditor
          key={`${workspaceId}:${layout.data.revision}`}
          initial={layout.data}
          profiles={profileRecords}
          workspaceId={workspaceId}
        />
      )}
    </SettingsPageFrame>
  );
}
