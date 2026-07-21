import { AlertTriangle, RefreshCw } from "lucide-react";
import type { ReactNode } from "react";

import { Button, Empty, Spinner } from "@agh/ui";

import { OnboardingWizard, useOnboardingStatus } from "@/systems/onboarding";

/**
 * Desktop-level onboarding gate (rewrite of the old `-onboarding-gate-frame`):
 * first-run setup renders before any desktop chrome exists.
 */
export function DesktopGate({ children }: { children: ReactNode }) {
  const onboarding = useOnboardingStatus();

  if (onboarding.data?.completed === true) {
    return children;
  }

  if (onboarding.data?.completed === false) {
    return <OnboardingWizard onComplete={() => void onboarding.refetch()} />;
  }

  if (onboarding.isError) {
    return (
      <GateFrame testId="onboarding-gate-error">
        <Empty
          className="max-w-xl"
          description={describeGateError(
            onboarding.error,
            "AGH could not confirm whether first-run setup is complete."
          )}
          icon={AlertTriangle}
          title="Unable to check onboarding"
          titleAs="h1"
          action={
            <Button
              onClick={() => void onboarding.refetch()}
              size="sm"
              type="button"
              variant="outline"
            >
              <RefreshCw className="size-3" />
              Retry
            </Button>
          }
        />
      </GateFrame>
    );
  }

  return (
    <GateFrame testId="onboarding-gate-loading">
      <Spinner />
    </GateFrame>
  );
}

function GateFrame({ children, testId }: { children: ReactNode; testId: string }) {
  return (
    <main
      id="app-content"
      data-testid={testId}
      className="flex min-h-0 flex-1 items-center justify-center bg-canvas"
    >
      {children}
    </main>
  );
}

function describeGateError(error: unknown, fallback: string) {
  if (error instanceof Error && error.message.trim().length > 0) {
    return error.message;
  }
  return fallback;
}
