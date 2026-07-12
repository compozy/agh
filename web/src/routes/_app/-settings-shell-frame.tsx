import type { ReactNode } from "react";

import { SettingsSectionNav } from "./-settings-section-nav";

export function SettingsShellFrame({
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
