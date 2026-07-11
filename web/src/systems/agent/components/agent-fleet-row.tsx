import { Link } from "@tanstack/react-router";

import { ListingRow, OwnerAvatar, Pill, cn } from "@agh/ui";

import type { AgentFleetRowModel } from "../lib/agent-fleet-projection";

export interface AgentFleetRowProps {
  row: AgentFleetRowModel;
}

function AgentFleetRow({ row }: AgentFleetRowProps) {
  const { agent, signals, meta, ariaLabel, hasDiagnostics, sessionsAvailable } = row;
  const sessionsTone =
    sessionsAvailable && signals && signals.active > 0 ? "text-success" : "text-muted";

  return (
    <ListingRow data-testid={`agent-fleet-row-${agent.name}`} data-agent={agent.name}>
      <ListingRow.Link
        className="col-span-3 grid-cols-[34px_minmax(0,1fr)_auto]"
        render={
          <Link
            to="/agents/$name"
            params={{ name: agent.name }}
            aria-label={ariaLabel}
            data-testid={`agent-fleet-row-link-${agent.name}`}
          />
        }
      >
        <ListingRow.Icon className="bg-transparent p-0">
          <OwnerAvatar ownerKind="agent" ownerId={agent.name} name={agent.name} size="default" />
        </ListingRow.Icon>
        <ListingRow.Main>
          <ListingRow.Name>
            <ListingRow.Title className="text-item-title">{agent.name}</ListingRow.Title>
          </ListingRow.Name>
          <ListingRow.Meta className="font-mono text-badge text-muted">{meta}</ListingRow.Meta>
        </ListingRow.Main>
        <ListingRow.Trail className="justify-self-end gap-2">
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
        </ListingRow.Trail>
      </ListingRow.Link>
    </ListingRow>
  );
}

export { AgentFleetRow };
