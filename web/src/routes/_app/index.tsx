import { Terminal } from "lucide-react";
import { createFileRoute } from "@tanstack/react-router";

import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty";
import { Kbd } from "@/components/ui/kbd";

export const Route = createFileRoute("/_app/")({
  component: AppIndexPage,
});

function AppIndexPage() {
  return (
    <div className="flex flex-1 items-center justify-center p-6">
      <Empty>
        <EmptyHeader>
          <EmptyMedia variant="icon">
            <Terminal className="size-5" />
          </EmptyMedia>
          <EmptyTitle>No session selected</EmptyTitle>
          <EmptyDescription>
            Start a new session from the sidebar or select an existing one to begin.
          </EmptyDescription>
        </EmptyHeader>
        <div className="flex items-center gap-1.5 text-xs text-[color:var(--ds-text-muted)]">
          <span>Press</span>
          <Kbd>⌘K</Kbd>
          <span>to search</span>
        </div>
      </Empty>
    </div>
  );
}
