import { Globe, ListChecks, Plus, RefreshCcw, Sparkles, UserCheck, Zap } from "lucide-react";
import type { ReactNode } from "react";

import { Pill } from "@/components/design-system";
import { Button } from "@agh/ui";
import { cn } from "@/lib/utils";

import { TASK_TEMPLATES, type TaskTemplate, type TaskTemplateId } from "../lib/task-templates";

const TEMPLATE_ICONS: Record<TaskTemplateId, ReactNode> = {
  one_shot: <Zap className="size-4" />,
  recurring: <RefreshCcw className="size-4" />,
  epic: <Sparkles className="size-4" />,
  remote_peer: <Globe className="size-4" />,
  human_in_loop: <UserCheck className="size-4" />,
  blank: <Plus className="size-4" />,
};

export interface TasksEmptyStateProps {
  workspaceName?: string | null;
  onSelectTemplate: (templateId: TaskTemplateId) => void;
  onCopyCli?: () => void;
}

export function TasksEmptyState({
  workspaceName,
  onSelectTemplate,
  onCopyCli,
}: TasksEmptyStateProps) {
  const headline = workspaceName ? `No tasks yet in ${workspaceName}` : "No tasks yet";

  return (
    <div
      className="flex min-h-0 flex-1 flex-col items-center justify-start overflow-y-auto px-6 py-10"
      data-testid="tasks-empty-state"
    >
      <div className="flex flex-col items-center text-center">
        <div className="relative mb-4 flex size-16 items-center justify-center rounded-2xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)] text-[color:var(--color-accent)]">
          <ListChecks className="size-7" />
          <span className="absolute -top-1.5 -right-1.5 flex size-5 items-center justify-center rounded-full border border-[color:var(--color-accent)] bg-[color:var(--color-accent)] text-[color:var(--color-accent-ink)]">
            <Plus className="size-3" />
          </span>
        </div>

        <h2 className="text-2xl font-semibold text-[color:var(--color-text-primary)]">
          {headline}
        </h2>
        <p className="mt-3 max-w-md text-sm leading-relaxed text-[color:var(--color-text-secondary)]">
          Tasks are durable contracts of work. Each task can spawn multiple runs across agents,
          respect dependencies, and live in global or workspace scope. Start from a template, or
          define one from scratch.
        </p>

        <div className="mt-6 flex flex-wrap items-center justify-center gap-3">
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
        </div>
      </div>

      <div className="mt-12 w-full max-w-5xl" data-testid="tasks-empty-templates">
        <div className="flex items-center justify-between">
          <p className="font-mono text-[0.66rem] uppercase tracking-[0.16em] text-[color:var(--color-text-label)]">
            Start from a template
          </p>
          <p className="font-mono text-[0.66rem] uppercase tracking-[0.16em] text-[color:var(--color-text-tertiary)]">
            {TASK_TEMPLATES.length} templates
          </p>
        </div>

        <div className="mt-4 grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {TASK_TEMPLATES.map(template => (
            <TemplateCard
              key={template.id}
              onSelect={() => onSelectTemplate(template.id)}
              template={template}
            />
          ))}
        </div>
      </div>
    </div>
  );
}

interface TemplateCardProps {
  template: TaskTemplate;
  onSelect: () => void;
}

function TemplateCard({ template, onSelect }: TemplateCardProps) {
  const isBlank = template.id === "blank";

  return (
    <button
      className={cn(
        "flex h-full min-h-[140px] flex-col gap-2 rounded-2xl border px-4 py-4 text-left transition-colors",
        isBlank
          ? "border-dashed border-[color:rgba(58,58,60,0.6)] hover:border-[color:var(--color-text-label)]"
          : "border-[color:var(--color-divider)] bg-[color:var(--color-surface)] hover:border-[color:var(--color-text-label)]"
      )}
      data-testid={`tasks-empty-template-${template.id}`}
      onClick={onSelect}
      type="button"
    >
      <div className="flex items-center gap-2">
        <span className="flex size-7 items-center justify-center rounded-lg border border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)] text-[color:var(--color-text-secondary)]">
          {TEMPLATE_ICONS[template.id]}
        </span>
        <span className="text-sm font-semibold text-[color:var(--color-text-primary)]">
          {template.label}
        </span>
      </div>
      <p className="text-xs leading-relaxed text-[color:var(--color-text-secondary)]">
        {template.description}
      </p>
      {template.badges.length > 0 ? (
        <div className="mt-auto flex flex-wrap gap-1.5">
          {template.badges.map(badge => (
            <Pill key={badge.label} kind="state" tone={badge.tone}>
              {badge.label}
            </Pill>
          ))}
        </div>
      ) : null}
    </button>
  );
}
