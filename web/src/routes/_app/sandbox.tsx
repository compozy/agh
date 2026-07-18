import { createFileRoute } from "@tanstack/react-router";

import type { TopbarRouteContext } from "@/types/topbar";
import { SandboxPage } from "./-sandbox-page";
import { preloadSandboxRoute } from "./-settings-preload";

export const Route = createFileRoute("/_app/sandbox")({
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { crumb: { label: "Sandbox", to: "/sandbox" } },
  }),
  loader: ({ context }) => preloadSandboxRoute(context.queryClient),
  component: SandboxPage,
});
