import type { ReactNode } from "react";

import { Users2 } from "lucide-react";

import { PageHead, Pill } from "@agh/ui";

import { formatAgentOriginLabel, formatCategoryMetaSegment } from "../lib/agent-fleet-projection";
import type { AgentPayload } from "../types";
import { AgentPageStatusPill } from "./agent-page-header";

export interface AgentDetailHeaderProps {
  agent: AgentPayload;
  activeCount: number;
  /** Hide Active/Idle status when catalog metrics are loading or unavailable. */
  metricsUnavailable?: boolean;
  /**
   * Runtime controls hosted on the hero's trailing edge (PageHead `actions` —
   * the provider·model·reasoning selector lives here, not in the topbar).
   */
  actions?: ReactNode;
}

export function AgentDetailHeader({
  agent,
  activeCount,
  metricsUnavailable = false,
  actions,
}: AgentDetailHeaderProps) {
  const category = formatCategoryMetaSegment(agent.category_path);
  const origin = formatAgentOriginLabel(agent.origin);
  const hasDiagnostics = Array.isArray(agent.diagnostics) && agent.diagnostics.length > 0;

  return (
    <div className="pt-5">
      <PageHead
        actions={actions}
        data-testid="agent-detail-header"
        icon={Users2}
        title={<span data-testid="agent-detail-header-name">{agent.name}</span>}
        variant="detail"
        pills={
          <>
            {!metricsUnavailable ? <AgentPageStatusPill activeCount={activeCount} /> : null}
            {origin ? (
              <Pill mono size="sm" data-testid="agent-detail-header-origin">
                {origin}
              </Pill>
            ) : null}
            {hasDiagnostics ? (
              <Pill tone="warning" size="sm" data-testid="agent-detail-header-invalid">
                Invalid
              </Pill>
            ) : null}
          </>
        }
        meta={
          category ? (
            <span className="truncate" data-testid="agent-detail-header-meta">
              {category}
            </span>
          ) : undefined
        }
      />
    </div>
  );
}
