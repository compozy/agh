import { createFileRoute } from "@tanstack/react-router";
import { Home } from "lucide-react";

import type { TopbarRouteContext } from "@/types/topbar";
import { preloadHomeRoute } from "./-app-preload";
import { AppHomePage } from "./-home-page";

export const Route = createFileRoute("/_app/")({
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { title: "Home", icon: Home },
  }),
  loader: ({ context }) => preloadHomeRoute(context.queryClient),
  component: AppHomePage,
});
