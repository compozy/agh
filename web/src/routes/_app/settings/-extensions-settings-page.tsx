import { AlertCircle } from "lucide-react";

import { useSettingsExtensionsPage } from "@/systems/settings/hooks/use-settings-extensions-page";
import { SettingsPageFrame } from "@/systems/settings";
import { Button, Spinner } from "@agh/ui";

import { PolicySection } from "./-extensions-policy-section";

export function ExtensionsSettingsPage() {
  const page = useSettingsExtensionsPage();
  if (page.isLoading)
    return (
      <div
        className="flex flex-1 items-center justify-center"
        data-testid="settings-page-extensions-loading"
      >
        <Spinner className="size-5 text-subtle" />
      </div>
    );
  if (page.error || !page.envelope || !page.draft)
    return (
      <div
        className="flex flex-1 items-center justify-center"
        data-testid="settings-page-extensions-error"
      >
        <div className="flex flex-col items-center gap-2 text-center">
          <AlertCircle className="size-6 text-danger" />
          <p className="text-sm text-subtle">
            {page.error?.message ?? "Failed to load extensions settings"}
          </p>
          <Button onClick={page.handleRetry} size="sm" type="button" variant="outline">
            Retry
          </Button>
        </div>
      </div>
    );
  return (
    <SettingsPageFrame
      description="What extensions are allowed to run on this daemon, and from where."
      restart={page.restart}
      slug="extensions"
    >
      <PolicySection
        canMutate={page.canMutatePolicy}
        draft={page.draft}
        error={page.savePolicyError}
        isDirty={page.isPolicyDirty}
        isSaving={page.isSavingPolicy}
        onReset={page.handleResetPolicy}
        onSave={page.handleSavePolicy}
        setDraft={value =>
          page.updatePolicyDraft(current => (typeof value === "function" ? value(current) : value))
        }
        warnings={page.policyWarnings}
      />
    </SettingsPageFrame>
  );
}
