import { AlertCircle } from "lucide-react";

import { useSettingsRolesPage } from "@/systems/settings/hooks/use-settings-roles-page";
import {
  RoleSettingsGroup,
  SettingsPageFrame,
  SettingsSaveBar,
  useSettingsSaveBarState,
  useSettingsTopbar,
} from "@/systems/settings";
import { Button, Spinner } from "@agh/ui";

const TEST_PREFIX = "settings-page-roles";

function RolesNotice({
  message,
  onRetry,
  testId,
}: {
  message: string;
  onRetry: () => void;
  testId: string;
}) {
  return (
    <div className="flex flex-1 items-center justify-center" data-testid={testId}>
      <div className="flex flex-col items-center gap-2 text-center">
        <AlertCircle className="size-6 text-danger" />
        <p className="max-w-settings-page-description text-sm text-subtle">{message}</p>
        <Button onClick={onRetry} size="sm" type="button" variant="outline">
          Retry
        </Button>
      </div>
    </div>
  );
}

export function RolesSettingsPage() {
  const page = useSettingsRolesPage();
  useSettingsTopbar("roles");
  const saveBarState = useSettingsSaveBarState({
    isDirty: page.isDirty,
    isSaving: page.isSaving,
    error: page.saveError,
    warnings: page.warnings,
    lastAppliedLabel: page.lastAppliedLabel,
  });

  if (page.isLoading) {
    return (
      <div
        className="flex flex-1 items-center justify-center"
        data-testid={`${TEST_PREFIX}-loading`}
      >
        <Spinner className="size-5 text-subtle" />
      </div>
    );
  }

  if (page.error) {
    return (
      <RolesNotice
        message={page.error.message}
        onRetry={page.handleRetry}
        testId={`${TEST_PREFIX}-error`}
      />
    );
  }

  // Empty projection is a protocol anomaly, not a normal empty state.
  if (page.isEmpty) {
    return (
      <RolesNotice
        message="Roles unavailable — no role projection was returned."
        onRetry={page.handleRetry}
        testId={`${TEST_PREFIX}-empty`}
      />
    );
  }

  return (
    <SettingsPageFrame
      wide
      slug="roles"
      description="Choose how each background role is routed."
      restart={page.restart}
      saveBar={
        <SettingsSaveBar
          slug="roles"
          state={saveBarState}
          onSave={page.handleSave}
          onReset={page.handleReset}
        />
      }
    >
      {page.roles.map(vm => (
        <RoleSettingsGroup
          key={vm.role}
          vm={vm}
          validationErrors={page.validationErrors}
          disabled={page.isSaving}
          setRoleField={page.setRoleField}
          setNumberFieldValidity={page.setNumberFieldValidity}
          addFallback={page.addFallback}
          removeFallback={page.removeFallback}
          updateFallback={page.updateFallback}
          registerFieldRef={page.registerFieldRef}
        />
      ))}
    </SettingsPageFrame>
  );
}
