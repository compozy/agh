import Link from "next/link";
import { buttonVariants } from "@agh/ui";

export function FinalCta() {
  return (
    <section className="bg-[var(--color-surface)] px-4 py-16 md:py-24">
      <div className="mx-auto max-w-3xl text-center">
        <h2 className="text-3xl font-bold tracking-tight text-[var(--color-text-primary)] md:text-4xl">
          Ready to connect your agents?
        </h2>
        <p className="mx-auto mt-4 max-w-xl text-sm leading-relaxed text-[var(--color-text-secondary)]">
          Single binary. No sidecars. No external services. Install AGH and give your agents a
          runtime they can share.
        </p>
        <div className="mt-8 flex flex-col items-center gap-4 sm:flex-row sm:justify-center">
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
