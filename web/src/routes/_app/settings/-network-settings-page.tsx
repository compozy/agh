import { AlertCircle } from "lucide-react";
import { useState } from "react";

import { useSettingsNetworkPage } from "@/hooks/routes/use-settings-network-page";
import {
  NetworkSettingsSections,
  restartBannerPropsFor,
  SettingsTopbarPublisher,
  SettingsSaveBar,
} from "@/systems/settings";
import { Button, PageShell, RestartBanner, Spinner, StatusLine } from "@agh/ui";

export function NetworkSettingsPage() {
  const page = useSettingsNetworkPage();
  const [validationErrors, setValidationErrors] = useState<Record<string, string | null>>({});
  const setValidationError = (key: string) => (message: string | null) => {
    setValidationErrors(current =>
      current[key] === message ? current : { ...current, [key]: message }
    );
  };
  const isInvalid = Object.values(validationErrors).some(message => message !== null);
  const runtime = page.envelope?.runtime;
  const statusLine = runtime ? (
    <StatusLine
      data-testid="settings-page-network-status-line"
      status={runtime.available ? "connected" : "error"}
      items={[
        {
          key: "status",
          value: runtime.status ?? (runtime.enabled ? "ready" : "disabled"),
          tone: "neutral",
        },
        {
          key: "participants",
          value: `${runtime.local_peers} Live participants`,
          tone: "neutral",
        },
      ]}
    />
  ) : null;

  if (page.isLoading) {
    return (
      <div
        aria-label="Loading network settings"
        className="flex flex-1 items-center justify-center"
        data-testid="settings-page-network-loading"
        role="status"
      >
        <Spinner aria-hidden="true" className="size-5 text-subtle" />
      </div>
    );
  }

  if (page.error || !page.envelope || !page.draft) {
    return (
      <div
        className="flex flex-1 items-center justify-center"
        data-testid="settings-page-network-error"
      >
        <div className="flex flex-col items-center gap-2 text-center">
          <AlertCircle className="size-6 text-danger" />
          <p className="text-sm text-subtle">
            {page.error?.message ?? "Failed to load network settings"}
          </p>
          <Button onClick={page.handleRetry} size="sm" type="button" variant="outline">
            Retry
          </Button>
        </div>
      </div>
    );
  }

  if (!runtime) {
    return null;
  }

  const bannerProps = restartBannerPropsFor("network", page.restart);

  return (
    <PageShell
      slug="network"
      banner={bannerProps ? <RestartBanner {...bannerProps} /> : null}
      head={<SettingsTopbarPublisher slug="network" statusLine={statusLine} />}
      footer={
        <SettingsSaveBar
          slug="network"
          isDirty={page.isDirty}
          isInvalid={isInvalid}
          isSaving={page.isSaving}
          error={page.saveError}
          warnings={page.warnings}
          lastAppliedLabel={page.lastAppliedLabel}
          onSave={page.handleSave}
          onReset={page.handleReset}
        />
      }
    >
      <NetworkSettingsSections
        runtime={runtime}
        draft={page.draft}
        setDraft={page.setDraft}
        validationErrors={validationErrors}
        setValidationError={setValidationError}
      />
    </PageShell>
  );
}
