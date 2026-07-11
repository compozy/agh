import { Link } from "@tanstack/react-router";

import { KindIcon, ListingRow, Pill, cn, providerKindIconRegistry } from "@agh/ui";

import type { AgentFleetRowModel } from "../lib/agent-fleet-projection";
import { AgentFleetNewSessionButton } from "./agent-fleet-new-session-button";

export interface AgentFleetRowProps {
  row: AgentFleetRowModel;
  newSessionDisabled?: boolean;
  onNewSession: (agentName: string) => void;
}

function AgentFleetRow({ row, newSessionDisabled = false, onNewSession }: AgentFleetRowProps) {
  const { agent, signals, meta, ariaLabel, hasDiagnostics, sessionsAvailable } = row;
  const sessionsTone =
    sessionsAvailable && signals && signals.active > 0 ? "text-success" : "text-muted";

  return (
    <ListingRow
      className="max-[920px]:grid-cols-[34px_minmax(0,1fr)] max-[920px]:gap-x-3 max-[920px]:gap-y-2"
      data-agent={agent.name}
      data-testid={`agent-fleet-row-${agent.name}`}
    >
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
        <ListingRow.Icon className="bg-elevated p-0 text-fg">
          <KindIcon
            className="size-[18px]"
            kind={agent.provider}
            registry={providerKindIconRegistry}
            size="sm"
            tone="default"
          />
        </ListingRow.Icon>
        <ListingRow.Main>
          <ListingRow.Name>
            <ListingRow.Title className="text-item-title">{agent.name}</ListingRow.Title>
          </ListingRow.Name>
          <ListingRow.Meta className="font-mono text-badge text-muted">{meta}</ListingRow.Meta>
        </ListingRow.Main>
      </ListingRow.Link>
      <ListingRow.Trail className="gap-2.5 max-[920px]:col-span-2 max-[920px]:col-start-2 max-[920px]:justify-self-start">
        <span
          aria-hidden="true"
          className={cn("font-mono text-badge tabular-nums", sessionsTone)}
          data-testid={`agent-fleet-sessions-${agent.name}`}
        >
          {sessionsAvailable && signals ? `● ${signals.active} / ${signals.total}` : "—"}
        </span>
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
