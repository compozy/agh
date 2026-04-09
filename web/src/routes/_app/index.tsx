import { Terminal } from "lucide-react";
import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/_app/")({
  component: AppIndexPage,
});

function AppIndexPage() {
  return (
    <div className="flex flex-1 items-center justify-center" data-testid="app-empty-state">
      <div className="flex flex-col items-center gap-3">
        <Terminal
          className="size-12 text-[color:var(--color-text-tertiary)]"
          data-testid="empty-terminal-icon"
        />
        <p className="text-[15px] font-medium text-[color:var(--color-text-secondary)]">
          Select a session to begin
        </p>
        <p className="text-[13px] text-[color:var(--color-text-tertiary)]">
          or create a new one from the sidebar
        </p>
      </div>
    </div>
  );
}
