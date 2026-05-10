import Link from "next/link";
import { Eyebrow } from "@agh/ui";

export default function NotFound() {
  return (
    <main id="main-content" className="flex min-h-[calc(100dvh-3.5rem)] items-center px-4 py-20">
      <section className="mx-auto w-full max-w-[760px] rounded-(--radius-diagram) border border-(--color-divider) bg-(--color-surface) p-8 md:p-10">
        <Eyebrow case="upper" tone="muted" weight="semibold" className="text-accent">
          Not found
        </Eyebrow>
        <h1 className="mt-5 max-w-[12ch] text-site-error-title leading-none font-normal tracking-tight text-(--color-text-primary)">
          This route is not in the runtime.
        </h1>
        <p className="mt-5 max-w-[58ch] text-base leading-7 text-(--color-text-secondary)">
          The requested page is not part of the published AGH site. Use the runtime docs or the
          network protocol reference to re-enter the catalog.
        </p>
        <div className="mt-8 flex flex-col gap-3 sm:flex-row">
          <Link
            href="/runtime/"
            className="inline-flex h-10 items-center justify-center rounded-lg bg-accent px-5 text-sm font-medium text-white transition-colors hover:bg-(--color-accent-hover) active:translate-y-px"
          >
            Runtime docs
          </Link>
          <Link
            href="/protocol/"
            className="inline-flex h-10 items-center justify-center rounded-lg border border-(--color-divider) px-5 text-sm font-medium text-(--color-text-primary) transition-colors hover:border-accent hover:text-accent active:translate-y-px"
          >
            Network protocol
          </Link>
        </div>
      </section>
    </main>
  );
}
