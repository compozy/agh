import {
  Button,
  Pill,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@agh/ui";

import type { SettingsMCPServerEntry } from "../types";

import { SettingsSourceBadge } from "./settings-source-badge";

export function MCPServersTable({
  servers,
  onEdit,
  onDelete,
}: {
  servers: SettingsMCPServerEntry[];
  onEdit: (entry: SettingsMCPServerEntry) => void;
  onDelete: (entry: SettingsMCPServerEntry) => void;
}) {
  return (
    <div
      className="overflow-hidden rounded-lg border border-line"
      data-testid="settings-page-mcp-servers-list"
    >
      <Table>
        <TableHeader>
          <TableRow className="bg-elevated">
            <TableHead className="eyebrow text-muted">Name</TableHead>
            <TableHead className="eyebrow text-muted">Endpoint</TableHead>
            <TableHead className="eyebrow text-muted">Source</TableHead>
            <TableHead className="eyebrow text-right text-muted">Env</TableHead>
            <TableHead className="eyebrow text-right text-muted">Args</TableHead>
            <TableHead className="eyebrow w-[1%] text-right text-muted">Actions</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {servers.map(server => (
            <MCPServerRow
              key={`${server.name}-${server.source_metadata.effective_source.kind}`}
              server={server}
              onEdit={onEdit}
              onDelete={onDelete}
            />
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

function MCPServerRow({
  server,
  onEdit,
  onDelete,
}: {
  server: SettingsMCPServerEntry;
  onEdit: (entry: SettingsMCPServerEntry) => void;
  onDelete: (entry: SettingsMCPServerEntry) => void;
}) {
  const source = server.source_metadata.effective_source;
  const shadowed = server.source_metadata.shadowed_sources ?? [];
  const envCount = server.env ? Object.keys(server.env).length : 0;
  const argsCount = server.args?.length ?? 0;
  const endpoint = server.transport === "stdio" ? server.command : server.url;
  const canEdit = server.transport === "stdio";

  return (
    <TableRow data-testid={`settings-page-mcp-servers-row-${server.name}`}>
      <TableCell>
        <div className="flex min-w-0 items-center gap-2.5">
          <Pill.Dot
            tone="success"
            size="md"
            data-testid={`settings-page-mcp-servers-row-${server.name}-status`}
            data-tone="configured"
          />
          <span className="font-mono text-sm text-fg">{server.name}</span>
        </div>
      </TableCell>
      <TableCell
        className="font-mono text-xs text-muted"
        data-testid={`settings-page-mcp-servers-row-${server.name}-command`}
      >
        {endpoint ?? "-"}
      </TableCell>
      <TableCell>
        <SettingsSourceBadge
          data-testid={`settings-page-mcp-servers-row-${server.name}-source`}
          source={source}
          shadowed={shadowed}
        />
      </TableCell>
      <TableCell
        className="text-right font-mono text-xs text-muted"
        data-testid={`settings-page-mcp-servers-row-${server.name}-env`}
      >
        {envCount}
      </TableCell>
      <TableCell
        className="text-right font-mono text-xs text-muted"
        data-testid={`settings-page-mcp-servers-row-${server.name}-args`}
      >
        {argsCount}
      </TableCell>
      <TableCell>
        <div className="flex items-center justify-end gap-2">
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={() => onEdit(server)}
            disabled={!canEdit}
            title={canEdit ? undefined : "Remote MCP servers are managed from config and CLI auth."}
            data-testid={`settings-page-mcp-servers-row-${server.name}-edit`}
          >
            Edit
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={() => onDelete(server)}
            data-testid={`settings-page-mcp-servers-row-${server.name}-delete`}
          >
            Delete
          </Button>
        </div>
      </TableCell>
    </TableRow>
  );
}
