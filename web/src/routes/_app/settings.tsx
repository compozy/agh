import {
  Link,
  Outlet,
  createFileRoute,
  useMatchRoute,
  useRouter,
  type ErrorComponentProps,
  type NotFoundRouteProps,
} from "@tanstack/react-router";
import { AlertTriangle, RefreshCw, Settings as SettingsIcon } from "lucide-react";
import type { ComponentType, ReactNode } from "react";

import { Button, Empty, buttonVariants, cn } from "@agh/ui";
import {
  SETTINGS_ROOT_PATH,
  SETTINGS_SECTIONS,
  settingsSectionPath,
  type SettingsSectionDescriptor,
} from "@/systems/settings";

export { SETTINGS_ROOT_PATH, SETTINGS_SECTIONS };
export type SettingsSection = SettingsSectionDescriptor;

export const Route = createFileRoute("/_app/settings")({
  component: SettingsShell,
  errorComponent: SettingsShellErrorBoundary,
  notFoundComponent: SettingsShellNotFoundBoundary,
});

function SettingsShell() {
  return (
    <SettingsShellFrame testId="settings-shell">
      <Outlet />
    </SettingsShellFrame>
  );
}

function SettingsShellErrorBoundary({ error, reset }: ErrorComponentProps) {
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
              <RefreshCw className="size-3.5" />
              Retry
            </Button>
            <Link
              className={buttonVariants({ variant: "outline", size: "sm" })}
              to={defaultSettingsSectionPath()}
            >
              <SettingsIcon className="size-3.5" />
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

function SettingsShellNotFoundBoundary({ routeId }: NotFoundRouteProps) {
  return (
    <SettingsShellFrame routeId={routeId} testId="settings-shell-not-found">
      <SettingsShellState
        action={
          <Link
            className={buttonVariants({ variant: "outline", size: "sm" })}
            to={defaultSettingsSectionPath()}
          >
            <SettingsIcon className="size-3.5" />
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

function SettingsSectionNav() {
  return (
    <nav
      aria-label="Settings sections"
      className="flex w-56 shrink-0 flex-col overflow-y-auto border-r border-[color:var(--color-divider)] bg-[color:var(--color-canvas-deep)]"
      data-testid="settings-section-nav"
    >
      <div className="px-4 pb-2 pt-5">
        <span className="font-mono text-[11px] font-semibold uppercase tracking-[var(--tracking-mono)] text-[color:var(--color-text-label)]">
          Settings
        </span>
      </div>
      <div className="flex flex-col gap-0.5 px-2 pb-4">
        {SETTINGS_SECTIONS.map(section => (
          <SettingsSectionLink key={section.slug} section={section} />
        ))}
      </div>
    </nav>
  );
}

function SettingsSectionLink({ section }: { section: SettingsSectionDescriptor }) {
  const matchRoute = useMatchRoute();
  const sectionPath = settingsSectionPath(section.slug);
  const isActive = !!matchRoute({ to: sectionPath, fuzzy: true });

  return (
    <Link
      to={sectionPath}
      data-testid={`settings-section-${section.slug}`}
      data-active={isActive ? "true" : "false"}
      aria-current={isActive ? "page" : undefined}
      className={cn(
        "relative flex items-center gap-2 rounded-md px-3 py-2 text-sm transition-colors",
        "text-[color:var(--color-text-secondary)] hover:bg-[color:var(--color-hover)]",
        isActive &&
          "bg-[color:var(--color-hover)] font-medium text-[color:var(--color-text-primary)]"
      )}
    >
      {isActive && (
        <span
          className="absolute left-0 top-1.5 bottom-1.5 w-[3px] rounded-r bg-[color:var(--color-accent)]"
          data-testid={`settings-section-active-${section.slug}`}
        />
      )}
      <span>{section.label}</span>
    </Link>
  );
}

function SettingsShellFrame({
  children,
  routeId,
  testId,
}: {
  children: ReactNode;
  routeId?: string;
  testId: string;
}) {
  return (
    <div className="flex flex-1 overflow-hidden" data-route-id={routeId} data-testid={testId}>
      <SettingsSectionNav />
      <div
        className="relative flex min-w-0 flex-1 flex-col overflow-hidden"
        data-testid="settings-shell-outlet"
      >
        {children}
      </div>
    </div>
  );
}

function SettingsShellState({
  action,
  description,
  icon,
  title,
}: {
  action?: ReactNode;
  description: string;
  icon: ComponentType<{ className?: string; size?: number }>;
  title: string;
}) {
  return (
    <div className="flex flex-1 items-center justify-center overflow-y-auto px-6 py-8">
      <Empty
        action={action}
        className="max-w-xl"
        description={description}
        icon={icon}
        title={title}
        titleAs="h1"
      />
    </div>
  );
}

function defaultSettingsSectionPath() {
  return settingsSectionPath(SETTINGS_SECTIONS[0].slug);
}

function describeRouteError(error: unknown, fallback: string) {
  if (error instanceof Error && error.message.trim().length > 0) {
    return error.message;
  }

  return fallback;
}
