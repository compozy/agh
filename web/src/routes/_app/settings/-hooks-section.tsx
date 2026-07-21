import { Webhook } from "lucide-react";

import { SettingsGroup, type SettingsHookEntry } from "@/systems/settings";
import {
  Empty,
  Pill,
  Spinner,
  Switch,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@agh/ui";

interface HooksSectionProps {
  hooks: SettingsHookEntry[];
  pendingHookName: string | null;
  hookError: string | null;
  canMutate: boolean;
  onToggle: (entry: SettingsHookEntry, nextEnabled: boolean) => void;
}

export function HooksSection({
  hooks,
  pendingHookName,
  hookError,
  canMutate,
  onToggle,
}: HooksSectionProps) {
  return (
    <SettingsGroup
      data-testid="settings-page-hooks-section"
      title="Lifecycle hooks"
      description="Restart the daemon to re-read hook declarations. Enablement changes persist immediately."
    >
      {hookError ? (
        <span className="text-xs text-danger" data-testid="settings-page-hooks-error-message">
          {hookError}
        </span>
      ) : null}
      {hooks.length === 0 ? (
        <Empty
          icon={Webhook}
          title="No hooks registered"
          description="Add a hook declaration to ~/.agh/config.toml or a workspace overlay to register one."
          data-testid="settings-page-hooks-empty"
        />
      ) : (
        <div
          className="overflow-hidden rounded-lg border border-line"
          data-testid="settings-page-hooks-list"
        >
          <Table>
            <TableHeader>
              <TableRow className="bg-elevated">
                <TableHead className="eyebrow text-muted">Name</TableHead>
                <TableHead className="eyebrow text-muted">Event</TableHead>
                <TableHead className="eyebrow text-muted">Mode</TableHead>
                <TableHead className="eyebrow text-muted">Matcher</TableHead>
                <TableHead className="eyebrow w-[1%] text-right text-muted">Enabled</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {hooks.map(entry => (
                <HookRow
                  key={entry.name}
                  entry={entry}
                  pending={pendingHookName === entry.name}
                  canMutate={canMutate}
                  onToggle={onToggle}
                />
              ))}
            </TableBody>
          </Table>
        </div>
      )}
    </SettingsGroup>
  );
}

function HookRow({
  entry,
  pending,
  canMutate,
  onToggle,
}: {
  entry: SettingsHookEntry;
  pending: boolean;
  canMutate: boolean;
  onToggle: (entry: SettingsHookEntry, nextEnabled: boolean) => void;
}) {
  const declaration = entry.declaration;
  const enabled = declaration.enabled !== false;
  const matcherSummary = summarizeMatcher(declaration.matcher);
  const mode = declaration.mode === "sync" ? "blocking" : (declaration.mode ?? "async");

  return (
    <TableRow data-testid={`settings-page-hooks-row-${entry.name}`}>
      <TableCell>
        <div className="flex flex-col gap-0.5">
          <span className="font-mono text-sm text-fg">{entry.name}</span>
          {declaration.command ? (
            <span className="font-mono text-badge text-subtle">
              {[declaration.command, ...(declaration.args ?? [])].join(" ")}
            </span>
          ) : null}
        </div>
      </TableCell>
      <TableCell>
        <Pill mono tone="info">
          {declaration.event}
        </Pill>
      </TableCell>
      <TableCell className="font-mono text-xs text-muted">{mode}</TableCell>
      <TableCell
        className="font-mono text-xs text-muted"
        data-testid={`settings-page-hooks-row-${entry.name}-matcher`}
      >
        {matcherSummary || "--"}
      </TableCell>
      <TableCell>
        <div className="flex items-center justify-end gap-2">
          {pending ? <Spinner className="size-3 text-subtle" /> : null}
          <Switch
            data-testid={`settings-page-hooks-row-${entry.name}-toggle`}
            checked={enabled}
            disabled={pending || !canMutate}
            onCheckedChange={checked => onToggle(entry, checked)}
            aria-label={`Toggle hook ${entry.name}`}
          />
        </div>
      </TableCell>
    </TableRow>
  );
}

function summarizeMatcher(matcher: SettingsHookEntry["declaration"]["matcher"]): string {
  const entries = Object.entries(matcher).filter(
    ([, value]) => value !== undefined && value !== null && value !== ""
  );
  if (entries.length === 0) return "";
  return entries.map(([key, value]) => `${key}=${String(value)}`).join(" · ");
}
