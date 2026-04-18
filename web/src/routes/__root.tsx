import { Outlet, createRootRoute } from "@tanstack/react-router";
import { ThemeProvider } from "next-themes";

import { Toaster, TooltipProvider } from "@agh/ui";

export const Route = createRootRoute({
  component: RootComponent,
});

function RootComponent() {
  return (
    <ThemeProvider attribute="class" defaultTheme="system" enableSystem disableTransitionOnChange>
      <TooltipProvider>
        <Outlet />
        <Toaster />
      </TooltipProvider>
    </ThemeProvider>
  );
}
