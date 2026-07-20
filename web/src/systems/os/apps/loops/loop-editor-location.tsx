import { useNavigate } from "@tanstack/react-router";

import { useTopbarSlot } from "@agh/ui";
import { LoopEditor } from "@/systems/loops";
import { useActiveWorkspace } from "@/systems/workspace";

export function LoopEditorLocation({ name }: { name: string }) {
  const { activeWorkspaceId } = useActiveWorkspace();
  const navigate = useNavigate();
  useTopbarSlot({ crumb: `Loops / ${name} / Editor` });

  return (
    <LoopEditor
      workspaceId={activeWorkspaceId ?? ""}
      name={name}
      // Publish → continue to the run form for the freshly published definition.
      onPublished={loop => void navigate({ to: "/loops/$name/run", params: { name: loop.name } })}
    />
  );
}
