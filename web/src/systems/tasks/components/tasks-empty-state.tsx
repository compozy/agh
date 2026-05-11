import { Globe, ListChecks, Plus, RefreshCcw, UserCheck, Zap } from "lucide-react";
import type { ReactNode } from "react";

import { Button, Empty, Eyebrow } from "@agh/ui";
import { cn } from "@/lib/utils";

import { getTaskTemplate, type TaskTemplate, type TaskTemplateId } from "../lib/task-templates";

export type TasksEmptyStateTone = "accent" | "info" | "warning" | "neutral";

interface TemplateSlot {
  id: TaskTemplateId;
  tone: TasksEmptyStateTone;
  icon: ReactNode;
}

/**
 * Four-card empty-state grid + §3 — `accent / info / warning /
 * neutral` tones only (the prior `violet / amber` palette is gone). Six
 * template definitions remain available to the editor; the empty state
 * surfaces a curated four that match the proposal reference.
 */
const TEMPLATE_SLOTS: TemplateSlot[] = [
  { id: "one_shot", tone: "accent", icon: <Zap className="size-4" /> },
  { id: "recurring", tone: "info", icon: <RefreshCcw className="size-4" /> },
  { id: "human_in_loop", tone: "warning", icon: <UserCheck className="size-4" /> },
  { id: "remote_peer", tone: "neutral", icon: <Globe className="size-4" /> },
];

const TONE_CLASS: Record<TasksEmptyStateTone, string> = {
  accent: "bg-(--accent-tint) text-(--accent)",
  info: "bg-(--info-tint) text-(--info)",
  warning: "bg-(--warning-tint) text-(--warning)",
  neutral: "bg-(--canvas-tint) text-(--muted)",
};

export interface TasksEmptyStateProps {
  workspaceName?: string | null;
  onSelectTemplate: (templateId: TaskTemplateId) => void;
  onCopyCli?: () => void;
}

/**
 * Empty-state for the Tasks domain — Empty primitive head + 4-card template
 * grid with the new tone palette (accent / info / warning / neutral). Eyebrow
 * head.
 */
export function TasksEmptyState({
  workspaceName,
  onSelectTemplate,
  onCopyCli,
}: TasksEmptyStateProps) {
  const headline = workspaceName ? `No tasks yet in ${workspaceName}` : "No tasks yet";

  return (
    <div className="flex min-h-0 flex-1 overflow-y-auto px-6 py-8" data-testid="tasks-empty-state">
      <div className="mx-auto flex w-full max-w-6xl flex-col gap-8">
        <Empty
          action={
            <>
              <Button
                data-testid="tasks-empty-cta-new"
                onClick={() => onSelectTemplate("one_shot")}
                size="lg"
                type="button"
              >
                <Plus className="size-4" />
                New task
              </Button>
              {onCopyCli ? (
                <Button
                  data-testid="tasks-empty-cta-cli"
                  onClick={onCopyCli}
                  size="lg"
                  type="button"
                  variant="outline"
                >
                  Copy CLI command
                </Button>
              ) : null}
            </>
          }
          description="Tasks are durable contracts of work. Each one can spawn runs across agents, respect dependencies, and live in workspace or global scope. Start from a template and keep the operational context visible as the queue grows."
          icon={ListChecks}
          title={headline}
        />

        <section data-testid="tasks-empty-templates" className="flex flex-col gap-4">
          <header className="flex items-baseline justify-between gap-2">
            <Eyebrow data-testid="tasks-empty-templates-eyebrow">Start from a template</Eyebrow>
            <Eyebrow className="text-(--muted)">{TEMPLATE_SLOTS.length} templates</Eyebrow>
          </header>
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-4">
            {TEMPLATE_SLOTS.map(slot => (
              <TemplateCard
                key={slot.id}
                onSelect={() => onSelectTemplate(slot.id)}
                slot={slot}
                template={getTaskTemplate(slot.id)}
              />
            ))}
          </div>
        </section>
      </div>
    </div>
  );
}

interface TemplateCardProps {
  template: TaskTemplate;
  slot: TemplateSlot;
  onSelect: () => void;
}

function TemplateCard({ template, slot, onSelect }: TemplateCardProps) {
  return (
    <button
      className={cn(
        "flex h-full min-h-[156px] flex-col gap-3 rounded-(--radius-lg) bg-(--canvas-soft) p-5 text-left transition-colors duration-(--dur) ease-(--ease)",
        "hover:bg-(--elevated) focus-visible:outline-none focus-visible:shadow-[inset_0_0_0_1px_var(--line-strong)]"
      )}
      data-testid={`tasks-empty-template-${template.id}`}
      data-tone={slot.tone}
      onClick={onSelect}
      type="button"
    >
      <span
        aria-hidden="true"
        className={cn(
          "flex size-7 items-center justify-center rounded-(--radius-md)",
          TONE_CLASS[slot.tone]
        )}
      >
        {slot.icon}
      </span>
      <span className="text-[13px] font-medium tracking-[-0.012em] text-(--fg-strong)">
        {template.label}
      </span>
      <p className="text-[12px] leading-relaxed text-(--muted)">{template.description}</p>
    </button>
  );
}
