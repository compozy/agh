import { Link } from "@tanstack/react-router";
import { ChevronLeft, Search } from "lucide-react";
import { useState, type Ref } from "react";

import { cn, ConnectionIndicator, Kbd } from "@agh/ui";

import {
  filterSettingsSections,
  SETTINGS_SECTION_GROUPS,
  settingsSectionPath,
  type SettingsSectionDescriptor,
} from "@/systems/settings";
import { useDaemonConnectionStatus } from "@/systems/status";

/**
 * Settings takeover sidebar (design system §02–03): back row, live search,
 * three nav groups, daemon status foot. Collapses to a horizontal chip strip
 * when the window is narrow. Active section comes from the WINDOW's location
 * so an unfocused settings window keeps its own highlight.
 */
export function SettingsWindowNav({
  activeSlug,
  onBackToApp,
  searchInputRef,
}: {
  activeSlug: string;
  onBackToApp: () => void;
  /** Focus target for the window-scoped `/` shortcut. */
  searchInputRef?: Ref<HTMLInputElement>;
}) {
  const [query, setQuery] = useState("");
  const connection = useDaemonConnectionStatus();
  const visible = filterSettingsSections(query);
  const visibleSlugs = new Set(visible.map(section => section.slug));

  return (
    <nav
      aria-label="Settings navigation"
      className={cn(
        "flex w-full shrink-0 flex-row items-center gap-1 overflow-x-auto border-b border-line-soft bg-canvas-soft px-2 py-1.5",
        "@min-[56rem]:w-[264px] @min-[56rem]:flex-col @min-[56rem]:items-stretch @min-[56rem]:gap-0 @min-[56rem]:overflow-y-auto @min-[56rem]:border-r @min-[56rem]:border-b-0 @min-[56rem]:px-2.5 @min-[56rem]:py-0"
      )}
      data-testid="settings-section-nav"
    >
      <div className="hidden @min-[56rem]:block">
        <button
          className={cn(
            "mt-2 flex h-[30px] w-full items-center gap-1.5 rounded-md px-2 text-small-body font-medium text-muted",
            "transition-colors duration-base hover:bg-row-hover hover:text-fg",
            "focus-visible:outline-none focus-visible:shadow-focus-ring"
          )}
          data-testid="settings-back-to-app"
          onClick={onBackToApp}
          type="button"
        >
          <ChevronLeft aria-hidden="true" className="size-3.5 text-subtle" />
          Back to app
        </button>
        <label
          className={cn(
            "mt-1.5 flex h-7 items-center gap-1.5 rounded-md border border-line-soft bg-canvas px-2",
            "focus-within:border-line-strong focus-within:shadow-focus-ring"
          )}
        >
          <Search aria-hidden="true" className="size-3 shrink-0 text-subtle" />
          <input
            aria-label="Search settings"
            autoComplete="off"
            className="min-w-0 flex-1 bg-transparent text-form-label text-fg outline-none placeholder:text-subtle"
            data-testid="settings-nav-search"
            onChange={event => setQuery(event.target.value)}
            placeholder="Search settings"
            ref={searchInputRef}
            type="search"
            value={query}
          />
          <Kbd className="shrink-0">/</Kbd>
        </label>
      </div>

      <div className="flex flex-row items-center gap-1 @min-[56rem]:flex-1 @min-[56rem]:flex-col @min-[56rem]:items-stretch @min-[56rem]:gap-0 @min-[56rem]:pb-2">
        {SETTINGS_SECTION_GROUPS.map(group => {
          const sections = visible.filter(section => section.group === group.id);
          if (sections.length === 0) return null;
          return (
            <div
              className="flex flex-row items-center gap-1 @min-[56rem]:flex-col @min-[56rem]:items-stretch @min-[56rem]:gap-0"
              key={group.id}
            >
              <span className="eyebrow hidden px-2 pt-3 pb-1 text-faint @min-[56rem]:block">
                {group.label}
              </span>
              {sections.map(section => (
                <SettingsSectionLink
                  isActive={section.slug === activeSlug}
                  key={section.slug}
                  section={section}
                />
              ))}
            </div>
          );
        })}
        {visibleSlugs.size === 0 ? (
          <p
            className="hidden px-2 py-3 text-form-label text-subtle @min-[56rem]:block"
            data-testid="settings-nav-empty"
          >
            No settings match “{query.trim()}”
          </p>
        ) : null}
      </div>

      <div
        className="hidden items-center gap-2 border-t border-line-soft px-2 py-2.5 @min-[56rem]:flex"
        data-testid="settings-daemon-status"
      >
        <ConnectionIndicator
          label={connection === "connected" ? "Daemon running" : "Daemon unreachable"}
          status={connection}
        />
      </div>
    </nav>
  );
}

function SettingsSectionLink({
  section,
  isActive,
}: {
  section: SettingsSectionDescriptor;
  isActive: boolean;
}) {
  const Icon = section.icon;

  return (
    <Link
      aria-current={isActive ? "page" : undefined}
      className={cn(
        "flex h-8 shrink-0 items-center gap-2.5 rounded-md px-2 text-ws-name font-medium",
        "transition-colors duration-base focus-visible:outline-none focus-visible:shadow-focus-ring",
        isActive
          ? "bg-accent-tint text-accent-strong"
          : "text-muted hover:bg-row-hover hover:text-fg"
      )}
      data-active={isActive ? "true" : "false"}
      data-testid={`settings-section-${section.slug}`}
      to={settingsSectionPath(section.slug)}
    >
      <Icon
        aria-hidden="true"
        className={cn("size-4 shrink-0", isActive ? "text-accent-strong" : "text-subtle")}
      />
      <span className="whitespace-nowrap @min-[56rem]:truncate" title={section.label}>
        {section.label}
      </span>
    </Link>
  );
}
