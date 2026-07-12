import { AlertTriangle, Webhook } from "lucide-react";

import type { SettingsHookEntry, SettingsHooksExtensionsTransportParity } from "@/systems/settings";
import {
  Alert,
  AlertDescription,
  Empty,
  Pill,
  Section,
  Spinner,
  Switch,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@agh/ui";

export function TransportParityBanner({
  parity,
}: {
  parity: SettingsHooksExtensionsTransportParity | null;
}) {
  if (!parity || !parity.known) return null;
  if (parity.extensions_http && parity.settings_http) return null;
  const unavailable = describeUnavailableHttpOperations(parity);

  return (
    <Alert
      variant="warning"
      role="status"
      data-testid="settings-page-hooks-extensions-transport-parity"
    >
      <AlertTriangle className="size-3" />
      <AlertDescription className="text-xs">
        <span className="font-medium text-warning">Some operations are unavailable over HTTP.</span>{" "}
        HTTP is bound outside the loopback host. {unavailable} stay available over UDS but return
        403 on HTTP. Use the CLI or rebind to loopback to edit from the web app.
      </AlertDescription>
    </Alert>
  );
}

function describeUnavailableHttpOperations(parity: SettingsHooksExtensionsTransportParity): string {
  const operations: string[] = [];
  if (parity.settings_http === false) {
    operations.push("Hook toggles and policy edits");
  }
  if (parity.extensions_http === false) {
    operations.push("Extension enable/disable");
  }

  if (operations.length === 0) return "These operations";
  if (operations.length === 1) return operations[0];
  return `${operations.slice(0, -1).join(", ")} and ${operations.at(-1)}`;
}

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
    <Section
      data-testid="settings-page-hooks-extensions-hooks-section"
      label="Lifecycle hooks"
      note="restart required to re-read declarations · toggles persist now"
    >
      {hookError ? (
        <span
          className="text-xs text-danger"
          data-testid="settings-page-hooks-extensions-hooks-error"
        >
          {hookError}
        </span>
      ) : null}
      {hooks.length === 0 ? (
        <Empty
          icon={Webhook}
          title="No hooks registered"
          description="Add a hook declaration to ~/.agh/config.toml or a workspace overlay to register one."
          data-testid="settings-page-hooks-extensions-hooks-empty"
        />
      ) : (
        <div
          className="overflow-hidden rounded-lg border border-line"
          data-testid="settings-page-hooks-extensions-hooks-list"
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
    </Section>
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
  const enabled = declaration.required !== false;
  const matcherSummary = summarizeMatcher(declaration.matcher);
  const mode = declaration.mode === "sync" ? "blocking" : (declaration.mode ?? "async");

  return (
    <TableRow data-testid={`settings-page-hooks-extensions-hooks-row-${entry.name}`}>
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
        data-testid={`settings-page-hooks-extensions-hooks-row-${entry.name}-matcher`}
      >
        {matcherSummary || "--"}
      </TableCell>
      <TableCell>
        <div className="flex items-center justify-end gap-2">
          {pending ? <Spinner className="size-3 text-subtle" /> : null}
          <Switch
            data-testid={`settings-page-hooks-extensions-hooks-row-${entry.name}-toggle`}
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
