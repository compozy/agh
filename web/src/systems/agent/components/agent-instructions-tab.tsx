import { useMemo } from "react";

import { Button, DescriptionCard, PillGroup, Skeleton, type PillGroupItem } from "@agh/ui";

import type { SessionPayload } from "@/systems/session";

import { useAgentInstructionsTab } from "../hooks/use-agent-instructions-tab";
import type { AgentInstructionFile } from "../lib/agent-detail-search";
import type { AgentPayload } from "../types";
import { AgentAuthoredFileEditor } from "./agent-authored-file-editor";
import { AgentHeartbeatOps } from "./agent-heartbeat-ops";

export interface AgentInstructionsTabProps {
  agent: AgentPayload;
  file: AgentInstructionFile;
  onFileChange: (file: AgentInstructionFile) => void;
  onEditAgentPrompt: () => void;
  workspaceId: string | null;
  sessions: SessionPayload[];
  onNewSession: () => void;
}

export function AgentInstructionsTab({
  agent,
  file,
  onFileChange,
  onEditAgentPrompt,
  workspaceId,
  sessions,
  onNewSession,
}: AgentInstructionsTabProps) {
  const page = useAgentInstructionsTab({ agent, file, workspaceId, sessions });

  const fileItems: PillGroupItem<AgentInstructionFile>[] = useMemo(
    () => [
      { value: "agent", label: "AGENT.md", testId: "agent-file-tab-agent" },
      {
        value: "soul",
        label: (
          <span className="inline-flex items-center gap-1.5">
            SOUL.md
            {page.soulMissing ? (
              <span className="text-warning" data-testid="agent-file-soul-missing-badge">
                missing
              </span>
            ) : null}
          </span>
        ),
        testId: "agent-file-tab-soul",
      },
      {
        value: "heartbeat",
        label: (
          <span className="inline-flex items-center gap-1.5">
            HEARTBEAT.md
            {page.heartbeatMissing ? (
              <span className="text-warning" data-testid="agent-file-heartbeat-missing-badge">
                missing
              </span>
            ) : null}
          </span>
        ),
        testId: "agent-file-tab-heartbeat",
      },
    ],
    [page.heartbeatMissing, page.soulMissing]
  );

  return (
    <div className="flex flex-col gap-4" data-testid="agent-instructions-tab">
      <PillGroup
        aria-label="Authored files"
        data-testid="agent-instructions-files"
        items={fileItems}
        onChange={onFileChange}
        size="md"
        value={file}
      />

      {file === "agent" ? (
        <div className="flex flex-col gap-3" data-testid="agent-file-agent">
          <div className="flex items-center justify-between gap-3">
            <p className="font-mono text-badge tracking-mono text-muted">{page.promptWordCount}</p>
            <Button
              type="button"
              size="sm"
              variant="ghost"
              onClick={onEditAgentPrompt}
              data-testid="agent-file-edit-prompt"
            >
              Edit ›
            </Button>
          </div>
          <DescriptionCard data-testid="agent-file-prompt">
            {agent.prompt || "No prompt."}
          </DescriptionCard>
        </div>
      ) : null}

      {file === "soul" ? (
        <AgentAuthoredFileEditor
          kind="soul"
          payload={page.soul.payload}
          isLoading={page.soul.isLoading}
          isError={page.soul.isError}
          history={page.soul.history}
          onValidate={page.soul.onValidate}
          onSave={page.soul.onSave}
          onRestore={page.soul.onRestore}
          onRetry={page.soul.onRetry}
        />
      ) : null}

      {file === "heartbeat" ? (
        <AgentAuthoredFileEditor
          kind="heartbeat"
          payload={page.heartbeat.payload}
          isLoading={page.heartbeat.isLoading}
          isError={page.heartbeat.isError}
          history={page.heartbeat.history}
          headerSlot={
            page.heartbeat.statusLoading && !page.heartbeat.status ? (
              <Skeleton className="h-32 rounded-md" />
            ) : (
              <AgentHeartbeatOps
                status={page.heartbeat.status}
                statusLoading={page.heartbeat.statusLoading}
                statusError={page.heartbeat.statusError}
                onRetryStatus={page.heartbeat.onRetryStatus}
                activeSessions={page.heartbeat.activeSessions}
                selectedSessionId={page.heartbeat.wakeSessionId}
                onSelectSessionId={page.heartbeat.setWakeSessionId}
                onWake={page.heartbeat.onWake}
                waking={page.heartbeat.waking}
                wakeDecision={page.heartbeat.wakeDecision}
                wakeError={page.heartbeat.wakeError}
                onNewSession={onNewSession}
              />
            )
          }
          onValidate={page.heartbeat.onValidate}
          onSave={page.heartbeat.onSave}
          onRestore={page.heartbeat.onRestore}
          onRetry={page.heartbeat.onRetry}
        />
      ) : null}
    </div>
  );
}
