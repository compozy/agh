import { Search } from "lucide-react";
import { useState } from "react";

import {
  BlockLoading,
  CodeBlock,
  LaneTabs,
  MetadataTile,
  MonoId,
  Pill,
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
  StatusCard,
  TabsContent,
  Time,
  type LaneTabsItem,
  type PillTone,
} from "@agh/ui";

import { taskRunStatusLabel } from "../lib/task-formatters";
import type {
  TaskBridgeNotificationSubscription,
  TaskBridgeNotificationSubscriptionCreateRequest,
  TaskDetailView,
  TaskInspectView,
} from "../types";
import { TaskBridgeSubscriptionsPane } from "./task-bridge-subscriptions-pane";
import { TaskRawPane } from "./task-raw-pane";

export type TaskInspectDrawerTab = "diagnostics" | "stream" | "bridges" | "raw";

export interface TaskInspectDrawerProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  detail: TaskDetailView;
  inspect: TaskInspectView | null;
  inspectLoading?: boolean;
  inspectErrorMessage?: string | null;
  stream: {
    state: "idle" | "connected" | "error" | "disabled";
    errorMessage: string | null;
    seedSequence: number;
    latestEventSeq: number | null;
  };
  bridges: {
    subscriptions: TaskBridgeNotificationSubscription[];
    isLoading?: boolean;
    errorMessage?: string | null;
    isCreatePending?: boolean;
    isDeletePending?: boolean;
    onCreate: (request: TaskBridgeNotificationSubscriptionCreateRequest) => Promise<void> | void;
    onDelete: (subscriptionId: string) => Promise<void> | void;
  };
}

const TABS: ReadonlyArray<LaneTabsItem<TaskInspectDrawerTab>> = [
  { value: "diagnostics", label: "Diagnostics", testId: "tasks-inspect-tab-diagnostics" },
  { value: "stream", label: "Stream", testId: "tasks-inspect-tab-stream" },
  { value: "bridges", label: "Bridges", testId: "tasks-inspect-tab-bridges" },
  { value: "raw", label: "Raw", testId: "tasks-inspect-tab-raw" },
];

/** §7.3 next-action vocabulary — plain language first, enum as microtext. */
const NEXT_ACTION_COPY: Record<string, string> = {
  claim_available: "A worker can pick this up now",
  waiting_for_session: "Waiting for the agent session to attach",
  stranded: "Run lost its worker. Retry or release it",
  recovery_required: "Stuck run needs recovery. Use Recover to requeue",
  running: "Running normally",
  terminal: "Nothing to do — the task is terminal",
};

const SEVERITY_TONE: Record<string, PillTone> = {
  ok: "success",
  info: "info",
  warn: "warning",
  error: "danger",
  critical: "danger",
};

function DiagnosticsPane({
  taskId,
  inspect,
  isLoading,
  errorMessage,
}: {
  taskId: string;
  inspect: TaskInspectView | null;
  isLoading: boolean;
  errorMessage: string | null;
}) {
  if (isLoading && !inspect) {
    return <BlockLoading label="Loading inspect snapshot" size="sm" surface="bare" />;
  }
  if (!inspect) {
    return (
      <p className="text-small-body text-muted" data-testid="tasks-inspect-diagnostics-empty">
        {errorMessage ?? "No inspect snapshot is available for this task state."}
      </p>
    );
  }

  const run = inspect.current_run ?? null;
  const session = inspect.bound_session ?? null;
  const diagnostics = inspect.diagnostics ?? [];
  const nextAction = inspect.next_action ?? null;

  return (
    <div className="flex flex-col gap-4" data-testid="tasks-inspect-diagnostics">
      <div className="grid grid-cols-2 gap-2.5">
        <MetadataTile
          className="border border-line-soft bg-input-fill"
          label="Next action"
          value={
            nextAction ? (NEXT_ACTION_COPY[nextAction] ?? nextAction.replaceAll("_", " ")) : "—"
          }
          detail={nextAction ?? undefined}
        />
        <MetadataTile
          className="border border-line-soft bg-input-fill"
          label="Current run"
          value={
            run ? `Attempt ${run.attempt} · ${taskRunStatusLabel(run.status)}` : "No current run"
          }
          detail={run ? run.run_id : undefined}
        />
        <MetadataTile
          className="border border-line-soft bg-input-fill"
          label="Session"
          value={session ? (session.state ?? "bound") : "No bound session"}
          detail={session ? session.session_id : undefined}
        />
        <MetadataTile
          className="border border-line-soft bg-input-fill"
          label="Scheduler"
          value={inspect.scheduler.paused ? "Paused" : "Active"}
        />
      </div>

      {diagnostics.length === 0 ? (
        <p className="rounded-md border border-line-soft bg-canvas-soft px-3.5 py-3 text-small-body leading-relaxed text-muted">
          No findings. The run is claiming, heartbeating, and reporting on schedule.
        </p>
      ) : (
        diagnostics.map(diagnostic => (
          <StatusCard
            className="border border-line-soft"
            data-testid={`tasks-inspect-diagnostic-${diagnostic.code}`}
            key={diagnostic.id}
            tone={SEVERITY_TONE[diagnostic.severity ?? ""] ?? "neutral"}
          >
            <StatusCard.Header label={diagnostic.title}>
              <MonoId className="ml-auto" size="sm" value={diagnostic.code} />
            </StatusCard.Header>
            <StatusCard.Body>{diagnostic.message}</StatusCard.Body>
            {diagnostic.suggested_command ? (
              <CodeBlock code={diagnostic.suggested_command} language="bash" />
            ) : null}
          </StatusCard>
        ))
      )}

      <CodeBlock
        code={`# same view from the CLI\nagh task inspect ${taskId} -o json`}
        language="bash"
      />
      <p className="text-form-hint text-subtle">
        Snapshot as of <Time iso={inspect.as_of} mode="relative" />.
      </p>
    </div>
  );
}

function StreamPane({ stream }: { stream: TaskInspectDrawerProps["stream"] }) {
  const connection =
    stream.state === "connected" ? "Connected" : stream.state === "error" ? "Error" : "Idle";
  const connectionTone: PillTone =
    stream.state === "connected" ? "success" : stream.state === "error" ? "danger" : "neutral";

  return (
    <div className="flex flex-col gap-4" data-testid="tasks-inspect-stream">
      <div className="grid grid-cols-2 gap-2.5">
        <MetadataTile
          className="border border-line-soft bg-input-fill"
          label="Connection"
          value={
            <span className="inline-flex items-center gap-1.5">
              <Pill.Dot pulse={stream.state === "connected"} tone={connectionTone} />
              {connection}
            </span>
          }
        />
        <MetadataTile
          className="border border-line-soft bg-input-fill"
          label="Latest event seq"
          value={stream.latestEventSeq ?? "—"}
        />
        <MetadataTile
          className="border border-line-soft bg-input-fill"
          label="Resume seed"
          value={stream.seedSequence}
        />
      </div>
      {stream.errorMessage ? (
        <p className="text-small-body text-danger">{stream.errorMessage}</p>
      ) : null}
      <p className="rounded-md border border-line-soft bg-canvas-soft px-3.5 py-3 text-small-body leading-relaxed text-muted">
        The task stream replays durable events for this task subtree, then tails new ones. The page
        refetches on every frame.
      </p>
    </div>
  );
}

/**
 * Operator drawer (§4.8): Diagnostics · Stream · Bridges · Raw. Everything the
 * old Orchestration tab exposed stays reachable here in ≤2 clicks; none of it
 * is the default view anymore.
 */
export function TaskInspectDrawer({
  open,
  onOpenChange,
  detail,
  inspect,
  inspectLoading = false,
  inspectErrorMessage = null,
  stream,
  bridges,
}: TaskInspectDrawerProps) {
  const [tab, setTab] = useState<TaskInspectDrawerTab>("diagnostics");

  return (
    <Sheet onOpenChange={onOpenChange} open={open}>
      <SheetContent
        className="w-[min(600px,calc(100vw-24px))] sm:max-w-none"
        data-testid="tasks-inspect-drawer"
        side="right"
      >
        <SheetHeader>
          <div className="flex items-start gap-3">
            <span
              aria-hidden="true"
              className="grid size-9 shrink-0 place-items-center rounded-md bg-accent-tint text-accent-strong"
            >
              <Search className="size-4" />
            </span>
            <div className="min-w-0">
              <span className="eyebrow font-mono text-subtle">Operator</span>
              <SheetTitle>Inspect</SheetTitle>
              <SheetDescription>
                Runtime internals for this task. Everything here maps to{" "}
                <span className="font-mono text-eyebrow">agh task inspect</span>.
              </SheetDescription>
            </div>
          </div>
        </SheetHeader>
        <LaneTabs<TaskInspectDrawerTab>
          ariaLabel="Inspect views"
          className="flex min-h-0 flex-1 flex-col gap-0 px-4"
          items={TABS}
          listClassName="w-full"
          onChange={setTab}
          value={tab}
        >
          <div className="min-h-0 flex-1 overflow-y-auto py-4">
            <TabsContent value="diagnostics">
              <DiagnosticsPane
                errorMessage={inspectErrorMessage}
                inspect={inspect}
                isLoading={inspectLoading}
                taskId={detail.task.id}
              />
            </TabsContent>
            <TabsContent value="stream">
              <StreamPane stream={stream} />
            </TabsContent>
            <TabsContent value="bridges">
              <TaskBridgeSubscriptionsPane
                errorMessage={bridges.errorMessage}
                isCreatePending={bridges.isCreatePending}
                isDeletePending={bridges.isDeletePending}
                isLoading={bridges.isLoading}
                onCreate={bridges.onCreate}
                onDelete={bridges.onDelete}
                subscriptions={bridges.subscriptions}
              />
            </TabsContent>
            <TabsContent value="raw">
              <TaskRawPane detail={detail} />
            </TabsContent>
          </div>
        </LaneTabs>
      </SheetContent>
    </Sheet>
  );
}
