import { useShallow } from "zustand/shallow";

import { useOsShell } from "../hooks/use-os-shell";
import { useDesktop } from "../hooks/use-desktop";
import { dockApps, OS_APPS } from "../lib/app-registry";
import type { OsAttentionBadges } from "../lib/attention-model";
import { osWindowId, type OsAppId } from "../lib/os-types";
import { dockBadgeFor } from "./dock-badges";
import { OsDockZone } from "./os-dock";
import { DockIcons, type DockIconId } from "./os-dock-icons";
import type { OsDockEntry } from "./os-dock-types";

export interface DesktopDockProps {
  onNewSession: () => void;
  badges: OsAttentionBadges;
}

/**
 * The wired dock: registry groups with running/minimized indicators and the
 * detached New Session control. Tab-bar prototype semantics: activating a
 * focused app minimizes it; anything else opens or restores it. Attention
 * badges bind to their runtime projections in the attention-surfaces task.
 */
export function DesktopDock({ onNewSession, badges }: DesktopDockProps) {
  const { coordinator, store } = useOsShell();
  const railOpen = useDesktop(state => state.railOpen);
  const windowStates = useDesktop(
    useShallow(state => {
      const byApp: Record<string, "open" | "focused" | "minimized"> = {};
      for (const win of Object.values(state.windows)) {
        if (win.instanceKey !== null) continue;
        byApp[win.app] = win.minimized
          ? "minimized"
          : state.focusedId === win.id
            ? "focused"
            : "open";
      }
      return byApp;
    })
  );

  const groups = dockApps();
  const sessionApp = OS_APPS.session;
  const entries: OsDockEntry[] = [
    {
      id: sessionApp.id,
      name: "Sessions",
      icon: DockIcons.sessions,
      running: railOpen,
      badge: dockBadgeFor(sessionApp, badges),
    },
  ];
  groups.forEach((group, index) => {
    // Prototype seams: groups 1+2 run together; separators split 2|3 and 3|4.
    if (index >= 2) entries.push({ id: `sep-${index}`, sep: true });
    for (const app of group) {
      const state = windowStates[app.id];
      entries.push({
        id: app.id,
        name: app.title,
        icon: DockIcons[app.id as DockIconId] ?? app.icon,
        running: state === "open" || state === "focused",
        minimized: state === "minimized",
        badge: dockBadgeFor(app, badges),
      });
    }
  });

  const handleSelect = (id: string) => {
    const appId = id as OsAppId;
    if (appId === "session") {
      store.getState().toggleRail();
      return;
    }
    if (windowStates[appId] === "focused") {
      coordinator.userMinimize(osWindowId(appId));
      return;
    }
    coordinator.userOpen({ app: appId });
  };

  return <OsDockZone items={entries} onSelect={handleSelect} onNewSession={onNewSession} />;
}
