import { Globe, ListChecks, Plus, RefreshCcw, Sparkles, UserCheck, Zap } from "lucide-react";
import type { ReactNode } from "react";

import { Button, Empty, Eyebrow, Pill, Section } from "@agh/ui";
import { cn } from "@/lib/utils";
import { pillToneFromLegacyTone } from "@/lib/pill-variant";

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

/**
 * Empty-state for the Tasks domain -- composes `@agh/ui` `Empty` + `Section` +
 * template-card grid. The `Empty` action slot owns the primary CTA; the Section
 * below lists the six task templates.
 */
export function TasksEmptyState({
  workspaceName,
  onSelectTemplate,
  onCopyCli,
}: TasksEmptyStateProps) {
  const headline = workspaceName ? `No tasks yet in ${workspaceName}` : "No tasks yet";

  return (
    <div className="flex min-h-0 flex-1 overflow-y-auto px-6 py-8" data-testid="tasks-empty-state">
      <div className="mx-auto flex w-full max-w-6xl flex-col gap-6">
        <Empty
          icon={ListChecks}
          title={headline}
          description="Tasks are durable contracts of work. Each one can spawn runs across agents, respect dependencies, and live in workspace or global scope. Start from a template and keep the operational context visible as the queue grows."
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
        />

        <Section
          data-testid="tasks-empty-templates"
          label="Start from a template"
          right={<Eyebrow>{TASK_TEMPLATES.length} templates</Eyebrow>}
        >
          <div className="mt-3 grid gap-4 md:grid-cols-2 xl:grid-cols-[1.2fr_1fr_1fr]">
            {TASK_TEMPLATES.map(template => (
              <TemplateCard
                key={template.id}
                onSelect={() => onSelectTemplate(template.id)}
                template={template}
              />
            ))}
          </div>
        </Section>
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
        "flex h-full min-h-[156px] flex-col gap-3 rounded-xl border px-5 py-5 text-left transition-colors",
        isBlank
          ? "border-dashed border-divider/60 hover:border-(--color-text-label)"
          : "border-(--color-divider) bg-(--color-surface) hover:border-(--color-text-label)"
      )}
      data-testid={`tasks-empty-template-${template.id}`}
      onClick={onSelect}
      type="button"
    >
      <div className="flex items-center gap-2">
        <span className="flex size-7 items-center justify-center rounded-lg border border-(--color-divider) bg-(--color-surface-panel) text-(--color-text-secondary)">
          {TEMPLATE_ICONS[template.id]}
        </span>
        <span className="text-sm font-semibold text-(--color-text-primary)">{template.label}</span>
      </div>
      <p className="text-sm leading-relaxed text-(--color-text-secondary)">
        {template.description}
      </p>
      {template.badges.length > 0 ? (
        <div className="mt-auto flex flex-wrap gap-1.5">
          {template.badges.map(badge => (
            <Pill key={badge.label} tone={pillToneFromLegacyTone(badge.tone)}>
              {badge.label}
            </Pill>
          ))}
        </div>
      ) : null}
    </button>
  );
}
