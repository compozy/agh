import {
  Link,
  useRouter,
  type ErrorComponentProps,
  type NotFoundRouteProps,
} from "@tanstack/react-router";
import { AlertTriangle, RefreshCw, Settings as SettingsIcon } from "lucide-react";

import { Button, buttonVariants } from "@agh/ui";

import { SETTINGS_SECTIONS, settingsSectionPath } from "@/systems/settings";
import { SettingsShellFrame } from "./-settings-shell-frame";
import { SettingsShellState } from "./-settings-shell-state";

function defaultSettingsSectionPath() {
  return settingsSectionPath(SETTINGS_SECTIONS[0].slug);
}

function describeRouteError(error: unknown, fallback: string) {
  if (error instanceof Error && error.message.trim().length > 0) return error.message;
  return fallback;
}

export function SettingsShellErrorBoundary({ error, reset }: ErrorComponentProps) {
  const router = useRouter();
  const handleRetry = () => {
    reset();
    void router.invalidate({ forcePending: true });
  };

  return (
    <SettingsShellFrame testId="settings-shell-error">
      <SettingsShellState
        action={
          <>
            <Button onClick={handleRetry} size="sm" type="button" variant="outline">
              <RefreshCw className="size-3" />
              Retry
            </Button>
            <Link
              className={buttonVariants({ variant: "outline", size: "sm" })}
              to={defaultSettingsSectionPath()}
            >
              <SettingsIcon className="size-3" />
              Open general settings
            </Link>
          </>
        }
        description={describeRouteError(
          error,
          "The selected settings section failed before it could render."
        )}
        icon={AlertTriangle}
        title="Unable to load this settings page"
      />
    </SettingsShellFrame>
  );
}

export function SettingsShellNotFoundBoundary({ routeId }: NotFoundRouteProps) {
  return (
    <SettingsShellFrame routeId={routeId} testId="settings-shell-not-found">
      <SettingsShellState
        action={
          <Link
            className={buttonVariants({ variant: "outline", size: "sm" })}
            to={defaultSettingsSectionPath()}
          >
            <SettingsIcon className="size-3" />
            Open general settings
          </Link>
        }
        description="The requested settings section does not exist in this build."
        icon={SettingsIcon}
        title="Settings section not found"
      />
    </SettingsShellFrame>
  );
}
