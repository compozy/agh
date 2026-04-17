import { Settings } from "lucide-react";
import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/_app/settings/")({
  component: SettingsIndexPage,
});

function SettingsIndexPage() {
  return (
    <div
      className="flex flex-1 items-center justify-center"
      data-testid="settings-index-placeholder"
    >
      <div className="flex flex-col items-center gap-3">
        <Settings
          className="size-10 text-[color:var(--color-text-tertiary)]"
          data-testid="settings-index-icon"
        />
        <p className="text-[15px] font-medium text-[color:var(--color-text-secondary)]">
          Select a settings section
        </p>
        <p className="text-[13px] text-[color:var(--color-text-tertiary)]">
          Choose a section from the left to configure AGH
        </p>
      </div>
    </div>
  );
}
