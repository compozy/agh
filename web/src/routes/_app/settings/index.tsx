import { Settings } from "lucide-react";
import { createFileRoute } from "@tanstack/react-router";

import { Empty } from "@agh/ui";

export const Route = createFileRoute("/_app/settings/")({
  component: SettingsIndexPage,
});

function SettingsIndexPage() {
  return (
    <div
      className="flex flex-1 items-center justify-center p-8"
      data-testid="settings-index-placeholder"
    >
      <Empty
        icon={Settings}
        title="Select a settings section"
        description="Choose a section from the left to configure AGH"
        data-testid="settings-index-empty"
        className="max-w-md"
      />
    </div>
  );
}
