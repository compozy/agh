import { AlertCircle } from "lucide-react";

import { useSettingsHooksPage } from "@/hooks/routes/use-settings-hooks-page";
import { NotificationPresetsPanel } from "@/systems/notifications";
import { restartBannerPropsFor, SettingsTopbarPublisher } from "@/systems/settings";
import { Button, PageShell, RestartBanner, Spinner, StatusLine } from "@agh/ui";

import { HooksSection } from "./-hooks-section";

export function HooksSettingsPage() {
  const page = useSettingsHooksPage();
  const statusLine = page.envelope ? (
    <StatusLine
      data-testid="settings-page-hooks-status-line"
      items={[
        {
          key: "hooks",
          value: `${page.hooksCounts.enabled}/${page.hooksCounts.total} hooks enabled`,
          tone: "neutral",
        },
      ]}
      status="connected"
    />
  ) : null;
  if (page.isLoading)
    return (
      <div
        className="flex flex-1 items-center justify-center"
        data-testid="settings-page-hooks-loading"
      >
        <Spinner className="size-5 text-subtle" />
      </div>
    );
  if (page.error || !page.envelope)
    return (
      <div
        className="flex flex-1 items-center justify-center"
        data-testid="settings-page-hooks-error"
      >
        <div className="flex flex-col items-center gap-2 text-center">
          <AlertCircle className="size-6 text-danger" />
          <p className="text-sm text-subtle">
            {page.error?.message ?? "Failed to load hooks settings"}
          </p>
          <Button onClick={page.handleRetry} size="sm" type="button" variant="outline">
            Retry
          </Button>
        </div>
      </div>
    );
  const banner = restartBannerPropsFor("hooks", page.restart);
  return (
    <PageShell
      banner={banner ? <RestartBanner {...banner} /> : null}
      head={<SettingsTopbarPublisher slug="hooks" statusLine={statusLine} />}
      slug="hooks"
    >
      <HooksSection
        canMutate={page.canMutateHooks}
        hookError={page.hookError}
        hooks={page.hooks}
        onToggle={page.toggleHookEnabled}
        pendingHookName={page.pendingHookName}
      />
      <NotificationPresetsPanel
        canMutate
        error={page.notificationPresetsError ?? page.notificationPresetActionError}
        isLoading={page.notificationPresetsLoading}
        onCreate={page.createNotificationPreset}
        onDelete={page.deleteNotificationPreset}
        onToggle={page.toggleNotificationPreset}
        pendingName={page.pendingNotificationPresetName}
        presets={page.notificationPresets}
      />
    </PageShell>
  );
}
