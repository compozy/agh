import { Plus, X } from "lucide-react";
import { useState } from "react";

import {
  Button,
  CommandEmpty,
  CommandItem,
  CommandList,
  CommandSelect,
  CommandSelectGroup,
  CommandSelectShell,
  CommandSelectTrigger,
  Field,
  FieldContent,
  FieldDescription,
  FieldLabel,
  FieldTitle,
  FormSection,
  Input,
  Pill,
} from "@agh/ui";

import { AgentCommandSelect, type AgentPayload } from "@/systems/agent";

import type { useWorkspaceSetupContent } from "../hooks/use-workspace-setup-content";

type SetupApi = ReturnType<typeof useWorkspaceSetupContent>;

interface SandboxOption {
  name: string;
  backend?: string;
}

interface WorkspaceSetupDefaultsPaneProps {
  setup: SetupApi;
  agents: AgentPayload[];
  sandboxes: SandboxOption[];
}

/**
 * Right pane of the split shell: the session defaults carried by
 * `CreateWorkspaceRequest`. Every field here is optional and editable later.
 */
export function WorkspaceSetupDefaultsPane({
  setup,
  agents,
  sandboxes,
}: WorkspaceSetupDefaultsPaneProps) {
  const [pendingDir, setPendingDir] = useState("");
  const [sandboxOpen, setSandboxOpen] = useState(false);
  const disabled = setup.submissionMode !== null;

  const commitDir = () => {
    setup.addDir(pendingDir);
    setPendingDir("");
  };

  return (
    <FormSection
      description="Optional — every one of these can change later."
      title="Session defaults"
    >
      <div className="flex flex-col gap-4">
        <Field>
          <FieldContent>
            <FieldLabel htmlFor="workspace-setup-default-agent">Default agent</FieldLabel>
            <FieldDescription>Preselected when a session starts here.</FieldDescription>
          </FieldContent>
          <AgentCommandSelect
            agents={agents}
            disabled={disabled || agents.length === 0}
            onChange={next => setup.setDefaultAgent(next ?? "")}
            placeholder={agents.length === 0 ? "No agents available" : "No default"}
            triggerId="workspace-setup-default-agent"
            triggerTestId="workspace-setup-default-agent-select"
            value={setup.draft.defaultAgent || null}
          />
        </Field>

        <Field>
          <FieldContent>
            <FieldTitle id="workspace-setup-sandbox-label">Sandbox profile</FieldTitle>
            <FieldDescription>Isolation applied to sessions in this workspace.</FieldDescription>
          </FieldContent>
          <CommandSelect onOpenChange={setSandboxOpen} open={sandboxOpen}>
            <CommandSelectTrigger
              aria-labelledby="workspace-setup-sandbox-label"
              data-testid="workspace-setup-sandbox-select"
              disabled={disabled || sandboxes.length === 0}
              placeholder={sandboxes.length === 0 ? "No sandbox profiles" : "No sandbox"}
              selected={setup.draft.sandboxRef !== ""}
            >
              {setup.draft.sandboxRef || null}
            </CommandSelectTrigger>
            <CommandSelectShell inputPlaceholder="Search sandbox profiles…">
              <CommandList>
                <CommandEmpty>No sandbox profiles match.</CommandEmpty>
                <CommandSelectGroup>
                  {sandboxes.map(sandbox => (
                    <CommandItem
                      data-testid={`workspace-setup-sandbox-${sandbox.name}`}
                      key={sandbox.name}
                      onSelect={() => {
                        setup.setSandboxRef(sandbox.name);
                        setSandboxOpen(false);
                      }}
                      value={sandbox.name}
                    >
                      <span className="min-w-0 flex-1 truncate">{sandbox.name}</span>
                      {sandbox.backend ? (
                        <Pill mono size="xs">
                          {sandbox.backend}
                        </Pill>
                      ) : null}
                    </CommandItem>
                  ))}
                </CommandSelectGroup>
              </CommandList>
            </CommandSelectShell>
          </CommandSelect>
        </Field>

        <Field>
          <FieldContent>
            <FieldLabel htmlFor="workspace-setup-add-dir">Additional directories</FieldLabel>
            <FieldDescription>Extra roots sessions may read.</FieldDescription>
          </FieldContent>
          <div className="flex items-center gap-2">
            <Input
              className="font-mono"
              data-testid="workspace-setup-add-dir-input"
              disabled={disabled}
              id="workspace-setup-add-dir"
              onChange={event => setPendingDir(event.target.value)}
              onKeyDown={event => {
                if (event.key !== "Enter") return;
                // The pane lives inside the dialog form; Enter adds a chip here
                // rather than submitting the workspace.
                event.preventDefault();
                commitDir();
              }}
              placeholder="/Users/you/Dev/shared-libs"
              value={pendingDir}
            />
            <Button
              data-testid="workspace-setup-add-dir-submit"
              disabled={disabled || pendingDir.trim() === ""}
              onClick={commitDir}
              size="sm"
              type="button"
              variant="outline"
            >
              <Plus className="size-3.5" />
              Add
            </Button>
          </div>
          {setup.draft.addDirs.length > 0 ? (
            <div className="mt-2 flex flex-wrap gap-1.5" data-testid="workspace-setup-add-dir-list">
              {setup.draft.addDirs.map(dir => (
                <span
                  className="inline-flex items-center gap-1 rounded-sm bg-canvas px-2 py-1 font-mono text-mono-id tracking-normal text-muted"
                  key={dir}
                >
                  <span className="min-w-0 truncate">{dir}</span>
                  <Button
                    aria-label={`Remove ${dir}`}
                    disabled={disabled}
                    onClick={() => setup.removeDir(dir)}
                    size="icon-xs"
                    type="button"
                    variant="ghost"
                  >
                    <X className="size-3" />
                  </Button>
                </span>
              ))}
            </div>
          ) : null}
        </Field>
      </div>
    </FormSection>
  );
}
