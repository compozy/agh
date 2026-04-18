import { Badge, NativeSelect, NativeSelectOption } from "@agh/ui";
import type { WorkspacePayload } from "../types";

interface WorkspaceSelectorProps {
  workspaces: WorkspacePayload[];
  value: string | null;
  onValueChange: (workspaceId: string) => void;
  disabled?: boolean;
}

function WorkspaceSelector({
  workspaces,
  value,
  onValueChange,
  disabled = false,
}: WorkspaceSelectorProps) {
  const selectedWorkspace =
    workspaces.find(workspace => workspace.id === value) ?? workspaces[0] ?? null;

  return (
    <div className="space-y-2">
      <NativeSelect
        aria-label="Workspace"
        className="w-full"
        value={selectedWorkspace?.id ?? ""}
        onChange={event => onValueChange(event.currentTarget.value)}
        disabled={disabled || workspaces.length === 0}
      >
        {workspaces.map(workspace => (
          <NativeSelectOption key={workspace.id} value={workspace.id}>
            {workspace.name}
          </NativeSelectOption>
        ))}
      </NativeSelect>

      {selectedWorkspace && (
        <div className="flex items-center gap-2 overflow-hidden">
          <Badge
            variant="outline"
            className="h-5 shrink-0 px-1.5 font-mono text-[0.55rem]"
            data-testid="workspace-selector-id"
          >
            {selectedWorkspace.id}
          </Badge>
          <span
            className="truncate text-[0.68rem] text-[color:var(--color-text-tertiary)]"
            data-testid="workspace-selector-root-dir"
            title={selectedWorkspace.root_dir}
          >
            {selectedWorkspace.root_dir}
          </span>
        </div>
      )}
    </div>
  );
}

export { WorkspaceSelector };
