import type { Meta } from "@storybook/react-vite";
import { useEffect } from "react";

import { useSidebarStore } from "@/hooks/use-sidebar-store";
import type { SettingsSectionName } from "@/systems/settings";
import { useSettingsRestartStore } from "@/systems/settings/stores/use-settings-restart-store";
import { useSessionStore } from "@/systems/session/hooks/use-session-store";
import { permissionRequestFixture } from "@/systems/session/mocks";
import { useActiveWorkspaceStore } from "@/systems/workspace/hooks/use-active-workspace-store";

export function StorybookRouteCanvas() {
  return null;
}

export function createRouteStoryMeta(
  title: string,
  description: string
): Meta<typeof StorybookRouteCanvas> {
  return {
    title,
    component: StorybookRouteCanvas,
    parameters: {
      layout: "fullscreen",
      docs: {
        description: {
          component: description,
        },
      },
    },
  };
}

export function appRouteParameters(path: string) {
  return {
    layout: "fullscreen" as const,
    router: {
      kind: "app" as const,
      initialEntries: [path],
    },
  };
}

export function StorybookWorkspaceSetup({
  workspaceId = "ws_storybook",
}: {
  workspaceId?: string;
}) {
  useEffect(() => {
    useSidebarStore.getState().setCollapsed(false);
    useActiveWorkspaceStore.getState().setSelectedWorkspaceId(workspaceId);
  }, [workspaceId]);

  return null;
}

export function StorybookSessionPermissionSetup({
  sessionId = "sess-storybook",
}: {
  sessionId?: string;
}) {
  useEffect(() => {
    useSessionStore.setState({
      activeSessionId: sessionId,
      isStreaming: false,
      historyMessages: [],
      liveMessages: [],
      awaitingTranscriptSync: false,
      pendingPermission: permissionRequestFixture,
    });
  }, [sessionId]);

  return null;
}

export function StorybookRestartBannerSetup({ section }: { section: SettingsSectionName }) {
  useEffect(() => {
    useSettingsRestartStore.getState().recordMutation({
      section,
      restartRequired: true,
      restartScope: "global",
      warnings: [],
      completedAt: "2026-04-18T01:00:00Z",
    });
  }, [section]);

  return null;
}
