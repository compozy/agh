import { PenLine } from "lucide-react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";

import type { TopbarRouteContext } from "@/types/topbar";
import { LoopEditor } from "@/systems/loops";
import { useActiveWorkspace } from "@/systems/workspace";

export const Route = createFileRoute("/_app/loops/$name/editor")({
  beforeLoad: ({ params }): { topbar: TopbarRouteContext } => ({
    topbar: { title: `Edit ${params.name}`, icon: PenLine },
  }),
  component: LoopEditorRoute,
});

function LoopEditorRoute() {
  const { name } = Route.useParams();
  const { activeWorkspaceId } = useActiveWorkspace();
  const navigate = useNavigate();

  return (
    <LoopEditor
      workspaceId={activeWorkspaceId ?? ""}
      name={name}
      // Publish → continue to the run form for the freshly published definition.
      onPublished={loop => void navigate({ to: "/loops/$name/run", params: { name: loop.name } })}
    />
  );
}
