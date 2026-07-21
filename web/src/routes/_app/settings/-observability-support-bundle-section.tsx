import { Download } from "lucide-react";
import { useState } from "react";

import { Button, Eyebrow, Spinner } from "@agh/ui";
import { SettingsGroup } from "@/systems/settings";
import { useSupportBundleDownload } from "@/systems/support";
import { formatBytes } from "./-observability-format";

export function ObservabilitySupportBundleSection() {
  const supportBundle = useSupportBundleDownload();
  const [approved, setApproved] = useState(false);
  const [consentError, setConsentError] = useState<string | null>(null);
  const operation = supportBundle.operation;
  const errorMessage =
    consentError ??
    (supportBundle.error instanceof Error ? supportBundle.error.message : undefined);

  const handleCreate = async () => {
    if (!approved) {
      setConsentError("Approval is required before creating a support bundle.");
      return;
    }
    setConsentError(null);
    try {
      await supportBundle.create({ includeStatus: true, yes: true });
    } catch (error) {
      setConsentError(
        error instanceof Error ? error.message : "Support bundle creation failed. Try again."
      );
    }
  };

  return (
    <SettingsGroup title="Support bundle" description="redacted daemon archive">
      <div
        className="flex flex-col gap-4 rounded-md border border-line bg-elevated px-4 py-3"
        data-testid="settings-page-observability-support-bundle"
      >
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div className="flex flex-col gap-1">
            <span className="text-sm text-fg">Create support bundle</span>
            <Eyebrow
              className="text-muted"
              data-testid="settings-page-observability-support-bundle-status"
            >
              status: {operation?.status ?? "idle"}
            </Eyebrow>
          </div>
          <Button
            data-testid="settings-page-observability-support-bundle-button"
            disabled={supportBundle.isPending}
            onClick={() => void handleCreate()}
            size="sm"
            type="button"
          >
            {supportBundle.isPending ? (
              <Spinner className="size-3.5" />
            ) : (
              <Download className="size-3.5" />
            )}
            {supportBundle.isPending ? "Preparing" : "Create bundle"}
          </Button>
        </div>
        <label className="flex items-start gap-3 text-sm text-subtle">
          <input
            checked={approved}
            className="mt-0.5 size-4 rounded border border-line bg-canvas-soft accent-[var(--accent)] focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent"
            data-testid="settings-page-observability-support-bundle-consent"
            onChange={event => {
              setApproved(event.currentTarget.checked);
              if (event.currentTarget.checked) setConsentError(null);
            }}
            type="checkbox"
          />
          <span>I approve creating a redacted diagnostics archive.</span>
        </label>
        {operation?.size_bytes ? (
          <Eyebrow className="text-muted" data-testid="settings-page-observability-support-size">
            size: {formatBytes(operation.size_bytes)}
          </Eyebrow>
        ) : null}
        {errorMessage ? (
          <p
            className="text-sm text-danger"
            data-testid="settings-page-observability-support-bundle-error"
            role="alert"
          >
            {errorMessage}
          </p>
        ) : null}
      </div>
    </SettingsGroup>
  );
}
