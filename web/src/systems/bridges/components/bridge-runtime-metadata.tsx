import type { ReactNode } from "react";
import { Plug } from "lucide-react";

import { Eyebrow, FormSection } from "@agh/ui";

export function RuntimeMissingProviderState() {
  return (
    <FormSection
      data-testid="bridge-wizard-section-runtime-missing"
      icon={Plug}
      title="Provider runtime"
    >
      <p className="text-small-body text-muted">
        Select a provider before configuring runtime details.
      </p>
    </FormSection>
  );
}

export function RuntimeMetadataTile({
  label,
  right,
  children,
}: {
  label: string;
  right?: ReactNode;
  children: ReactNode;
}) {
  return (
    <div className="flex flex-col gap-1 rounded bg-canvas-tint px-3 py-2.5">
      <div className="flex items-center justify-between gap-2">
        <Eyebrow className="text-muted">{label}</Eyebrow>
        {right ?? null}
      </div>
      <div className="text-small-body text-fg">{children}</div>
    </div>
  );
}
