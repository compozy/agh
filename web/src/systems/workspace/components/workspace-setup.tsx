import { FolderPlus, Home, Sparkles } from "lucide-react";

import {
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  Eyebrow,
  Field,
  FieldDescription,
  FieldError,
  FieldLabel,
  Input,
  Pill,
  Spinner,
} from "@agh/ui";

import { OptionCard } from "./option-card";
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
  const size = variant === "onboarding" ? "comfortable" : "compact";

  const globalCard = (
    <OptionCard size={size} data-testid="workspace-setup-global-card">
      <OptionCard.Header
        eyebrow="Global"
        right={<Pill tone="success">{WORKSPACE_SETUP_COPY.global.badge}</Pill>}
      />
      <OptionCard.Body>
        <OptionCard.Icon tone="neutral">
          <Home className="size-4" />
        </OptionCard.Icon>
        <OptionCard.Content>
          <OptionCard.Title>{WORKSPACE_SETUP_COPY.global.title}</OptionCard.Title>
          <OptionCard.Description>{WORKSPACE_SETUP_COPY.global.description}</OptionCard.Description>
          {globalMeta ? (
            <OptionCard.Meta data-testid="workspace-global-meta">{globalMeta}</OptionCard.Meta>
          ) : null}
        </OptionCard.Content>
      </OptionCard.Body>
      <OptionCard.Action>
        <Button
          className="w-full justify-between text-accent-ink"
          disabled={isGlobalDisabled}
          onClick={setup.handleUseGlobalWorkspace}
          data-testid="workspace-use-global"
        >
          <span>{WORKSPACE_SETUP_COPY.global.action}</span>
          {isSubmittingGlobal ? <Spinner /> : <Sparkles />}
        </Button>
      </OptionCard.Action>
    </OptionCard>
  );

  const manualCard = (
    <OptionCard size={size} data-testid="workspace-setup-manual-card">
      <OptionCard.Header eyebrow="Path" right={<Pill>{WORKSPACE_SETUP_COPY.manual.badge}</Pill>} />
      <OptionCard.Body>
        <OptionCard.Icon tone="neutral">
          <FolderPlus className="size-4" />
        </OptionCard.Icon>
        <OptionCard.Content>
          <OptionCard.Title>{WORKSPACE_SETUP_COPY.manual.title}</OptionCard.Title>
          <OptionCard.Description>{WORKSPACE_SETUP_COPY.manual.description}</OptionCard.Description>
        </OptionCard.Content>
      </OptionCard.Body>
      <OptionCard.Action>
        <form className="flex flex-col gap-3" onSubmit={setup.handleManualSubmit}>
          <Field data-invalid={manualInvalid || undefined}>
            <FieldLabel htmlFor="workspace-manual-path" className="sr-only">
              {WORKSPACE_SETUP_COPY.manual.inputLabel}
            </FieldLabel>
            <Input
              id="workspace-manual-path"
              aria-label={WORKSPACE_SETUP_COPY.manual.inputLabel}
              aria-invalid={manualInvalid || undefined}
              className="border-line bg-canvas-soft"
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
            className="w-full justify-between text-accent-ink"
            disabled={setup.submissionMode !== null}
            type="submit"
            data-testid="workspace-register-manual"
          >
            <span>{WORKSPACE_SETUP_COPY.manual.action}</span>
            {isSubmittingManual ? <Spinner /> : <FolderPlus />}
          </Button>
        </form>
      </OptionCard.Action>
    </OptionCard>
  );

  if (variant === "dialog") {
    return (
      <div className="flex flex-col gap-4 p-5" data-testid="workspace-setup-dialog-body">
        {globalCard}
        <div className="flex items-center gap-3 px-1">
          <div className="h-px flex-1 bg-line" />
          <Eyebrow className="text-muted">{WORKSPACE_SETUP_COPY.manual.dividerLabel}</Eyebrow>
          <div className="h-px flex-1 bg-line" />
        </div>
        {manualCard}
      </div>
    );
  }

  return (
    <div
      className="flex w-full flex-col gap-4 lg:max-w-96 lg:justify-self-end"
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
        unframed
        className="max-w-xl border border-line bg-canvas sm:max-w-xl"
        showCloseButton
        data-testid="workspace-setup-dialog"
      >
        <DialogHeader variant="ruled">
          <DialogTitle className="text-item-title font-medium text-fg">
            {WORKSPACE_SETUP_COPY.dialog.title}
          </DialogTitle>
          <DialogDescription className="text-small-body leading-6 text-muted">
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
      className="flex min-h-0 flex-1 items-start justify-center overflow-y-auto bg-background p-6 lg:items-center lg:py-10"
      data-testid="workspace-onboarding"
    >
      <div className="w-full max-w-5xl rounded-xl border border-line bg-canvas p-6 sm:p-8 lg:p-10">
        <div className="grid gap-8 lg:grid-cols-[minmax(0,1.15fr)_minmax(22rem,24rem)] lg:gap-8 xl:gap-10">
          <div className="flex flex-col justify-between gap-6">
            <div className="space-y-4">
              <span
                aria-hidden="true"
                data-testid="workspace-onboarding-hero-icon"
                className="inline-flex size-14 items-center justify-center rounded-icon-well bg-surface-glaze text-accent"
              >
                <Sparkles className="size-6" />
              </span>
              <Eyebrow className="text-muted">{copy.eyebrow}</Eyebrow>
              <div className="space-y-3">
                <h1
                  data-testid="workspace-onboarding-hero-title"
                  className="max-w-xl text-detail-h1 font-medium tracking-detail-h1 text-fg"
                >
                  {copy.title}
                </h1>
                <p className="max-w-xl text-item-title leading-7 text-muted">{copy.description}</p>
              </div>
            </div>

            <div className="rounded-2xl border border-line bg-canvas-soft p-4">
              <Eyebrow className="text-muted">{copy.noteLabel}</Eyebrow>
              <p className="mt-2 text-sm leading-6 text-muted">{copy.noteBody}</p>
            </div>
          </div>

          <WorkspaceSetupContent variant="onboarding" onWorkspaceResolved={onWorkspaceResolved} />
        </div>
      </div>
    </div>
  );
}

export { WorkspaceOnboarding, WorkspaceSetupDialog };
