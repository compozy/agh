import { Empty } from "@agh/ui";

import { useDesktop } from "../hooks/use-desktop";
import { getOsApp } from "../lib/app-registry";

/**
 * Interim body for apps whose desktop port has not landed yet (the app-port
 * wave replaces this per app). States the situation plainly — no dead
 * controls, no placeholder data (SD-007).
 */
export function PendingAppWindow({ windowId }: { windowId: string }) {
  const appId = useDesktop(state => state.windows[windowId]?.app ?? null);
  if (appId === null) return null;
  const app = getOsApp(appId);

  return (
    <div className="flex min-h-full items-center justify-center p-6" data-testid="os-pending-app">
      <Empty
        className="max-w-md"
        icon={app.icon}
        title={`${app.title} is not windowed yet`}
        description="This surface is being rehosted into the desktop shell. Its CLI and API surfaces remain available."
      />
    </div>
  );
}
