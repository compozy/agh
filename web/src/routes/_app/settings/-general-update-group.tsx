import { ExternalLink } from "lucide-react";

import type { useSettingsGeneralPage } from "@/systems/settings/hooks/use-settings-general-page";
import { SettingRow, SettingValue, SettingsGroup } from "@/systems/settings";
import { Button, Pill } from "@agh/ui";

type UpdateQuery = ReturnType<typeof useSettingsGeneralPage>["update"];

export function GeneralUpdateGroup({ update }: { update: UpdateQuery }) {
  const snapshot = update.data ?? null;
  const transportError = update.error
    ? update.error instanceof Error
      ? update.error.message
      : "Failed to load update status"
    : null;
  const lastUpdateError = snapshot?.last_error ?? (snapshot ? null : transportError);

  return (
    <SettingsGroup
      action={
        update.error ? (
          <Button
            data-testid="settings-page-general-update-retry"
            onClick={() => void update.refetch()}
            size="sm"
            type="button"
            variant="neutral"
          >
            Retry
          </Button>
        ) : undefined
      }
      title="Updates"
    >
      <SettingRow
        data-testid="settings-page-general-update-status"
        description={
          snapshot?.managed
            ? `Installed with ${snapshot.install_method ?? "a package manager"}. Updates are managed for you.`
            : "AGH updates direct-binary installs on its own."
        }
        label="AGH version"
        control={
          <span className="flex items-center gap-2">
            <SettingValue mono>{snapshot?.current_version ?? "—"}</SettingValue>
            {snapshot ? (
              snapshot.status === "available" ? (
                <Pill tone="info">Update available</Pill>
              ) : snapshot.status === "current" || snapshot.status === "updated" ? (
                <Pill tone="success">
                  <Pill.Dot tone="success" />
                  Up to date
                </Pill>
              ) : snapshot.status === "failed" ? (
                <Pill tone="danger">Update failed</Pill>
              ) : (
                <Pill tone="neutral">{snapshot.status}</Pill>
              )
            ) : null}
            {snapshot?.release_url ? (
              <Button
                nativeButton={false}
                render={
                  <a
                    aria-label="Open release notes"
                    data-testid="settings-page-general-update-release-link"
                    href={snapshot.release_url}
                    rel="noreferrer"
                    target="_blank"
                  />
                }
                size="sm"
                variant="ghost"
              >
                <ExternalLink aria-hidden="true" className="size-3 text-subtle" />
              </Button>
            ) : null}
          </span>
        }
      />
      {snapshot?.recommendation ? (
        <SettingRow
          data-testid="settings-page-general-update-recommendation"
          description="Exact command or package-manager path for this install."
          label="Next action"
          control={<SettingValue mono>{snapshot.recommendation}</SettingValue>}
        />
      ) : null}
      {lastUpdateError ? (
        <SettingRow
          data-testid="settings-page-general-update-last-error"
          description={<span className="text-danger">{lastUpdateError}</span>}
          label="Last error"
        />
      ) : null}
    </SettingsGroup>
  );
}

export function GeneralUpdateAdvancedRow({ update }: { update: UpdateQuery }) {
  const snapshot = update.data ?? null;
  if (!snapshot) return null;

  return (
    <SettingRow
      data-testid="settings-page-general-update-detail"
      description={
        snapshot.recommendation ?? "Latest stable and install-method detail for this machine."
      }
      label="Update detail"
      control={
        <SettingValue mono>
          {snapshot.latest_version ?? "—"} · {snapshot.install_method ?? "—"}
        </SettingValue>
      }
    />
  );
}
