import { Outlet, createRootRoute } from "@tanstack/react-router";

import { Toaster, TooltipProvider } from "@agh/ui";

import { AppHeader } from "@/components/app-header";

export const Route = createRootRoute({
  component: RootComponent,
});

function RootComponent() {
  return (
    <TooltipProvider>
      <div
        data-testid="app-shell"
        className="flex h-screen flex-col overflow-hidden bg-background text-foreground"
      >
        <AppHeader />
        <div className="flex min-h-0 flex-1 overflow-hidden">
          <Outlet />
        </div>
      </div>
      <Toaster />
    </TooltipProvider>
  );
}
