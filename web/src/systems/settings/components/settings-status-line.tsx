import { Children, isValidElement, type ReactNode } from "react";
import { ConnectionIndicator, type ConnectionStatus } from "@agh/ui";

interface SettingsStatusLineProps {
  status: ConnectionStatus;
  daemonLabel?: string;
  items: ReactNode[];
  "data-testid"?: string;
}

function SettingsStatusLine({
  status,
  daemonLabel,
  items,
  "data-testid": testId,
}: SettingsStatusLineProps) {
  const keyedItems = Children.toArray(items);

  return (
    <div className="flex flex-wrap items-center gap-x-4 gap-y-1" data-testid={testId}>
      <ConnectionIndicator label={daemonLabel} status={status} />
      {keyedItems.map(item => (
        <span
          key={isValidElement(item) ? item.key : String(item)}
          className="flex items-center gap-1"
        >
          <span aria-hidden="true" className="text-(--color-text-tertiary)">
            ·
          </span>
          {item}
        </span>
      ))}
    </div>
  );
}

export { SettingsStatusLine };
