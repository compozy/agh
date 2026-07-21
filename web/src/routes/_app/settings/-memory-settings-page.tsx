import { AlertCircle } from "lucide-react";
import { useState } from "react";

import { useSettingsMemoryPage } from "@/systems/settings/hooks/use-settings-memory-page";
import { SettingsAdvancedFold, SettingsPageFrame, SettingsSaveBar } from "@/systems/settings";
import { Button, Spinner, Time } from "@agh/ui";
import { ControllerLLMSection, ControllerSection } from "./-memory-controller-sections";
import { DreamSection } from "./-memory-dream-section";
import {
  DailyLogsSection,
  FileCapsSection,
  SessionLedgerSection,
  WorkspaceIdentitySection,
} from "./-memory-file-sections";
import { DecisionsSection, ExtractorSection } from "./-memory-processing-sections";
import { RecallSection } from "./-memory-recall-section";
import { TEST_PREFIX, type ValidationSetter } from "./-memory-settings-types";
import { MemorySystemSection, ProviderResilienceSection } from "./-memory-system-sections";

export function MemorySettingsPage() {
  const page = useSettingsMemoryPage();
  const [validationErrors, setValidationErrors] = useState<Record<string, string | null>>({});
  const setValidationError: ValidationSetter = (key: string) => (message: string | null) => {
    setValidationErrors(current =>
      current[key] === message ? current : { ...current, [key]: message }
    );
  };
  const isInvalid = Object.values(validationErrors).some(message => message !== null);

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

  if (page.error || !page.envelope || !page.draft) {
    return (
      <div className="flex flex-1 items-center justify-center" data-testid={`${TEST_PREFIX}-error`}>
        <div className="flex flex-col items-center gap-2 text-center">
          <AlertCircle className="size-6 text-danger" />
          <p className="text-sm text-subtle">
            {page.error?.message ?? "Failed to load memory settings"}
          </p>
          <Button onClick={page.handleRetry} size="sm" type="button" variant="outline">
            Retry
          </Button>
        </div>
      </div>
    );
  }

  const { envelope, draft, setDraft, restart } = page;
  const health = envelope.health;
  const dreamAvailable =
    envelope.actions.consolidate.available && envelope.health.dream_enabled && draft.dream.enabled;

  return (
    <SettingsPageFrame
      description="What your agents remember across sessions, and how those memories are made."
      meta={[
        {
          key: "files",
          content: (
            <span>
              <span className="font-medium text-muted">{health.file_count}</span> memory files
            </span>
          ),
        },
        {
          key: "dream",
          content: health.last_consolidated_at ? (
            <span
              className="inline-flex items-center gap-1"
              data-testid={`${TEST_PREFIX}-last-consolidated`}
            >
              last dream <Time iso={health.last_consolidated_at} mode="relative" />
            </span>
          ) : (
            <span data-testid={`${TEST_PREFIX}-last-consolidated`}>no dream runs yet</span>
          ),
        },
      ]}
      restart={restart}
      saveBar={
        <SettingsSaveBar
          slug="memory"
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
      slug="memory"
    >
      <MemorySystemSection draft={draft} setDraft={setDraft} />
      <RecallSection
        draft={draft}
        setDraft={setDraft}
        validationErrors={validationErrors}
        setValidationError={setValidationError}
      />
      <DreamSection
        draft={draft}
        setDraft={setDraft}
        validationErrors={validationErrors}
        setValidationError={setValidationError}
        dreamAvailable={dreamAvailable}
        dreamPending={page.isTriggeringDream}
        onTriggerDream={page.handleTriggerDream}
        actionMessage={page.actionMessage}
      />
      <SessionLedgerSection
        draft={draft}
        setDraft={setDraft}
        validationErrors={validationErrors}
        setValidationError={setValidationError}
      />
      <DailyLogsSection
        draft={draft}
        setDraft={setDraft}
        validationErrors={validationErrors}
        setValidationError={setValidationError}
      />
      <WorkspaceIdentitySection draft={draft} setDraft={setDraft} />

      <SettingsAdvancedFold data-testid={`${TEST_PREFIX}-advanced`} padded>
        <ProviderResilienceSection
          draft={draft}
          setDraft={setDraft}
          validationErrors={validationErrors}
          setValidationError={setValidationError}
        />
        <ControllerSection
          draft={draft}
          setDraft={setDraft}
          validationErrors={validationErrors}
          setValidationError={setValidationError}
        />
        <ControllerLLMSection
          draft={draft}
          setDraft={setDraft}
          validationErrors={validationErrors}
          setValidationError={setValidationError}
        />
        <DecisionsSection
          draft={draft}
          setDraft={setDraft}
          validationErrors={validationErrors}
          setValidationError={setValidationError}
        />
        <ExtractorSection
          draft={draft}
          setDraft={setDraft}
          validationErrors={validationErrors}
          setValidationError={setValidationError}
        />
        <FileCapsSection
          draft={draft}
          setDraft={setDraft}
          validationErrors={validationErrors}
          setValidationError={setValidationError}
        />
      </SettingsAdvancedFold>
    </SettingsPageFrame>
  );
}
