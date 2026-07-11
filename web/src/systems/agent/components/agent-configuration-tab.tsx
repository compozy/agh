import { Pill, Section } from "@agh/ui";

import { formatAbsentOverride } from "../lib/agent-absent-value";
import type { AgentPayload } from "../types";
import { AgentMcpServersPanel } from "./agent-mcp-servers-panel";

export interface AgentConfigurationTabProps {
  agent: AgentPayload;
  onEditSection: (section: "runtime" | "access" | "mcp") => void;
}

export function AgentConfigurationTab({ agent, onEditSection }: AgentConfigurationTabProps) {
  const tools = agent.tools ?? [];
  const denyTools = agent.deny_tools ?? [];
  const toolsets = agent.toolsets ?? [];

  return (
    <div className="flex flex-col gap-6" data-testid="agent-configuration-tab">
      <Section
        label="Runtime"
        data-testid="agent-config-runtime"
        right={
          <EditLink onClick={() => onEditSection("runtime")} testId="agent-config-edit-runtime" />
        }
      >
        <dl className="grid gap-3 sm:grid-cols-2">
          <ConfigField label="Provider" value={agent.provider} />
          <ConfigField
            label="Model"
            value={formatAbsentOverride(agent.model)}
            muted={!agent.model}
          />
          <ConfigField
            label="Command"
            value={formatAbsentOverride(agent.command)}
            muted={!agent.command}
            mono={Boolean(agent.command)}
          />
          <ConfigField
            label="Permissions"
            value={agent.permissions?.trim() || "Default"}
            muted={!agent.permissions?.trim()}
          />
        </dl>
      </Section>

      <Section
        label="Access"
        data-testid="agent-config-access"
        right={
          <EditLink onClick={() => onEditSection("access")} testId="agent-config-edit-access" />
        }
      >
        <TokenRow label="Tools" values={tools} />
        <TokenRow label="Deny tools" values={denyTools} />
        <TokenRow label="Toolsets" values={toolsets} />
      </Section>

      <Section
        label="MCP servers"
        data-testid="agent-config-mcp"
        right={<EditLink onClick={() => onEditSection("mcp")} testId="agent-config-edit-mcp" />}
      >
        <AgentMcpServersPanel agent={agent} bare />
      </Section>
    </div>
  );
}

function EditLink({ onClick, testId }: { onClick: () => void; testId: string }) {
  return (
    <button
      type="button"
      className="text-small-body text-muted hover:text-fg"
      onClick={onClick}
      data-testid={testId}
    >
      Edit ›
    </button>
  );
}

function ConfigField({
  label,
  value,
  muted,
  mono,
}: {
  label: string;
  value: string;
  muted?: boolean;
  mono?: boolean;
}) {
  return (
    <div>
      <dt className="eyebrow text-muted">{label}</dt>
      <dd
        className={[
          "mt-1 text-body",
          muted ? "text-muted" : "text-fg",
          mono ? "font-mono" : "",
        ].join(" ")}
      >
        {value}
      </dd>
    </div>
  );
}

function TokenRow({ label, values }: { label: string; values: readonly string[] }) {
  return (
    <div
      className="mb-4 last:mb-0"
      data-testid={`agent-config-${label.toLowerCase().replace(/\s+/g, "-")}`}
    >
      <p className="eyebrow mb-2 text-muted">{label}</p>
      {values.length === 0 ? (
        <p className="text-small-body text-muted">None</p>
      ) : (
        <div className="flex flex-wrap gap-1.5">
          {values.map(value => (
            <Pill key={value} mono size="sm" tone="neutral">
              {value}
            </Pill>
          ))}
        </div>
      )}
    </div>
  );
}
