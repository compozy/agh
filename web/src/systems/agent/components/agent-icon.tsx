import { Bot, BrainCircuit, Code, Sparkles, Terminal, type LucideIcon } from "lucide-react";
import type { ComponentProps } from "react";

import { cn } from "@/lib/utils";

const providerIconMap: Record<string, LucideIcon> = {
  claude: BrainCircuit,
  codex: Code,
  gemini: Sparkles,
  openai: Bot,
  ollama: Terminal,
};

interface AgentIconProps extends ComponentProps<"svg"> {
  provider: string;
}

function AgentIcon({ provider, className, ...props }: AgentIconProps) {
  const Icon = providerIconMap[provider.toLowerCase()] ?? Bot;
  return <Icon className={cn("size-4", className)} {...props} />;
}

export { AgentIcon, providerIconMap };
