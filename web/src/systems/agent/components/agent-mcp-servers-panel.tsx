import { Plug } from "lucide-react";

import {
  Empty,
  Eyebrow,
  Item,
  ItemActions,
  ItemContent,
  ItemDescription,
  ItemGroup,
  ItemTitle,
  Pill,
  Section,
  cn,
} from "@agh/ui";

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
      />
    ) : (
      <ItemGroup className="gap-2" data-testid="agent-mcp-list">
        {mcpServers.map(server => {
          const transport = server.transport ?? "stdio";
          const commandOrUrl = server.url ?? server.command;
          const envKeys = envKeyNames(server);
          const secretKeys = secretKeyNames(server);
          return (
            <Item
              key={server.name}
              role="listitem"
              variant="outline"
              size="sm"
              data-testid={`agent-mcp-row-${server.name}`}
            >
              <ItemContent>
                <ItemTitle>{server.name}</ItemTitle>
                {commandOrUrl ? (
                  <ItemDescription className="truncate font-mono text-badge tracking-mono text-muted">
                    {commandOrUrl}
                  </ItemDescription>
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
              </ItemContent>
              <ItemActions>
                <Eyebrow data-testid={`agent-mcp-transport-${server.name}`} className="text-muted">
                  {transport.toUpperCase()}
                </Eyebrow>
              </ItemActions>
            </Item>
          );
        })}
      </ItemGroup>
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
