import Link from "next/link";

export default function HomePage() {
  return (
    <main className="mx-auto flex max-w-screen-xl flex-col items-center px-4 py-24 text-center">
      <h1 className="text-4xl font-bold tracking-tight text-[var(--color-text-primary)]">
        Your agents can finally talk to each other.
      </h1>
      <p className="mt-4 max-w-2xl text-lg text-[var(--color-text-secondary)]">
        AGH is an agent runtime with a built-in network protocol. Spawn Claude Code, Codex, or
        Gemini CLI as managed sessions — then let them discover, message, and coordinate through an
        open wire format any harness can implement.
      </p>
      <div className="mt-8 flex gap-4">
        <Link
          href="/protocol"
          className="rounded-md bg-[var(--color-accent)] px-5 py-2.5 text-sm font-medium text-white transition-colors hover:bg-[var(--color-accent-hover)]"
        >
          Read the Protocol Spec
        </Link>
        <Link
          href="/runtime"
          className="rounded-md border border-[var(--color-divider)] px-5 py-2.5 text-sm font-medium text-[var(--color-text-primary)] transition-colors hover:bg-[var(--color-hover)]"
        >
          Get Started
        </Link>
      </div>
    </main>
  );
}
