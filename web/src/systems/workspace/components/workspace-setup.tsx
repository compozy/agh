import { FolderPlus, Home, Loader2, Sparkles } from "lucide-react";

import { PillButton } from "@/components/design-system/pill-button";
import { Button, Input } from "@agh/ui";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { cn } from "@/lib/utils";

import {
  useWorkspaceSetupContent,
  type WorkspaceSetupVariant,
} from "../hooks/use-workspace-setup-content";

interface WorkspaceSetupSharedProps {
  onWorkspaceResolved: (workspaceId: string) => void;
}

interface WorkspaceSetupDialogProps extends WorkspaceSetupSharedProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

interface WorkspaceOnboardingProps extends WorkspaceSetupSharedProps {}

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

  const globalCard = (
    <section
      className={cn(
        "rounded-2xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface)]",
        variant === "onboarding" ? "p-5" : "p-4"
      )}
    >
      <div className="flex items-start gap-3">
        <div className="flex size-10 shrink-0 items-center justify-center rounded-2xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)] text-[color:var(--color-accent)]">
          <Home className="size-4" />
        </div>
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <span className="text-sm font-semibold text-[color:var(--color-text-primary)]">
              Use global workspace
            </span>
            <PillButton active size="dense" className="pointer-events-none">
              HOME
            </PillButton>
          </div>
          <p className="mt-1 text-sm leading-6 text-[color:var(--color-text-secondary)]">
            Register your OS home directory as the default AGH workspace and start with one click.
          </p>
          <p className="mt-3 truncate font-mono text-[0.68rem] text-[color:var(--color-text-tertiary)]">
            {setup.userHomeDir || setup.globalUnavailableReason}
          </p>
        </div>
      </div>
      <Button
        className="mt-4 w-full justify-between text-[color:var(--color-accent-ink)]"
        disabled={setup.submissionMode !== null || setup.globalUnavailableReason !== null}
        onClick={setup.handleUseGlobalWorkspace}
      >
        <span>Use global workspace</span>
        {setup.submissionMode === "global" ? <Loader2 className="animate-spin" /> : <Sparkles />}
      </Button>
    </section>
  );

  const manualCard = (
    <section
      className={cn(
        "rounded-2xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface)]",
        variant === "onboarding" ? "p-5" : "p-4"
      )}
    >
      <div className="flex items-start gap-3">
        <div className="flex size-10 shrink-0 items-center justify-center rounded-2xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)] text-[color:var(--color-text-primary)]">
          <FolderPlus className="size-4" />
        </div>
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <span className="text-sm font-semibold text-[color:var(--color-text-primary)]">
              Register workspace
            </span>
            <PillButton size="dense" className="pointer-events-none">
              PATH
            </PillButton>
          </div>
          <p className="mt-1 text-sm leading-6 text-[color:var(--color-text-secondary)]">
            Add any project root by absolute path. AGH will resolve and register it as a workspace.
          </p>
        </div>
      </div>

      <form className="mt-4 flex flex-col gap-3" onSubmit={setup.handleManualSubmit}>
        <Input
          aria-label="Workspace path"
          className="border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)]"
          disabled={setup.submissionMode !== null}
          onChange={event => setup.setManualPath(event.currentTarget.value)}
          placeholder="/Users/name/project"
          value={setup.manualPath}
        />
        {setup.manualError && (
          <p
            className="text-sm text-[color:var(--color-danger)]"
            data-testid="workspace-path-error"
          >
            {setup.manualError}
          </p>
        )}
        <Button
          className="w-full justify-between text-[color:var(--color-accent-ink)]"
          disabled={setup.submissionMode !== null}
          type="submit"
        >
          <span>Register workspace</span>
          {setup.submissionMode === "manual" ? (
            <Loader2 className="animate-spin" />
          ) : (
            <FolderPlus />
          )}
        </Button>
      </form>
    </section>
  );

  if (variant === "dialog") {
    return (
      <div className="flex flex-col gap-4 p-5">
        {globalCard}
        <div className="flex items-center gap-3 px-1">
          <div className="h-px flex-1 bg-[color:var(--color-divider)]" />
          <span className="font-mono text-[0.62rem] uppercase tracking-[0.14em] text-[color:var(--color-text-label)]">
            or
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
        className="max-w-xl border border-[color:var(--color-divider)] bg-[color:var(--color-canvas)] p-0"
        showCloseButton
      >
        <DialogHeader className="border-b border-[color:var(--color-divider)] px-5 py-4">
          <DialogTitle className="text-[15px] font-semibold text-[color:var(--color-text-primary)]">
            Add workspace
          </DialogTitle>
          <DialogDescription className="text-[13px] leading-6 text-[color:var(--color-text-secondary)]">
            Choose the fastest way to bring a workspace into AGH without leaving the shell.
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
  return (
    <div
      className="flex min-h-screen items-center justify-center bg-background px-6 py-10"
      data-testid="workspace-onboarding"
    >
      <div className="w-full max-w-5xl rounded-[28px] border border-[color:var(--color-divider)] bg-[color:var(--color-canvas)] p-6 sm:p-8 lg:p-10">
        <div className="grid gap-8 lg:grid-cols-[minmax(0,1.15fr)_minmax(22rem,24rem)] lg:gap-8 xl:gap-10">
          <div className="flex flex-col justify-between gap-6">
            <div className="space-y-4">
              <PillButton active size="dense" className="pointer-events-none">
                Workspace setup
              </PillButton>
              <div className="space-y-3">
                <h1 className="max-w-xl text-3xl font-semibold tracking-[-0.03em] text-[color:var(--color-text-primary)] sm:text-4xl">
                  Start AGH with a real workspace, not an empty shell.
                </h1>
                <p className="max-w-xl text-[15px] leading-7 text-[color:var(--color-text-secondary)]">
                  Register your global workspace to anchor AGH immediately, or point it at a
                  specific project root if this machine already has a working directory in play.
                </p>
              </div>
            </div>

            <div className="rounded-2xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)] p-4">
              <p className="font-mono text-[0.62rem] uppercase tracking-[0.14em] text-[color:var(--color-text-label)]">
                First-run note
              </p>
              <p className="mt-2 text-sm leading-6 text-[color:var(--color-text-secondary)]">
                AGH needs at least one registered workspace before sessions, knowledge, and
                workspace-local skills can behave predictably.
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
