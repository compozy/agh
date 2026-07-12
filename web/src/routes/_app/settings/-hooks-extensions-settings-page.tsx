import { AlertCircle } from "lucide-react";

import { useSettingsHooksExtensionsPage } from "@/hooks/routes/use-settings-hooks-extensions-page";
import { NotificationPresetsPanel } from "@/systems/notifications";
import { restartBannerPropsFor } from "@/systems/settings";
import {
  Button,
  PageShell,
  RestartBanner,
  Spinner,
  StatusLineTopbarSlot,
  useTopbarSlot,
} from "@agh/ui";
import { LastActionAlert } from "./-hooks-extensions-action-alert";
import { HooksSection, TransportParityBanner } from "./-hooks-extensions-hooks-section";
import { ExtensionsSection } from "./-hooks-extensions-installed-section";
import { MarketplaceSection } from "./-hooks-extensions-marketplace-section";
import { PolicySection } from "./-hooks-extensions-policy-section";

export function HooksExtensionsSettingsPage() {
  const page = useSettingsHooksExtensionsPage();
  const notificationPresets = page.notificationPresets ?? [];
  const envelopeForSlot = page.envelope;
  useTopbarSlot({
    tabs: envelopeForSlot ? (
      <StatusLineTopbarSlot
        data-testid="settings-page-hooks-extensions-status-line"
        status="connected"
        items={[
          {
            key: "hooks",
            value: (
              <span data-testid="settings-page-hooks-extensions-hooks-total">
                {page.hooksCounts.enabled}/{page.hooksCounts.total} hooks enabled
              </span>
            ),
            tone: "neutral",
          },
          {
            key: "extensions",
            value: (
              <span data-testid="settings-page-hooks-extensions-extensions-total">
                {page.extensionsCounts.enabled}/{page.extensionsCounts.total} extensions enabled
              </span>
            ),
            tone: "neutral",
          },
        ]}
      />
    ) : undefined,
  });

  if (page.isLoading) {
    return (
      <div
        className="flex flex-1 items-center justify-center"
        data-testid="settings-page-hooks-extensions-loading"
      >
        <Spinner className="size-5 text-subtle" />
      </div>
    );
  }

  if (page.error || !page.envelope || !page.draft) {
    return (
      <div
        className="flex flex-1 items-center justify-center"
        data-testid="settings-page-hooks-extensions-error"
      >
        <div className="flex flex-col items-center gap-2 text-center">
          <AlertCircle className="size-6 text-danger" />
          <p className="text-sm text-subtle">
            {page.error?.message ?? "Failed to load hooks & extensions settings"}
          </p>
          <Button onClick={page.handleRetry} size="sm" type="button" variant="outline">
            Retry
          </Button>
        </div>
      </div>
    );
  }

  const { draft, hooks, extensions, transportParity } = page;
  const bannerProps = restartBannerPropsFor("hooks-extensions", page.restart);

  return (
    <PageShell
      slug="hooks-extensions"
      banner={bannerProps ? <RestartBanner {...bannerProps} /> : null}
    >
      {page.lastAction ? (
        <LastActionAlert action={page.lastAction} onDismiss={page.dismissLastAction} />
      ) : null}
      <TransportParityBanner parity={transportParity} />
      <HooksSection
        hooks={hooks}
        pendingHookName={page.pendingHookName}
        hookError={page.hookError}
        canMutate={page.canMutateHooks}
        onToggle={page.toggleHookEnabled}
      />
      <ExtensionsSection
        extensions={extensions}
        pendingExtensionName={page.pendingExtensionName}
        error={page.extensionActionError ?? page.extensionsError}
        isLoading={page.extensionsLoading}
        canMutate={page.canMutateExtensions}
        onToggle={page.toggleExtensionEnabled}
        onUpdate={page.updateExtension}
        onRemove={page.removeExtension}
        onOpenProvenance={page.openExtensionProvenance}
        selectedProvenanceName={page.selectedProvenanceName}
        selectedProvenance={page.selectedProvenance}
        provenanceLoading={page.provenanceLoading}
        provenanceError={page.provenanceError}
        onCloseProvenance={page.closeExtensionProvenance}
      />
      <MarketplaceSection
        entries={page.marketplaceEntries}
        query={page.marketplaceSearch}
        setQuery={page.setMarketplaceSearch}
        allowUnverified={page.marketplaceAllowUnverified}
        setAllowUnverified={page.setMarketplaceAllowUnverified}
        pendingSlug={page.pendingMarketplaceSlug}
        error={page.marketplaceError}
        isLoading={page.marketplaceLoading}
        canMutate={page.canMutateExtensions}
        onSearch={page.searchMarketplace}
        onInstall={page.installMarketplaceExtension}
      />
      <NotificationPresetsPanel
        presets={notificationPresets}
        isLoading={page.notificationPresetsLoading}
        error={page.notificationPresetsError ?? page.notificationPresetActionError}
        pendingName={page.pendingNotificationPresetName}
        canMutate={page.canMutateNotificationPresets}
        onCreate={page.createNotificationPreset}
        onToggle={page.toggleNotificationPreset}
        onDelete={page.deleteNotificationPreset}
      />
      <PolicySection
        draft={draft}
        setDraft={value =>
          page.updatePolicyDraft(current => (typeof value === "function" ? value(current) : value))
        }
        onToggleAllowedKind={page.toggleAllowedKind}
        isDirty={page.isPolicyDirty}
        isSaving={page.isSavingPolicy}
        error={page.savePolicyError}
        warnings={page.policyWarnings}
        canMutate={page.canMutatePolicy}
        onSave={page.handleSavePolicy}
        onReset={page.handleResetPolicy}
      />
    </PageShell>
  );
}
