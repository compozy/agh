import { ScrollArea, Section, Empty, Pill, cn } from "@agh/ui";
import { Plug } from "lucide-react";

import type { AgentPayload } from "../types";

export interface AgentInfoPanelProps {
  agent: AgentPayload;
  className?: string;
}

const PANEL_WIDTH = 320;

export function AgentInfoPanel({ agent, className }: AgentInfoPanelProps) {
  const mcpServers = agent.mcp_servers ?? [];
  return (
    <aside
      data-testid="agent-info-panel"
      aria-label={`${agent.name} agent details`}
      style={{ width: PANEL_WIDTH }}
      className={cn(
        "hidden shrink-0 flex-col overflow-hidden border-l border-(--color-divider) bg-(--color-canvas) xl:flex",
        className
      )}
    >
      <ScrollArea className="flex-1 min-h-0">
        <div className="flex flex-col gap-6 px-4 py-5">
          <Section label="MCP Servers" data-testid="agent-info-mcp-servers">
            {mcpServers.length === 0 ? (
              <Empty
                icon={Plug}
                title="No MCP servers"
                description="This agent does not declare any MCP servers."
                data-testid="agent-info-mcp-empty"
                fill={false}
              />
            ) : (
              <ul className="flex flex-col gap-2" data-testid="agent-info-mcp-list">
                {mcpServers.map(server => {
                  const transport = server.transport ?? "stdio";
                  return (
                    <li
                      key={server.name}
                      data-testid={`agent-info-mcp-row-${server.name}`}
                      className="flex items-center justify-between gap-2 rounded-md border border-(--color-divider) bg-(--color-surface) px-3 py-2"
                    >
                      <div className="flex min-w-0 flex-col gap-0.5">
                        <span className="truncate text-small-body font-medium text-(--color-text-primary)">
                          {server.name}
                        </span>
                        {server.command || server.url ? (
                          <span className="truncate font-mono text-badge tracking-mono text-(--color-text-tertiary)">
                            {server.url ?? server.command}
                          </span>
                        ) : null}
                      </div>
                      <Pill mono tone="info" data-testid={`agent-info-mcp-kind-${server.name}`}>
                        {transport}
                      </Pill>
                    </li>
                  );
                })}
              </ul>
            )}
          </Section>
        </div>
      </ScrollArea>
    </aside>
  );
}
