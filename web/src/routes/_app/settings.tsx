import { Outlet, createFileRoute, Link, useMatchRoute } from "@tanstack/react-router";

import { cn } from "@/lib/utils";
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
});

function SettingsShell() {
  return (
    <div className="flex flex-1 overflow-hidden" data-testid="settings-shell">
      <SettingsSectionNav />
      <div
        className="relative flex min-w-0 flex-1 flex-col overflow-hidden"
        data-testid="settings-shell-outlet"
      >
        <Outlet />
      </div>
    </div>
  );
}

function SettingsSectionNav() {
  return (
    <nav
      aria-label="Settings sections"
      className="flex w-56 shrink-0 flex-col overflow-y-auto border-r border-[color:var(--color-divider)] bg-[color:var(--color-surface)]"
      data-testid="settings-section-nav"
    >
      <div className="px-4 pb-2 pt-5">
        <span className="font-mono text-[0.6rem] uppercase tracking-[0.22em] text-[color:var(--color-text-label)]">
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
