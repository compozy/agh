import { RotateCcw, Save, SlidersHorizontal, X } from "lucide-react";

import { Button, Eyebrow, Sheet, SheetContent } from "@agh/ui";

import { useLoopConfigure } from "../../hooks/use-loop-configure";
import type { LoopConfig, LoopDetail, LoopEffectiveConfig } from "../../types";
import { LoopConfigureChecks } from "./loop-configure-checks";
import { LoopConfigureLimits } from "./loop-configure-limits";
import { LoopConfigureStrategy } from "./loop-configure-strategy";
import { LoopConfigureSwitchRow } from "./loop-configure-switch-row";

interface LoopConfigureSheetProps {
  open: boolean;
  workspaceId: string;
  loop: LoopDetail;
  config: LoopConfig | null;
  effectiveConfig: LoopEffectiveConfig;
  onOpenChange: (open: boolean) => void;
  /** Opens the builder for structural edits. */
  onOpenEditor?: () => void;
}

/**
 * The no-fork Configure sheet (design §4.7): a right-side drawer over a dimmed backdrop that
 * writes the per-loop `loop_config` store (verification-check toggles + command overrides, the
 * human-approval-gate flag, the re-attempt strategy, and the 6 clamped limit overrides).
 * Structural fields (node order / kinds / inputs / contract / goal) are non-editable here —
 * they belong in the builder (ADR-009). There is NO cost-cap field; cost is display-only (ADR-017).
 */
export function LoopConfigureSheet({
  open,
  workspaceId,
  loop,
  config,
  effectiveConfig,
  onOpenChange,
  onOpenEditor,
}: LoopConfigureSheetProps) {
  const model = useLoopConfigure({
    workspaceId,
    loop,
    config,
    effectiveConfig,
    onSaved: () => onOpenChange(false),
  });

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent
        side="right"
        showCloseButton={false}
        className="grid w-full grid-rows-[auto_1fr_auto] gap-0 p-0 sm:max-w-[36rem]"
        data-testid="loop-configure-sheet"
      >
        <header className="flex items-start gap-3 border-b border-line-soft px-5 py-4">
          <span className="flex size-9 shrink-0 items-center justify-center rounded-icon-well bg-accent-tint text-accent-strong ring-1 ring-accent-dim ring-inset">
            <SlidersHorizontal aria-hidden="true" className="size-4" />
          </span>
          <div className="min-w-0 flex-1">
            <Eyebrow className="text-accent-strong">Loops · Configure</Eyebrow>
            <h2 className="mt-1 text-item-title font-medium text-fg-strong">
              Configure {loop.name}
            </h2>
            <p className="mt-0.5 flex items-center gap-2 text-[11.5px] text-subtle">
              <span className="truncate font-mono text-[11px] text-muted">{loop.name}</span>
              <span aria-hidden="true" className="size-0.5 rounded-full bg-faint" />
              no-fork tweaks
            </p>
          </div>
          <Button
            type="button"
            variant="ghost"
            size="icon-sm"
            onClick={() => onOpenChange(false)}
            disabled={model.busy}
            aria-label="Close"
            data-testid="loop-configure-close"
          >
            <X aria-hidden="true" className="size-4" />
          </Button>
        </header>

        <div className="min-h-0 overflow-y-auto px-5 py-4">
          <p
            className="mb-5 rounded-md border border-line-soft bg-canvas-tint px-3 py-2.5 text-[11.5px] leading-relaxed text-subtle"
            data-testid="loop-configure-structural-note"
          >
            Adjust how this loop runs without changing its structure. Steps, inputs, node kinds and
            the goal stay fixed here. To change those,{" "}
            <button
              type="button"
              onClick={onOpenEditor}
              disabled={!onOpenEditor || model.busy}
              className="font-medium text-accent-strong underline-offset-2 hover:underline disabled:no-underline disabled:opacity-70"
              data-testid="loop-configure-edit-link"
            >
              {loop.source === "workspace" ? "Edit" : "Fork & edit"}
            </button>{" "}
            the definition in the builder.
          </p>

          <ConfigureGroup label="Review gate" lock="declared in the loop">
            <LoopConfigureChecks
              descriptors={model.descriptors}
              states={model.draft.checks}
              disabled={model.busy}
              onToggle={model.toggleCheck}
              onCommandChange={model.setCheckCommand}
            />
          </ConfigureGroup>

          <ConfigureGroup label="Human approval gate">
            <div className="overflow-hidden rounded-lg border border-line-soft bg-canvas-tint">
              <LoopConfigureSwitchRow
                testId="loop-configure-human-gate"
                typeLabel="human"
                title="Pause for a human decision"
                description="Pauses the run as needs-approval for a human decision before it completes. Off by default."
                checked={model.draft.humanGateEnabled}
                disabled={model.busy}
                onCheckedChange={model.setHumanGate}
              />
            </div>
          </ConfigureGroup>

          <ConfigureGroup label="Re-attempt strategy">
            <LoopConfigureStrategy
              value={model.draft.reattemptStrategy}
              disabled={model.busy}
              onChange={model.setStrategy}
            />
          </ConfigureGroup>

          <ConfigureGroup label="Stop limits" lock="per-loop defaults" last>
            <LoopConfigureLimits
              effectiveConfig={effectiveConfig}
              draft={model.draft.limits}
              disabled={model.busy}
              onChange={model.setLimits}
            />
          </ConfigureGroup>
        </div>

        <footer className="flex items-center gap-2 border-t border-line-soft px-5 py-3.5">
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={model.handleReset}
            disabled={model.busy}
            data-testid="loop-configure-reset"
          >
            <RotateCcw aria-hidden="true" className="size-3.5" />
            Reset to defaults
          </Button>
          <span className="flex-1" />
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => onOpenChange(false)}
            disabled={model.busy}
            data-testid="loop-configure-cancel"
          >
            Cancel
          </Button>
          <Button
            type="button"
            variant="primary"
            size="sm"
            onClick={model.handleSave}
            disabled={model.busy}
            data-testid="loop-configure-save"
          >
            <Save aria-hidden="true" className="size-3.5" />
            Save configuration
          </Button>
        </footer>
      </SheetContent>
    </Sheet>
  );
}

interface ConfigureGroupProps {
  label: string;
  lock?: string;
  last?: boolean;
  children: React.ReactNode;
}

/** One labelled group of the configure body: an uppercase eyebrow + optional lock badge. */
function ConfigureGroup({ label, lock, last = false, children }: ConfigureGroupProps) {
  return (
    <section className={last ? undefined : "mb-6"}>
      <div className="mb-3 flex items-center gap-2">
        <Eyebrow className="text-subtle">{label}</Eyebrow>
        {lock ? (
          <span className="ml-auto rounded-xs bg-badge-fill px-1.5 py-0.5 font-mono text-[9px] text-faint">
            {lock}
          </span>
        ) : null}
      </div>
      {children}
    </section>
  );
}
