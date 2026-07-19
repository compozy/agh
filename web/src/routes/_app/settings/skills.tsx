import { createFileRoute } from "@tanstack/react-router";

import type { TopbarRouteContext } from "@/types/topbar";
import { preloadSettingsSkillsRoute } from "../-settings-preload";
import { SkillsSettingsPage } from "./-skills-settings-page";

export const Route = createFileRoute("/_app/settings/skills")({
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { crumb: { label: "Skills" } },
  }),
  loader: ({ context }) => preloadSettingsSkillsRoute(context.queryClient),
  component: SkillsSettingsPage,
});
