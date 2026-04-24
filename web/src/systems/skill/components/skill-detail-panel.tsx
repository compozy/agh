import { AlertCircle, Loader2, Wrench } from "lucide-react";

import {
  Button,
  Empty,
  MonoBadge,
  PageHeader,
  Section,
  StatusDot,
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
        <MonoBadge data-testid="detail-version-badge">{`v${skill.version}`}</MonoBadge>
      ) : null}
      {author ? <MonoBadge data-testid="detail-author-badge">{`@${author}`}</MonoBadge> : null}
      <MonoBadge data-testid="source-badge" tone={skillSourceTone(skill.source)}>
        {skill.source}
      </MonoBadge>
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
      <div
        className="rounded-lg border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] p-4"
        data-testid="content-body"
      >
        <pre className="max-h-96 overflow-auto whitespace-pre-wrap font-mono text-[12px] leading-relaxed text-[color:var(--color-text-secondary)]">
          {content}
        </pre>
      </div>
    );
  }
  if (isLoading) {
    return (
      <div
        className="flex items-center gap-2 rounded-lg border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] px-4 py-3 text-[13px] text-[color:var(--color-text-secondary)]"
        data-testid="content-loading"
      >
        <Loader2
          aria-hidden="true"
          className="size-4 animate-spin text-[color:var(--color-text-tertiary)]"
        />
        Loading full skill content…
      </div>
    );
  }
  if (error) {
    return (
      <div
        className="flex flex-col gap-2 rounded-lg border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] px-4 py-3"
        data-testid="content-error"
      >
        <p className="text-[13px] text-[color:var(--color-danger)]">
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
      </div>
    );
  }
  return (
    <div
      className="flex flex-col gap-3 rounded-lg border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] px-4 py-3"
      data-testid="content-empty"
    >
      <p className="text-[13px] leading-relaxed text-[color:var(--color-text-secondary)]">
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
    </div>
  );
}

function SkillCapabilitiesSection({ skill }: { skill: SkillPayload }) {
  const capabilities = deriveSkillCapabilities(skill);
  return (
    <Section label="Capabilities">
      {capabilities.length === 0 ? (
        <div data-testid="skill-capabilities-empty">
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
            <MonoBadge
              data-testid={`skill-capability-${capability}`}
              key={capability}
              uppercase={false}
            >
              {capability}
            </MonoBadge>
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
        <div data-testid="skill-recent-calls-empty">
          <Empty
            className="max-w-sm"
            description="No recent invocations recorded."
            title="No recent calls"
            fill={false}
          />
        </div>
      ) : (
        <div
          className="overflow-hidden rounded-lg border border-[color:var(--color-divider)]"
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
                  key={`${call.label}-${index}`}
                >
                  <TableCell>
                    <StatusDot
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
                  <TableCell className="font-mono text-[12px] text-[color:var(--color-text-secondary)]">
                    {call.label}
                  </TableCell>
                  <TableCell className="text-right font-mono text-[11px] text-[color:var(--color-text-tertiary)]">
                    {call.timestamp ? formatSkillRelativeTime(call.timestamp) : "—"}
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
        <Loader2
          aria-hidden="true"
          className="size-5 animate-spin text-[color:var(--color-text-tertiary)]"
        />
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
          description={error.message ?? "Failed to load skill details"}
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
      <PageHeader
        count={undefined}
        icon={() => <Wrench className="size-3.5" data-testid="skill-detail-icon" />}
        meta={<SkillDetailMeta skill={skill} />}
        title={<span data-testid="skill-detail-title">{skill.name}</span>}
      />

      <div className="flex flex-col gap-6 px-6 py-5">
        <Section
          label="Overview"
          right={
            <div className="flex items-center gap-2" data-testid="skill-enabled-toggle">
              <span
                className="font-mono text-[11px] uppercase tracking-[0.08em] text-[color:var(--color-text-label)]"
                id="skill-enabled-label"
              >
                {skill.enabled ? "Enabled" : "Disabled"}
              </span>
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
            <p className="text-[13px] leading-relaxed text-[color:var(--color-text-secondary)]">
              {skill.description}
            </p>
            <span className="truncate font-mono text-[11px] text-[color:var(--color-text-tertiary)]">
              {skill.dir}
            </span>
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
