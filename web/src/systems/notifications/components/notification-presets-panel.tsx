import { Bell, Plus, Trash2 } from "lucide-react";
import { useState } from "react";

import type { CreateNotificationPresetRequest, NotificationPresetEntry } from "../types";
import {
  Button,
  Empty,
  Input,
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

interface NotificationPresetsPanelProps {
  presets: NotificationPresetEntry[];
  isLoading: boolean;
  error: string | null;
  pendingName: string | null;
  canMutate: boolean;
  onCreate: (body: CreateNotificationPresetRequest) => void;
  onToggle: (preset: NotificationPresetEntry, nextEnabled: boolean) => void;
  onDelete: (preset: NotificationPresetEntry) => void;
}

export function NotificationPresetsPanel({
  presets,
  isLoading,
  error,
  pendingName,
  canMutate,
  onCreate,
  onToggle,
  onDelete,
}: NotificationPresetsPanelProps) {
  const [form, setForm] = useState({
    name: "",
    events: "task.run_*",
    target: "",
    filter: "",
    enabled: false,
  });
  const [localError, setLocalError] = useState<string | null>(null);

  const submit = () => {
    const eventList = form.events
      .split(",")
      .map(item => item.trim())
      .filter(Boolean);
    const targets = parseNotificationPresetTargetEntry(form.target);
    if (form.name.trim() === "" || eventList.length === 0) {
      setLocalError("Name and event are required.");
      return;
    }
    if (targets === null) {
      setLocalError("Target must use bridge_id:canonical_route.");
      return;
    }
    setLocalError(null);
    onCreate({
      name: form.name.trim(),
      events: eventList,
      targets,
      filter: form.filter.trim(),
      enabled: form.enabled,
    });
  };

  return (
    <Section
      data-testid="settings-page-hooks-extensions-notification-presets-section"
      label="Notification presets"
      note="SQLite-backed fanout policies"
      right={<Bell className="size-4 text-muted" />}
    >
      <div className="flex flex-col gap-3">
        <div className="grid gap-2 lg:grid-cols-[minmax(0,1fr)_minmax(0,1.2fr)_minmax(0,1.2fr)_minmax(0,1fr)_auto_auto]">
          <Input
            aria-label="Notification preset name"
            className="font-mono"
            data-testid="settings-page-hooks-extensions-notification-preset-name"
            placeholder="custom_alert"
            value={form.name}
            onChange={event => setForm(current => ({ ...current, name: event.target.value }))}
          />
          <Input
            aria-label="Notification preset events"
            className="font-mono"
            data-testid="settings-page-hooks-extensions-notification-preset-events"
            value={form.events}
            onChange={event => setForm(current => ({ ...current, events: event.target.value }))}
          />
          <Input
            aria-label="Notification preset target"
            className="font-mono"
            data-testid="settings-page-hooks-extensions-notification-preset-target"
            placeholder="bridge_slack_ops:channel:ops"
            value={form.target}
            onChange={event => setForm(current => ({ ...current, target: event.target.value }))}
          />
          <Input
            aria-label="Notification preset filter"
            className="font-mono"
            data-testid="settings-page-hooks-extensions-notification-preset-filter"
            placeholder="outcome >= warning"
            value={form.filter}
            onChange={event => setForm(current => ({ ...current, filter: event.target.value }))}
          />
          <label className="flex min-h-9 items-center gap-2 text-xs text-muted">
            <Switch
              aria-label="Create notification preset enabled"
              checked={form.enabled}
              disabled={!canMutate}
              onCheckedChange={next => setForm(current => ({ ...current, enabled: next }))}
              data-testid="settings-page-hooks-extensions-notification-preset-enabled"
            />
            enabled
          </label>
          <Button
            data-testid="settings-page-hooks-extensions-notification-preset-create"
            disabled={!canMutate || pendingName === form.name.trim()}
            onClick={submit}
            type="button"
            size="sm"
          >
            {pendingName === form.name.trim() ? (
              <Spinner className="size-3.5" />
            ) : (
              <Plus className="size-3.5" />
            )}
            Create
          </Button>
        </div>

        {localError || error ? (
          <span
            className="text-xs text-danger"
            data-testid="settings-page-hooks-extensions-notification-presets-error"
          >
            {localError ?? error}
          </span>
        ) : null}

        {isLoading && presets.length === 0 ? (
          <div
            className="flex items-center gap-2 text-xs text-subtle"
            data-testid="settings-page-hooks-extensions-notification-presets-loading"
          >
            <Spinner className="size-3" />
            Loading presets...
          </div>
        ) : presets.length === 0 ? (
          <Empty
            icon={Bell}
            title="No notification presets"
            description="SQLite has no preset rows for this runtime."
            data-testid="settings-page-hooks-extensions-notification-presets-empty"
          />
        ) : (
          <>
            <div className="grid gap-2 md:hidden">
              {presets.map(preset => (
                <NotificationPresetCard
                  key={preset.name}
                  preset={preset}
                  pending={pendingName === preset.name}
                  canMutate={canMutate}
                  onToggle={onToggle}
                  onDelete={onDelete}
                />
              ))}
            </div>
            <div
              className="hidden overflow-hidden rounded-lg border border-line md:block"
              data-testid="settings-page-hooks-extensions-notification-presets-table"
            >
              <Table className="table-fixed">
                <TableHeader>
                  <TableRow className="bg-elevated">
                    <TableHead className="eyebrow w-[22%] text-muted">Preset</TableHead>
                    <TableHead className="eyebrow w-[30%] text-muted">Events</TableHead>
                    <TableHead className="eyebrow w-[24%] text-muted">Targets</TableHead>
                    <TableHead className="eyebrow w-[14%] text-muted">Default</TableHead>
                    <TableHead className="eyebrow w-[10%] text-right text-muted">State</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {presets.map(preset => (
                    <NotificationPresetRow
                      key={preset.name}
                      preset={preset}
                      pending={pendingName === preset.name}
                      canMutate={canMutate}
                      onToggle={onToggle}
                      onDelete={onDelete}
                    />
                  ))}
                </TableBody>
              </Table>
            </div>
          </>
        )}
      </div>
    </Section>
  );
}

function NotificationPresetCard({
  preset,
  pending,
  canMutate,
  onToggle,
  onDelete,
}: {
  preset: NotificationPresetEntry;
  pending: boolean;
  canMutate: boolean;
  onToggle: (preset: NotificationPresetEntry, nextEnabled: boolean) => void;
  onDelete: (preset: NotificationPresetEntry) => void;
}) {
  return (
    <article
      className="flex flex-col gap-3 rounded-md border border-line bg-elevated px-3 py-2"
      data-testid={"settings-page-hooks-extensions-notification-preset-card-" + preset.name}
    >
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <span className="block truncate font-mono text-sm text-fg">{preset.name}</span>
          <div className="mt-1 flex flex-wrap gap-1">
            {preset.built_in ? <Pill mono>built-in</Pill> : <Pill mono>custom</Pill>}
            {preset.user_modified ? (
              <Pill tone="warning" mono>
                modified
              </Pill>
            ) : null}
          </div>
        </div>
        <div className="flex shrink-0 items-center gap-2">
          {pending ? <Spinner className="size-3.5 text-muted" /> : null}
          <Switch
            aria-label={"Toggle " + preset.name}
            checked={preset.enabled}
            disabled={!canMutate || pending}
            onCheckedChange={next => onToggle(preset, next)}
            data-testid={
              "settings-page-hooks-extensions-notification-preset-card-" + preset.name + "-toggle"
            }
          />
          <Button
            aria-label={"Delete " + preset.name}
            data-testid={
              "settings-page-hooks-extensions-notification-preset-card-" + preset.name + "-delete"
            }
            disabled={!canMutate || pending || preset.built_in}
            onClick={() => onDelete(preset)}
            size="icon"
            type="button"
            variant="ghost"
          >
            <Trash2 className="size-3.5" />
          </Button>
        </div>
      </div>
      <dl className="grid gap-2 text-xs">
        <div>
          <dt className="eyebrow text-muted">Events</dt>
          <dd className="break-words font-mono text-subtle">{preset.events.join(", ")}</dd>
        </div>
        <div>
          <dt className="eyebrow text-muted">Targets</dt>
          <dd className="break-words font-mono text-subtle">
            {notificationPresetTargetLabel(preset.targets)}
          </dd>
        </div>
        <div>
          <dt className="eyebrow text-muted">Default</dt>
          <dd className="flex flex-wrap items-center gap-1 font-mono text-subtle">
            <span>{preset.default_version ?? "custom"}</span>
            {preset.default_update_available ? (
              <Pill tone="warning" mono>
                default drift
              </Pill>
            ) : null}
          </dd>
        </div>
      </dl>
    </article>
  );
}

function NotificationPresetRow({
  preset,
  pending,
  canMutate,
  onToggle,
  onDelete,
}: {
  preset: NotificationPresetEntry;
  pending: boolean;
  canMutate: boolean;
  onToggle: (preset: NotificationPresetEntry, nextEnabled: boolean) => void;
  onDelete: (preset: NotificationPresetEntry) => void;
}) {
  return (
    <TableRow data-testid={"settings-page-hooks-extensions-notification-preset-row-" + preset.name}>
      <TableCell>
        <div className="flex min-w-0 flex-col gap-1">
          <span className="truncate font-mono text-sm text-fg">{preset.name}</span>
          <div className="flex flex-wrap gap-1">
            {preset.built_in ? <Pill mono>built-in</Pill> : <Pill mono>custom</Pill>}
            {preset.user_modified ? (
              <Pill tone="warning" mono>
                modified
              </Pill>
            ) : null}
          </div>
        </div>
      </TableCell>
      <TableCell className="whitespace-normal">
        <span className="break-words font-mono text-xs text-subtle">
          {preset.events.join(", ")}
        </span>
      </TableCell>
      <TableCell className="whitespace-normal">
        <span className="break-words font-mono text-xs text-subtle">
          {notificationPresetTargetLabel(preset.targets)}
        </span>
      </TableCell>
      <TableCell className="whitespace-normal">
        <div className="flex flex-col gap-1">
          <span className="font-mono text-xs text-subtle">
            {preset.default_version ?? "custom"}
          </span>
          {preset.default_update_available ? (
            <Pill tone="warning" mono>
              default drift
            </Pill>
          ) : null}
        </div>
      </TableCell>
      <TableCell>
        <div className="flex items-center justify-end gap-2">
          {pending ? <Spinner className="size-3.5 text-muted" /> : null}
          <Switch
            aria-label={"Toggle " + preset.name}
            checked={preset.enabled}
            disabled={!canMutate || pending}
            onCheckedChange={next => onToggle(preset, next)}
            data-testid={
              "settings-page-hooks-extensions-notification-preset-row-" + preset.name + "-toggle"
            }
          />
          <Button
            aria-label={"Delete " + preset.name}
            data-testid={
              "settings-page-hooks-extensions-notification-preset-row-" + preset.name + "-delete"
            }
            disabled={!canMutate || pending || preset.built_in}
            onClick={() => onDelete(preset)}
            size="icon"
            type="button"
            variant="ghost"
          >
            <Trash2 className="size-3.5" />
          </Button>
        </div>
      </TableCell>
    </TableRow>
  );
}

function parseNotificationPresetTargetEntry(
  value: string
): CreateNotificationPresetRequest["targets"] | null {
  const trimmed = value.trim();
  if (trimmed === "") return [];
  const split = trimmed.indexOf(":");
  if (split <= 0 || split === trimmed.length - 1) return null;
  return [
    {
      bridge_id: trimmed.slice(0, split).trim(),
      canonical_route: trimmed.slice(split + 1).trim(),
      delivery_mode: "direct-send",
    },
  ];
}

function notificationPresetTargetLabel(targets: NotificationPresetEntry["targets"]) {
  if (targets.length === 0) return "none";
  return targets
    .map(target => {
      const route = target.canonical_route ?? target.display_name ?? "unresolved";
      return target.bridge_id + ":" + route;
    })
    .join(", ");
}
