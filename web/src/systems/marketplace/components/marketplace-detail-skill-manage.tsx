import {
  Button,
  Card,
  CodeBlock,
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

import { skillSourceTone } from "@/systems/skill";

import { useMarketplaceDetailSkillManage } from "../hooks/use-marketplace-detail-skill-manage";

interface MarketplaceDetailSkillManageProps {
  name: string;
}

function MarketplaceDetailSkillManage({ name }: MarketplaceDetailSkillManageProps) {
  const state = useMarketplaceDetailSkillManage(name);

  return (
    <>
      <Section label="Manage">
        <div className="flex items-center justify-between rounded-lg bg-canvas-soft px-4 py-3">
          <Eyebrow className="text-muted" id="marketplace-skill-enabled-label">
            {state.skill?.enabled ? "Enabled" : "Disabled"}
          </Eyebrow>
          <Switch
            aria-labelledby="marketplace-skill-enabled-label"
            checked={state.skill?.enabled === true}
            data-testid="skill-enabled-switch"
            disabled={state.acting || !state.skill}
            onCheckedChange={state.toggleEnabled}
          />
        </div>
      </Section>
      <SkillManageContent
        content={state.contentQuery.data}
        error={state.contentQuery.error}
        isLoading={state.contentRequested && state.contentQuery.isLoading}
        name={name}
        onRetry={() => void state.contentQuery.refetch()}
        onView={() => state.setContentRequested(true)}
      />
      <SkillManageResolution
        error={state.shadowsQuery.error}
        isLoading={state.shadowsQuery.isLoading}
        shadows={state.shadowsQuery.data}
      />
    </>
  );
}

function SkillManageContent({
  content,
  error,
  isLoading,
  name,
  onRetry,
  onView,
}: {
  content: string | undefined;
  error: Error | null;
  isLoading: boolean;
  name: string;
  onRetry: () => void;
  onView: () => void;
}) {
  return (
    <Section label="Content">
      {content ? (
        <Card className="p-0" data-testid="content-body" size="sm">
          <CodeBlock code={content} copyable={false} showPrompt={false} truncateLines={16} />
        </Card>
      ) : isLoading ? (
        <Card
          className="flex-row items-center px-4 py-3 text-small-body text-muted"
          data-testid="content-loading"
          size="sm"
        >
          <Spinner aria-hidden="true" className="size-4 text-subtle" />
          Loading full skill content…
        </Card>
      ) : error ? (
        <Card className="px-4 py-3" data-testid="content-error" size="sm">
          <p className="text-small-body text-danger">
            {error.message ?? "Failed to load full content."}
          </p>
          <Button
            data-testid="retry-view-content-btn"
            onClick={onRetry}
            size="sm"
            type="button"
            variant="outline"
          >
            Try again
          </Button>
        </Card>
      ) : (
        <Card className="px-4 py-3" data-testid="content-empty" size="sm">
          <p className="text-small-body leading-relaxed text-muted">
            Full skill instructions are loaded on demand.
          </p>
          <div>
            <Button
              data-testid="view-full-content-btn"
              onClick={() => onView()}
              size="sm"
              type="button"
              variant="outline"
            >
              View full content
            </Button>
          </div>
        </Card>
      )}
      <span className="sr-only">{name}</span>
    </Section>
  );
}

function SkillManageResolution({
  error,
  isLoading,
  shadows,
}: {
  error: Error | null;
  isLoading: boolean;
  shadows:
    | {
        shadows: {
          detected_at: string;
          path: string;
          resolved_to_winner: boolean;
          tier: string;
        }[];
      }
    | undefined;
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

export { MarketplaceDetailSkillManage };
export type { MarketplaceDetailSkillManageProps };
