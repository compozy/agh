import { Star } from "lucide-react";
import { baseOptions } from "@/lib/layout.shared";
import { CtaButton } from "./primitives/cta-button";
import { SectionFrame } from "./primitives/section-frame";

export function FinalCta() {
  return (
    <SectionFrame background="surface" padY="lg">
      <div className="grid gap-8 rounded-(--radius-diagram) border border-(--color-divider) bg-(--color-canvas) px-6 py-10 lg:grid-cols-[minmax(0,1fr)_340px] lg:items-center lg:px-10">
        <div>
          <p className="font-mono text-[11px] font-semibold uppercase tracking-(--tracking-mono) text-(--color-accent)">
            Ship it
          </p>
          <h2 className="mt-4 max-w-[18ch] text-[clamp(2rem,4.5vw,3.2rem)] leading-[1.02] font-normal tracking-[-0.03em] text-(--color-text-primary)">
            Install AGH. Run a session. Join the network.
          </h2>
          <p className="mt-5 max-w-[52ch] text-sm leading-7 text-(--color-text-secondary)">
            One binary. No infrastructure. Shipped today.
          </p>
        </div>

        <div className="flex flex-col items-start gap-3">
          <CtaButton
            href="/runtime/core/getting-started/installation"
            variant="primary"
            className="w-full justify-center sm:w-auto"
          >
            Install AGH
          </CtaButton>
          <CtaButton href="/protocol" variant="ghost" className="w-full justify-center sm:w-auto">
            Read agh-network/v0 spec
          </CtaButton>
          <a
            href={baseOptions.githubUrl}
            target="_blank"
            rel="noreferrer"
            className="mt-1 inline-flex items-center gap-2 font-mono text-[12px] uppercase tracking-(--tracking-mono) text-(--color-text-secondary) transition-colors hover:text-(--color-accent)"
          >
            <Star className="h-3.5 w-3.5" />
            Star on GitHub
          </a>
        </div>
      </div>
    </SectionFrame>
  );
}
