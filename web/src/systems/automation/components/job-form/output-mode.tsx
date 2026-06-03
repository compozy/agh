import { Bot, SquareCheck } from "lucide-react";

import { ChoiceCard } from "@/components/choice-card";

import type { JobOutputMode } from "../../lib/automation-drafts";

interface OutputModeProps {
  output: JobOutputMode;
  onOutputChange: (output: JobOutputMode) => void;
}

/** "Run" step output mode: prompt an agent directly, or delegate to a durable task. */
export function OutputMode({ output, onOutputChange }: OutputModeProps) {
  return (
    <div aria-label="Output mode" className="grid grid-cols-2 gap-3" role="radiogroup">
      <ChoiceCard
        data-testid="job-output-agent"
        description="Opens a session and sends the prompt directly."
        icon={Bot}
        onSelect={() => onOutputChange("agent")}
        selected={output === "agent"}
        title="Prompt an agent"
      />
      <ChoiceCard
        data-testid="job-output-task"
        description="Materializes a durable task run each tick."
        icon={SquareCheck}
        onSelect={() => onOutputChange("task")}
        selected={output === "task"}
        title="Delegate to a task"
      />
    </div>
  );
}
