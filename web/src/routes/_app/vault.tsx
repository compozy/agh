import { createFileRoute } from "@tanstack/react-router";
import { KeyRound } from "lucide-react";

import type { TopbarRouteContext } from "@/types/topbar";
import { VaultPage } from "./-vault-page";
import { preloadVaultRoute } from "./-vault-preload";

export const Route = createFileRoute("/_app/vault")({
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { title: "Vault", icon: KeyRound },
  }),
  loader: ({ context }) => preloadVaultRoute(context.queryClient),
  component: VaultPage,
});
