import type { Dispatch, SetStateAction } from "react";

import { SettingsFieldRow, type SettingsGeneralSection } from "@/systems/settings";
import { Input, Section, Switch } from "@agh/ui";

type GeneralConfig = SettingsGeneralSection["config"];

interface DraftSectionProps {
  draft: GeneralConfig;
  setDraft: Dispatch<SetStateAction<GeneralConfig | null>>;
}

export function DaemonSection({ draft, setDraft }: DraftSectionProps) {
  return (
    <Section divided label="Runtime memory reporting" note="restart required">
      <SettingsFieldRow
        data-testid="settings-page-general-memory-report-interval"
        label="Report interval"
        description="Cadence for daemon process-memory snapshots in logs and the runtime.memory probe. Set 0s to disable memory reporting."
        hint="DEFAULT"
        control={
          <Input
            className="w-32 font-mono"
            data-testid="settings-page-general-memory-report-interval-input"
            value={draft.daemon.memory_report_interval}
            placeholder="5m"
            onChange={event =>
              setDraft(prev => {
                const current = prev ?? draft;
                return {
                  ...current,
                  daemon: { ...current.daemon, memory_report_interval: event.target.value },
                };
              })
            }
          />
        }
      />
    </Section>
  );
}

export function RedactionSection({ draft, setDraft }: DraftSectionProps) {
  return (
    <Section
      divided
      label="Secret redaction"
      note="restart required"
      data-testid="settings-page-general-redact"
    >
      <SettingsFieldRow
        data-testid="settings-page-general-redact-enabled"
        label="Secret redaction heuristics"
        description="Redacts likely credentials in agent-visible text, logs, and event content. Exact secret protections remain active when disabled."
        control={
          <Switch
            data-testid="settings-page-general-redact-enabled-switch"
            checked={draft.redact.enabled}
            onCheckedChange={checked =>
              setDraft(prev => {
                const current = prev ?? draft;
                return { ...current, redact: { ...current.redact, enabled: checked } };
              })
            }
          />
        }
      />
    </Section>
  );
}
