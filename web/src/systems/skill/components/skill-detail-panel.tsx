import { AlertCircle, Wrench } from "lucide-react";

import {
  Button,
  Card,
  CodeBlock,
  DetailHeader,
  Empty,
  Eyebrow,
  Pill,
  Section,
  Spinner,
  StatusDot,
  Switch,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  Time,
} from "@agh/ui";

import {
  deriveSkillAuthor,
  deriveSkillCapabilities,
  deriveSkillRecentCalls,
  skillSourceTone,
} from "../lib/skill-formatters";
import type { SkillPayload, SkillShadowsResponse } from "../types";

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
  shadows: SkillShadowsResponse | undefined;
  isShadowsLoading: boolean;
  shadowsError: Error | null;
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
        className="flex-row items-center px-4 py-3 text-small-body text-muted"
        data-testid="content-loading"
        size="sm"
      >
        <Spinner aria-hidden="true" className="size-4 text-subtle" />
        Loading full skill content…
      </Card>
    );
  }
  if (error) {
    return (
      <Card className="px-4 py-3" data-testid="content-error" size="sm">
        <p className="text-small-body text-danger">
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
      <p className="text-small-body leading-relaxed text-muted">
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
            <Pill mono data-testid={`skill-capability-${capability}`} key={capability}>
              {capability}
            </Pill>
          ))}
        </div>
      )}
    </Section>
  );
}

const RECENT_STATUS_TONE = {
  success: "faint",
  error: "danger",
  pending: "accent",
} as const;

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
          className="overflow-hidden rounded-lg border border-line"
          data-testid="skill-recent-calls-table"
        >
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-18">Status</TableHead>
                <TableHead>Call</TableHead>
                <TableHead className="w-30 text-right">When</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {calls.map((call, index) => (
                <TableRow
                  data-testid={`skill-recent-call-row-${index}`}
                  key={`${call.label}-${call.timestamp ?? call.status}`}
                >
                  <TableCell>
                    <StatusDot
                      label={call.status}
                      tone={RECENT_STATUS_TONE[call.status]}
                      variant={call.status === "pending" ? "ring" : "solid"}
                    />
                  </TableCell>
                  <TableCell className="font-mono text-xs text-muted">{call.label}</TableCell>
                  <TableCell className="text-right font-mono text-eyebrow text-subtle">
                    {call.timestamp ? <Time iso={call.timestamp} /> : "--"}
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

function SkillProvenanceSection({ skill }: { skill: SkillPayload }) {
  const provenance = skill.provenance;
  if (!provenance) return null;
  const rows = [
    ["Tier", provenance.precedence_tier],
    ["Extension", provenance.installed_from_extension],
    ["Bundle", provenance.installed_from_bundle],
    ["Registry", provenance.registry],
    ["Slug", provenance.slug],
    ["Version", provenance.version],
  ].filter(([, value]) => typeof value === "string" && value.trim() !== "");

  return (
    <Section label="Provenance">
      <div
        className="overflow-hidden rounded-lg border border-line"
        data-testid="skill-provenance-table"
      >
        <Table>
          <TableBody>
            {rows.map(([label, value]) => (
              <TableRow key={label}>
                <TableCell className="w-32 text-eyebrow text-subtle">{label}</TableCell>
                <TableCell className="font-mono text-xs text-muted">{value}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    </Section>
  );
}

function SkillShadowSection({
  shadows,
  isLoading,
  error,
}: {
  shadows: SkillShadowsResponse | undefined;
  isLoading: boolean;
  error: Error | null;
}) {
  if (isLoading) {
    return (
      <Section label="Resolution">
        <div
          className="flex items-center gap-2 rounded-lg border border-line px-4 py-3 text-small-body text-muted"
          data-testid="skill-shadows-loading"
        >
          <Spinner aria-hidden="true" className="size-4 text-subtle" />
          Loading resolver paths…
        </div>
      </Section>
    );
  }
  if (error) {
    return (
      <Section label="Resolution">
        <p className="rounded-lg border border-danger/40 px-4 py-3 text-small-body text-danger">
          {error.message ?? "Failed to load skill resolution."}
        </p>
      </Section>
    );
  }
  if (!shadows || shadows.shadows.length === 0) return null;

  return (
    <Section label="Resolution">
      <div
        className="overflow-hidden rounded-lg border border-line"
        data-testid="skill-shadow-table"
      >
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-20">Winner</TableHead>
              <TableHead className="w-28">Tier</TableHead>
              <TableHead>Path</TableHead>
              <TableHead className="w-30 text-right">Detected</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {shadows.shadows.map(entry => (
              <TableRow key={`${entry.tier}-${entry.path}`}>
                <TableCell>
                  <StatusDot
                    label={entry.resolved_to_winner ? "winner" : "shadowed"}
                    tone={entry.resolved_to_winner ? "accent" : "faint"}
                    variant={entry.resolved_to_winner ? "solid" : "ring"}
                  />
                </TableCell>
                <TableCell>
                  <Pill mono tone={skillSourceTone(entry.tier)}>
                    {entry.tier}
                  </Pill>
                </TableCell>
                <TableCell className="max-w-0 truncate font-mono text-xs text-muted">
                  {entry.path}
                </TableCell>
                <TableCell className="text-right font-mono text-eyebrow text-subtle">
                  <Time iso={entry.detected_at} />
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
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
  shadows,
  isShadowsLoading,
  shadowsError,
}: SkillDetailPanelProps) {
  if (isLoading) {
    return (
      <div
        className="flex min-h-0 flex-1 items-center justify-center"
        data-testid="skill-detail-loading"
      >
        <Spinner aria-hidden="true" className="size-5 text-subtle" />
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

  const author = deriveSkillAuthor(skill);

  return (
    <div className="flex min-h-0 flex-1 flex-col overflow-y-auto" data-testid="skill-detail-panel">
      <DetailHeader
        data-testid="skill-detail-header"
        title={<span data-testid="skill-detail-title">{skill.name}</span>}
        pills={
          <>
            {skill.version ? (
              <Pill mono data-testid="detail-version-badge">{`v${skill.version}`}</Pill>
            ) : null}
            {author ? <Pill mono data-testid="detail-author-badge">{`@${author}`}</Pill> : null}
            <Pill mono data-testid="source-badge" tone={skillSourceTone(skill.source)}>
              {skill.provenance?.precedence_tier ?? skill.source}
            </Pill>
          </>
        }
        actions={
          <div className="flex items-center gap-2" data-testid="skill-enabled-toggle">
            <Eyebrow className="text-muted" id="skill-enabled-label">
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
      />

      <div className="flex flex-col gap-6 px-6 py-5">
        <Section label="Overview">
          <div className="flex flex-col gap-3">
            <p className="text-small-body leading-relaxed text-muted">{skill.description}</p>
            <span className="truncate font-mono text-eyebrow text-subtle">{skill.dir}</span>
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
        <SkillProvenanceSection skill={skill} />
        <SkillShadowSection error={shadowsError} isLoading={isShadowsLoading} shadows={shadows} />
        <SkillRecentCallsSection skill={skill} />
      </div>
    </div>
  );
}

export { SkillDetailPanel };
export type { SkillDetailPanelProps };
