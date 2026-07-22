import { Brain } from "lucide-react";
import { useId, type ReactNode } from "react";

import { type ReasoningEffort } from "@/lib/api-contract";
import { cn, Eyebrow, IntensityMeter } from "@agh/ui";

import { reasoningEffortLabel, reasoningEffortPosition, type RuntimeReasoningState } from "./types";

export interface ReasoningBarProps {
  reasoning: RuntimeReasoningState;
  value: ReasoningEffort | "";
  providerName: string;
  modelName: string;
  onSelect: (effort: ReasoningEffort | "") => void;
}

function Note({ children }: { children: ReactNode }) {
  return (
    <div className="flex items-center gap-2.5 px-px py-0.5 text-small-body text-subtle">
      <Brain aria-hidden="true" className="size-4 shrink-0 text-faint" />
      <span>{children}</span>
    </div>
  );
}

const SEG_BUTTON_CLASS =
  "inline-flex flex-col items-center gap-1 rounded-sm px-1 py-1.5 text-muted outline-none transition-colors hover:bg-row-hover hover:text-fg-strong focus-visible:bg-row-hover focus-visible:text-fg-strong focus-visible:ring-2 focus-visible:ring-accent data-[on=true]:bg-accent-tint-strong data-[on=true]:text-accent-strong";

export function ReasoningBar({
  reasoning,
  value,
  providerName,
  modelName,
  onSelect,
}: ReasoningBarProps) {
  // Per-instance id so multiple mounted selectors never collide on one DOM id
  // (the label→group `aria-labelledby` relationship must stay local to this bar).
  const labelId = useId();
  if (reasoning.mode === "no-model") {
    return (
      <div
        className="shrink-0 border-t border-line-soft bg-canvas px-[13px] py-[11px]"
        data-testid="runtime-selector-reasoning"
        data-reasoning-mode="no-model"
      >
        <Note>Select a model to choose its reasoning effort.</Note>
      </div>
    );
  }

  if (reasoning.mode === "none") {
    return (
      <div
        className="shrink-0 border-t border-line-soft bg-canvas px-[13px] py-[11px]"
        data-testid="runtime-selector-reasoning"
        data-reasoning-mode="none"
      >
        <Note>This model doesn’t use reasoning effort; the provider handles it.</Note>
      </div>
    );
  }

  if (reasoning.mode === "supported-nolevels") {
    return (
      <div
        className="shrink-0 border-t border-line-soft bg-canvas px-[13px] py-[11px]"
        data-testid="runtime-selector-reasoning"
        data-reasoning-mode="supported-nolevels"
      >
        <Note>
          <b className="font-medium text-muted">Reasoning is on.</b> {providerName} doesn’t expose
          selectable effort for {modelName}.
        </Note>
      </div>
    );
  }

  const isDefault = value === "";
  const current = value || reasoning.defaultEffort;
  const levels = [...reasoning.levels].sort(
    (a, b) => reasoningEffortPosition(a) - reasoningEffortPosition(b)
  );
  const sourceLabel = reasoning.source === "acp" ? "ACP" : "CATALOG";

  return (
    <div
      className="shrink-0 border-t border-line-soft bg-canvas px-[13px] py-[11px]"
      data-testid="runtime-selector-reasoning"
      data-reasoning-mode="levels"
    >
      <div className="mb-[9px] flex items-center gap-2">
        <Eyebrow id={labelId} className="shrink-0 text-subtle">
          Reasoning effort
        </Eyebrow>
        <span className="min-w-0 truncate font-mono text-badge text-faint">
          {providerName} · {modelName}
        </span>
        <span className="ml-auto shrink-0 rounded-xxs bg-badge-fill px-[5px] py-px font-mono text-micro font-semibold tracking-mono text-faint">
          {sourceLabel}
        </span>
      </div>
      <div
        role="group"
        aria-labelledby={labelId}
        className="flex gap-0.5 rounded-md border border-line-soft bg-canvas p-0.5"
      >
        <button
          type="button"
          data-rz=""
          data-on={isDefault}
          title="Use provider default"
          aria-pressed={isDefault}
          className={cn(SEG_BUTTON_CLASS, "shrink-0 px-[9px]")}
          onClick={() => onSelect("")}
        >
          <span
            aria-hidden="true"
            className={cn(
              "size-3 rounded-full ring-2 ring-inset",
              isDefault ? "ring-accent-strong" : "ring-faint"
            )}
          />
          <span className="text-eyebrow font-medium">Default</span>
        </button>
        {levels.map(effort => {
          const active = !isDefault && current === effort;
          return (
            <button
              key={effort}
              type="button"
              data-rz={effort}
              data-on={active}
              aria-pressed={active}
              className={cn(SEG_BUTTON_CLASS, "flex-1")}
              onClick={() => onSelect(effort)}
            >
              <IntensityMeter position={reasoningEffortPosition(effort)} />
              <span className="text-eyebrow font-medium">{reasoningEffortLabel(effort)}</span>
            </button>
          );
        })}
      </div>
    </div>
  );
}
