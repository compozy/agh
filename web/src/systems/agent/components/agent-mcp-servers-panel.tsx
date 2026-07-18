import { Plug } from "lucide-react";

import { Empty, Pill, Section, cn } from "@agh/ui";

import type { AgentMCPServer, AgentPayload } from "../types";

export interface AgentMcpServersPanelProps {
  agent: AgentPayload;
  className?: string;
  /** When true, omit the section chrome (used inside Configuration tab Section). */
  bare?: boolean;
}

function envKeyNames(server: AgentMCPServer): string[] {
  return Object.keys(server.env ?? {});
}

function secretKeyNames(server: AgentMCPServer): string[] {
  return Object.keys(server.secret_env ?? {});
}

export function AgentMcpServersPanel({
  agent,
  className,
  bare = false,
}: AgentMcpServersPanelProps) {
  const mcpServers = agent.mcp_servers ?? [];

  const list =
    mcpServers.length === 0 ? (
      <Empty
        icon={Plug}
        title="No MCP servers"
        description="This agent does not declare any MCP servers."
        data-testid="agent-mcp-empty"
        fill={false}
        className="px-4 py-8"
      />
    ) : (
      <ul data-testid="agent-mcp-list">
        {mcpServers.map(server => {
          const transport = server.transport ?? "stdio";
          const commandOrUrl = server.url ?? server.command;
          const envKeys = envKeyNames(server);
          const secretKeys = secretKeyNames(server);
          return (
            <li
              key={server.name}
              className="grid grid-cols-[minmax(0,1fr)_auto] items-start gap-3 border-t border-line-soft px-4 py-3.5 first:border-t-0"
              data-testid={`agent-mcp-row-${server.name}`}
            >
              <div className="min-w-0">
                <p className="truncate font-mono text-small-body font-medium tracking-mono text-fg-strong">
                  {server.name}
                </p>
                {commandOrUrl ? (
                  <p className="mt-1 truncate font-mono text-badge tracking-mono text-muted">
                    {commandOrUrl}
                  </p>
                ) : null}
                {envKeys.length > 0 || secretKeys.length > 0 ? (
                  <div
                    className="mt-2 flex flex-wrap gap-1"
                    data-testid={`agent-mcp-keys-${server.name}`}
                  >
                    {envKeys.map(key => (
                      <Pill key={`env:${key}`} mono size="sm" tone="neutral">
                        {key}
                      </Pill>
                    ))}
                    {secretKeys.map(key => (
                      <Pill key={`secret:${key}`} mono size="sm" tone="warning">
                        {key}
                      </Pill>
                    ))}
                  </div>
                ) : null}
              </div>
              <Pill mono size="sm" tone="info" data-testid={`agent-mcp-transport-${server.name}`}>
                {transport}
              </Pill>
            </li>
          );
        })}
      </ul>
    );

  if (bare) {
    return (
      <div className={cn("min-w-0", className)} data-testid="agent-mcp-servers-panel">
        {list}
      </div>
    );
  }

  return (
    <Section
      label="MCP servers"
      className={cn("min-w-0", className)}
      data-testid="agent-mcp-servers-panel"
    >
      {list}
    </Section>
  );
}
