import { Link } from "@tanstack/react-router";

import { KindIcon, ListingRow, Pill, providerKindIconRegistry } from "@agh/ui";

import type { AgentFleetRowModel } from "../lib/agent-fleet-projection";
import { AgentFleetNewSessionButton } from "./agent-fleet-new-session-button";

export interface AgentFleetRowProps {
  row: AgentFleetRowModel;
  newSessionDisabled?: boolean;
  onNewSession: (agentName: string) => void;
}

function AgentFleetRow({ row, newSessionDisabled = false, onNewSession }: AgentFleetRowProps) {
  const { agent, signals, ariaLabel, hasDiagnostics, sessionsAvailable, cardCategory, cardOrigin } =
    row;
  const model = agent.model?.trim() || null;
  const category = cardCategory && cardCategory !== agent.provider ? cardCategory : null;

  return (
    <ListingRow data-agent={agent.name} data-testid={`agent-fleet-row-${agent.name}`}>
      <ListingRow.Link
        render={
          <Link
            to="/agents/$name"
            params={{ name: agent.name }}
            aria-label={ariaLabel}
            data-testid={`agent-fleet-row-link-${agent.name}`}
          />
        }
      >
        <ListingRow.Icon>
          <KindIcon
            className="size-4"
            kind={agent.provider}
            registry={providerKindIconRegistry}
            size="sm"
            tone="default"
          />
        </ListingRow.Icon>
        <ListingRow.Main>
          <ListingRow.Name>
            <ListingRow.Title>{agent.name}</ListingRow.Title>
            {agent.provider ? (
              <Pill size="xs" tone="neutral">
                {agent.provider}
              </Pill>
            ) : null}
            {model ? <ListingRow.Slug>{model}</ListingRow.Slug> : null}
          </ListingRow.Name>
          <ListingRow.Meta>
            {category ? <span>{category}</span> : null}
            {category ? <ListingRow.MetaDot /> : null}
            <span>{cardOrigin}</span>
          </ListingRow.Meta>
        </ListingRow.Main>
      </ListingRow.Link>
      <ListingRow.Trail className="gap-3">
        <ListingRow.Stat className="w-16" data-testid={`agent-fleet-sessions-${agent.name}`}>
          <ListingRow.Stat.Value>
            {sessionsAvailable && signals ? `${signals.active}/${signals.total}` : "--"}
          </ListingRow.Stat.Value>
          <ListingRow.Stat.Label>sessions</ListingRow.Stat.Label>
        </ListingRow.Stat>
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
        <AgentFleetNewSessionButton
          agentName={agent.name}
          disabled={newSessionDisabled}
          onNewSession={onNewSession}
        />
      </ListingRow.Trail>
    </ListingRow>
  );
}

export { AgentFleetRow };
