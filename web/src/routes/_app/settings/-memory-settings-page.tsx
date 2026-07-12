import { AlertCircle } from "lucide-react";
import { useState } from "react";

import { useSettingsMemoryPage } from "@/hooks/routes/use-settings-memory-page";
import { restartBannerPropsFor, SettingsSaveBar } from "@/systems/settings";
import {
  Button,
  PageShell,
  RestartBanner,
  Spinner,
  StatusLineTopbarSlot,
  useTopbarSlot,
} from "@agh/ui";
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

function formatHealthTimestamp(timestamp: string): string {
  return timestamp.replace("T", " ").replace(/\.\d+Z$/, "Z");
}

export function MemorySettingsPage() {
  const page = useSettingsMemoryPage();
  const [validationErrors, setValidationErrors] = useState<Record<string, string | null>>({});
  const setValidationError: ValidationSetter = (key: string) => (message: string | null) => {
    setValidationErrors(current =>
      current[key] === message ? current : { ...current, [key]: message }
    );
  };
  const isInvalid = Object.values(validationErrors).some(message => message !== null);
  const healthForSlot = page.envelope?.health;
  useTopbarSlot({
    tabs: healthForSlot ? (
      <StatusLineTopbarSlot
        data-testid={`${TEST_PREFIX}-status-line`}
        status={healthForSlot.available ? "connected" : "error"}
        items={[
          {
            key: "files",
            value: `${healthForSlot.file_count} memory files`,
            tone: "neutral",
          },
          {
            key: "last",
            value: (
              <span data-testid={`${TEST_PREFIX}-last-consolidated`}>
                {healthForSlot.last_consolidated_at
                  ? `last dream ${formatHealthTimestamp(healthForSlot.last_consolidated_at)}`
                  : "no dream runs yet"}
              </span>
            ),
            tone: "neutral",
          },
          {
            key: "dream-state",
            value: healthForSlot.dream_enabled ? "dreaming enabled" : "dreaming disabled",
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
  const dreamAvailable =
    envelope.actions.consolidate.available && envelope.health.dream_enabled && draft.dream.enabled;

  const bannerProps = restartBannerPropsFor("memory", restart);

  return (
    <PageShell
      slug="memory"
      banner={bannerProps ? <RestartBanner {...bannerProps} /> : null}
      footer={
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
    >
      <MemorySystemSection draft={draft} setDraft={setDraft} />
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
      <RecallSection
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
      <FileCapsSection
        draft={draft}
        setDraft={setDraft}
        validationErrors={validationErrors}
        setValidationError={setValidationError}
      />
      <WorkspaceIdentitySection draft={draft} setDraft={setDraft} />
    </PageShell>
  );
}
