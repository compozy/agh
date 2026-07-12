import type { ComponentType, ReactNode } from "react";

import { Empty } from "@agh/ui";

export function SettingsShellState({
  action,
  description,
  icon,
  title,
}: {
  action?: ReactNode;
  description: string;
  icon: ComponentType<{ className?: string; size?: number }>;
  title: string;
}) {
  return (
    <div className="flex flex-1 items-center justify-center overflow-y-auto px-6 py-8">
      <Empty
        action={action}
        className="max-w-xl"
        description={description}
        icon={icon}
        title={title}
        titleAs="h1"
      />
    </div>
  );
}
