import { useCallback, useMemo, useState, type ReactNode } from "react";
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
  Eyebrow,
  Pill,
  Spinner,
  Textarea,
  type PillTone,
  Section,
} from "@agh/ui";

import { useProfileEditor } from "../hooks/use-profile-editor";
import { formatRelativeTime } from "../lib/task-formatters";
import type { TaskExecutionProfile, TaskExecutionProfileSetRequest } from "../types";

export interface TasksExecutionProfileCardProps {
  taskId: string;
  profile: TaskExecutionProfile | null;
  state?: {
    isLoading?: boolean;
    hasActiveRun?: boolean;
    isSetPending?: boolean;
    isDeletePending?: boolean;
  };
  errorMessage?: string | null;
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
    <div className="flex flex-col gap-1.5">
      <Eyebrow className="text-faint">{label}</Eyebrow>
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
      <Eyebrow className="text-faint">{label}</Eyebrow>
      <Pill tone={tone ?? "neutral"}>{value}</Pill>
    </div>
  );
}

export function TasksExecutionProfileCard({
  taskId,
  profile,
  errorMessage = null,
  state,
  onSetProfile,
  onDeleteProfile,
}: TasksExecutionProfileCardProps) {
  const {
    isLoading = false,
    hasActiveRun = false,
    isSetPending = false,
    isDeletePending = false,
  } = state ?? {};
  return TasksExecutionProfileCardView({
    taskId,
    profile,
    errorMessage,
    state: { isLoading, hasActiveRun, isSetPending, isDeletePending },
    onSetProfile,
    onDeleteProfile,
  });
}

function TasksExecutionProfileCardView({
  taskId,
  profile,
  errorMessage,
  state,
  onSetProfile,
  onDeleteProfile,
}: TasksExecutionProfileCardProps) {
  const {
    isLoading = false,
    hasActiveRun = false,
    isSetPending = false,
    isDeletePending = false,
  } = state ?? {};
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
            variant="neutral"
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
            variant="ghost"
          >
            {isDeletePending ? <Spinner className="size-3.5" /> : <Trash2 className="size-3.5" />}
            Delete
          </Button>
        </div>
      }
    >
      {hasActiveRun ? (
        <div
          className="flex items-start gap-2 rounded bg-warning-tint px-3 py-2 text-[12px] leading-relaxed text-warning"
          data-testid="tasks-execution-profile-active-run-warning"
        >
          <AlertCircle className="mt-0.5 size-3.5 shrink-0" />
          <span>
            Profile mutation is blocked while this task has an active run. Cancel or wait for the
            current run to terminate before editing or deleting the profile.
          </span>
        </div>
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
          className="flex flex-col gap-4 rounded-lg bg-canvas-soft px-4 py-4"
          data-testid="tasks-execution-profile-summary"
        >
          <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
            {summaryPills.map(item => (
              <PillRow key={item.label} label={item.label} value={item.value} />
            ))}
          </div>
          <div className="grid gap-4 md:grid-cols-2">
            <ProfilePane title="Worker">
              {profile.worker?.agent_name ? (
                <ProfileLine label="agent" value={profile.worker.agent_name} primary />
              ) : null}
              {profile.worker?.provider ? (
                <ProfileLine label="provider" value={profile.worker.provider} />
              ) : null}
              {profile.worker?.model ? (
                <ProfileLine label="model" value={profile.worker.model} />
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
            </ProfilePane>
            <ProfilePane title="Coordinator">
              {profile.coordinator?.agent_name ? (
                <ProfileLine label="agent" value={profile.coordinator.agent_name} primary />
              ) : null}
              {profile.coordinator?.provider ? (
                <ProfileLine label="provider" value={profile.coordinator.provider} />
              ) : null}
              {profile.coordinator?.model ? (
                <ProfileLine label="model" value={profile.coordinator.model} />
              ) : null}
              {profile.coordinator?.guidance ? (
                <p className="whitespace-pre-wrap text-[12px] leading-relaxed text-muted">
                  {profile.coordinator.guidance}
                </p>
              ) : null}
            </ProfilePane>
            <ProfilePane title="Review selectors">
              {profile.review?.agent_name ? (
                <ProfileLine label="reviewer" value={profile.review.agent_name} primary />
              ) : null}
              {profile.review?.provider ? (
                <ProfileLine label="provider" value={profile.review.provider} />
              ) : null}
              {profile.review?.model ? (
                <ProfileLine label="model" value={profile.review.model} />
              ) : null}
              <ListSlot label="Allowed agents" values={profile.review?.allowed_agent_names} />
              <ListSlot label="Preferred agents" values={profile.review?.preferred_agent_names} />
              <ListSlot label="Allowed peers" values={profile.review?.allowed_peer_ids} />
              <ListSlot label="Allowed channels" values={profile.review?.allowed_channel_ids} />
              <ListSlot
                label="Required capabilities"
                values={profile.review?.required_capabilities}
              />
            </ProfilePane>
            <ProfilePane title="Sandbox + participants">
              {profile.sandbox?.sandbox_ref ? (
                <ProfileLine label="sandbox" value={profile.sandbox.sandbox_ref} primary />
              ) : null}
              <ListSlot
                label="Preferred agents"
                values={profile.participants?.preferred_agent_names}
              />
              <ListSlot label="Allowed agents" values={profile.participants?.allowed_agent_names} />
              <ListSlot label="Allowed peers" values={profile.participants?.allowed_peer_ids} />
              <ListSlot
                label="Allowed channels"
                values={profile.participants?.allowed_channel_ids}
              />
              <ListSlot
                label="Required capabilities"
                values={profile.participants?.required_capabilities}
              />
            </ProfilePane>
          </div>
          <div className="flex flex-wrap items-center gap-x-4 gap-y-1 border-t border-line pt-3">
            <Eyebrow className="text-faint">
              created {formatRelativeTime(profile.created_at)}
            </Eyebrow>
            <Eyebrow className="text-faint">
              updated {formatRelativeTime(profile.updated_at)}
            </Eyebrow>
          </div>
        </div>
      ) : null}

      <ExecutionProfileEditorDialog
        editor={editor}
        isSetPending={isSetPending}
        title={editorTitle}
      />
      <ExecutionProfileDeleteDialog
        deleteOpen={deleteOpen}
        isDeletePending={isDeletePending}
        onConfirm={handleDeleteConfirm}
        setDeleteOpen={setDeleteOpen}
      />
    </Section>
  );
}

function ExecutionProfileEditorDialog({
  editor,
  isSetPending,
  title,
}: {
  editor: ReturnType<typeof useProfileEditor>;
  isSetPending: boolean;
  title: string;
}) {
  return (
    <Dialog open={editor.open} onOpenChange={editor.setOpen}>
      <DialogContent
        className="max-w-2xl"
        data-testid="tasks-execution-profile-editor-dialog"
        showCloseButton={!isSetPending}
      >
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>
            Profile JSON must match the typed task execution profile contract. The runtime rejects
            edits while this task has an active run.
          </DialogDescription>
        </DialogHeader>
        <Textarea
          aria-label="Execution profile JSON"
          className="min-h-[280px]"
          data-testid="tasks-execution-profile-editor-input"
          disabled={isSetPending}
          onChange={event => editor.setValue(event.target.value)}
          spellCheck={false}
          value={editor.value}
          variant="mono"
        />
        {editor.error ? (
          <p className="text-xs text-danger" data-testid="tasks-execution-profile-editor-error">
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
  );
}

function ExecutionProfileDeleteDialog({
  deleteOpen,
  isDeletePending,
  onConfirm,
  setDeleteOpen,
}: {
  deleteOpen: boolean;
  isDeletePending: boolean;
  onConfirm: () => Promise<void>;
  setDeleteOpen: (open: boolean) => void;
}) {
  return (
    <Dialog open={deleteOpen} onOpenChange={setDeleteOpen}>
      <DialogContent
        className="max-w-md"
        data-testid="tasks-execution-profile-delete-dialog"
        showCloseButton={!isDeletePending}
      >
        <DialogHeader>
          <DialogTitle>Delete execution profile?</DialogTitle>
          <DialogDescription>
            This removes the task-owned execution profile. Workspace defaults will apply on the next
            session start. Delete is rejected while this task has an active run.
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
            onClick={onConfirm}
            type="button"
            variant="destructive"
          >
            {isDeletePending ? <Spinner className="size-3.5" /> : null}
            Delete profile
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

interface ProfilePaneProps {
  title: string;
  children: ReactNode;
}

function ProfilePane({ title, children }: ProfilePaneProps) {
  return (
    <div className="flex flex-col gap-2.5">
      <Eyebrow>{title}</Eyebrow>
      <div className="flex flex-col gap-2">{children}</div>
    </div>
  );
}

interface ProfileLineProps {
  label: string;
  value: string;
  primary?: boolean;
}

function ProfileLine({ label, value, primary = false }: ProfileLineProps) {
  return (
    <span className="inline-flex min-w-0 items-baseline gap-1.5 font-mono text-[11px] tabular-nums">
      <span className="text-faint">{label}</span>
      <span className={primary ? "min-w-0 truncate text-fg-strong" : "min-w-0 truncate text-muted"}>
        {value}
      </span>
    </span>
  );
}
