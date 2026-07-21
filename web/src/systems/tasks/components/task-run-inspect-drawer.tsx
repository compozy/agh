import { Search } from "lucide-react";

import {
  MetadataTile,
  Pill,
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
  Time,
} from "@agh/ui";

import type { TaskRunDetailView, TaskRunInspectView } from "../types";

export interface TaskRunInspectDrawerProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  run: TaskRunDetailView;
  inspect: TaskRunInspectView | null;
}

/**
 * Run operator drawer (§4.9): claim and lease internals — heartbeat, lease
 * window, truncated claim-token hash, idempotency key. Raw claim tokens do
 * not exist in any DTO and are never rendered.
 */
export function TaskRunInspectDrawer({
  open,
  onOpenChange,
  run,
  inspect,
}: TaskRunInspectDrawerProps) {
  const record = run.run;
  const inspectRun = inspect?.current_run ?? null;
  const heartbeatAt = inspectRun?.heartbeat_at ?? record.heartbeat_at ?? null;
  const leaseUntil = inspectRun?.lease_until ?? record.lease_until ?? null;
  const claimHash = inspectRun?.claim_token_hash_truncated ?? record.claim_token_hash ?? null;
  const idempotencyKey = record.idempotency_key ?? null;

  return (
    <Sheet onOpenChange={onOpenChange} open={open}>
      <SheetContent
        className="w-[min(560px,calc(100vw-24px))] sm:max-w-none"
        data-testid="tasks-run-inspect-drawer"
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
              <SheetTitle>Inspect run</SheetTitle>
              <SheetDescription>
                Claim and lease internals for{" "}
                <span className="font-mono text-eyebrow">{record.id}</span>.
              </SheetDescription>
            </div>
          </div>
        </SheetHeader>
        <div className="min-h-0 flex-1 overflow-y-auto px-4 py-4">
          <div className="grid grid-cols-2 gap-2.5">
            <MetadataTile
              className="border border-line-soft bg-input-fill"
              label="Heartbeat"
              value={
                heartbeatAt ? (
                  <span className="inline-flex items-center gap-1.5">
                    <Pill.Dot tone="success" />
                    <Time iso={heartbeatAt} mode="relative" />
                  </span>
                ) : (
                  "—"
                )
              }
            />
            <MetadataTile
              className="border border-line-soft bg-input-fill"
              label="Lease until"
              value={leaseUntil ? <Time iso={leaseUntil} mode="absolute" /> : "—"}
            />
            <MetadataTile
              className="border border-line-soft bg-input-fill"
              label="Claim token"
              value={claimHash ? `sha256 · ${claimHash}` : "—"}
            />
            <MetadataTile
              className="border border-line-soft bg-input-fill"
              label="Idempotency key"
              value={idempotencyKey ?? "—"}
            />
          </div>
          <p className="mt-4 rounded-md border border-line-soft bg-canvas-soft px-3.5 py-3 text-small-body leading-relaxed text-muted">
            The claim lease renews on every heartbeat. If heartbeats stop, the scheduler escalates
            this run for recovery.
          </p>
        </div>
      </SheetContent>
    </Sheet>
  );
}
