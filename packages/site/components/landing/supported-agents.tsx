import { Eyebrow } from "@agh/ui";
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

const logoClassName = "size-6 text-(--fg)";

export const PROVIDERS: Provider[] = [
  { id: "claude", name: "Claude Code", logo: <ClaudeLogo aria-hidden className="size-6" /> },
  {
    id: "codex",
    name: "Codex",
    logo: <OpenAILogo aria-hidden className="size-6" mode="dark" />,
  },
  { id: "gemini", name: "Gemini CLI", logo: <GeminiLogo aria-hidden className="size-6" /> },
  {
    id: "opencode",
    name: "OpenCode",
    logo: <OpenCodeLogo aria-hidden className={logoClassName} />,
  },
  {
    id: "copilot",
    name: "GitHub Copilot CLI",
    logo: <GithubLogo aria-hidden className={logoClassName} />,
  },
  {
    id: "cursor",
    name: "Cursor Agent",
    logo: <CursorLogo aria-hidden className={logoClassName} />,
  },
  { id: "kiro", name: "Kiro CLI", logo: <KiroLogo aria-hidden className="size-6" /> },
  { id: "pi", name: "Pi", logo: <PiLogo aria-hidden className="size-6" /> },
  {
    id: "blackbox",
    name: "BLACKBOX AI",
    logo: <BlackboxLogo aria-hidden className={logoClassName} />,
  },
  { id: "cline", name: "Cline", logo: <ClineLogo aria-hidden className={logoClassName} /> },
  { id: "goose", name: "Goose", logo: <GooseLogo aria-hidden className={logoClassName} /> },
  { id: "hermes", name: "Hermes", logo: <HermesLogo aria-hidden className={logoClassName} /> },
  { id: "junie", name: "Junie", logo: <JunieLogo aria-hidden className="size-6" /> },
  { id: "kimi-cli", name: "Kimi CLI", logo: <KimiLogo aria-hidden className={logoClassName} /> },
  { id: "openclaw", name: "OpenClaw", logo: <OpenClawLogo aria-hidden className="size-6" /> },
  {
    id: "openhands",
    name: "OpenHands",
    logo: <OpenHandsLogo aria-hidden className={logoClassName} />,
  },
  { id: "qoder", name: "Qoder CLI", logo: <QoderLogo aria-hidden className={logoClassName} /> },
  { id: "qwen-code", name: "Qwen Code", logo: <QwenLogo aria-hidden className={logoClassName} /> },
  {
    id: "openrouter",
    name: "OpenRouter",
    logo: <OpenRouterLogo aria-hidden className={logoClassName} />,
  },
  { id: "zai", name: "z.ai", logo: <ZAILogo aria-hidden className={logoClassName} /> },
  {
    id: "moonshot",
    name: "Moonshot / Kimi",
    logo: <KimiLogo aria-hidden className={logoClassName} />,
  },
  {
    id: "vercel-ai-gateway",
    name: "Vercel AI Gateway",
    logo: <VercelLogo aria-hidden className={logoClassName} />,
  },
  { id: "xai", name: "xAI", logo: <XAILogo aria-hidden className={logoClassName} /> },
  { id: "minimax", name: "MiniMax", logo: <MinimaxLogo aria-hidden className="size-6" /> },
  { id: "mistral", name: "Mistral", logo: <MistralLogo aria-hidden className="size-6" /> },
  { id: "groq", name: "Groq", logo: <GroqLogo aria-hidden className="size-6" /> },
];

/**
 * Compact strip showing which agent CLIs are supported. Frames each CLI as a
 * peer on AGH Network , the strip's job is to make the operator see their
 * existing CLI as the entry point to the network.
 */
export function SupportedAgents() {
  return (
    <SectionFrame background="canvas" padY="md" className="border-b border-(--line)">
      <div className="flex flex-col gap-8 lg:flex-row lg:items-center lg:justify-between lg:gap-12">
        <div className="max-w-[40ch]">
          <Eyebrow className="text-accent">Your CLI on the network</Eyebrow>
          <p className="mt-2 text-base leading-snug text-(--fg)">
            AGH runs the CLIs you already use as durable sessions and joins them to the workplace as
            peers. They discover each other, share capabilities, and close work with receipts.
          </p>
          <Link
            href="/runtime/core/agents/providers"
            className="eyebrow mt-3 inline-flex items-center gap-1.5 text-(--muted) transition-colors hover:text-accent"
          >
            Read more about providers
            <ArrowUpRight aria-hidden className="size-3" />
          </Link>
        </div>

        <ul className="grid w-full grid-cols-3 gap-2 sm:grid-cols-6 lg:w-auto lg:grid-cols-9">
          {PROVIDERS.map(provider => (
            <li key={provider.id}>
              <div
                className="flex h-16 w-full min-w-[76px] flex-col items-center justify-center gap-1 rounded-icon-well border border-(--line) bg-(--canvas-soft) px-2 transition-colors hover:border-accent/35"
                title={provider.name}
              >
                <span aria-hidden="true" className="flex size-6 items-center justify-center">
                  {provider.logo}
                </span>
                <Eyebrow className="text-(--subtle)">{provider.id}</Eyebrow>
              </div>
            </li>
          ))}
        </ul>
      </div>
    </SectionFrame>
  );
}
