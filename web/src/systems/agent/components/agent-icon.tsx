import { Bot, BrainCircuit, Code, Sparkles, Terminal, type LucideIcon } from "lucide-react";
import type { ComponentProps } from "react";

import { cn } from "@agh/ui";

const providerIconMap: Record<string, LucideIcon> = {
  claude: BrainCircuit,
  codex: Code,
  gemini: Sparkles,
  openai: Bot,
  ollama: Terminal,
};

type AgentIconTone = "default" | "muted" | "accent";

const AGENT_ICON_TONE: Record<AgentIconTone, string> = {
  default: "text-[color:var(--color-text-primary)]",
  muted: "text-[color:var(--color-text-tertiary)]",
  accent: "text-[color:var(--color-accent)]",
};

interface AgentIconProps extends ComponentProps<"svg"> {
  provider: string;
  tone?: AgentIconTone;
}

function AgentIcon({ provider, tone = "muted", className, ...props }: AgentIconProps) {
  const Icon = providerIconMap[provider.toLowerCase()] ?? Bot;
  return (
    <Icon
      data-slot="agent-icon"
      data-provider={provider.toLowerCase()}
      className={cn("size-4 shrink-0", AGENT_ICON_TONE[tone], className)}
      {...props}
    />
  );
}

export { AgentIcon, providerIconMap };
export type { AgentIconTone };
