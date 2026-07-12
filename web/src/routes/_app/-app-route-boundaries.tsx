import {
  Link,
  useRouter,
  type ErrorComponentProps,
  type NotFoundRouteProps,
} from "@tanstack/react-router";
import { AlertTriangle, Compass, RefreshCw } from "lucide-react";

import { Button, Empty, buttonVariants } from "@agh/ui";

import { AppRouteBoundaryFrame } from "./-app-route-boundary-frame";

function describeRouteError(error: unknown, fallback: string) {
  if (error instanceof Error && error.message.trim().length > 0) return error.message;
  return fallback;
}

export function AppRouteErrorBoundary({ error, reset }: ErrorComponentProps) {
  const router = useRouter();
  const handleRetry = () => {
    reset();
    void router.invalidate({ forcePending: true });
  };

  return (
    <AppRouteBoundaryFrame testId="app-route-error">
      <Empty
        className="max-w-xl"
        description={describeRouteError(error, "The requested app route could not be rendered.")}
        icon={AlertTriangle}
        title="Unable to load this page"
        titleAs="h1"
        action={
          <>
            <Button onClick={handleRetry} size="sm" type="button" variant="outline">
              <RefreshCw className="size-3" />
              Retry
            </Button>
            <Link className={buttonVariants({ variant: "outline", size: "sm" })} to="/">
              <Compass className="size-3" />
              Go home
            </Link>
          </>
        }
      />
    </AppRouteBoundaryFrame>
  );
}

export function AppRouteNotFoundBoundary({ routeId }: NotFoundRouteProps) {
  return (
    <AppRouteBoundaryFrame routeId={routeId} testId="app-route-not-found">
      <Empty
        className="max-w-xl"
        description="The requested app route does not exist."
        icon={Compass}
        title="Page not found"
        titleAs="h1"
        action={
          <Link className={buttonVariants({ variant: "outline", size: "sm" })} to="/">
            <Compass className="size-3" />
            Go home
          </Link>
        }
      />
    </AppRouteBoundaryFrame>
  );
}
