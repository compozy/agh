import {
  ArrowLeft,
  ArrowRight,
  Bot,
  Check,
  ChevronRight,
  NotebookText,
  Plus,
  Settings2,
  ShieldCheck,
  X,
} from "lucide-react";
import { useId, useState, type KeyboardEvent } from "react";

import {
  Button,
  Dialog,
  DialogContent,
  DialogTitle,
  Eyebrow,
  Field,
  FieldDescription,
  FieldError,
  FieldLabel,
  FormSection,
  Input,
  InputGroup,
  InputGroupAddon,
  InputGroupButton,
  InputGroupInput,
  Pill,
  PillGroup,
  RadioCard,
  Spinner,
  Textarea,
  type PillGroupItem,
} from "@agh/ui";

import {
  AGENT_CREATE_PERMISSION_OPTIONS,
  appendAgentCreateTokens,
  removeAgentCreateToken,
  updateAgentCreateScope,
  type AgentCreateDialogDraft,
  type AgentCreatePermissionChoice,
  type AgentCreateProviderOption,
  type AgentCreateScope,
  type AgentCreateStep,
} from "../lib/agent-create-draft";
import { AgentModelCommandSelect } from "./agent-model-command-select";
import { AgentProviderCommandSelect } from "./agent-provider-command-select";
import { useAgentCreateDialogViewState } from "../hooks/use-agent-create-dialog-view-state";

const PERMISSION_DESCRIPTIONS: Record<AgentCreatePermissionChoice, string> = {
  "": "Use the runtime's default approval mode.",
  "deny-all": "Ask before every tool call.",
  "approve-reads": "Auto-approve read-only tools; ask for the rest.",
  "approve-all": "Auto-approve every allowed tool call.",
};

interface AgentCreateDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  draft: AgentCreateDialogDraft;
  onDraftChange: (draft: AgentCreateDialogDraft) => void;
  onSubmit: () => void;
  providerOptions: AgentCreateProviderOption[];
  providersLoading: boolean;
  providersError: string | null;
  modelOptions: string[];
  modelCatalogLoading: boolean;
  modelCatalogError: string | null;
  submitError: string | null;
  isSubmitting: boolean;
  hasActiveWorkspace: boolean;
  workspaceName: string | null;
  initialStep?: AgentCreateStep;
}

interface WizardStepDescriptor {
  id: AgentCreateStep;
  label: string;
  testId: string;
}

const WIZARD_STEPS: readonly WizardStepDescriptor[] = [
  { id: "basics", label: "Basics", testId: "agent-create-step-basics" },
  { id: "runtime", label: "Runtime", testId: "agent-create-step-runtime" },
  { id: "instructions", label: "Instructions", testId: "agent-create-step-instructions" },
  { id: "access", label: "Access", testId: "agent-create-step-access" },
] as const;

function AgentCreateDialog({
  open,
  onOpenChange,
  draft,
  onDraftChange,
  onSubmit,
  providerOptions,
  providersLoading,
  providersError,
  modelOptions,
  modelCatalogLoading,
  modelCatalogError,
  submitError,
  isSubmitting,
  hasActiveWorkspace,
  workspaceName,
  initialStep = "basics",
}: AgentCreateDialogProps) {
  const titleId = useId();
  const {
    activeProvider,
    canAdvance,
    currentIndex,
    handleOpenChange,
    nextStep,
    previousStep,
    setStep,
    step,
    validation,
    visibleErrors,
  } = useAgentCreateDialogViewState({
    draft,
    hasActiveWorkspace,
    initialStep,
    onOpenChange,
    open,
    providerOptions,
    providersError,
    providersLoading,
  });

  return (
    <Dialog onOpenChange={handleOpenChange} open={open}>
      <DialogContent
        aria-labelledby={titleId}
        className="w-(--width-modal-lg) max-w-[calc(100vw-2rem)] sm:max-w-(--width-modal-lg) grid-rows-[auto_auto_1fr_auto] max-h-[min(var(--height-modal-tall),calc(100vh-2rem))]"
        data-testid="agent-create-dialog"
        showCloseButton={!isSubmitting}
        unframed
      >
        <header
          className="flex items-center justify-between gap-3 border-b border-line px-5 py-3.5"
          data-slot="agent-create-head"
        >
          <DialogTitle
            id={titleId}
            className="text-modal-title font-medium tracking-modal-title text-fg-strong"
          >
            Create agent
          </DialogTitle>
          {activeProvider ? (
            <span
              className="min-w-0 truncate font-mono text-form-label text-muted"
              data-testid="agent-create-active-provider"
            >
              {providerDisplayName(activeProvider)}
            </span>
          ) : null}
        </header>

        <nav
          aria-label="Agent create steps"
          className="flex items-center gap-2 overflow-x-auto border-b border-line bg-canvas-tint px-5 py-2.5 text-eyebrow"
          data-testid="agent-create-stepper"
        >
          {WIZARD_STEPS.map((item, index) => {
            const status = stepStatus(index, currentIndex);
            return (
              <span
                key={item.id}
                className="flex items-center gap-2"
                data-status={status}
                data-testid={item.testId}
              >
                <span
                  aria-hidden="true"
                  className={stepBadgeClassName(status)}
                  data-slot="agent-create-step-badge"
                >
                  {status === "complete" ? (
                    <Check width={11} height={11} strokeWidth={2} />
                  ) : (
                    <span className="font-mono text-eyebrow">{index + 1}</span>
                  )}
                </span>
                <Eyebrow className={status === "current" ? "text-fg-strong" : "text-muted"}>
                  {item.label}
                </Eyebrow>
                {index < WIZARD_STEPS.length - 1 ? (
                  <ChevronRight
                    aria-hidden="true"
                    className="ml-1 text-faint"
                    width={12}
                    height={12}
                    strokeWidth={1.75}
                  />
                ) : null}
              </span>
            );
          })}
        </nav>

        <div
          className="flex min-h-0 flex-col gap-4 overflow-y-auto p-5"
          data-testid="agent-create-body"
        >
          {step === "basics" ? (
            <BasicsStep
              draft={draft}
              errors={visibleErrors}
              hasActiveWorkspace={hasActiveWorkspace}
              onDraftChange={onDraftChange}
              workspaceName={workspaceName}
            />
          ) : null}
          {step === "runtime" ? (
            <RuntimeStep
              draft={draft}
              errors={visibleErrors}
              modelCatalogError={modelCatalogError}
              modelCatalogLoading={modelCatalogLoading}
              modelOptions={modelOptions}
              onDraftChange={onDraftChange}
              providerOptions={providerOptions}
              providersLoading={providersLoading}
            />
          ) : null}
          {step === "instructions" ? (
            <InstructionsStep draft={draft} errors={visibleErrors} onDraftChange={onDraftChange} />
          ) : null}
          {step === "access" ? (
            <AccessStep draft={draft} errors={visibleErrors} onDraftChange={onDraftChange} />
          ) : null}
        </div>

        <footer
          className="flex flex-wrap items-center gap-3 border-t border-line bg-canvas-soft px-5 py-3.5"
          data-slot="agent-create-footer"
        >
          <span
            className="font-mono text-form-label text-muted"
            data-testid="agent-create-progress"
          >
            Step {currentIndex + 1} of {WIZARD_STEPS.length}
          </span>
          {submitError ? (
            <FieldError className="min-w-0 flex-1" data-testid="agent-create-submit-error">
              {submitError}
            </FieldError>
          ) : null}
          <div className="ml-auto flex flex-wrap items-center gap-2">
            <Button
              data-testid="agent-create-cancel"
              disabled={isSubmitting}
              onClick={() => handleOpenChange(false)}
              size="sm"
              type="button"
              variant="outline"
            >
              Cancel
            </Button>
            {previousStep ? (
              <Button
                data-testid="agent-create-back"
                disabled={isSubmitting}
                onClick={() => setStep(previousStep)}
                size="sm"
                type="button"
                variant="outline"
              >
                <ArrowLeft className="size-3" />
                Back
              </Button>
            ) : null}
            {nextStep ? (
              <Button
                data-testid="agent-create-next"
                disabled={!canAdvance || isSubmitting}
                onClick={() => setStep(nextStep)}
                size="sm"
                type="button"
              >
                Continue
                <ArrowRight className="size-3" />
              </Button>
            ) : (
              <Button
                data-testid="submit-agent-create"
                disabled={!validation.canSubmit || isSubmitting}
                onClick={onSubmit}
                size="sm"
                type="button"
              >
                {isSubmitting ? (
                  <>
                    <Spinner className="size-3" />
                    Creating...
                  </>
                ) : (
                  "Create agent"
                )}
              </Button>
            )}
          </div>
        </footer>
      </DialogContent>
    </Dialog>
  );
}

function BasicsStep({
  draft,
  errors,
  hasActiveWorkspace,
  onDraftChange,
  workspaceName,
}: {
  draft: AgentCreateDialogDraft;
  errors: Record<string, string | undefined>;
  hasActiveWorkspace: boolean;
  onDraftChange: (draft: AgentCreateDialogDraft) => void;
  workspaceName: string | null;
}) {
  return (
    <FormSection
      data-testid="agent-create-basics"
      icon={Bot}
      size="compact"
      title="Basics"
      description="Name the definition and choose where AGH writes its AGENT.md."
    >
      <Field data-invalid={Boolean(errors.scope)}>
        <FieldLabel id="agent-create-scope-label">Scope</FieldLabel>
        <FieldDescription>
          Workspace scope writes to the active workspace. Global scope writes to AGH home.
        </FieldDescription>
        <PillGroup
          aria-labelledby="agent-create-scope-label"
          data-testid="agent-create-scope"
          items={
            [
              {
                value: "workspace",
                label: workspaceName ? "Workspace · " + workspaceName : "Workspace",
                disabled: !hasActiveWorkspace,
                testId: "agent-create-scope-workspace",
              },
              { value: "global", label: "Global", testId: "agent-create-scope-global" },
            ] satisfies PillGroupItem<AgentCreateScope>[]
          }
          onChange={next => onDraftChange(updateAgentCreateScope(draft, next))}
          value={draft.scope}
        />
        <FieldError data-testid="agent-create-scope-error">{errors.scope}</FieldError>
      </Field>

      <Field data-invalid={Boolean(errors.name)}>
        <FieldLabel htmlFor="agent-create-name">Name</FieldLabel>
        <FieldDescription>
          Use the canonical agent id that will become the folder name.
        </FieldDescription>
        <Input
          aria-invalid={Boolean(errors.name)}
          autoFocus
          data-testid="agent-create-name"
          id="agent-create-name"
          onChange={event => onDraftChange({ ...draft, name: event.target.value })}
          placeholder="release-captain"
          value={draft.name}
        />
        <FieldError data-testid="agent-create-name-error">{errors.name}</FieldError>
      </Field>

      <Field data-invalid={Boolean(errors.categoryPath)}>
        <FieldLabel htmlFor="agent-create-category-path">Category path</FieldLabel>
        <FieldDescription>Optional slash-separated sidebar grouping.</FieldDescription>
        <Input
          aria-invalid={Boolean(errors.categoryPath)}
          data-testid="agent-create-category-path"
          id="agent-create-category-path"
          onChange={event => onDraftChange({ ...draft, categoryPath: event.target.value })}
          placeholder="Engineering/Release"
          value={draft.categoryPath}
        />
        <FieldError data-testid="agent-create-category-path-error">
          {errors.categoryPath}
        </FieldError>
      </Field>
    </FormSection>
  );
}

function RuntimeStep({
  draft,
  errors,
  modelCatalogError,
  modelCatalogLoading,
  modelOptions,
  onDraftChange,
  providerOptions,
  providersLoading,
}: {
  draft: AgentCreateDialogDraft;
  errors: Record<string, string | undefined>;
  modelCatalogError: string | null;
  modelCatalogLoading: boolean;
  modelOptions: string[];
  onDraftChange: (draft: AgentCreateDialogDraft) => void;
  providerOptions: AgentCreateProviderOption[];
  providersLoading: boolean;
}) {
  const providerSelected = draft.provider.trim().length > 0;
  return (
    <FormSection
      data-testid="agent-create-runtime"
      icon={Settings2}
      size="compact"
      title="Runtime"
      description="Choose the provider and optional runtime overrides for new sessions."
    >
      <Field data-invalid={Boolean(errors.provider)}>
        <FieldLabel id="agent-create-provider-label">Provider</FieldLabel>
        <FieldDescription>Provider options come from the selected scope.</FieldDescription>
        <AgentProviderCommandSelect
          options={providerOptions}
          value={draft.provider}
          onChange={provider => onDraftChange({ ...draft, provider, model: "" })}
          disabled={providersLoading || providerOptions.length === 0}
          placeholder={providersLoading ? "Loading providers..." : "Select a provider"}
          triggerId="agent-create-provider"
          triggerTestId="agent-create-provider"
        />
        <FieldError data-testid="agent-create-provider-error">{errors.provider}</FieldError>
      </Field>

      <Field>
        <FieldLabel id="agent-create-model-label">Model</FieldLabel>
        <FieldDescription>
          Pick a catalog model when available, or type a custom model id.
        </FieldDescription>
        <AgentModelCommandSelect
          options={modelOptions}
          value={draft.model}
          onChange={model => onDraftChange({ ...draft, model })}
          disabled={!providerSelected}
          loading={modelCatalogLoading}
          triggerId="agent-create-model"
          triggerTestId="agent-create-model"
        />
        {modelCatalogError ? (
          <p className="text-small-body text-warning" data-testid="agent-create-model-error">
            {modelCatalogError}
          </p>
        ) : null}
      </Field>

      <Field>
        <FieldLabel htmlFor="agent-create-command">Command</FieldLabel>
        <FieldDescription>Optional provider command override for this agent.</FieldDescription>
        <Input
          data-testid="agent-create-command"
          id="agent-create-command"
          onChange={event => onDraftChange({ ...draft, command: event.target.value })}
          placeholder="codex --model gpt-5.4"
          value={draft.command}
        />
      </Field>
    </FormSection>
  );
}

function InstructionsStep({
  draft,
  errors,
  onDraftChange,
}: {
  draft: AgentCreateDialogDraft;
  errors: Record<string, string | undefined>;
  onDraftChange: (draft: AgentCreateDialogDraft) => void;
}) {
  return (
    <FormSection
      data-testid="agent-create-instructions"
      icon={NotebookText}
      size="compact"
      title="Instructions"
      description="Write the system prompt that defines this agent's role and behavior."
    >
      <Field data-invalid={Boolean(errors.prompt)}>
        <FieldLabel htmlFor="agent-create-prompt">Prompt</FieldLabel>
        <Textarea
          aria-invalid={Boolean(errors.prompt)}
          autoFocus
          className="min-h-52"
          data-testid="agent-create-prompt"
          id="agent-create-prompt"
          onChange={event => onDraftChange({ ...draft, prompt: event.target.value })}
          placeholder="You are responsible for release readiness..."
          value={draft.prompt}
        />
        <FieldError data-testid="agent-create-prompt-error">{errors.prompt}</FieldError>
      </Field>
    </FormSection>
  );
}

function AccessStep({
  draft,
  errors,
  onDraftChange,
}: {
  draft: AgentCreateDialogDraft;
  errors: Record<string, string | undefined>;
  onDraftChange: (draft: AgentCreateDialogDraft) => void;
}) {
  return (
    <FormSection
      data-testid="agent-create-access"
      icon={ShieldCheck}
      size="compact"
      title="Access"
      description="Constrain the tools and skills available to sessions started from this agent."
    >
      <Field>
        <FieldLabel id="agent-create-permissions-label">Permissions</FieldLabel>
        <FieldDescription>Optional default approval posture for this agent.</FieldDescription>
        <div
          aria-labelledby="agent-create-permissions-label"
          className="grid gap-2 sm:grid-cols-2"
          data-testid="agent-create-permissions"
          role="radiogroup"
        >
          {AGENT_CREATE_PERMISSION_OPTIONS.map(option => (
            <RadioCard
              key={option.value || "inherit"}
              data-testid={"agent-create-permissions-" + (option.value || "inherit")}
              description={PERMISSION_DESCRIPTIONS[option.value]}
              onSelect={() => onDraftChange({ ...draft, permissions: option.value })}
              selected={draft.permissions === option.value}
              title={option.label}
            />
          ))}
        </div>
      </Field>

      <div className="grid gap-3.5 md:grid-cols-2">
        <TokenListField
          description="Canonical tool IDs or namespace wildcards."
          error={errors.tools}
          label="Tools"
          onChange={tools => onDraftChange({ ...draft, tools })}
          placeholder="agh__skill_view, mcp__github__*"
          testId="agent-create-tools"
          values={draft.tools}
        />
        <TokenListField
          description="Tool groups enabled for the agent."
          error={errors.toolsets}
          label="Toolsets"
          onChange={toolsets => onDraftChange({ ...draft, toolsets })}
          placeholder="agh__catalog"
          testId="agent-create-toolsets"
          values={draft.toolsets}
        />
        <TokenListField
          description="Canonical tools to deny after allow rules."
          error={errors.denyTools}
          label="Denied tools"
          onChange={denyTools => onDraftChange({ ...draft, denyTools })}
          placeholder="agh__task_*"
          testId="agent-create-deny-tools"
          values={draft.denyTools}
        />
        <TokenListField
          description="Skill names disabled only for this agent."
          label="Disabled skills"
          onChange={disabledSkills => onDraftChange({ ...draft, disabledSkills })}
          placeholder="code-review, release-notes"
          testId="agent-create-disabled-skills"
          values={draft.disabledSkills}
        />
      </div>
    </FormSection>
  );
}

function TokenListField({
  description,
  error,
  label,
  onChange,
  placeholder,
  testId,
  values,
}: {
  description: string;
  error?: string;
  label: string;
  onChange: (values: string[]) => void;
  placeholder: string;
  testId: string;
  values: string[];
}) {
  const inputId = useId();
  const [inputValue, setInputValue] = useState("");

  const commit = () => {
    if (inputValue.trim().length === 0) return;
    onChange(appendAgentCreateTokens(values, inputValue));
    setInputValue("");
  };

  const handleKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === "Enter" || event.key === ",") {
      event.preventDefault();
      commit();
    }
  };

  return (
    <Field data-invalid={Boolean(error)}>
      <FieldLabel htmlFor={inputId}>{label}</FieldLabel>
      <FieldDescription>{description}</FieldDescription>
      <InputGroup>
        <InputGroupInput
          aria-invalid={Boolean(error)}
          data-testid={testId + "-input"}
          id={inputId}
          onBlur={commit}
          onChange={event => {
            const next = event.target.value;
            if (/[,\n]/.test(next)) {
              onChange(appendAgentCreateTokens(values, next));
              setInputValue("");
              return;
            }
            setInputValue(next);
          }}
          onKeyDown={handleKeyDown}
          placeholder={placeholder}
          value={inputValue}
        />
        <InputGroupAddon align="inline-end">
          <InputGroupButton
            aria-label={"Add " + label.toLowerCase()}
            data-testid={testId + "-add"}
            disabled={inputValue.trim().length === 0}
            onClick={commit}
            size="icon-xs"
          >
            <Plus aria-hidden="true" className="size-3" />
          </InputGroupButton>
        </InputGroupAddon>
      </InputGroup>
      {values.length > 0 ? (
        <div className="flex flex-wrap gap-1.5" data-testid={testId + "-tokens"}>
          {values.map(value => (
            <Pill key={value} className="gap-1 pr-1" size="sm">
              <span className="max-w-44 truncate">{value}</span>
              <button
                aria-label={"Remove " + value}
                className="inline-flex size-4 items-center justify-center rounded-sm text-subtle transition-colors hover:bg-hover hover:text-fg focus-visible:outline-none focus-visible:shadow-focus-ring"
                onClick={() => onChange(removeAgentCreateToken(values, value))}
                type="button"
              >
                <X aria-hidden="true" className="size-3" />
              </button>
            </Pill>
          ))}
        </div>
      ) : null}
      <FieldError data-testid={testId + "-error"}>{error}</FieldError>
    </Field>
  );
}

function providerDisplayName(provider: AgentCreateProviderOption): string {
  return provider.display_name?.trim() || provider.name;
}

function stepStatus(index: number, currentIndex: number): "complete" | "current" | "pending" {
  if (index < currentIndex) return "complete";
  if (index === currentIndex) return "current";
  return "pending";
}

function stepBadgeClassName(status: "complete" | "current" | "pending"): string {
  const base =
    "inline-flex size-5 shrink-0 items-center justify-center rounded-full text-fg-strong transition-colors duration-base ease-out";
  if (status === "complete") return base + " bg-success-tint text-success";
  if (status === "current") return base + " bg-surface-glaze text-fg-strong";
  return base + " bg-canvas-soft text-subtle";
}

export { AgentCreateDialog };
export type { AgentCreateDialogProps };
