import { createFileRoute } from "@tanstack/react-router";
import { Wrench } from "lucide-react";

import type { TopbarRouteContext } from "@/types/topbar";
import { preloadSettingsSkillsRoute } from "../-settings-preload";
import { SkillsSettingsPage } from "./-skills-settings-page";

export const Route = createFileRoute("/_app/settings/skills")({
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { title: "Skills settings", icon: Wrench },
  }),
  loader: ({ context }) => preloadSettingsSkillsRoute(context.queryClient),
  component: SkillsSettingsPage,
});
