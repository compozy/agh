import { cn } from "@agh/ui";
import { ClaudeLogo, GeminiLogo, OpenAILogo, OpenCodeLogo } from "@agh/ui/logos";
import { Bot, BrainCircuit, Code, Sparkles, Terminal, type LucideIcon } from "lucide-react";
import type { ReactNode } from "react";

type LogoKind = "claude" | "openai" | "gemini" | "opencode";

const BRAND_LOGO_KIND: Record<string, LogoKind> = {
  claude: "claude",
  codex: "openai",
  openai: "openai",
  gemini: "gemini",
  opencode: "opencode",
};

const FALLBACK_ICON: Record<string, LucideIcon> = {
  blackbox: Bot,
  claude: BrainCircuit,
  cline: Code,
  codex: Code,
  gemini: Sparkles,
  goose: Terminal,
  hermes: BrainCircuit,
  junie: Sparkles,
  "kimi-cli": Terminal,
  ollama: Terminal,
  openai: Bot,
  openclaw: Bot,
  openhands: Code,
  qoder: Code,
  "qwen-code": Sparkles,
};

interface ProviderLogoProps {
  provider: string;
  className?: string;
}

export function ProviderLogo({ provider, className }: ProviderLogoProps) {
  const key = provider.toLowerCase();
  const glyphClass = cn("size-6 text-(--color-text-primary)", className);
  const brand = BRAND_LOGO_KIND[key];
  return (
    <span
      data-slot="provider-logo"
      data-provider={key}
      className="inline-flex items-center justify-center"
    >
      {brand ? renderBrand(brand, glyphClass) : renderFallback(key, glyphClass)}
    </span>
  );
}

function renderBrand(kind: LogoKind, className: string): ReactNode {
  switch (kind) {
    case "claude":
      return <ClaudeLogo className={className} />;
    case "openai":
      return <OpenAILogo className={className} mode="dark" />;
    case "gemini":
      return <GeminiLogo className={className} />;
    case "opencode":
      return <OpenCodeLogo className={className} />;
  }
}

function renderFallback(key: string, className: string): ReactNode {
  const Icon = FALLBACK_ICON[key] ?? Bot;
  return <Icon className={className} />;
}
