import { createFileRoute } from "@tanstack/react-router";

import type { TopbarRouteContext } from "@/types/topbar";
import {
  SettingsShellErrorBoundary,
  SettingsShellNotFoundBoundary,
} from "./-settings-shell-boundaries";
import { SettingsShell } from "./-settings-shell";

export const Route = createFileRoute("/_app/settings")({
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { crumb: { label: "Settings", to: "/settings" } },
  }),
  component: SettingsShell,
  errorComponent: SettingsShellErrorBoundary,
  notFoundComponent: SettingsShellNotFoundBoundary,
});
