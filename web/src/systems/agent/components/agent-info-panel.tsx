import {
  ScrollArea,
  Section,
  Empty,
  Pill,
  cn,
  Item,
  ItemActions,
  ItemContent,
  ItemDescription,
  ItemGroup,
  ItemTitle,
} from "@agh/ui";
import { Plug } from "lucide-react";

import type { AgentPayload } from "../types";

export interface AgentInfoPanelProps {
  agent: AgentPayload;
  className?: string;
}

export function AgentInfoPanel({ agent, className }: AgentInfoPanelProps) {
  const mcpServers = agent.mcp_servers ?? [];
  return (
    <aside
      data-testid="agent-info-panel"
      aria-label={`${agent.name} agent details`}
      style={{ width: "var(--rail-inspector-w, 320px)" }}
      className={cn(
        "hidden shrink-0 flex-col overflow-hidden border-l border-(--line) bg-(--canvas) xl:flex",
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
              <ItemGroup className="gap-2" data-testid="agent-info-mcp-list">
                {mcpServers.map(server => {
                  const transport = server.transport ?? "stdio";
                  return (
                    <Item
                      key={server.name}
                      role="listitem"
                      variant="outline"
                      size="sm"
                      data-testid={`agent-info-mcp-row-${server.name}`}
                    >
                      <ItemContent>
                        <ItemTitle>{server.name}</ItemTitle>
                        {server.command || server.url ? (
                          <ItemDescription className="truncate font-mono text-badge tracking-mono text-(--subtle)">
                            {server.url ?? server.command}
                          </ItemDescription>
                        ) : null}
                      </ItemContent>
                      <ItemActions>
                        <Pill mono tone="info" data-testid={`agent-info-mcp-kind-${server.name}`}>
                          {transport}
                        </Pill>
                      </ItemActions>
                    </Item>
                  );
                })}
              </ItemGroup>
            )}
          </Section>
        </div>
      </ScrollArea>
    </aside>
  );
}
