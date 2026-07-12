import { ArrowLeft, ArrowRight, Check, ChevronRight } from "lucide-react";
import { useId, useState } from "react";

import { Button, Dialog, DialogContent, DialogTitle, Eyebrow, Spinner } from "@agh/ui";

import { parseBridgeProviderConfig } from "../lib/bridge-drafts";
import { findBridgeProviderByKey, isBridgeProviderSelectable } from "../lib/bridge-formatters";
import type { BridgeCreateDraft, BridgeProvider } from "../types";
import { DeliveryStep } from "./bridge-delivery-step";
import { ProviderStep } from "./bridge-provider-step";
import { RuntimeMissingProviderState } from "./bridge-runtime-metadata";
import { RuntimeStep } from "./bridge-runtime-step";

type WizardStep = "provider" | "runtime" | "delivery";

const WIZARD_STEPS = [
  { id: "provider", label: "Provider", testId: "bridge-wizard-step-provider" },
  { id: "runtime", label: "Runtime", testId: "bridge-wizard-step-runtime" },
  { id: "delivery", label: "Delivery", testId: "bridge-wizard-step-delivery" },
] as const satisfies ReadonlyArray<{ id: WizardStep; label: string; testId: string }>;

interface BridgeCreateDialogProps {
  activeWorkspaceId?: string | null;
  activeWorkspaceName?: string | null;
  draft: BridgeCreateDraft;
  isPending: boolean;
  onDraftChange: (draft: BridgeCreateDraft) => void;
  onOpenChange: (open: boolean) => void;
  onSubmit: () => void;
  open: boolean;
  providers: BridgeProvider[];
}

export function BridgeCreateDialog(props: BridgeCreateDialogProps) {
  return <BridgeCreateDialogContent {...props} />;
}

function BridgeCreateDialogContent({
  activeWorkspaceId,
  activeWorkspaceName,
  draft,
  isPending,
  onDraftChange,
  onOpenChange,
  onSubmit,
  open,
  providers,
}: BridgeCreateDialogProps) {
  const titleId = useId();
  const [step, setStep] = useState<WizardStep>("provider");

  const selectedProvider = findBridgeProviderByKey(providers, draft.selectedProviderKey);
  const providerConfigError = parseBridgeProviderConfig(draft.providerConfigText).error;
  const stepValidity: Record<WizardStep, boolean> = {
    provider: Boolean(selectedProvider && isBridgeProviderSelectable(selectedProvider)),
    runtime: Boolean(draft.displayName.trim()) && !providerConfigError,
    delivery: true,
  };
  const canSubmit = stepValidity.provider && stepValidity.runtime && stepValidity.delivery;
  const currentIndex = WIZARD_STEPS.findIndex(item => item.id === step);
  const nextStep =
    currentIndex < WIZARD_STEPS.length - 1 ? WIZARD_STEPS[currentIndex + 1] : undefined;
  const previousStep = currentIndex > 0 ? WIZARD_STEPS[currentIndex - 1] : undefined;

  const handleOpenChange = (next: boolean) => {
    if (!next) setStep("provider");
    onOpenChange(next);
  };

  return (
    <Dialog onOpenChange={handleOpenChange} open={open}>
      <DialogContent
        aria-labelledby={titleId}
        className="w-(--width-modal-lg) max-w-[calc(100vw-2rem)] sm:max-w-(--width-modal-lg) grid-rows-[auto_auto_1fr_auto] max-h-[min(var(--height-modal-tall),calc(100vh-2rem))]"
        data-testid="bridge-create-dialog"
        showCloseButton={false}
        unframed
      >
        <header
          data-slot="bridge-wizard-head"
          className="flex items-center justify-between gap-3 border-b border-line px-5 py-3.5"
        >
          <DialogTitle
            id={titleId}
            data-testid="bridge-wizard-title"
            className="text-modal-title font-medium tracking-modal-title text-fg-strong"
          >
            Create bridge
          </DialogTitle>
          {selectedProvider ? (
            <span
              className="font-mono text-form-label text-muted"
              data-testid="bridge-wizard-active-provider"
            >
              {selectedProvider.display_name}
            </span>
          ) : null}
        </header>

        <nav
          aria-label="Bridge create steps"
          className="flex items-center gap-2 border-b border-line bg-canvas-tint px-5 py-2.5 text-eyebrow"
          data-testid="bridge-wizard-stepper"
        >
          {WIZARD_STEPS.map((item, index) => {
            const status = stepStatus(index, currentIndex);
            return (
              <span
                key={item.id}
                className="flex items-center gap-2"
                data-testid={item.testId}
                data-status={status}
              >
                <span
                  aria-hidden="true"
                  className={stepBadgeClass(status)}
                  data-slot="bridge-wizard-step-badge"
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
                    width={12}
                    height={12}
                    strokeWidth={1.75}
                    className="ml-1 text-faint"
                  />
                ) : null}
              </span>
            );
          })}
        </nav>

        <div
          className="flex min-h-0 flex-col gap-4 overflow-y-auto p-5"
          data-testid="bridge-wizard-body"
        >
          {step === "provider" ? (
            <ProviderStep
              onSelect={key => {
                const provider = findBridgeProviderByKey(providers, key);
                onDraftChange({
                  ...draft,
                  displayName:
                    !draft.displayName.trim() ||
                    draft.displayName.trim() === selectedProvider?.display_name
                      ? (provider?.display_name ?? draft.displayName)
                      : draft.displayName,
                  selectedProviderKey: key,
                });
              }}
              providers={providers}
              selectedProviderKey={draft.selectedProviderKey}
            />
          ) : null}
          {step === "runtime" && selectedProvider ? (
            <RuntimeStep
              activeWorkspaceId={activeWorkspaceId}
              activeWorkspaceName={activeWorkspaceName}
              draft={draft}
              onDraftChange={onDraftChange}
              provider={selectedProvider}
              providerConfigError={providerConfigError}
            />
          ) : null}
          {step === "runtime" && !selectedProvider ? <RuntimeMissingProviderState /> : null}
          {step === "delivery" ? (
            <DeliveryStep draft={draft} onDraftChange={onDraftChange} />
          ) : null}
        </div>

        <footer
          className="flex flex-wrap items-center gap-3 border-t border-line bg-canvas-soft px-5 py-3.5"
          data-slot="bridge-wizard-footer"
        >
          <span
            className="font-mono text-form-label text-muted"
            data-testid="bridge-wizard-progress"
          >
            Step {currentIndex + 1} of {WIZARD_STEPS.length}
          </span>
          <div className="ml-auto flex flex-wrap items-center gap-2">
            <Button
              data-testid="bridge-wizard-cancel"
              onClick={() => handleOpenChange(false)}
              size="sm"
              type="button"
              variant="outline"
            >
              Cancel
            </Button>
            {previousStep ? (
              <Button
                data-testid="bridge-wizard-back"
                onClick={() => setStep(previousStep.id)}
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
                data-testid="bridge-wizard-next"
                disabled={!stepValidity[step]}
                onClick={() => setStep(nextStep.id)}
                size="sm"
                type="button"
              >
                Continue
                <ArrowRight className="size-3" />
              </Button>
            ) : (
              <Button
                data-testid="submit-bridge-create"
                disabled={!canSubmit || isPending}
                onClick={onSubmit}
                size="sm"
                type="button"
              >
                {isPending ? (
                  <>
                    <Spinner className="size-3" />
                    Creating…
                  </>
                ) : (
                  "Create Bridge"
                )}
              </Button>
            )}
          </div>
        </footer>
      </DialogContent>
    </Dialog>
  );
}

function stepStatus(index: number, currentIndex: number): "complete" | "current" | "pending" {
  if (index < currentIndex) return "complete";
  if (index === currentIndex) return "current";
  return "pending";
}

function stepBadgeClass(status: "complete" | "current" | "pending"): string {
  const base =
    "inline-flex size-5 shrink-0 items-center justify-center rounded-full text-fg-strong transition-colors duration-base ease-out";
  if (status === "complete") return `${base} bg-success-tint text-success`;
  if (status === "current") return `${base} bg-surface-glaze text-fg-strong`;
  return `${base} bg-canvas-soft text-subtle`;
}
