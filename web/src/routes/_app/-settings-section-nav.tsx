import { Link, useMatchRoute } from "@tanstack/react-router";

import { SidebarSectionLabel, cn } from "@agh/ui";

import {
  ACTIVE_NAV_INDICATOR_CLASS,
  ACTIVE_NAV_ROW_CLASS,
  NAV_ROW_CLASS,
} from "@/components/sidebar-nav-classes";
import {
  SETTINGS_SECTIONS,
  settingsSectionPath,
  type SettingsSectionDescriptor,
} from "@/systems/settings";

export function SettingsSectionNav() {
  return (
    <nav
      aria-label="Settings sections"
      className="flex w-full shrink-0 flex-wrap gap-1 overflow-y-auto border-b border-line bg-canvas px-2 py-2 xl:w-56 xl:flex-col xl:flex-nowrap xl:border-r xl:border-b-0 xl:py-3"
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
      <Icon aria-hidden="true" className="size-3 shrink-0" />
      <span className="whitespace-nowrap xl:truncate" title={section.label}>
        {section.label}
      </span>
    </Link>
  );
}
