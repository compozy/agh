import {
  BlackboxLogo,
  ClaudeLogo,
  ClineLogo,
  CursorLogo,
  GeminiLogo,
  GithubLogo,
  GooseLogo,
  GroqLogo,
  HermesLogo,
  JunieLogo,
  KimiLogo,
  KiroLogo,
  MinimaxLogo,
  MistralLogo,
  OpenAILogo,
  OpenClawLogo,
  OpenCodeLogo,
  OpenHandsLogo,
  OpenRouterLogo,
  PiLogo,
  QoderLogo,
  QwenLogo,
  VercelLogo,
  XAILogo,
  ZAILogo,
} from "@agh/ui/logos";
import { ArrowUpRight } from "lucide-react";
import Link from "next/link";
import type { ReactNode } from "react";
import { SectionFrame } from "./primitives/section-frame";

type Provider = { id: string; name: string; logo: ReactNode };

const logoClassName = "h-6 w-6 text-(--color-text-primary)";

export const PROVIDERS: Provider[] = [
  { id: "claude", name: "Claude Code", logo: <ClaudeLogo className="h-6 w-6" /> },
  { id: "codex", name: "Codex", logo: <OpenAILogo className="h-6 w-6" mode="dark" /> },
  { id: "gemini", name: "Gemini CLI", logo: <GeminiLogo className="h-6 w-6" /> },
  { id: "opencode", name: "OpenCode", logo: <OpenCodeLogo className={logoClassName} /> },
  {
    id: "copilot",
    name: "GitHub Copilot CLI",
    logo: <GithubLogo aria-hidden className={logoClassName} />,
  },
  { id: "cursor", name: "Cursor Agent", logo: <CursorLogo className={logoClassName} /> },
  { id: "kiro", name: "Kiro CLI", logo: <KiroLogo className="h-6 w-6" /> },
  { id: "pi", name: "Pi", logo: <PiLogo className="h-6 w-6" /> },
  { id: "blackbox", name: "BLACKBOX AI", logo: <BlackboxLogo className={logoClassName} /> },
  { id: "cline", name: "Cline", logo: <ClineLogo className={logoClassName} /> },
  { id: "goose", name: "Goose", logo: <GooseLogo className={logoClassName} /> },
  { id: "hermes", name: "Hermes", logo: <HermesLogo className={logoClassName} /> },
  { id: "junie", name: "Junie", logo: <JunieLogo className="h-6 w-6" /> },
  { id: "kimi-cli", name: "Kimi CLI", logo: <KimiLogo className={logoClassName} /> },
  { id: "openclaw", name: "OpenClaw", logo: <OpenClawLogo className="h-6 w-6" /> },
  { id: "openhands", name: "OpenHands", logo: <OpenHandsLogo className={logoClassName} /> },
  { id: "qoder", name: "Qoder CLI", logo: <QoderLogo className={logoClassName} /> },
  { id: "qwen-code", name: "Qwen Code", logo: <QwenLogo className={logoClassName} /> },
  { id: "openrouter", name: "OpenRouter", logo: <OpenRouterLogo className={logoClassName} /> },
  { id: "zai", name: "z.ai", logo: <ZAILogo className={logoClassName} /> },
  { id: "moonshot", name: "Moonshot / Kimi", logo: <KimiLogo className={logoClassName} /> },
  {
    id: "vercel-ai-gateway",
    name: "Vercel AI Gateway",
    logo: <VercelLogo className={logoClassName} />,
  },
  { id: "xai", name: "xAI", logo: <XAILogo className={logoClassName} /> },
  { id: "minimax", name: "MiniMax", logo: <MinimaxLogo className="h-6 w-6" /> },
  { id: "mistral", name: "Mistral", logo: <MistralLogo className="h-6 w-6" /> },
  { id: "groq", name: "Groq", logo: <GroqLogo className="h-6 w-6" /> },
];

/**
 * Compact strip showing which agent CLIs are supported. Frames each CLI as a
 * peer on AGH Network — the strip's job is to make the operator see their
 * existing CLI as the entry point to the network.
 */
export function SupportedAgents() {
  return (
    <SectionFrame background="canvas" padY="md" className="border-b border-(--color-divider)">
      <div className="flex flex-col gap-8 lg:flex-row lg:items-center lg:justify-between lg:gap-12">
        <div className="max-w-[40ch]">
          <p className="font-mono text-[10px] font-semibold uppercase tracking-(--tracking-mono) text-(--color-accent)">
            Your CLI on the network
          </p>
          <p className="mt-2 text-[1rem] leading-snug text-(--color-text-primary)">
            AGH runs the CLIs you already use as durable sessions and joins them to the workplace as
            peers. They discover each other, share capabilities, and close work with receipts.
          </p>
          <Link
            href="/runtime/core/agents/providers"
            className="mt-3 inline-flex items-center gap-1.5 font-mono text-[11px] uppercase tracking-(--tracking-mono) text-(--color-text-secondary) transition-colors hover:text-(--color-accent)"
          >
            Read more about providers
            <ArrowUpRight aria-hidden className="h-3 w-3" />
          </Link>
        </div>

        <ul className="grid w-full grid-cols-3 gap-2 sm:grid-cols-6 lg:w-auto lg:grid-cols-9">
          {PROVIDERS.map(provider => (
            <li key={provider.id}>
              <div
                className="flex h-16 w-full min-w-[76px] flex-col items-center justify-center gap-1 rounded-[10px] border border-(--color-divider) bg-(--color-surface) px-2 transition-colors hover:border-[color-mix(in_srgb,var(--color-accent)_35%,var(--color-divider))]"
                title={provider.name}
              >
                <span aria-hidden="true" className="flex h-6 w-6 items-center justify-center">
                  {provider.logo}
                </span>
                <span className="font-mono text-[9px] uppercase tracking-(--tracking-mono) text-(--color-text-tertiary)">
                  {provider.id}
                </span>
              </div>
            </li>
          ))}
        </ul>
      </div>
    </SectionFrame>
  );
}
