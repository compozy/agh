import { AlertCircle, Wrench } from "lucide-react";

import {
  Button,
  Card,
  CodeBlock,
  Empty,
  Eyebrow,
  Pill,
  Section,
  Spinner,
  Switch,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@agh/ui";

import {
  deriveSkillAuthor,
  deriveSkillCapabilities,
  deriveSkillRecentCalls,
  formatSkillRelativeTime,
  skillSourceTone,
} from "../lib/skill-formatters";
import type { SkillPayload } from "../types";

interface SkillDetailPanelProps {
  skill: SkillPayload | undefined;
  isLoading: boolean;
  error: Error | null;
  content: string | undefined;
  isContentLoading: boolean;
  contentError: Error | null;
  onViewContent: (name: string) => void;
  onRetryContent: () => void;
  onDisable: (name: string) => void;
  onEnable: (name: string) => void;
  isActionPending: boolean;
}

function SkillDetailMeta({ skill }: { skill: SkillPayload }) {
  const author = deriveSkillAuthor(skill);
  return (
    <div className="flex flex-wrap items-center gap-1.5">
      {skill.version ? (
        <Pill mono data-testid="detail-version-badge">{`v${skill.version}`}</Pill>
      ) : null}
      {author ? <Pill mono data-testid="detail-author-badge">{`@${author}`}</Pill> : null}
      <Pill mono data-testid="source-badge" tone={skillSourceTone(skill.source)}>
        {skill.source}
      </Pill>
    </div>
  );
}

interface SkillContentSectionProps {
  skill: SkillPayload;
  content: string | undefined;
  isLoading: boolean;
  error: Error | null;
  onViewContent: (name: string) => void;
  onRetryContent: () => void;
}

function SkillContentSection({
  skill,
  content,
  isLoading,
  error,
  onViewContent,
  onRetryContent,
}: SkillContentSectionProps) {
  if (content) {
    return (
      <Card className="p-0" data-testid="content-body" size="sm">
        <CodeBlock code={content} copyable={false} showPrompt={false} truncateLines={16} />
      </Card>
    );
  }
  if (isLoading) {
    return (
      <Card
        className="flex-row items-center px-4 py-3 text-small-body text-(--muted)"
        data-testid="content-loading"
        size="sm"
      >
        <Spinner aria-hidden="true" className="size-4 text-(--subtle)" />
        Loading full skill content…
      </Card>
    );
  }
  if (error) {
    return (
      <Card className="px-4 py-3" data-testid="content-error" size="sm">
        <p className="text-small-body text-(--danger)">
          {error.message ?? "Failed to load full content."}
        </p>
        <Button
          data-testid="retry-view-content-btn"
          onClick={onRetryContent}
          size="sm"
          type="button"
          variant="outline"
        >
          Try again
        </Button>
      </Card>
    );
  }
  return (
    <Card className="px-4 py-3" data-testid="content-empty" size="sm">
      <p className="text-small-body leading-relaxed text-(--muted)">
        Full skill instructions are loaded on demand.
      </p>
      <div>
        <Button
          data-testid="view-full-content-btn"
          onClick={() => onViewContent(skill.name)}
          size="sm"
          type="button"
          variant="outline"
        >
          View full content
        </Button>
      </div>
    </Card>
  );
}

function SkillCapabilitiesSection({ skill }: { skill: SkillPayload }) {
  const capabilities = deriveSkillCapabilities(skill);
  return (
    <Section label="Capabilities">
      {capabilities.length === 0 ? (
        <div className="flex justify-center px-2 py-6" data-testid="skill-capabilities-empty">
          <Empty
            className="max-w-sm"
            description="This skill has not declared any capabilities."
            title="No capabilities"
            fill={false}
          />
        </div>
      ) : (
        <div className="flex flex-wrap items-center gap-1.5" data-testid="skill-capabilities-list">
          {capabilities.map(capability => (
            <Pill
              mono
              data-testid={`skill-capability-${capability}`}
              key={capability}
              uppercase={false}
            >
              {capability}
            </Pill>
          ))}
        </div>
      )}
    </Section>
  );
}

function SkillRecentCallsSection({ skill }: { skill: SkillPayload }) {
  const calls = deriveSkillRecentCalls(skill);
  return (
    <Section label="Recent calls">
      {calls.length === 0 ? (
        <div className="flex justify-center px-2 py-6" data-testid="skill-recent-calls-empty">
          <Empty
            className="max-w-sm"
            description="No recent invocations recorded."
            title="No recent calls"
            fill={false}
          />
        </div>
      ) : (
        <div
          className="overflow-hidden rounded-lg border border-(--line)"
          data-testid="skill-recent-calls-table"
        >
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-[72px]">Status</TableHead>
                <TableHead>Call</TableHead>
                <TableHead className="w-[120px] text-right">When</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {calls.map((call, index) => (
                <TableRow
                  data-testid={`skill-recent-call-row-${index}`}
                  key={`${call.label}-${call.timestamp ?? call.status}`}
                >
                  <TableCell>
                    <Pill.Dot
                      pulse={call.status === "pending"}
                      tone={
                        call.status === "error"
                          ? "danger"
                          : call.status === "pending"
                            ? "accent"
                            : "success"
                      }
                    />
                  </TableCell>
                  <TableCell className="font-mono text-xs text-(--muted)">{call.label}</TableCell>
                  <TableCell className="text-right font-mono text-eyebrow text-(--subtle)">
                    {call.timestamp ? formatSkillRelativeTime(call.timestamp) : "--"}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}
    </Section>
  );
}

function SkillDetailPanel({
  skill,
  isLoading,
  error,
  content,
  isContentLoading,
  contentError,
  onViewContent,
  onRetryContent,
  onDisable,
  onEnable,
  isActionPending,
}: SkillDetailPanelProps) {
  if (isLoading) {
    return (
      <div
        className="flex min-h-0 flex-1 items-center justify-center"
        data-testid="skill-detail-loading"
      >
        <Spinner aria-hidden="true" className="size-5 text-(--subtle)" />
      </div>
    );
  }

  if (error) {
    return (
      <div
        className="flex min-h-0 flex-1 items-center justify-center px-6 py-10"
        data-testid="skill-detail-error"
      >
        <Empty
          className="max-w-md"
          description={error.message}
          icon={AlertCircle}
          title="Failed to load skill details"
        />
      </div>
    );
  }

  if (!skill) {
    return (
      <div
        className="flex min-h-0 flex-1 items-center justify-center px-6 py-10"
        data-testid="skill-detail-empty"
      >
        <Empty
          className="max-w-md"
          description="Select a skill to view details"
          icon={Wrench}
          title="Select a skill to view details"
        />
      </div>
    );
  }

  const handleToggle = (next: boolean) => {
    if (isActionPending) return;
    if (next) {
      onEnable(skill.name);
    } else {
      onDisable(skill.name);
    }
  };

  return (
    <div className="flex min-h-0 flex-1 flex-col overflow-y-auto" data-testid="skill-detail-panel">
      <header
        data-slot="page-header"
        className="flex min-h-11 flex-col gap-2 border-b border-(--line) px-4 py-2.5"
      >
        <div
          data-slot="page-header-main"
          className="flex min-w-0 flex-wrap items-center gap-2 sm:gap-3"
        >
          <div data-slot="page-header-title" className="flex min-w-0 items-center gap-2">
            <span
              aria-hidden="true"
              data-slot="page-header-icon"
              className="inline-flex size-6 shrink-0 items-center justify-center rounded-(--radius-sm) bg-(--elevated) text-(--accent)"
            >
              <Wrench className="size-3.5" data-testid="skill-detail-icon" />
            </span>
            <h1 className="truncate text-[22px] font-medium tracking-[-0.026em] text-(--fg-strong)">
              <span data-testid="skill-detail-title">{skill.name}</span>
            </h1>
          </div>
          <div
            data-slot="page-header-meta"
            className="ml-auto flex shrink-0 items-center gap-2 text-[13px] text-(--muted)"
          >
            <SkillDetailMeta skill={skill} />
          </div>
        </div>
      </header>

      <div className="flex flex-col gap-6 px-6 py-5">
        <Section
          label="Overview"
          right={
            <div className="flex items-center gap-2" data-testid="skill-enabled-toggle">
              <Eyebrow case="upper" tone="muted" id="skill-enabled-label">
                {skill.enabled ? "Enabled" : "Disabled"}
              </Eyebrow>
              <Switch
                aria-labelledby="skill-enabled-label"
                checked={skill.enabled}
                data-testid="skill-enabled-switch"
                disabled={isActionPending}
                onCheckedChange={handleToggle}
              />
            </div>
          }
        >
          <div className="flex flex-col gap-3">
            <p className="text-small-body leading-relaxed text-(--muted)">{skill.description}</p>
            <span className="truncate font-mono text-eyebrow text-(--subtle)">{skill.dir}</span>
            <SkillContentSection
              content={content}
              error={contentError}
              isLoading={isContentLoading}
              onRetryContent={onRetryContent}
              onViewContent={onViewContent}
              skill={skill}
            />
          </div>
        </Section>

        <SkillCapabilitiesSection skill={skill} />
        <SkillRecentCallsSection skill={skill} />
      </div>
    </div>
  );
}

export { SkillDetailPanel };
export type { SkillDetailPanelProps };
