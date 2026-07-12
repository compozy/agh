import { Outlet } from "@tanstack/react-router";

import { SettingsShellFrame } from "./-settings-shell-frame";

export function SettingsShell() {
  return (
    <SettingsShellFrame testId="settings-shell">
      <Outlet />
    </SettingsShellFrame>
  );
}
