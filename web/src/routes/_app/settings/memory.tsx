import { AlertCircle, Loader2, Play } from "lucide-react";
import { createFileRoute } from "@tanstack/react-router";
import { useCallback, useMemo, useState, type Dispatch, type SetStateAction } from "react";

import { Button, Input, Switch } from "@agh/ui";
import { useSettingsMemoryPage } from "@/hooks/routes/use-settings-memory-page";
import type { SettingsMemorySection } from "@/systems/settings";
import {
  SettingsDecimalInput,
  SettingsFieldRow,
  SettingsNumberInput,
  SettingsPageActions,
  SettingsPageShell,
  SettingsRestartBanner,
  SettingsSaveBar,
  SettingsSectionCard,
  SettingsStatusLine,
} from "@/systems/settings/components";

export const Route = createFileRoute("/_app/settings/memory")({
  component: MemorySettingsPage,
});

type MemoryConfig = SettingsMemorySection["config"];
type ValidationSetter = (key: string) => (message: string | null) => void;

const TEST_PREFIX = "settings-page-memory";

export function MemorySettingsPage() {
  const page = useSettingsMemoryPage();
  const [validationErrors, setValidationErrors] = useState<Record<string, string | null>>({});
  const setValidationError = useCallback<ValidationSetter>(
    (key: string) => (message: string | null) => {
      setValidationErrors(current =>
        current[key] === message ? current : { ...current, [key]: message }
      );
    },
    []
  );
  const isInvalid = useMemo(
    () => Object.values(validationErrors).some(message => message !== null),
    [validationErrors]
  );

  if (page.isLoading) {
    return (
      <div
        className="flex flex-1 items-center justify-center"
        data-testid={`${TEST_PREFIX}-loading`}
      >
        <Loader2 className="size-5 animate-spin text-(--color-text-tertiary)" />
      </div>
    );
  }

  if (page.error || !page.envelope || !page.draft) {
    return (
      <div className="flex flex-1 items-center justify-center" data-testid={`${TEST_PREFIX}-error`}>
        <div className="flex flex-col items-center gap-2 text-center">
          <AlertCircle className="size-6 text-(--color-danger)" />
          <p className="text-sm text-(--color-text-tertiary)">
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
    <SettingsPageShell
      slug="memory"
      title="Memory"
      statusLine={
        <SettingsStatusLine
          data-testid={`${TEST_PREFIX}-status-line`}
          daemonAvailable={health.available}
          items={[
            <span key="files">{health.file_count} memory files</span>,
            <span key="last" data-testid={`${TEST_PREFIX}-last-consolidated`}>
              {health.last_consolidated_at
                ? `last dream ${new Date(health.last_consolidated_at).toLocaleString()}`
                : "no dream runs yet"}
            </span>,
            <span key="dream-state">
              {health.dream_enabled ? "dreaming enabled" : "dreaming disabled"}
            </span>,
          ]}
        />
      }
      actions={<SettingsPageActions slug="memory" restart={restart} />}
      banner={<SettingsRestartBanner slug="memory" restart={restart} />}
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
    </SettingsPageShell>
  );
}

interface DraftSectionProps {
  draft: MemoryConfig;
  setDraft: Dispatch<SetStateAction<MemoryConfig | null>>;
}

interface ValidatedSectionProps extends DraftSectionProps {
  validationErrors: Record<string, string | null>;
  setValidationError: ValidationSetter;
}

function MemorySystemSection({ draft, setDraft }: DraftSectionProps) {
  return (
    <SettingsSectionCard eyebrow="Memory system">
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-enabled`}
        label="Memory persistence"
        description="Persist curated recall across sessions"
        control={
          <Switch
            data-testid={`${TEST_PREFIX}-enabled-switch`}
            checked={draft.enabled}
            onCheckedChange={checked => setDraft({ ...draft, enabled: checked })}
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-global-dir`}
        label="Global memory directory"
        description="Root for global-scope memory files"
        hint="DEFAULT"
        control={
          <Input
            className="w-72 font-mono"
            data-testid={`${TEST_PREFIX}-global-dir-input`}
            value={draft.global_dir ?? ""}
            placeholder="~/.agh/memory"
            onChange={event =>
              setDraft({
                ...draft,
                global_dir: event.target.value === "" ? undefined : event.target.value,
              })
            }
          />
        }
      />
    </SettingsSectionCard>
  );
}

function ProviderResilienceSection({
  draft,
  setDraft,
  validationErrors,
  setValidationError,
}: ValidatedSectionProps) {
  return (
    <SettingsSectionCard
      eyebrow="Memory provider"
      note="circuit-breaker policy when an external memory provider is configured"
    >
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-provider-name`}
        label="Provider name"
        description="Empty falls back to the bundled local provider"
        control={
          <Input
            className="w-56 font-mono"
            data-testid={`${TEST_PREFIX}-provider-name-input`}
            value={draft.provider.name}
            placeholder="local"
            onChange={event =>
              setDraft({
                ...draft,
                provider: { ...draft.provider, name: event.target.value },
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-provider-timeout`}
        label="Per-call timeout"
        description="Deadline for each provider method before failing open to local"
        control={
          <Input
            className="w-32 font-mono"
            data-testid={`${TEST_PREFIX}-provider-timeout-input`}
            value={draft.provider.timeout}
            placeholder="2s"
            onChange={event =>
              setDraft({
                ...draft,
                provider: { ...draft.provider, timeout: event.target.value },
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-provider-failure-threshold`}
        label="Failure threshold"
        description="Consecutive failures before the breaker opens"
        error={validationErrors.providerFailureThreshold ?? undefined}
        control={
          <SettingsNumberInput
            min={1}
            className="w-24"
            data-testid={`${TEST_PREFIX}-provider-failure-threshold-input`}
            value={draft.provider.failure_threshold}
            onValidityChange={setValidationError("providerFailureThreshold")}
            onValueChange={value =>
              setDraft({
                ...draft,
                provider: { ...draft.provider, failure_threshold: value },
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-provider-cooldown`}
        label="Cooldown"
        description="How long the breaker stays open before retrying"
        control={
          <Input
            className="w-32 font-mono"
            data-testid={`${TEST_PREFIX}-provider-cooldown-input`}
            value={draft.provider.cooldown}
            placeholder="30s"
            onChange={event =>
              setDraft({
                ...draft,
                provider: { ...draft.provider, cooldown: event.target.value },
              })
            }
          />
        }
      />
    </SettingsSectionCard>
  );
}

function ControllerSection({
  draft,
  setDraft,
  validationErrors,
  setValidationError,
}: ValidatedSectionProps) {
  const allowOrigins = draft.controller.policy.allow_origins.join(", ");
  return (
    <SettingsSectionCard
      eyebrow="Write controller"
      note="lexical/entity-only ADD / UPDATE / DELETE / NOOP / REJECT pipeline"
    >
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-controller-mode`}
        label="Controller mode"
        description="hybrid uses rules with an LLM tiebreaker; rules and llm pin a single strategy"
        control={
          <Input
            className="w-40 font-mono"
            data-testid={`${TEST_PREFIX}-controller-mode-input`}
            value={draft.controller.mode}
            placeholder="hybrid"
            onChange={event =>
              setDraft({
                ...draft,
                controller: { ...draft.controller, mode: event.target.value },
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-controller-max-latency`}
        label="Max latency"
        description="Hard deadline before the controller falls back to default-op"
        control={
          <Input
            className="w-32 font-mono"
            data-testid={`${TEST_PREFIX}-controller-max-latency-input`}
            value={draft.controller.max_latency}
            placeholder="300ms"
            onChange={event =>
              setDraft({
                ...draft,
                controller: { ...draft.controller, max_latency: event.target.value },
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-controller-default-op`}
        label="Default op on fail"
        description="Decision used when the controller bails (e.g. timeout, schema drift)"
        control={
          <Input
            className="w-32 font-mono"
            data-testid={`${TEST_PREFIX}-controller-default-op-input`}
            value={draft.controller.default_op_on_fail}
            placeholder="noop"
            onChange={event =>
              setDraft({
                ...draft,
                controller: {
                  ...draft.controller,
                  default_op_on_fail: event.target.value,
                },
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-controller-policy-max-content`}
        label="Max content chars"
        description="Per-candidate body cap enforced before the controller decides"
        error={validationErrors.policyMaxContentChars ?? undefined}
        control={
          <SettingsNumberInput
            min={0}
            className="w-32"
            data-testid={`${TEST_PREFIX}-controller-policy-max-content-input`}
            value={draft.controller.policy.max_content_chars}
            onValidityChange={setValidationError("policyMaxContentChars")}
            onValueChange={value =>
              setDraft({
                ...draft,
                controller: {
                  ...draft.controller,
                  policy: { ...draft.controller.policy, max_content_chars: value },
                },
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-controller-policy-max-writes`}
        label="Max writes per minute"
        description="Soft rate limit applied at controller entry"
        error={validationErrors.policyMaxWritesPerMin ?? undefined}
        control={
          <SettingsNumberInput
            min={0}
            className="w-32"
            data-testid={`${TEST_PREFIX}-controller-policy-max-writes-input`}
            value={draft.controller.policy.max_writes_per_min}
            onValidityChange={setValidationError("policyMaxWritesPerMin")}
            onValueChange={value =>
              setDraft({
                ...draft,
                controller: {
                  ...draft.controller,
                  policy: { ...draft.controller.policy, max_writes_per_min: value },
                },
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-controller-policy-allow-origins`}
        label="Allowed origins"
        description="Read-only roster of write origins permitted by this build"
        control={
          <Input
            readOnly
            className="w-full font-mono"
            data-testid={`${TEST_PREFIX}-controller-policy-allow-origins-input`}
            value={allowOrigins}
          />
        }
      />
    </SettingsSectionCard>
  );
}

function ControllerLLMSection({
  draft,
  setDraft,
  validationErrors,
  setValidationError,
}: ValidatedSectionProps) {
  const llmDisabled = !draft.controller.llm.enabled;
  return (
    <SettingsSectionCard
      eyebrow="Controller LLM tiebreaker"
      note="entity-slot ambiguity escalations"
    >
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-controller-llm-enabled`}
        label="LLM tiebreaker"
        description="Allow the controller to escalate ambiguous slot matches to the configured LLM"
        control={
          <Switch
            data-testid={`${TEST_PREFIX}-controller-llm-enabled-switch`}
            checked={draft.controller.llm.enabled}
            onCheckedChange={checked =>
              setDraft({
                ...draft,
                controller: {
                  ...draft.controller,
                  llm: { ...draft.controller.llm, enabled: checked },
                },
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-controller-llm-model`}
        label="Model"
        description="Provider-prefixed identifier (e.g. anthropic/claude-haiku-4)"
        control={
          <Input
            className="w-72 font-mono"
            disabled={llmDisabled}
            data-testid={`${TEST_PREFIX}-controller-llm-model-input`}
            value={draft.controller.llm.model}
            placeholder="anthropic/claude-haiku-4"
            onChange={event =>
              setDraft({
                ...draft,
                controller: {
                  ...draft.controller,
                  llm: { ...draft.controller.llm, model: event.target.value },
                },
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-controller-llm-top-k`}
        label="Tiebreaker top-K"
        description="Candidate slugs passed to the LLM when slots are ambiguous"
        error={validationErrors.controllerLlmTopK ?? undefined}
        control={
          <SettingsNumberInput
            min={1}
            disabled={llmDisabled}
            className="w-24"
            data-testid={`${TEST_PREFIX}-controller-llm-top-k-input`}
            value={draft.controller.llm.top_k}
            onValidityChange={setValidationError("controllerLlmTopK")}
            onValueChange={value =>
              setDraft({
                ...draft,
                controller: {
                  ...draft.controller,
                  llm: { ...draft.controller.llm, top_k: value },
                },
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-controller-llm-max-tokens`}
        label="Max output tokens"
        description="Caps the tiebreaker response so it stays within budget"
        error={validationErrors.controllerLlmMaxTokens ?? undefined}
        control={
          <SettingsNumberInput
            min={1}
            disabled={llmDisabled}
            className="w-24"
            data-testid={`${TEST_PREFIX}-controller-llm-max-tokens-input`}
            value={draft.controller.llm.max_tokens_out}
            onValidityChange={setValidationError("controllerLlmMaxTokens")}
            onValueChange={value =>
              setDraft({
                ...draft,
                controller: {
                  ...draft.controller,
                  llm: { ...draft.controller.llm, max_tokens_out: value },
                },
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-controller-llm-timeout`}
        label="Timeout"
        description="Tiebreaker call deadline; expiry counts as a controller fallback"
        control={
          <Input
            className="w-32 font-mono"
            disabled={llmDisabled}
            data-testid={`${TEST_PREFIX}-controller-llm-timeout-input`}
            value={draft.controller.llm.timeout}
            placeholder="250ms"
            onChange={event =>
              setDraft({
                ...draft,
                controller: {
                  ...draft.controller,
                  llm: { ...draft.controller.llm, timeout: event.target.value },
                },
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-controller-llm-prompt-version`}
        label="Prompt version"
        description="Pinned controller-prompt revision for reproducible decisions"
        control={
          <Input
            className="w-32 font-mono"
            disabled={llmDisabled}
            data-testid={`${TEST_PREFIX}-controller-llm-prompt-version-input`}
            value={draft.controller.llm.prompt_version}
            placeholder="v1"
            onChange={event =>
              setDraft({
                ...draft,
                controller: {
                  ...draft.controller,
                  llm: { ...draft.controller.llm, prompt_version: event.target.value },
                },
              })
            }
          />
        }
      />
    </SettingsSectionCard>
  );
}

function RecallSection({
  draft,
  setDraft,
  validationErrors,
  setValidationError,
}: ValidatedSectionProps) {
  return (
    <SettingsSectionCard
      eyebrow="Recall pipeline"
      note="deterministic FTS5 + scope-shadow + freshness banner"
    >
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-recall-top-k`}
        label="Top-K"
        description="Curated entries surfaced per recall after fusion"
        error={validationErrors.recallTopK ?? undefined}
        control={
          <SettingsNumberInput
            min={1}
            className="w-24"
            data-testid={`${TEST_PREFIX}-recall-top-k-input`}
            value={draft.recall.top_k}
            onValidityChange={setValidationError("recallTopK")}
            onValueChange={value =>
              setDraft({
                ...draft,
                recall: { ...draft.recall, top_k: value },
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-recall-raw-candidates`}
        label="Raw candidates"
        description="Pre-fusion candidate pool size pulled from each FTS lane"
        error={validationErrors.recallRawCandidates ?? undefined}
        control={
          <SettingsNumberInput
            min={1}
            className="w-24"
            data-testid={`${TEST_PREFIX}-recall-raw-candidates-input`}
            value={draft.recall.raw_candidates}
            onValidityChange={setValidationError("recallRawCandidates")}
            onValueChange={value =>
              setDraft({
                ...draft,
                recall: { ...draft.recall, raw_candidates: value },
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-recall-fusion`}
        label="Fusion strategy"
        description="weighted is the only strategy in Slice 1; rrf is reserved for Slice 3"
        control={
          <Input
            className="w-32 font-mono"
            data-testid={`${TEST_PREFIX}-recall-fusion-input`}
            value={draft.recall.fusion}
            placeholder="weighted"
            onChange={event =>
              setDraft({
                ...draft,
                recall: { ...draft.recall, fusion: event.target.value },
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-recall-include-already-surfaced`}
        label="Include already surfaced"
        description="Re-include entries already injected this session"
        control={
          <Switch
            data-testid={`${TEST_PREFIX}-recall-include-already-surfaced-switch`}
            checked={draft.recall.include_already_surfaced}
            onCheckedChange={checked =>
              setDraft({
                ...draft,
                recall: { ...draft.recall, include_already_surfaced: checked },
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-recall-include-system`}
        label="Include _system entries"
        description="Surface dreaming, extractor, and ad-hoc files (normally hidden)"
        control={
          <Switch
            data-testid={`${TEST_PREFIX}-recall-include-system-switch`}
            checked={draft.recall.include_system}
            onCheckedChange={checked =>
              setDraft({
                ...draft,
                recall: { ...draft.recall, include_system: checked },
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-recall-weight-bm25-unicode`}
        label="Weight · BM25 unicode"
        description="Score blend coefficient for the unicode FTS lane"
        error={validationErrors.recallWeightUnicode ?? undefined}
        control={
          <SettingsDecimalInput
            min={0}
            max={1}
            precision={2}
            className="w-24"
            data-testid={`${TEST_PREFIX}-recall-weight-bm25-unicode-input`}
            value={draft.recall.weights.bm25_unicode}
            onValidityChange={setValidationError("recallWeightUnicode")}
            onValueChange={value =>
              setDraft({
                ...draft,
                recall: {
                  ...draft.recall,
                  weights: { ...draft.recall.weights, bm25_unicode: value },
                },
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-recall-weight-bm25-trigram`}
        label="Weight · BM25 trigram"
        description="Score blend coefficient for the trigram FTS lane"
        error={validationErrors.recallWeightTrigram ?? undefined}
        control={
          <SettingsDecimalInput
            min={0}
            max={1}
            precision={2}
            className="w-24"
            data-testid={`${TEST_PREFIX}-recall-weight-bm25-trigram-input`}
            value={draft.recall.weights.bm25_trigram}
            onValidityChange={setValidationError("recallWeightTrigram")}
            onValueChange={value =>
              setDraft({
                ...draft,
                recall: {
                  ...draft.recall,
                  weights: { ...draft.recall.weights, bm25_trigram: value },
                },
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-recall-weight-recency`}
        label="Weight · recency"
        description="Score blend coefficient for the recency signal"
        error={validationErrors.recallWeightRecency ?? undefined}
        control={
          <SettingsDecimalInput
            min={0}
            max={1}
            precision={2}
            className="w-24"
            data-testid={`${TEST_PREFIX}-recall-weight-recency-input`}
            value={draft.recall.weights.recency}
            onValidityChange={setValidationError("recallWeightRecency")}
            onValueChange={value =>
              setDraft({
                ...draft,
                recall: {
                  ...draft.recall,
                  weights: { ...draft.recall.weights, recency: value },
                },
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-recall-weight-recall-signal`}
        label="Weight · recall signal"
        description="Score blend coefficient for prior-recall reinforcement"
        error={validationErrors.recallWeightRecallSignal ?? undefined}
        control={
          <SettingsDecimalInput
            min={0}
            max={1}
            precision={2}
            className="w-24"
            data-testid={`${TEST_PREFIX}-recall-weight-recall-signal-input`}
            value={draft.recall.weights.recall_signal}
            onValidityChange={setValidationError("recallWeightRecallSignal")}
            onValueChange={value =>
              setDraft({
                ...draft,
                recall: {
                  ...draft.recall,
                  weights: { ...draft.recall.weights, recall_signal: value },
                },
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-recall-banner-after-days`}
        label="Freshness banner after"
        description="Days before a surfaced entry shows a staleness banner"
        error={validationErrors.recallBannerAfter ?? undefined}
        control={
          <SettingsNumberInput
            min={0}
            className="w-24"
            data-testid={`${TEST_PREFIX}-recall-banner-after-days-input`}
            value={draft.recall.freshness.banner_after_days}
            onValidityChange={setValidationError("recallBannerAfter")}
            onValueChange={value =>
              setDraft({
                ...draft,
                recall: {
                  ...draft.recall,
                  freshness: { ...draft.recall.freshness, banner_after_days: value },
                },
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-recall-signals-queue`}
        label="Signal queue capacity"
        description="Bounded post-recall signal queue; oldest entries drop on overflow"
        error={validationErrors.recallSignalQueue ?? undefined}
        control={
          <SettingsNumberInput
            min={1}
            className="w-32"
            data-testid={`${TEST_PREFIX}-recall-signals-queue-input`}
            value={draft.recall.signals.queue_capacity}
            onValidityChange={setValidationError("recallSignalQueue")}
            onValueChange={value =>
              setDraft({
                ...draft,
                recall: {
                  ...draft.recall,
                  signals: { ...draft.recall.signals, queue_capacity: value },
                },
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-recall-signals-retry`}
        label="Signal retry max"
        description="Per-update attempts before emitting a failed-signal event"
        error={validationErrors.recallSignalRetry ?? undefined}
        control={
          <SettingsNumberInput
            min={0}
            className="w-24"
            data-testid={`${TEST_PREFIX}-recall-signals-retry-input`}
            value={draft.recall.signals.worker_retry_max}
            onValidityChange={setValidationError("recallSignalRetry")}
            onValueChange={value =>
              setDraft({
                ...draft,
                recall: {
                  ...draft.recall,
                  signals: { ...draft.recall.signals, worker_retry_max: value },
                },
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-recall-signals-metrics`}
        label="Signal metrics"
        description="Emit recall-signal counters for observability dashboards"
        control={
          <Switch
            data-testid={`${TEST_PREFIX}-recall-signals-metrics-switch`}
            checked={draft.recall.signals.metrics_enabled}
            onCheckedChange={checked =>
              setDraft({
                ...draft,
                recall: {
                  ...draft.recall,
                  signals: { ...draft.recall.signals, metrics_enabled: checked },
                },
              })
            }
          />
        }
      />
    </SettingsSectionCard>
  );
}

function DecisionsSection({
  draft,
  setDraft,
  validationErrors,
  setValidationError,
}: ValidatedSectionProps) {
  return (
    <SettingsSectionCard eyebrow="Decisions retention" note="memory_decisions WAL housekeeping">
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-decisions-prune-after`}
        label="Prune after applied (days)"
        description="Delete applied decisions older than this; 0 disables pruning"
        error={validationErrors.decisionsPruneAfter ?? undefined}
        control={
          <SettingsNumberInput
            min={0}
            className="w-24"
            data-testid={`${TEST_PREFIX}-decisions-prune-after-input`}
            value={draft.decisions.prune_after_applied_days}
            onValidityChange={setValidationError("decisionsPruneAfter")}
            onValueChange={value =>
              setDraft({
                ...draft,
                decisions: { ...draft.decisions, prune_after_applied_days: value },
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-decisions-keep-summary`}
        label="Keep audit summary on prune"
        description="Emit memory.decisions.audit_summarized before deleting old rows"
        control={
          <Switch
            data-testid={`${TEST_PREFIX}-decisions-keep-summary-switch`}
            checked={draft.decisions.keep_audit_summary}
            onCheckedChange={checked =>
              setDraft({
                ...draft,
                decisions: { ...draft.decisions, keep_audit_summary: checked },
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-decisions-max-post-content`}
        label="Max post_content bytes"
        description="Per-row body cap; oversize rows store a content-hash reference instead"
        error={validationErrors.decisionsMaxPostBytes ?? undefined}
        control={
          <SettingsNumberInput
            min={0}
            className="w-32"
            data-testid={`${TEST_PREFIX}-decisions-max-post-content-input`}
            value={draft.decisions.max_post_content_bytes}
            onValidityChange={setValidationError("decisionsMaxPostBytes")}
            onValueChange={value =>
              setDraft({
                ...draft,
                decisions: { ...draft.decisions, max_post_content_bytes: value },
              })
            }
          />
        }
      />
    </SettingsSectionCard>
  );
}

function ExtractorSection({
  draft,
  setDraft,
  validationErrors,
  setValidationError,
}: ValidatedSectionProps) {
  const extractorDisabled = !draft.extractor.enabled;
  return (
    <SettingsSectionCard eyebrow="Extractor" note="post-message proposal generation">
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-extractor-enabled`}
        label="Extractor"
        description="Spawn the extractor sub-agent on session.message_persisted"
        control={
          <Switch
            data-testid={`${TEST_PREFIX}-extractor-enabled-switch`}
            checked={draft.extractor.enabled}
            onCheckedChange={checked =>
              setDraft({
                ...draft,
                extractor: { ...draft.extractor, enabled: checked },
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-extractor-mode`}
        label="Mode"
        description="post_message is the only Slice 1 mode; compaction_flush ships in Slice 2"
        control={
          <Input
            className="w-40 font-mono"
            disabled={extractorDisabled}
            data-testid={`${TEST_PREFIX}-extractor-mode-input`}
            value={draft.extractor.mode}
            placeholder="post_message"
            onChange={event =>
              setDraft({
                ...draft,
                extractor: { ...draft.extractor, mode: event.target.value },
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-extractor-throttle`}
        label="Throttle turns"
        description="Skip N turns between extractor invocations"
        error={validationErrors.extractorThrottle ?? undefined}
        control={
          <SettingsNumberInput
            min={0}
            disabled={extractorDisabled}
            className="w-24"
            data-testid={`${TEST_PREFIX}-extractor-throttle-input`}
            value={draft.extractor.throttle_turns}
            onValidityChange={setValidationError("extractorThrottle")}
            onValueChange={value =>
              setDraft({
                ...draft,
                extractor: { ...draft.extractor, throttle_turns: value },
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-extractor-deadline`}
        label="Deadline"
        description="Per-extraction wall clock budget"
        control={
          <Input
            className="w-32 font-mono"
            disabled={extractorDisabled}
            data-testid={`${TEST_PREFIX}-extractor-deadline-input`}
            value={draft.extractor.deadline}
            placeholder="60s"
            onChange={event =>
              setDraft({
                ...draft,
                extractor: { ...draft.extractor, deadline: event.target.value },
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-extractor-sandbox`}
        label="Sandbox to inbox only"
        description="Restrict the extractor sub-agent to writes under _inbox/"
        control={
          <Switch
            disabled={extractorDisabled}
            data-testid={`${TEST_PREFIX}-extractor-sandbox-switch`}
            checked={draft.extractor.sandbox_inbox_only}
            onCheckedChange={checked =>
              setDraft({
                ...draft,
                extractor: { ...draft.extractor, sandbox_inbox_only: checked },
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-extractor-model`}
        label="Model override"
        description="Empty inherits the parent session model"
        control={
          <Input
            className="w-72 font-mono"
            disabled={extractorDisabled}
            data-testid={`${TEST_PREFIX}-extractor-model-input`}
            value={draft.extractor.model}
            placeholder="(inherit)"
            onChange={event =>
              setDraft({
                ...draft,
                extractor: { ...draft.extractor, model: event.target.value },
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-extractor-queue-capacity`}
        label="Queue capacity"
        description="Per-session in-flight extraction slots"
        error={validationErrors.extractorQueueCapacity ?? undefined}
        control={
          <SettingsNumberInput
            min={1}
            disabled={extractorDisabled}
            className="w-24"
            data-testid={`${TEST_PREFIX}-extractor-queue-capacity-input`}
            value={draft.extractor.queue.capacity}
            onValidityChange={setValidationError("extractorQueueCapacity")}
            onValueChange={value =>
              setDraft({
                ...draft,
                extractor: {
                  ...draft.extractor,
                  queue: { ...draft.extractor.queue, capacity: value },
                },
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-extractor-coalesce-max`}
        label="Coalesce ceiling"
        description="Maximum coalesced batches before drop-oldest kicks in"
        error={validationErrors.extractorCoalesce ?? undefined}
        control={
          <SettingsNumberInput
            min={1}
            disabled={extractorDisabled}
            className="w-24"
            data-testid={`${TEST_PREFIX}-extractor-coalesce-max-input`}
            value={draft.extractor.queue.coalesce_max}
            onValidityChange={setValidationError("extractorCoalesce")}
            onValueChange={value =>
              setDraft({
                ...draft,
                extractor: {
                  ...draft.extractor,
                  queue: { ...draft.extractor.queue, coalesce_max: value },
                },
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-extractor-inbox-path`}
        label="Inbox path"
        description="Read-only daemon-managed location for extractor JSONL output"
        control={
          <Input
            readOnly
            className="w-full font-mono"
            data-testid={`${TEST_PREFIX}-extractor-inbox-path-input`}
            value={draft.extractor.inbox_path}
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-extractor-dlq-path`}
        label="DLQ path"
        description="Read-only daemon-managed location for extractor failure records"
        control={
          <Input
            readOnly
            className="w-full font-mono"
            data-testid={`${TEST_PREFIX}-extractor-dlq-path-input`}
            value={draft.extractor.dlq_path}
          />
        }
      />
    </SettingsSectionCard>
  );
}

interface DreamSectionProps extends ValidatedSectionProps {
  dreamAvailable: boolean;
  dreamPending: boolean;
  onTriggerDream: () => void;
  actionMessage: string | null;
}

function DreamSection({
  draft,
  setDraft,
  validationErrors,
  setValidationError,
  dreamAvailable,
  dreamPending,
  onTriggerDream,
  actionMessage,
}: DreamSectionProps) {
  const dreamDisabled = !draft.dream.enabled;
  return (
    <SettingsSectionCard
      eyebrow="Memory dreaming"
      note="background recall-signal scoring + curated promotion"
      headerAction={
        <Button
          type="button"
          variant="outline"
          size="sm"
          data-testid={`${TEST_PREFIX}-dream-trigger`}
          disabled={!dreamAvailable || dreamPending}
          onClick={onTriggerDream}
        >
          {dreamPending ? (
            <Loader2 className="size-3.5 animate-spin" />
          ) : (
            <Play className="size-3.5" />
          )}
          Trigger dream
        </Button>
      }
    >
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-dream-enabled`}
        label="Automatic dreaming"
        description="Run dreaming on idle when the recall-signal gate is satisfied"
        control={
          <Switch
            data-testid={`${TEST_PREFIX}-dream-enabled-switch`}
            checked={draft.dream.enabled}
            onCheckedChange={checked =>
              setDraft({ ...draft, dream: { ...draft.dream, enabled: checked } })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-dream-agent`}
        label="Dream agent"
        description="Dedicated curator agent (defaults to dreaming-curator)"
        control={
          <Input
            className="w-56 font-mono"
            disabled={dreamDisabled}
            data-testid={`${TEST_PREFIX}-dream-agent-input`}
            value={draft.dream.agent}
            placeholder="dreaming-curator"
            onChange={event =>
              setDraft({ ...draft, dream: { ...draft.dream, agent: event.target.value } })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-dream-min-hours`}
        label="Min idle hours"
        description="Wait at least this many hours since the last dream run"
        error={validationErrors.dreamMinHours ?? undefined}
        control={
          <SettingsDecimalInput
            min={0}
            precision={1}
            disabled={dreamDisabled}
            className="w-24"
            data-testid={`${TEST_PREFIX}-dream-min-hours-input`}
            value={draft.dream.min_hours}
            onValidityChange={setValidationError("dreamMinHours")}
            onValueChange={value =>
              setDraft({
                ...draft,
                dream: { ...draft.dream, min_hours: value },
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-dream-min-sessions`}
        label="Min sessions"
        description="Sessions required since the last dream run"
        error={validationErrors.dreamMinSessions ?? undefined}
        control={
          <SettingsNumberInput
            min={0}
            disabled={dreamDisabled}
            className="w-24"
            data-testid={`${TEST_PREFIX}-dream-min-sessions-input`}
            value={draft.dream.min_sessions}
            onValidityChange={setValidationError("dreamMinSessions")}
            onValueChange={value =>
              setDraft({
                ...draft,
                dream: { ...draft.dream, min_sessions: value },
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-dream-debounce`}
        label="Debounce"
        description="Anti-thrash debounce after a no-op tick"
        control={
          <Input
            className="w-32 font-mono"
            disabled={dreamDisabled}
            data-testid={`${TEST_PREFIX}-dream-debounce-input`}
            value={draft.dream.debounce}
            placeholder="10m"
            onChange={event =>
              setDraft({ ...draft, dream: { ...draft.dream, debounce: event.target.value } })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-dream-check-interval`}
        label="Check interval"
        description="How often the dreaming runtime evaluates idle gates"
        control={
          <Input
            className="w-32 font-mono"
            disabled={dreamDisabled}
            data-testid={`${TEST_PREFIX}-dream-check-interval-input`}
            value={draft.dream.check_interval}
            placeholder="30m"
            onChange={event =>
              setDraft({
                ...draft,
                dream: { ...draft.dream, check_interval: event.target.value },
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-dream-prompt-version`}
        label="Prompt version"
        description="Pinned dreaming-prompt revision; bumping invalidates idempotency keys"
        control={
          <Input
            className="w-32 font-mono"
            disabled={dreamDisabled}
            data-testid={`${TEST_PREFIX}-dream-prompt-version-input`}
            value={draft.dream.prompt_version}
            placeholder="v1"
            onChange={event =>
              setDraft({
                ...draft,
                dream: { ...draft.dream, prompt_version: event.target.value },
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-dream-gate-min-unpromoted`}
        label="Gate · min unpromoted"
        description="Recall-signal candidates that must be unpromoted to start a run"
        error={validationErrors.dreamGateMinUnpromoted ?? undefined}
        control={
          <SettingsNumberInput
            min={0}
            disabled={dreamDisabled}
            className="w-24"
            data-testid={`${TEST_PREFIX}-dream-gate-min-unpromoted-input`}
            value={draft.dream.gates.min_unpromoted}
            onValidityChange={setValidationError("dreamGateMinUnpromoted")}
            onValueChange={value =>
              setDraft({
                ...draft,
                dream: { ...draft.dream, gates: { ...draft.dream.gates, min_unpromoted: value } },
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-dream-gate-min-recall-count`}
        label="Gate · min recall count"
        description="Recall events per candidate required to qualify"
        error={validationErrors.dreamGateMinRecallCount ?? undefined}
        control={
          <SettingsNumberInput
            min={0}
            disabled={dreamDisabled}
            className="w-24"
            data-testid={`${TEST_PREFIX}-dream-gate-min-recall-count-input`}
            value={draft.dream.gates.min_recall_count}
            onValidityChange={setValidationError("dreamGateMinRecallCount")}
            onValueChange={value =>
              setDraft({
                ...draft,
                dream: { ...draft.dream, gates: { ...draft.dream.gates, min_recall_count: value } },
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-dream-gate-min-score`}
        label="Gate · min score"
        description="Promotion score threshold (0–1)"
        error={validationErrors.dreamGateMinScore ?? undefined}
        control={
          <SettingsDecimalInput
            min={0}
            max={1}
            precision={2}
            disabled={dreamDisabled}
            className="w-24"
            data-testid={`${TEST_PREFIX}-dream-gate-min-score-input`}
            value={draft.dream.gates.min_score}
            onValidityChange={setValidationError("dreamGateMinScore")}
            onValueChange={value =>
              setDraft({
                ...draft,
                dream: { ...draft.dream, gates: { ...draft.dream.gates, min_score: value } },
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-dream-scoring-recency-half-life`}
        label="Scoring · recency half-life (days)"
        description="Half-life applied to the recency component"
        error={validationErrors.dreamScoringHalfLife ?? undefined}
        control={
          <SettingsNumberInput
            min={1}
            disabled={dreamDisabled}
            className="w-24"
            data-testid={`${TEST_PREFIX}-dream-scoring-recency-half-life-input`}
            value={draft.dream.scoring.recency_half_life_days}
            onValidityChange={setValidationError("dreamScoringHalfLife")}
            onValueChange={value =>
              setDraft({
                ...draft,
                dream: {
                  ...draft.dream,
                  scoring: { ...draft.dream.scoring, recency_half_life_days: value },
                },
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-dream-scoring-weight-frequency`}
        label="Score weight · frequency"
        error={validationErrors.dreamScoreWeightFrequency ?? undefined}
        control={
          <SettingsDecimalInput
            min={0}
            max={1}
            precision={2}
            disabled={dreamDisabled}
            className="w-24"
            data-testid={`${TEST_PREFIX}-dream-scoring-weight-frequency-input`}
            value={draft.dream.scoring.weights.frequency}
            onValidityChange={setValidationError("dreamScoreWeightFrequency")}
            onValueChange={value =>
              setDraft({
                ...draft,
                dream: {
                  ...draft.dream,
                  scoring: {
                    ...draft.dream.scoring,
                    weights: { ...draft.dream.scoring.weights, frequency: value },
                  },
                },
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-dream-scoring-weight-relevance`}
        label="Score weight · relevance"
        error={validationErrors.dreamScoreWeightRelevance ?? undefined}
        control={
          <SettingsDecimalInput
            min={0}
            max={1}
            precision={2}
            disabled={dreamDisabled}
            className="w-24"
            data-testid={`${TEST_PREFIX}-dream-scoring-weight-relevance-input`}
            value={draft.dream.scoring.weights.relevance}
            onValidityChange={setValidationError("dreamScoreWeightRelevance")}
            onValueChange={value =>
              setDraft({
                ...draft,
                dream: {
                  ...draft.dream,
                  scoring: {
                    ...draft.dream.scoring,
                    weights: { ...draft.dream.scoring.weights, relevance: value },
                  },
                },
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-dream-scoring-weight-recency`}
        label="Score weight · recency"
        error={validationErrors.dreamScoreWeightRecency ?? undefined}
        control={
          <SettingsDecimalInput
            min={0}
            max={1}
            precision={2}
            disabled={dreamDisabled}
            className="w-24"
            data-testid={`${TEST_PREFIX}-dream-scoring-weight-recency-input`}
            value={draft.dream.scoring.weights.recency}
            onValidityChange={setValidationError("dreamScoreWeightRecency")}
            onValueChange={value =>
              setDraft({
                ...draft,
                dream: {
                  ...draft.dream,
                  scoring: {
                    ...draft.dream.scoring,
                    weights: { ...draft.dream.scoring.weights, recency: value },
                  },
                },
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-dream-scoring-weight-freshness`}
        label="Score weight · freshness"
        error={validationErrors.dreamScoreWeightFreshness ?? undefined}
        control={
          <SettingsDecimalInput
            min={0}
            max={1}
            precision={2}
            disabled={dreamDisabled}
            className="w-24"
            data-testid={`${TEST_PREFIX}-dream-scoring-weight-freshness-input`}
            value={draft.dream.scoring.weights.freshness}
            onValidityChange={setValidationError("dreamScoreWeightFreshness")}
            onValueChange={value =>
              setDraft({
                ...draft,
                dream: {
                  ...draft.dream,
                  scoring: {
                    ...draft.dream.scoring,
                    weights: { ...draft.dream.scoring.weights, freshness: value },
                  },
                },
              })
            }
          />
        }
      />
      {actionMessage ? (
        <p
          className="text-xs text-(--color-text-tertiary)"
          data-testid={`${TEST_PREFIX}-action-message`}
        >
          {actionMessage}
        </p>
      ) : null}
    </SettingsSectionCard>
  );
}

function SessionLedgerSection({
  draft,
  setDraft,
  validationErrors,
  setValidationError,
}: ValidatedSectionProps) {
  return (
    <SettingsSectionCard eyebrow="Session ledger" note="forensic JSONL ledger materialization">
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-session-ledger-format`}
        label="Ledger format"
        description="jsonl is the only Slice 1 ledger format"
        control={
          <Input
            className="w-32 font-mono"
            data-testid={`${TEST_PREFIX}-session-ledger-format-input`}
            value={draft.session.ledger_format}
            placeholder="jsonl"
            onChange={event =>
              setDraft({
                ...draft,
                session: { ...draft.session, ledger_format: event.target.value },
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-session-events-purge-grace`}
        label="Events purge grace"
        description="Hold events.db rows this long before purging materialized ledgers"
        control={
          <Input
            className="w-32 font-mono"
            data-testid={`${TEST_PREFIX}-session-events-purge-grace-input`}
            value={draft.session.events_purge_grace}
            placeholder="24h"
            onChange={event =>
              setDraft({
                ...draft,
                session: { ...draft.session, events_purge_grace: event.target.value },
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-session-cold-archive-days`}
        label="Cold archive (days)"
        description="Move ledgers to cold archive after this many days"
        error={validationErrors.sessionColdArchive ?? undefined}
        control={
          <SettingsNumberInput
            min={0}
            className="w-24"
            data-testid={`${TEST_PREFIX}-session-cold-archive-days-input`}
            value={draft.session.cold_archive_days}
            onValidityChange={setValidationError("sessionColdArchive")}
            onValueChange={value =>
              setDraft({
                ...draft,
                session: { ...draft.session, cold_archive_days: value },
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-session-hard-delete-days`}
        label="Hard delete (days)"
        description="0 means never auto-delete; ledgers prune only via explicit CLI"
        error={validationErrors.sessionHardDelete ?? undefined}
        control={
          <SettingsNumberInput
            min={0}
            className="w-24"
            data-testid={`${TEST_PREFIX}-session-hard-delete-days-input`}
            value={draft.session.hard_delete_days}
            onValidityChange={setValidationError("sessionHardDelete")}
            onValueChange={value =>
              setDraft({
                ...draft,
                session: { ...draft.session, hard_delete_days: value },
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-session-max-archive-bytes`}
        label="Max archive bytes"
        description="Safety valve for cold archive size"
        error={validationErrors.sessionMaxArchive ?? undefined}
        control={
          <SettingsNumberInput
            min={0}
            className="w-40"
            data-testid={`${TEST_PREFIX}-session-max-archive-bytes-input`}
            value={draft.session.max_archive_bytes}
            onValidityChange={setValidationError("sessionMaxArchive")}
            onValueChange={value =>
              setDraft({
                ...draft,
                session: { ...draft.session, max_archive_bytes: value },
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-session-ledger-root`}
        label="Ledger root"
        description="Read-only daemon-managed ledger root"
        control={
          <Input
            readOnly
            className="w-full font-mono"
            data-testid={`${TEST_PREFIX}-session-ledger-root-input`}
            value={draft.session.ledger_root}
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-session-unbound-partition`}
        label="Unbound partition"
        description="Read-only directory for sessions without a workspace binding"
        control={
          <Input
            readOnly
            className="w-full font-mono"
            data-testid={`${TEST_PREFIX}-session-unbound-partition-input`}
            value={draft.session.unbound_partition}
          />
        }
      />
    </SettingsSectionCard>
  );
}

function DailyLogsSection({
  draft,
  setDraft,
  validationErrors,
  setValidationError,
}: ValidatedSectionProps) {
  return (
    <SettingsSectionCard eyebrow="Daily logs" note="rotation, dreaming window, and archival">
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-daily-max-bytes`}
        label="Max bytes per file"
        description="Daily log rotates when it crosses this byte budget"
        error={validationErrors.dailyMaxBytes ?? undefined}
        control={
          <SettingsNumberInput
            min={0}
            className="w-32"
            data-testid={`${TEST_PREFIX}-daily-max-bytes-input`}
            value={draft.daily.max_bytes}
            onValidityChange={setValidationError("dailyMaxBytes")}
            onValueChange={value =>
              setDraft({
                ...draft,
                daily: { ...draft.daily, max_bytes: value },
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-daily-max-lines`}
        label="Max lines per file"
        description="Hard line ceiling before rotation"
        error={validationErrors.dailyMaxLines ?? undefined}
        control={
          <SettingsNumberInput
            min={0}
            className="w-32"
            data-testid={`${TEST_PREFIX}-daily-max-lines-input`}
            value={draft.daily.max_lines}
            onValidityChange={setValidationError("dailyMaxLines")}
            onValueChange={value =>
              setDraft({
                ...draft,
                daily: { ...draft.daily, max_lines: value },
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-daily-sweep-hour`}
        label="Sweep hour"
        description="Local hour for the daily housekeeping sweep"
        error={validationErrors.dailySweepHour ?? undefined}
        control={
          <SettingsNumberInput
            min={0}
            className="w-24"
            data-testid={`${TEST_PREFIX}-daily-sweep-hour-input`}
            value={draft.daily.sweep_hour}
            onValidityChange={setValidationError("dailySweepHour")}
            onValueChange={value =>
              setDraft({
                ...draft,
                daily: { ...draft.daily, sweep_hour: value },
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-daily-dreaming-window`}
        label="Dreaming window (days)"
        description="How many days of daily logs feed dreaming candidates"
        error={validationErrors.dailyDreamingWindow ?? undefined}
        control={
          <SettingsNumberInput
            min={0}
            className="w-24"
            data-testid={`${TEST_PREFIX}-daily-dreaming-window-input`}
            value={draft.daily.dreaming_window}
            onValidityChange={setValidationError("dailyDreamingWindow")}
            onValueChange={value =>
              setDraft({
                ...draft,
                daily: { ...draft.daily, dreaming_window: value },
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-daily-cold-archive-days`}
        label="Cold archive (days)"
        description="Move daily logs to cold archive after this many days"
        error={validationErrors.dailyColdArchive ?? undefined}
        control={
          <SettingsNumberInput
            min={0}
            className="w-24"
            data-testid={`${TEST_PREFIX}-daily-cold-archive-days-input`}
            value={draft.daily.cold_archive_days}
            onValidityChange={setValidationError("dailyColdArchive")}
            onValueChange={value =>
              setDraft({
                ...draft,
                daily: { ...draft.daily, cold_archive_days: value },
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-daily-hard-delete-days`}
        label="Hard delete (days)"
        description="0 means never auto-delete; daily logs prune only via explicit CLI"
        error={validationErrors.dailyHardDelete ?? undefined}
        control={
          <SettingsNumberInput
            min={0}
            className="w-24"
            data-testid={`${TEST_PREFIX}-daily-hard-delete-days-input`}
            value={draft.daily.hard_delete_days}
            onValidityChange={setValidationError("dailyHardDelete")}
            onValueChange={value =>
              setDraft({
                ...draft,
                daily: { ...draft.daily, hard_delete_days: value },
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-daily-max-archive-bytes`}
        label="Max archive bytes"
        description="Safety valve cap for daily-log cold archive size"
        error={validationErrors.dailyMaxArchiveBytes ?? undefined}
        control={
          <SettingsNumberInput
            min={0}
            className="w-40"
            data-testid={`${TEST_PREFIX}-daily-max-archive-bytes-input`}
            value={draft.daily.max_archive_bytes}
            onValidityChange={setValidationError("dailyMaxArchiveBytes")}
            onValueChange={value =>
              setDraft({
                ...draft,
                daily: { ...draft.daily, max_archive_bytes: value },
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-daily-rotate-format`}
        label="Rotate format"
        description="Read-only daemon-managed daily-log rotation pattern"
        control={
          <Input
            readOnly
            className="w-full font-mono"
            data-testid={`${TEST_PREFIX}-daily-rotate-format-input`}
            value={draft.daily.rotate_format}
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-daily-archive-path`}
        label="Archive path"
        description="Read-only relative path for daily-log cold archive"
        control={
          <Input
            readOnly
            className="w-full font-mono"
            data-testid={`${TEST_PREFIX}-daily-archive-path-input`}
            value={draft.daily.archive_path}
          />
        }
      />
    </SettingsSectionCard>
  );
}

function FileCapsSection({
  draft,
  setDraft,
  validationErrors,
  setValidationError,
}: ValidatedSectionProps) {
  return (
    <SettingsSectionCard eyebrow="File caps" note="MEMORY.md projection ceilings">
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-file-max-lines`}
        label="Max lines"
        description="Soft cap for the MEMORY.md projection"
        error={validationErrors.fileMaxLines ?? undefined}
        control={
          <SettingsNumberInput
            min={0}
            className="w-24"
            data-testid={`${TEST_PREFIX}-file-max-lines-input`}
            value={draft.file.max_lines}
            onValidityChange={setValidationError("fileMaxLines")}
            onValueChange={value =>
              setDraft({
                ...draft,
                file: { ...draft.file, max_lines: value },
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-file-max-bytes`}
        label="Max bytes"
        description="Hard byte budget for the MEMORY.md projection"
        error={validationErrors.fileMaxBytes ?? undefined}
        control={
          <SettingsNumberInput
            min={0}
            className="w-32"
            data-testid={`${TEST_PREFIX}-file-max-bytes-input`}
            value={draft.file.max_bytes}
            onValidityChange={setValidationError("fileMaxBytes")}
            onValueChange={value =>
              setDraft({
                ...draft,
                file: { ...draft.file, max_bytes: value },
              })
            }
          />
        }
      />
    </SettingsSectionCard>
  );
}

function WorkspaceIdentitySection({ draft, setDraft }: DraftSectionProps) {
  return (
    <SettingsSectionCard eyebrow="Workspace identity" note=".agh/workspace.toml lifecycle">
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-workspace-toml-path`}
        label="Workspace toml path"
        description="Informational; validation locks this to <workspace>/.agh/workspace.toml"
        control={
          <Input
            readOnly
            className="w-full font-mono"
            data-testid={`${TEST_PREFIX}-workspace-toml-path-input`}
            value={draft.workspace.toml_path}
          />
        }
      />
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-workspace-auto-create`}
        label="Auto-create workspace.toml"
        description="Create <workspace>/.agh/workspace.toml on first touch"
        control={
          <Switch
            data-testid={`${TEST_PREFIX}-workspace-auto-create-switch`}
            checked={draft.workspace.auto_create}
            onCheckedChange={checked =>
              setDraft({
                ...draft,
                workspace: { ...draft.workspace, auto_create: checked },
              })
            }
          />
        }
      />
    </SettingsSectionCard>
  );
}
