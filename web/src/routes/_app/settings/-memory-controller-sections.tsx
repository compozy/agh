import { SettingsFieldRow, SettingsNumberInput } from "@/systems/settings";
import { Input, Section, Switch } from "@agh/ui";
import { type ValidatedSectionProps, TEST_PREFIX } from "./-memory-settings-types";

export function ControllerSection({
  draft,
  setDraft,
  validationErrors,
  setValidationError,
}: ValidatedSectionProps) {
  const allowOrigins = draft.controller.policy.allow_origins.join(", ");
  return (
    <Section
      divided
      label="Write controller"
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
              setDraft(prev => {
                const current = prev ?? draft;
                return {
                  ...current,
                  controller: { ...current.controller, mode: event.target.value },
                };
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
              setDraft(prev => {
                const current = prev ?? draft;
                return {
                  ...current,
                  controller: { ...current.controller, max_latency: event.target.value },
                };
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
              setDraft(prev => {
                const current = prev ?? draft;
                return {
                  ...current,
                  controller: {
                    ...current.controller,
                    default_op_on_fail: event.target.value,
                  },
                };
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
              setDraft(prev => {
                const current = prev ?? draft;
                return {
                  ...current,
                  controller: {
                    ...current.controller,
                    policy: { ...current.controller.policy, max_content_chars: value },
                  },
                };
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
              setDraft(prev => {
                const current = prev ?? draft;
                return {
                  ...current,
                  controller: {
                    ...current.controller,
                    policy: { ...current.controller.policy, max_writes_per_min: value },
                  },
                };
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
    </Section>
  );
}

export function ControllerLLMSection({
  draft,
  setDraft,
  validationErrors,
  setValidationError,
}: ValidatedSectionProps) {
  const llmDisabled = !draft.controller.llm.enabled;
  return (
    <Section divided label="Controller LLM tiebreaker" note="entity-slot ambiguity escalations">
      <SettingsFieldRow
        data-testid={`${TEST_PREFIX}-controller-llm-enabled`}
        label="LLM tiebreaker"
        description="Allow the controller to escalate ambiguous slot matches to the configured LLM"
        control={
          <Switch
            data-testid={`${TEST_PREFIX}-controller-llm-enabled-switch`}
            checked={draft.controller.llm.enabled}
            onCheckedChange={checked =>
              setDraft(prev => {
                const current = prev ?? draft;
                return {
                  ...current,
                  controller: {
                    ...current.controller,
                    llm: { ...current.controller.llm, enabled: checked },
                  },
                };
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
              setDraft(prev => {
                const current = prev ?? draft;
                return {
                  ...current,
                  controller: {
                    ...current.controller,
                    llm: { ...current.controller.llm, model: event.target.value },
                  },
                };
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
              setDraft(prev => {
                const current = prev ?? draft;
                return {
                  ...current,
                  controller: {
                    ...current.controller,
                    llm: { ...current.controller.llm, top_k: value },
                  },
                };
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
              setDraft(prev => {
                const current = prev ?? draft;
                return {
                  ...current,
                  controller: {
                    ...current.controller,
                    llm: { ...current.controller.llm, max_tokens_out: value },
                  },
                };
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
              setDraft(prev => {
                const current = prev ?? draft;
                return {
                  ...current,
                  controller: {
                    ...current.controller,
                    llm: { ...current.controller.llm, timeout: event.target.value },
                  },
                };
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
              setDraft(prev => {
                const current = prev ?? draft;
                return {
                  ...current,
                  controller: {
                    ...current.controller,
                    llm: { ...current.controller.llm, prompt_version: event.target.value },
                  },
                };
              })
            }
          />
        }
      />
    </Section>
  );
}
