import { useCallback, useMemo, useState } from "react";
import { AlertCircle, Edit3, Settings2, Trash2 } from "lucide-react";

import {
  Button,
  BlockLoading,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Empty,
  Pill,
  Spinner,
  type PillTone,
  Section,
} from "@agh/ui";

import { useProfileEditor } from "../hooks/use-profile-editor";
import { formatRelativeTime } from "../lib/task-formatters";
import type { TaskExecutionProfile, TaskExecutionProfileSetRequest } from "../types";

export interface TasksExecutionProfileCardProps {
  taskId: string;
  profile: TaskExecutionProfile | null;
  isLoading?: boolean;
  errorMessage?: string | null;
  hasActiveRun?: boolean;
  isSetPending?: boolean;
  isDeletePending?: boolean;
  onSetProfile: (data: TaskExecutionProfileSetRequest) => Promise<void>;
  onDeleteProfile: () => Promise<void>;
}

interface ListSlotProps {
  label: string;
  values?: string[] | null;
}

function ListSlot({ label, values }: ListSlotProps) {
  const items = (values ?? []).filter(value => value.trim() !== "");
  if (items.length === 0) {
    return null;
  }
  return (
    <div className="flex flex-col gap-1">
      <span className="text-badge text-(--color-text-tertiary)">{label}</span>
      <div className="flex flex-wrap gap-1.5">
        {items.map(value => (
          <Pill key={`${label}:${value}`} mono>
            {value}
          </Pill>
        ))}
      </div>
    </div>
  );
}

interface PillRowProps {
  label: string;
  value: string;
  tone?: PillTone;
}

function PillRow({ label, value, tone }: PillRowProps) {
  return (
    <div className="flex items-center gap-2">
      <span className="text-badge text-(--color-text-tertiary)">{label}</span>
      <Pill tone={tone ?? "neutral"}>{value}</Pill>
    </div>
  );
}

export function TasksExecutionProfileCard({
  taskId,
  profile,
  isLoading = false,
  errorMessage = null,
  hasActiveRun = false,
  isSetPending = false,
  isDeletePending = false,
  onSetProfile,
  onDeleteProfile,
}: TasksExecutionProfileCardProps) {
  const editor = useProfileEditor({ taskId, profile, onSetProfile });
  const [deleteOpen, setDeleteOpen] = useState(false);

  const editDisabled = hasActiveRun || isSetPending || isDeletePending;
  const deleteDisabled = !profile || hasActiveRun || isSetPending || isDeletePending;

  const summaryPills = useMemo(() => {
    if (!profile) {
      return [];
    }
    return [
      { label: "Worker mode", value: profile.worker?.mode ?? "inherit" },
      { label: "Coordinator mode", value: profile.coordinator?.mode ?? "inherit" },
      { label: "Sandbox mode", value: profile.sandbox?.mode ?? "inherit" },
    ];
  }, [profile]);

  const handleDeleteConfirm = useCallback(async () => {
    try {
      await onDeleteProfile();
      setDeleteOpen(false);
    } catch {
      // toast surfaced by route hook; keep dialog open so the operator can retry.
    }
  }, [onDeleteProfile]);

  const editorTitle = profile ? "Edit execution profile" : "Create execution profile";
  const cardTestId = "tasks-execution-profile-card";

  return (
    <Section
      aria-label="Execution profile"
      className="w-full gap-4"
      data-testid={cardTestId}
      label="Execution profile"
      right={
        <div className="flex items-center gap-2">
          <Button
            data-testid="tasks-execution-profile-edit"
            disabled={editDisabled}
            onClick={() => editor.setOpen(true)}
            size="sm"
            type="button"
            variant="outline"
          >
            <Edit3 className="size-3.5" />
            {profile ? "Edit" : "Create"}
          </Button>
          <Button
            data-testid="tasks-execution-profile-delete"
            disabled={deleteDisabled}
            onClick={() => setDeleteOpen(true)}
            size="sm"
            type="button"
            variant="outline"
          >
            {isDeletePending ? <Spinner className="size-3.5" /> : <Trash2 className="size-3.5" />}
            Delete
          </Button>
        </div>
      }
    >
      {hasActiveRun ? (
        <p
          className="rounded-xl border border-(--color-divider) bg-(--color-warning-tint) px-3 py-2 text-xs text-(--color-warning)"
          data-testid="tasks-execution-profile-active-run-warning"
        >
          Profile mutation is blocked while this task has an active run. Cancel or wait for the
          current run to terminate before editing or deleting the profile.
        </p>
      ) : null}
      {isLoading && !profile ? (
        <BlockLoading
          label="Loading execution profile"
          size="sm"
          surface="bare"
          data-testid="tasks-execution-profile-loading"
        />
      ) : null}
      {errorMessage && !profile ? (
        <Empty
          data-testid="tasks-execution-profile-error"
          description={errorMessage}
          icon={AlertCircle}
          title="Unable to load execution profile"
        />
      ) : null}
      {!isLoading && !errorMessage && !profile ? (
        <Empty
          data-testid="tasks-execution-profile-empty"
          description="No task-owned execution profile is set. Workspace defaults apply at session start."
          icon={Settings2}
          title="No execution profile"
        />
      ) : null}
      {profile ? (
        <div
          className="flex flex-col gap-4 rounded-xl border border-(--color-divider) bg-(--color-surface-elevated) px-4 py-3"
          data-testid="tasks-execution-profile-summary"
        >
          <div className="flex flex-wrap items-center gap-3">
            {summaryPills.map(item => (
              <PillRow key={item.label} label={item.label} value={item.value} />
            ))}
          </div>
          <div className="grid gap-4 md:grid-cols-2">
            <div className="flex flex-col gap-2">
              <h3 className="text-eyebrow text-(--color-text-secondary)">Worker</h3>
              <div className="flex flex-col gap-2 text-small-body text-(--color-text-primary)">
                {profile.worker?.agent_name ? (
                  <span className="font-mono text-xs">agent {profile.worker.agent_name}</span>
                ) : null}
                {profile.worker?.provider ? (
                  <span className="font-mono text-xs text-(--color-text-secondary)">
                    provider {profile.worker.provider}
                  </span>
                ) : null}
                {profile.worker?.model ? (
                  <span className="font-mono text-xs text-(--color-text-secondary)">
                    model {profile.worker.model}
                  </span>
                ) : null}
                <ListSlot label="Allowed agents" values={profile.worker?.allowed_agent_names} />
                <ListSlot label="Preferred agents" values={profile.worker?.preferred_agent_names} />
                <ListSlot
                  label="Required capabilities"
                  values={profile.worker?.required_capabilities}
                />
                <ListSlot
                  label="Preferred capabilities"
                  values={profile.worker?.preferred_capabilities}
                />
              </div>
            </div>
            <div className="flex flex-col gap-2">
              <h3 className="text-eyebrow text-(--color-text-secondary)">Coordinator</h3>
              <div className="flex flex-col gap-2 text-small-body text-(--color-text-primary)">
                {profile.coordinator?.agent_name ? (
                  <span className="font-mono text-xs">agent {profile.coordinator.agent_name}</span>
                ) : null}
                {profile.coordinator?.provider ? (
                  <span className="font-mono text-xs text-(--color-text-secondary)">
                    provider {profile.coordinator.provider}
                  </span>
                ) : null}
                {profile.coordinator?.model ? (
                  <span className="font-mono text-xs text-(--color-text-secondary)">
                    model {profile.coordinator.model}
                  </span>
                ) : null}
                {profile.coordinator?.guidance ? (
                  <p className="whitespace-pre-wrap text-xs text-(--color-text-secondary)">
                    {profile.coordinator.guidance}
                  </p>
                ) : null}
              </div>
            </div>
            <div className="flex flex-col gap-2">
              <h3 className="text-eyebrow text-(--color-text-secondary)">Review selectors</h3>
              <div className="flex flex-col gap-2 text-small-body">
                {profile.review?.agent_name ? (
                  <span className="font-mono text-xs">reviewer {profile.review.agent_name}</span>
                ) : null}
                {profile.review?.provider ? (
                  <span className="font-mono text-xs text-(--color-text-secondary)">
                    provider {profile.review.provider}
                  </span>
                ) : null}
                {profile.review?.model ? (
                  <span className="font-mono text-xs text-(--color-text-secondary)">
                    model {profile.review.model}
                  </span>
                ) : null}
                <ListSlot label="Allowed agents" values={profile.review?.allowed_agent_names} />
                <ListSlot label="Preferred agents" values={profile.review?.preferred_agent_names} />
                <ListSlot label="Allowed peers" values={profile.review?.allowed_peer_ids} />
                <ListSlot label="Allowed channels" values={profile.review?.allowed_channel_ids} />
                <ListSlot
                  label="Required capabilities"
                  values={profile.review?.required_capabilities}
                />
              </div>
            </div>
            <div className="flex flex-col gap-2">
              <h3 className="text-eyebrow text-(--color-text-secondary)">Sandbox + participants</h3>
              <div className="flex flex-col gap-2 text-small-body">
                {profile.sandbox?.sandbox_ref ? (
                  <span className="font-mono text-xs">sandbox {profile.sandbox.sandbox_ref}</span>
                ) : null}
                <ListSlot
                  label="Preferred agents"
                  values={profile.participants?.preferred_agent_names}
                />
                <ListSlot
                  label="Allowed agents"
                  values={profile.participants?.allowed_agent_names}
                />
                <ListSlot label="Allowed peers" values={profile.participants?.allowed_peer_ids} />
                <ListSlot
                  label="Allowed channels"
                  values={profile.participants?.allowed_channel_ids}
                />
                <ListSlot
                  label="Required capabilities"
                  values={profile.participants?.required_capabilities}
                />
              </div>
            </div>
          </div>
          <div className="flex flex-wrap items-center gap-3 border-t border-(--color-divider) pt-3 text-eyebrow text-(--color-text-tertiary)">
            <span>created {formatRelativeTime(profile.created_at)}</span>
            <span>updated {formatRelativeTime(profile.updated_at)}</span>
          </div>
        </div>
      ) : null}

      <Dialog open={editor.open} onOpenChange={editor.setOpen}>
        <DialogContent
          className="max-w-2xl"
          data-testid="tasks-execution-profile-editor-dialog"
          showCloseButton={!isSetPending}
        >
          <DialogHeader>
            <DialogTitle>{editorTitle}</DialogTitle>
            <DialogDescription>
              Profile JSON must match the typed task execution profile contract. The runtime rejects
              edits while this task has an active run.
            </DialogDescription>
          </DialogHeader>
          <textarea
            aria-label="Execution profile JSON"
            className="min-h-[280px] w-full rounded-xl border border-(--color-divider) bg-(--color-surface) p-3 font-mono text-xs text-(--color-text-primary) focus:outline-none focus:ring-1 focus:ring-accent"
            data-testid="tasks-execution-profile-editor-input"
            disabled={isSetPending}
            onChange={event => editor.setValue(event.target.value)}
            spellCheck={false}
            value={editor.value}
          />
          {editor.error ? (
            <p
              className="text-xs text-(--color-danger)"
              data-testid="tasks-execution-profile-editor-error"
            >
              {editor.error}
            </p>
          ) : null}
          <DialogFooter className="gap-2">
            <Button
              data-testid="tasks-execution-profile-editor-cancel"
              disabled={isSetPending}
              onClick={() => editor.setOpen(false)}
              type="button"
              variant="ghost"
            >
              Cancel
            </Button>
            <Button
              data-testid="tasks-execution-profile-editor-submit"
              disabled={isSetPending}
              onClick={editor.submit}
              type="button"
              variant="default"
            >
              {isSetPending ? <Spinner className="size-3.5" /> : null}
              Save profile
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={deleteOpen} onOpenChange={setDeleteOpen}>
        <DialogContent
          className="max-w-md"
          data-testid="tasks-execution-profile-delete-dialog"
          showCloseButton={!isDeletePending}
        >
          <DialogHeader>
            <DialogTitle>Delete execution profile?</DialogTitle>
            <DialogDescription>
              This removes the task-owned execution profile. Workspace defaults will apply on the
              next session start. Delete is rejected while this task has an active run.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter className="gap-2">
            <Button
              data-testid="tasks-execution-profile-delete-cancel"
              disabled={isDeletePending}
              onClick={() => setDeleteOpen(false)}
              type="button"
              variant="ghost"
            >
              Cancel
            </Button>
            <Button
              data-testid="tasks-execution-profile-delete-confirm"
              disabled={isDeletePending}
              onClick={handleDeleteConfirm}
              type="button"
              variant="destructive"
            >
              {isDeletePending ? <Spinner className="size-3.5" /> : null}
              Delete profile
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </Section>
  );
}
