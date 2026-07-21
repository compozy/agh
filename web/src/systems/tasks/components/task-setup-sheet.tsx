import { Lock, Settings2 } from "lucide-react";
import { useState, type ReactNode } from "react";
import { toast } from "sonner";

import {
  Button,
  JsonViewer,
  PillGroup,
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
  Textarea,
  type PillGroupItem,
} from "@agh/ui";

import { useProfileEditor } from "../hooks/use-profile-editor";
import type { TaskExecutionProfile, TaskExecutionProfileSetRequest } from "../types";

export interface TaskSetupSheetProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  taskId: string;
  profile: TaskExecutionProfile | null;
  profileLoading?: boolean;
  profileErrorMessage?: string | null;
  hasActiveRun: boolean;
  isSetPending?: boolean;
  isDeletePending?: boolean;
  onSetProfile: (data: TaskExecutionProfileSetRequest) => Promise<void>;
  onDeleteProfile: () => Promise<void>;
}

function SetupGroup({ label, children }: { label: string; children: ReactNode }) {
  return (
    <section className="mb-5">
      <h3 className="eyebrow mb-2.5 text-subtle">{label}</h3>
      <div className="flex flex-col">{children}</div>
    </section>
  );
}

function SetupRow({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="grid grid-cols-[128px_minmax(0,1fr)] items-center gap-3 py-1.5">
      <span className="text-form-label text-muted">{label}</span>
      <div className="flex min-w-0 flex-wrap items-center gap-1.5 text-small-body text-fg">
        {children}
      </div>
    </div>
  );
}

function Chip({ children }: { children: ReactNode }) {
  return (
    <span className="inline-flex h-5 items-center rounded-xs bg-badge-fill px-2 font-mono text-eyebrow text-muted">
      {children}
    </span>
  );
}

const WORKER_MODE_ITEMS: ReadonlyArray<PillGroupItem<"inherit" | "select">> = [
  { value: "inherit", label: "Inherit", disabled: true },
  { value: "select", label: "Select", disabled: true },
];

function ProfileReadView({ profile }: { profile: TaskExecutionProfile }) {
  const worker = profile.worker;
  const coordinator = profile.coordinator;
  const sandbox = profile.sandbox;
  const review = profile.review;
  const participants = profile.participants;
  const reviewConfigured = Boolean(
    review.agent_name ||
    (review.allowed_agent_names?.length ?? 0) > 0 ||
    (review.preferred_agent_names?.length ?? 0) > 0
  );

  return (
    <>
      <SetupGroup label="Worker">
        <SetupRow label="Mode">
          <PillGroup
            aria-label="Worker mode"
            items={WORKER_MODE_ITEMS}
            onChange={() => undefined}
            size="sm"
            value={worker.mode}
          />
        </SetupRow>
        {worker.agent_name ? (
          <SetupRow label="Agent">
            <Chip>{worker.agent_name}</Chip>
          </SetupRow>
        ) : null}
        {worker.allowed_agent_names && worker.allowed_agent_names.length > 0 ? (
          <SetupRow label="Allowed agents">
            {worker.allowed_agent_names.map(name => (
              <Chip key={name}>{name}</Chip>
            ))}
          </SetupRow>
        ) : null}
        {worker.model ? (
          <SetupRow label="Runtime">
            <span className="font-mono text-eyebrow text-fg">
              {worker.provider ? `${worker.provider} · ` : null}
              {worker.model}
            </span>
          </SetupRow>
        ) : null}
      </SetupGroup>

      <SetupGroup label="Coordinator">
        <SetupRow label="Mode">
          <span className="capitalize">{coordinator.mode}</span>
        </SetupRow>
        {coordinator.agent_name ? (
          <SetupRow label="Agent">
            <Chip>{coordinator.agent_name}</Chip>
          </SetupRow>
        ) : null}
        {coordinator.guidance ? (
          <SetupRow label="Guidance">
            <span className="text-small-body leading-relaxed text-muted">
              {coordinator.guidance}
            </span>
          </SetupRow>
        ) : null}
      </SetupGroup>

      <SetupGroup label="Review">
        <SetupRow label="Policy">
          {reviewConfigured ? (
            <>
              {review.agent_name ? <Chip>{review.agent_name}</Chip> : <span>Configured</span>}
              {review.model ? (
                <span className="font-mono text-eyebrow text-muted">
                  {review.provider ? `${review.provider} · ` : null}
                  {review.model}
                </span>
              ) : null}
            </>
          ) : (
            <span className="text-muted">Off</span>
          )}
        </SetupRow>
      </SetupGroup>

      <SetupGroup label="Sandbox & participants">
        <SetupRow label="Sandbox">
          {sandbox.mode === "ref" && sandbox.sandbox_ref ? (
            <Chip>{sandbox.sandbox_ref}</Chip>
          ) : (
            <span className="capitalize text-muted">{sandbox.mode}</span>
          )}
        </SetupRow>
        {participants.allowed_agent_names && participants.allowed_agent_names.length > 0 ? (
          <SetupRow label="Participants">
            {participants.allowed_agent_names.map(name => (
              <Chip key={name}>{name}</Chip>
            ))}
          </SetupRow>
        ) : null}
        {participants.required_capabilities && participants.required_capabilities.length > 0 ? (
          <SetupRow label="Capabilities">
            {participants.required_capabilities.map(capability => (
              <Chip key={capability}>{capability}</Chip>
            ))}
          </SetupRow>
        ) : null}
      </SetupGroup>
    </>
  );
}

/**
 * Setup sheet (§4.7): who works on this task and where it runs. Read view is
 * form-shaped; editing is locked while a run is active. The raw profile stays
 * one toggle away for operators, and JSON editing remains the write path.
 */
export function TaskSetupSheet({
  open,
  onOpenChange,
  taskId,
  profile,
  profileLoading = false,
  profileErrorMessage = null,
  hasActiveRun,
  isSetPending = false,
  isDeletePending = false,
  onSetProfile,
  onDeleteProfile,
}: TaskSetupSheetProps) {
  const [showJson, setShowJson] = useState(false);
  const editor = useProfileEditor({ taskId, profile, onSetProfile });

  const handleDelete = async () => {
    try {
      await onDeleteProfile();
      toast.success("Setup cleared.");
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Couldn't clear the setup");
    }
  };

  return (
    <Sheet onOpenChange={onOpenChange} open={open}>
      <SheetContent
        className="w-[min(640px,calc(100vw-24px))] sm:max-w-none"
        data-testid="tasks-setup-sheet"
        side="right"
      >
        <SheetHeader>
          <div className="flex items-start gap-3">
            <span
              aria-hidden="true"
              className="grid size-9 shrink-0 place-items-center rounded-md bg-accent-tint text-accent-strong"
            >
              <Settings2 className="size-4" />
            </span>
            <div className="min-w-0">
              <span className="eyebrow font-mono text-subtle">Execution profile</span>
              <SheetTitle>Task setup</SheetTitle>
              <SheetDescription>Who works on this task and where it runs.</SheetDescription>
            </div>
          </div>
        </SheetHeader>

        <div className="min-h-0 flex-1 overflow-y-auto px-4 py-4">
          {hasActiveRun ? (
            <div
              className="mb-4 flex items-start gap-2.5 rounded-md border border-line-soft bg-canvas-soft px-3.5 py-3 text-small-body leading-relaxed text-muted"
              data-testid="tasks-setup-locked"
            >
              <Lock aria-hidden="true" className="mt-0.5 size-3.5 shrink-0 text-subtle" />
              Editing is locked while a run is active. Pause or cancel the current run to change the
              setup.
            </div>
          ) : null}

          {profileLoading && !profile ? (
            <p className="text-small-body text-muted">Loading setup…</p>
          ) : profileErrorMessage && !profile ? (
            <p className="text-small-body text-danger">{profileErrorMessage}</p>
          ) : profile ? (
            <ProfileReadView profile={profile} />
          ) : (
            <p className="text-small-body text-muted" data-testid="tasks-setup-empty">
              No custom setup. Runs inherit the workspace defaults for worker, model, and sandbox.
            </p>
          )}

          {editor.open ? (
            <div className="mt-2 flex flex-col gap-2" data-testid="tasks-setup-editor">
              <Textarea
                aria-invalid={Boolean(editor.error)}
                aria-label="Execution profile JSON"
                data-testid="tasks-setup-editor-input"
                onChange={event => editor.setValue(event.target.value)}
                rows={14}
                value={editor.value}
                variant="mono"
              />
              {editor.error ? (
                <p className="text-form-hint text-danger" data-testid="tasks-setup-editor-error">
                  {editor.error}
                </p>
              ) : null}
              <div className="flex items-center justify-end gap-2">
                <Button
                  disabled={isSetPending}
                  onClick={() => editor.setOpen(false)}
                  size="sm"
                  type="button"
                  variant="neutral"
                >
                  Cancel
                </Button>
                <Button
                  data-testid="tasks-setup-editor-save"
                  disabled={isSetPending}
                  onClick={() => void editor.submit()}
                  size="sm"
                  type="button"
                >
                  Save setup
                </Button>
              </div>
            </div>
          ) : showJson && profile ? (
            <div className="mt-2" data-testid="tasks-setup-json">
              <JsonViewer value={profile} />
            </div>
          ) : null}
        </div>

        <SheetFooter className="flex-row items-center justify-between gap-2 border-t border-line px-4 py-3">
          <div className="flex items-center gap-2">
            {profile && !editor.open ? (
              <Button
                data-testid="tasks-setup-toggle-json"
                onClick={() => setShowJson(current => !current)}
                size="sm"
                type="button"
                variant="ghost"
              >
                {showJson ? "Hide JSON" : "View JSON"}
              </Button>
            ) : null}
            {profile && !hasActiveRun && !editor.open ? (
              <Button
                data-testid="tasks-setup-clear"
                disabled={isDeletePending}
                onClick={() => void handleDelete()}
                size="sm"
                type="button"
                variant="ghost"
              >
                Clear setup
              </Button>
            ) : null}
          </div>
          <div className="flex items-center gap-2">
            {!hasActiveRun && !editor.open ? (
              <Button
                data-testid="tasks-setup-edit"
                onClick={() => editor.setOpen(true)}
                size="sm"
                type="button"
                variant="neutral"
              >
                {profile ? "Edit setup" : "Create setup"}
              </Button>
            ) : null}
            <Button
              data-testid="tasks-setup-close"
              onClick={() => onOpenChange(false)}
              size="sm"
              type="button"
              variant={hasActiveRun || editor.open ? "neutral" : "ghost"}
            >
              Close
            </Button>
          </div>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
