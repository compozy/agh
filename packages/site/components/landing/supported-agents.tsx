"use client";

import {
  Eyebrow,
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
  UIProvider,
} from "@agh/ui";
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

const logoClassName = "size-6 text-fg";

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

// 14×4 scatter ribbon, '*' = logo slot, '.' = empty slot. Logo slots align 1:1
// with PROVIDERS order; keep the '*' count in sync whenever PROVIDERS grows or
// shrinks so the radial mask continues to fade the band ends instead of orphans.
const QUILT_PATTERN = [
  ".*..*.*.*..*.*",
  "*.*.**.*.*..*.",
  ".*.**.*.*.*.*.",
  "*..*.*.*..*..*",
] as const;
const QUILT_COLS = QUILT_PATTERN[0].length;

const QUILT_LAYOUT: readonly ("logo" | "empty")[] = QUILT_PATTERN.flatMap(row =>
  Array.from(row, ch => (ch === "*" ? ("logo" as const) : ("empty" as const)))
);

/**
 * Compact strip showing which agent CLIs are supported. Frames each CLI as a
 * peer on AGH Network , the strip's job is to make the operator see their
 * existing CLI as the entry point to the network.
 */
export function SupportedAgents() {
  return (
    <SectionFrame background="canvas" padY="md" className="border-b border-line">
      <div className="flex flex-col gap-8 lg:flex-row lg:items-center lg:justify-between lg:gap-12">
        <div className="max-w-[40ch]">
          <Eyebrow className="text-accent">Your CLI on the network</Eyebrow>
          <p className="mt-2 text-base leading-snug text-fg">
            AGH runs the CLIs you already use as durable sessions and joins them to the workplace as
            peers. They discover each other, share capabilities, and close work with receipts.
          </p>
          <Link
            href="/runtime/core/agents/providers"
            className="eyebrow mt-3 inline-flex items-center gap-1.5 text-muted transition-colors hover:text-accent"
          >
            Read more about providers
            <ArrowUpRight aria-hidden className="size-3" />
          </Link>
        </div>

        <ProviderQuilt providers={PROVIDERS} />
      </div>
    </SectionFrame>
  );
}

function ProviderQuilt({ providers }: { providers: Provider[] }) {
  let logoCursor = 0;
  return (
    <div
      // Radial mask fades only the band ends — vertical fade is muted because the ribbon is short.
      className="relative mx-auto w-full max-w-md md:max-w-2xl lg:mx-0 lg:max-w-[40rem]"
      style={{
        maskImage: "radial-gradient(ellipse 85% 120% at center, black 60%, transparent 100%)",
        WebkitMaskImage: "radial-gradient(ellipse 85% 120% at center, black 60%, transparent 100%)",
      }}
    >
      <UIProvider>
        <TooltipProvider delay={120}>
          <ul
            aria-label="Supported agent CLIs"
            className="grid w-full gap-1.5"
            style={{ gridTemplateColumns: `repeat(${QUILT_COLS}, minmax(0, 1fr))` }}
          >
            {QUILT_LAYOUT.map((slot, idx) => {
              if (slot === "empty") {
                return (
                  <li
                    key={`empty-${idx}`}
                    aria-hidden="true"
                    className="aspect-square rounded-icon-well border border-line-soft bg-canvas"
                  />
                );
              }
              const provider = providers[logoCursor];
              logoCursor += 1;
              if (!provider) {
                return (
                  <li
                    key={`logo-missing-${idx}`}
                    aria-hidden="true"
                    className="aspect-square rounded-icon-well border border-line-soft bg-canvas"
                  />
                );
              }
              return (
                <Tooltip key={provider.id}>
                  <TooltipTrigger
                    render={
                      <li
                        aria-label={provider.name}
                        tabIndex={0}
                        className="flex aspect-square cursor-default items-center justify-center rounded-icon-well border border-line bg-canvas-soft transition-colors hover:border-accent/35 focus-visible:border-accent/35 focus-visible:outline-none"
                      />
                    }
                  >
                    <span aria-hidden="true" className="flex items-center justify-center">
                      {provider.logo}
                    </span>
                  </TooltipTrigger>
                  <TooltipContent>{provider.name}</TooltipContent>
                </Tooltip>
              );
            })}
          </ul>
        </TooltipProvider>
      </UIProvider>
    </div>
  );
}
