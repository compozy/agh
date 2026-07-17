import { Link } from "@tanstack/react-router";

import { CatalogCard, KindIcon, Pill, providerKindIconRegistry } from "@agh/ui";

import type { AgentFleetRowModel } from "../lib/agent-fleet-projection";
import { AgentFleetNewSessionButton } from "./agent-fleet-new-session-button";

export interface AgentFleetCardProps {
  row: AgentFleetRowModel;
  newSessionDisabled?: boolean;
  onNewSession: (agentName: string) => void;
}

function AgentFleetCard({ row, newSessionDisabled = false, onNewSession }: AgentFleetCardProps) {
  const { agent, signals, ariaLabel, hasDiagnostics, sessionsAvailable } = row;
  const category = row.cardCategory;
  const model = agent.model?.trim() || null;
  const origin = row.cardOrigin;

  return (
    <CatalogCard actionable data-agent={agent.name} data-testid={`agent-fleet-card-${agent.name}`}>
      <Link
        aria-label={ariaLabel}
        className="flex min-w-0 flex-col gap-3 outline-none focus-visible:shadow-focus-inset"
        data-testid={`agent-fleet-card-link-${agent.name}`}
        params={{ name: agent.name }}
        to="/agents/$name"
      >
        <div className="flex min-w-0 items-start gap-3">
          <CatalogCard.Logo>
            <KindIcon
              className="size-4"
              kind={agent.provider}
              registry={providerKindIconRegistry}
              size="sm"
              tone="default"
            />
          </CatalogCard.Logo>
          <div className="flex min-w-0 flex-1 flex-col gap-1">
            <CatalogCard.Title>{agent.name}</CatalogCard.Title>
            <CatalogCard.Meta data-testid={`agent-fleet-card-meta-${agent.name}`}>
              <span className="flex min-w-0 items-center gap-1.5 truncate text-form-label text-subtle">
                {category ? <span className="truncate">{category}</span> : null}
                {category && model ? (
                  <span aria-hidden="true" className="text-faint">
                    ·
                  </span>
                ) : null}
                {model ? (
                  <span className="truncate font-mono text-badge tracking-mono text-faint">
                    {model}
                  </span>
                ) : null}
                {(category || model) && origin ? (
                  <span aria-hidden="true" className="text-faint">
                    ·
                  </span>
                ) : null}
                {origin ? <span className="truncate text-faint">{origin}</span> : null}
              </span>
            </CatalogCard.Meta>
          </div>
        </div>
      </Link>
      <CatalogCard.Actions className="justify-between">
        <div className="flex min-w-0 flex-wrap items-center gap-2">
          {sessionsAvailable && signals ? (
            <Pill
              size="sm"
              tone={signals.status === "active" ? "success" : "neutral"}
              data-testid={`agent-fleet-status-${agent.name}`}
            >
              <Pill.Dot tone={signals.status === "active" ? "success" : "neutral"} size="sm" />
              {signals.status === "active" ? "Active" : "Idle"}
            </Pill>
          ) : null}
          {hasDiagnostics ? (
            <Pill tone="warning" size="sm" data-testid={`agent-fleet-invalid-${agent.name}`}>
              Invalid
            </Pill>
          ) : null}
        </div>
        <AgentFleetNewSessionButton
          agentName={agent.name}
          disabled={newSessionDisabled}
          onNewSession={onNewSession}
        />
      </CatalogCard.Actions>
    </CatalogCard>
  );
}

export { AgentFleetCard };
