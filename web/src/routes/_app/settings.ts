import { createFileRoute } from "@tanstack/react-router";
import { Settings as SettingsIcon } from "lucide-react";

import type { TopbarRouteContext } from "@/types/topbar";
import {
  SettingsShellErrorBoundary,
  SettingsShellNotFoundBoundary,
} from "./-settings-shell-boundaries";
import { SettingsShell } from "./-settings-shell";

export const Route = createFileRoute("/_app/settings")({
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { title: "Settings", icon: SettingsIcon },
  }),
  component: SettingsShell,
  errorComponent: SettingsShellErrorBoundary,
  notFoundComponent: SettingsShellNotFoundBoundary,
});
