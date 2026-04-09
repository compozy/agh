import { Wrench } from "lucide-react";
import { createFileRoute } from "@tanstack/react-router";

import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty";

export const Route = createFileRoute("/_app/skills")({
  component: SkillsPage,
});

function SkillsPage() {
  return (
    <div className="flex flex-1 items-center justify-center p-6">
      <Empty>
        <EmptyHeader>
          <EmptyMedia variant="icon">
            <Wrench className="size-5" />
          </EmptyMedia>
          <EmptyTitle>Skills</EmptyTitle>
          <EmptyDescription>
            Manage installed skills and browse the marketplace. Coming soon.
          </EmptyDescription>
        </EmptyHeader>
      </Empty>
    </div>
  );
}
