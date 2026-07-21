import { useDesktopDock } from "../hooks/use-desktop-dock";
import type { OsAttentionBadges } from "../lib/attention-model";
import { OsDockZone } from "./os-dock";
import { OsDockTabBar } from "./os-dock-tab-bar";

export interface DesktopDockProps {
  onNewSession: () => void;
  badges: OsAttentionBadges;
}

/**
 * The wired dock: floating renders the centered glass strip with proximity
 * magnification; compact renders the full-width bottom tab bar (os-v2.css
 * mobile block). Entries, activation semantics, and the magnification gates
 * live in `useDesktopDock`.
 */
export function DesktopDock({ onNewSession, badges }: DesktopDockProps) {
  const { entries, presentation, magnify, handleSelect } = useDesktopDock(badges);

  if (presentation === "compact") {
    return <OsDockTabBar items={entries} onSelect={handleSelect} onNewSession={onNewSession} />;
  }

  return (
    <OsDockZone
      items={entries}
      onSelect={handleSelect}
      onNewSession={onNewSession}
      magnify={magnify}
    />
  );
}
