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

import { Button, Empty, SidebarSectionLabel, buttonVariants, cn } from "@agh/ui";
import {
  ACTIVE_NAV_INDICATOR_CLASS,
  ACTIVE_NAV_ROW_CLASS,
  NAV_ROW_CLASS,
} from "@/components/sidebar-nav-classes";
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
      className="flex w-full shrink-0 flex-wrap gap-1 overflow-y-auto border-b border-[color:var(--color-divider)] bg-[color:var(--color-canvas-deep)] px-2 py-2 xl:w-56 xl:flex-col xl:flex-nowrap xl:border-r xl:border-b-0 xl:py-3"
      data-testid="settings-section-nav"
    >
      <SidebarSectionLabel className="hidden px-2 pt-2 pb-1 xl:block">Settings</SidebarSectionLabel>
      <div className="flex flex-wrap gap-1 xl:flex-col xl:flex-nowrap xl:gap-0.5">
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
  const Icon = section.icon;

  return (
    <Link
      to={sectionPath}
      data-testid={`settings-section-${section.slug}`}
      data-active={isActive ? "true" : "false"}
      aria-current={isActive ? "page" : undefined}
      className={cn(NAV_ROW_CLASS, "shrink-0", isActive && ACTIVE_NAV_ROW_CLASS)}
    >
      {isActive && (
        <span
          aria-hidden="true"
          className={cn(ACTIVE_NAV_INDICATOR_CLASS, "hidden xl:block")}
          data-testid={`settings-section-active-${section.slug}`}
        />
      )}
      <Icon aria-hidden="true" className="size-3.5 shrink-0" />
      <span className="whitespace-nowrap xl:truncate" title={section.label}>
        {section.label}
      </span>
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
    <div
      className="flex flex-1 flex-col overflow-hidden xl:flex-row"
      data-route-id={routeId}
      data-testid={testId}
    >
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
