import { FolderPlus, Home, Loader2, Sparkles } from "lucide-react";
import type { ReactNode } from "react";

import {
  Button,
  cn,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  Field,
  FieldDescription,
  FieldError,
  FieldLabel,
  Input,
  Pill,
  Section,
} from "@agh/ui";

import {
  useWorkspaceSetupContent,
  type WorkspaceSetupVariant,
} from "../hooks/use-workspace-setup-content";
import { WORKSPACE_SETUP_COPY } from "../lib/workspace-setup-copy";

interface WorkspaceSetupSharedProps {
  onWorkspaceResolved: (workspaceId: string) => void;
}

interface WorkspaceSetupDialogProps extends WorkspaceSetupSharedProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

type WorkspaceOnboardingProps = WorkspaceSetupSharedProps;

interface SetupOptionCardProps {
  variant: WorkspaceSetupVariant;
  eyebrow: string;
  right: ReactNode;
  icon: ReactNode;
  iconTone: "accent" | "neutral";
  title: ReactNode;
  description: ReactNode;
  meta?: ReactNode;
  testId?: string;
  children: ReactNode;
}

function SetupOptionCard({
  variant,
  eyebrow,
  right,
  icon,
  iconTone,
  title,
  description,
  meta,
  testId,
  children,
}: SetupOptionCardProps) {
  return (
    <Section
      label={eyebrow}
      right={right}
      data-testid={testId}
      className={cn(
        "rounded-2xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface)]",
        variant === "onboarding" ? "p-5" : "p-4"
      )}
    >
      <div className="flex items-start gap-3">
        <span
          aria-hidden="true"
          className={cn(
            "inline-flex size-10 shrink-0 items-center justify-center rounded-2xl border border-[color:var(--color-divider)]",
            iconTone === "accent"
              ? "bg-[color:var(--color-surface-panel)] text-[color:var(--color-accent)]"
              : "bg-[color:var(--color-surface-panel)] text-[color:var(--color-text-primary)]"
          )}
        >
          {icon}
        </span>
        <div className="min-w-0 flex-1">
          <p className="text-sm font-semibold text-[color:var(--color-text-primary)]">{title}</p>
          <p className="mt-1 text-sm leading-6 text-[color:var(--color-text-secondary)]">
            {description}
          </p>
          {meta ? (
            <p
              className="mt-3 truncate font-mono text-[0.68rem] text-[color:var(--color-text-tertiary)]"
              data-testid="workspace-global-meta"
            >
              {meta}
            </p>
          ) : null}
        </div>
      </div>
      {children}
    </Section>
  );
}

function WorkspaceSetupContent({
  variant,
  onWorkspaceResolved,
  onSuccessClose,
}: WorkspaceSetupSharedProps & {
  variant: WorkspaceSetupVariant;
  onSuccessClose?: () => void;
}) {
  const setup = useWorkspaceSetupContent({
    onSuccessClose,
    onWorkspaceResolved,
  });

  const isSubmittingGlobal = setup.submissionMode === "global";
  const isSubmittingManual = setup.submissionMode === "manual";
  const isGlobalDisabled = setup.submissionMode !== null || setup.globalUnavailableReason !== null;
  const globalMeta = setup.userHomeDir || setup.globalUnavailableReason || "";
  const manualInvalid = Boolean(setup.manualError);

  const globalCard = (
    <SetupOptionCard
      variant={variant}
      eyebrow="Global"
      right={<Pill tone="accent">{WORKSPACE_SETUP_COPY.global.badge}</Pill>}
      icon={<Home className="size-4" />}
      iconTone="accent"
      title={WORKSPACE_SETUP_COPY.global.title}
      description={WORKSPACE_SETUP_COPY.global.description}
      meta={globalMeta}
      testId="workspace-setup-global-card"
    >
      <Button
        className="mt-4 w-full justify-between text-[color:var(--color-accent-ink)]"
        disabled={isGlobalDisabled}
        onClick={setup.handleUseGlobalWorkspace}
        data-testid="workspace-use-global"
      >
        <span>{WORKSPACE_SETUP_COPY.global.action}</span>
        {isSubmittingGlobal ? <Loader2 className="animate-spin" /> : <Sparkles />}
      </Button>
    </SetupOptionCard>
  );

  const manualCard = (
    <SetupOptionCard
      variant={variant}
      eyebrow="Path"
      right={<Pill>{WORKSPACE_SETUP_COPY.manual.badge}</Pill>}
      icon={<FolderPlus className="size-4" />}
      iconTone="neutral"
      title={WORKSPACE_SETUP_COPY.manual.title}
      description={WORKSPACE_SETUP_COPY.manual.description}
      testId="workspace-setup-manual-card"
    >
      <form className="mt-4 flex flex-col gap-3" onSubmit={setup.handleManualSubmit}>
        <Field data-invalid={manualInvalid || undefined}>
          <FieldLabel htmlFor="workspace-manual-path" className="sr-only">
            {WORKSPACE_SETUP_COPY.manual.inputLabel}
          </FieldLabel>
          <Input
            id="workspace-manual-path"
            aria-label={WORKSPACE_SETUP_COPY.manual.inputLabel}
            aria-invalid={manualInvalid || undefined}
            className="border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)]"
            disabled={setup.submissionMode !== null}
            onChange={event => setup.setManualPath(event.currentTarget.value)}
            placeholder={WORKSPACE_SETUP_COPY.manual.inputPlaceholder}
            value={setup.manualPath}
            data-testid="workspace-manual-path-input"
          />
          {setup.manualError ? (
            <FieldError data-testid="workspace-path-error">{setup.manualError}</FieldError>
          ) : (
            <FieldDescription className="sr-only">Absolute path required.</FieldDescription>
          )}
        </Field>
        <Button
          className="w-full justify-between text-[color:var(--color-accent-ink)]"
          disabled={setup.submissionMode !== null}
          type="submit"
          data-testid="workspace-register-manual"
        >
          <span>{WORKSPACE_SETUP_COPY.manual.action}</span>
          {isSubmittingManual ? <Loader2 className="animate-spin" /> : <FolderPlus />}
        </Button>
      </form>
    </SetupOptionCard>
  );

  if (variant === "dialog") {
    return (
      <div className="flex flex-col gap-4 p-5" data-testid="workspace-setup-dialog-body">
        {globalCard}
        <div className="flex items-center gap-3 px-1">
          <div className="h-px flex-1 bg-[color:var(--color-divider)]" />
          <span className="font-mono text-[0.62rem] uppercase tracking-[0.14em] text-[color:var(--color-text-label)]">
            {WORKSPACE_SETUP_COPY.manual.dividerLabel}
          </span>
          <div className="h-px flex-1 bg-[color:var(--color-divider)]" />
        </div>
        {manualCard}
      </div>
    );
  }

  return (
    <div
      className="flex w-full flex-col gap-4 lg:max-w-[24rem] lg:justify-self-end"
      data-testid="workspace-setup-options"
    >
      {globalCard}
      {manualCard}
    </div>
  );
}

function WorkspaceSetupDialog({
  open,
  onOpenChange,
  onWorkspaceResolved,
}: WorkspaceSetupDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="max-w-xl border border-[color:var(--color-divider)] bg-[color:var(--color-canvas)] p-0 sm:max-w-xl"
        showCloseButton
        data-testid="workspace-setup-dialog"
      >
        <DialogHeader className="gap-2 border-b border-[color:var(--color-divider)] px-5 py-4">
          <DialogTitle className="text-[15px] font-semibold text-[color:var(--color-text-primary)]">
            {WORKSPACE_SETUP_COPY.dialog.title}
          </DialogTitle>
          <DialogDescription className="text-[13px] leading-6 text-[color:var(--color-text-secondary)]">
            {WORKSPACE_SETUP_COPY.dialog.description}
          </DialogDescription>
        </DialogHeader>
        <WorkspaceSetupContent
          variant="dialog"
          onSuccessClose={() => onOpenChange(false)}
          onWorkspaceResolved={onWorkspaceResolved}
        />
      </DialogContent>
    </Dialog>
  );
}

function WorkspaceOnboarding({ onWorkspaceResolved }: WorkspaceOnboardingProps) {
  const copy = WORKSPACE_SETUP_COPY.onboarding;

  return (
    <div
      className="flex min-h-0 flex-1 items-start justify-center overflow-y-auto bg-background px-6 py-6 lg:items-center lg:py-10"
      data-testid="workspace-onboarding"
    >
      <div className="w-full max-w-5xl rounded-[28px] border border-[color:var(--color-divider)] bg-[color:var(--color-canvas)] p-6 sm:p-8 lg:p-10">
        <div className="grid gap-8 lg:grid-cols-[minmax(0,1.15fr)_minmax(22rem,24rem)] lg:gap-8 xl:gap-10">
          <div className="flex flex-col justify-between gap-6">
            <div className="space-y-4">
              <Pill tone="accent">{copy.eyebrow}</Pill>
              <div className="space-y-3">
                <h1 className="max-w-xl text-3xl font-semibold tracking-[-0.03em] text-[color:var(--color-text-primary)] sm:text-4xl">
                  {copy.title}
                </h1>
                <p className="max-w-xl text-[15px] leading-7 text-[color:var(--color-text-secondary)]">
                  {copy.description}
                </p>
              </div>
            </div>

            <div className="rounded-2xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)] p-4">
              <p className="font-mono text-[0.62rem] uppercase tracking-[0.14em] text-[color:var(--color-text-label)]">
                {copy.noteLabel}
              </p>
              <p className="mt-2 text-sm leading-6 text-[color:var(--color-text-secondary)]">
                {copy.noteBody}
              </p>
            </div>
          </div>

          <WorkspaceSetupContent variant="onboarding" onWorkspaceResolved={onWorkspaceResolved} />
        </div>
      </div>
    </div>
  );
}

export { WorkspaceOnboarding, WorkspaceSetupDialog };
