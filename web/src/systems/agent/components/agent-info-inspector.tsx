import { useCallback, useState } from "react";
import { Plug } from "lucide-react";

import {
  DetailInspector,
  Empty,
  Eyebrow,
  Item,
  ItemActions,
  ItemContent,
  ItemDescription,
  ItemGroup,
  ItemTitle,
  Section,
  cn,
} from "@agh/ui";

import type { AgentPayload } from "../types";

const AGENT_INFO_INSPECTOR_TABS = [{ id: "mcp", label: "MCP" }] as const;

export interface AgentInfoInspectorProps {
  agent: AgentPayload;
  className?: string;
  drawerOpen?: boolean;
  onDrawerOpenChange?: (open: boolean) => void;
}

/**
 * Agent-detail right rail consuming the `<DetailInspector>` primitive per
 * Renders one tab (MCP) listing the agent's declared MCP servers
 * with transport rendered via `<Eyebrow>`
 */
export function AgentInfoInspector({
  agent,
  className,
  drawerOpen,
  onDrawerOpenChange,
}: AgentInfoInspectorProps) {
  const [activeTab, setActiveTab] = useState<string>("mcp");
  const handleTabChange = useCallback((id: string) => setActiveTab(id), []);
  const mcpServers = agent.mcp_servers ?? [];

  return (
    <DetailInspector
      data-testid="agent-info-inspector"
      aria-label={`${agent.name} agent details`}
      tabs={AGENT_INFO_INSPECTOR_TABS}
      activeTab={activeTab}
      onTabChange={handleTabChange}
      open={drawerOpen}
      onOpenChange={onDrawerOpenChange}
      className={cn("min-w-0", className)}
    >
      <div className="flex min-h-full flex-col gap-6 px-4 py-5">
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
                      <Eyebrow
                        data-testid={`agent-info-mcp-kind-${server.name}`}
                        className="text-(--muted)"
                      >
                        {transport.toUpperCase()}
                      </Eyebrow>
                    </ItemActions>
                  </Item>
                );
              })}
            </ItemGroup>
          )}
        </Section>
      </div>
    </DetailInspector>
  );
}
