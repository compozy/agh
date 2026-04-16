import Link from "next/link";
import { buttonVariants } from "@agh/ui";

export function Hero() {
  return (
    <section className="flex min-h-[calc(100vh-3.5rem)] flex-col items-center justify-center px-4 py-24 text-center">
      <div className="mx-auto max-w-4xl">
        <h1 className="text-5xl font-bold tracking-tight text-[var(--color-text-primary)] md:text-6xl lg:text-7xl">
          Your agents can finally talk to each other.
        </h1>
        <p className="mx-auto mt-6 max-w-2xl text-lg leading-relaxed text-[var(--color-text-secondary)] md:text-xl">
          AGH is an agent runtime with a built-in network protocol. Spawn Claude Code, Codex, or
          Gemini CLI as managed sessions — then let them discover, message, and coordinate through
          an open wire format any harness can implement.
        </p>
        <div className="mt-10 flex flex-col items-center gap-4 sm:flex-row sm:justify-center">
          <Link
            href="/protocol"
            className={buttonVariants({
              variant: "default",
              size: "lg",
              className: "h-11 px-6 text-base font-medium",
            })}
          >
            Read the Protocol Spec
          </Link>
          <Link
            href="/runtime"
            className={buttonVariants({
              variant: "outline",
              size: "lg",
              className: "h-11 px-6 text-base font-medium",
            })}
          >
            Get Started
          </Link>
        </div>
      </div>
    </section>
  );
}
