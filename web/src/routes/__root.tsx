import type { ReactNode } from "react";

import {
  Link,
  Outlet,
  createRootRoute,
  useRouter,
  type ErrorComponentProps,
  type NotFoundRouteProps,
} from "@tanstack/react-router";
import { AlertTriangle, Compass, RefreshCw } from "lucide-react";

import { Button, Empty, Toaster, TooltipProvider, buttonVariants } from "@agh/ui";

import { AppHeader } from "@/components/app-header";

export const Route = createRootRoute({
  component: RootComponent,
  errorComponent: RootRouteErrorBoundary,
  notFoundComponent: RootRouteNotFoundBoundary,
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

function RootRouteErrorBoundary({ error, reset }: ErrorComponentProps) {
  const router = useRouter();

  const handleRetry = () => {
    reset();
    void router.invalidate({ forcePending: true });
  };

  return (
    <RootBoundaryFrame testId="root-route-error">
      <Empty
        className="max-w-xl"
        description={describeRouteError(
          error,
          "The application shell failed before the route could render."
        )}
        icon={AlertTriangle}
        title="Unable to render this route"
        titleAs="h1"
        action={
          <>
            <Button onClick={handleRetry} size="sm" type="button" variant="outline">
              <RefreshCw className="size-3.5" />
              Retry
            </Button>
            <Link className={buttonVariants({ variant: "outline", size: "sm" })} to="/">
              <Compass className="size-3.5" />
              Go home
            </Link>
          </>
        }
      />
    </RootBoundaryFrame>
  );
}

function RootRouteNotFoundBoundary({ routeId }: NotFoundRouteProps) {
  return (
    <RootBoundaryFrame routeId={routeId} testId="root-route-not-found">
      <Empty
        className="max-w-xl"
        description="The requested route does not exist in this build."
        icon={Compass}
        title="Route not found"
        titleAs="h1"
        action={
          <Link className={buttonVariants({ variant: "outline", size: "sm" })} to="/">
            <Compass className="size-3.5" />
            Go home
          </Link>
        }
      />
    </RootBoundaryFrame>
  );
}

function RootBoundaryFrame({
  children,
  routeId,
  testId,
}: {
  children: ReactNode;
  routeId?: string;
  testId: string;
}) {
  return (
    <TooltipProvider>
      <div className="flex min-h-dvh flex-col bg-background text-foreground">
        <AppHeader />
        <main
          data-route-id={routeId}
          data-testid={testId}
          className="flex flex-1 items-center justify-center overflow-y-auto px-6 py-8"
        >
          {children}
        </main>
      </div>
      <Toaster />
    </TooltipProvider>
  );
}

function describeRouteError(error: unknown, fallback: string) {
  if (error instanceof Error && error.message.trim().length > 0) {
    return error.message;
  }

  return fallback;
}
