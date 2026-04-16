import Link from "next/link";

const terminalLines = [
  { prompt: true, text: "agh daemon start" },
  { prompt: false, text: "daemon  pid=42871  status=online  uptime=0s" },
  { prompt: false, text: "" },
  { prompt: true, text: "agh session new --agent coder" },
  { prompt: false, text: "session  id=s_7kx9  agent=coder  driver=claude-code" },
  { prompt: false, text: "memory   workspace=/projects/api  skills=3 loaded" },
  { prompt: false, text: "" },
  { prompt: true, text: "agh network peers" },
  { prompt: false, text: "AGENT         DRIVER       CAPABILITIES" },
  { prompt: false, text: "deployer      codex-cli    deploy, infra, rollback" },
  { prompt: false, text: "analyst       gemini-cli   analysis, transforms" },
  { prompt: false, text: "reviewer      claude-code  review, architecture" },
  { prompt: false, text: "" },
  { prompt: true, text: "agh network delegate deployer --task 'deploy staging'" },
  { prompt: false, text: "delegate  agent=deployer  task=t_3mp1  status=accepted" },
];

const signalItems = [
  { label: "RUNTIME", value: "Local-first control plane" },
  { label: "NETWORK", value: "Open agent coordination" },
  { label: "STATE", value: "Sessions that survive restarts" },
  { label: "EDGE", value: "Cross-runtime delegation" },
];

export function Hero() {
  return (
    <section className="relative overflow-hidden border-b border-[var(--color-divider)] px-4 pt-8 pb-16 md:pt-12 md:pb-24">
      {/* Background mesh */}
      <div
        className="pointer-events-none absolute inset-0 bg-[length:100%_auto] bg-[position:0%_0%] bg-no-repeat opacity-20 mix-blend-screen"
        style={{ backgroundImage: "url('/hero-bg.png')" }}
      />
      <div className="relative mx-auto max-w-[var(--site-layout-width)]">
        <div className="grid gap-10 lg:grid-cols-[minmax(0,1.1fr)_minmax(340px,480px)] lg:items-start lg:gap-14">
          <div className="pt-4">
            <div className="flex items-center gap-3 font-mono text-[11px] font-medium uppercase tracking-[0.06em] text-[var(--color-text-tertiary)]">
              <span className="text-[var(--color-accent)]">AGH</span>
              <span className="h-px w-10 bg-[var(--color-divider)]" />
              <span>Agent Operating System</span>
            </div>

            <h1 className="mt-6 max-w-[14ch] text-[clamp(3.2rem,8vw,7rem)] leading-[0.88] font-normal tracking-[-0.04em] text-(--color-text-primary)">
              Your agents can finally talk to each other
            </h1>

            <p className="mt-6 max-w-[54ch] text-base md:text-lg leading-relaxed text-[var(--color-text-secondary)]">
              Run your agents as sessions you can inspect, resume, and govern. When one needs
              another, AGH Network lets them discover peers, delegate work, and exchange results
              across runtimes.
            </p>

            <div className="mt-8 flex flex-col items-start gap-3 sm:flex-row sm:flex-wrap">
              <Link
                href="/docs/getting-started"
                className="inline-flex h-9 items-center justify-center rounded-[8px] bg-[var(--color-accent)] px-5 text-sm font-medium text-(--color-accent-ink) transition-colors hover:bg-[var(--color-accent-hover)]"
              >
                Get Started
              </Link>
              <Link
                href="/protocol"
                className="inline-flex h-9 items-center justify-center rounded-[8px] border border-[var(--color-divider)] bg-[var(--color-surface)] px-5 text-sm font-medium text-[var(--color-text-primary)] transition-colors hover:border-[var(--color-accent)] hover:text-[var(--color-accent)]"
              >
                Explore AGH Network
              </Link>
            </div>

            <dl className="mt-10 grid gap-4 border-t border-[var(--color-divider)] pt-5 sm:grid-cols-2 xl:grid-cols-4">
              {signalItems.map(item => (
                <div key={item.label} className="min-w-0">
                  <dt className="font-mono text-[11px] font-medium uppercase tracking-[0.06em] text-[var(--color-text-tertiary)]">
                    {item.label}
                  </dt>
                  <dd className="mt-2 text-sm font-medium text-[var(--color-text-primary)]">
                    {item.value}
                  </dd>
                </div>
              ))}
            </dl>
          </div>

          <div className="relative pt-20">
            <div className="overflow-hidden rounded-[12px] border border-[var(--color-divider)] bg-[var(--color-canvas)]">
              <div className="flex items-center gap-2 border-b border-[var(--color-divider)] px-4 py-3">
                <span className="h-3 w-3 rounded-full bg-[var(--color-danger)] opacity-60" />
                <span className="h-3 w-3 rounded-full bg-[var(--color-warning)] opacity-60" />
                <span className="h-3 w-3 rounded-full bg-[var(--color-success)] opacity-60" />
                <span className="ml-3 font-mono text-[11px] font-medium uppercase tracking-[0.06em] text-[var(--color-text-tertiary)]">
                  terminal
                </span>
              </div>
              <div className="p-4 md:p-5">
                <pre className="font-mono text-[13px] leading-[1.7]">
                  {terminalLines.map((line, i) => (
                    <div key={i} className={line.text === "" ? "h-2" : ""}>
                      {line.prompt && <span className="text-[var(--color-accent)]">$ </span>}
                      <span
                        className={
                          line.prompt
                            ? "text-[var(--color-text-primary)]"
                            : "text-[var(--color-text-secondary)]"
                        }
                      >
                        {line.text}
                      </span>
                    </div>
                  ))}
                </pre>
              </div>
            </div>

            <div className="mt-4 pl-4 sm:pl-6">
              <p className="text-sm leading-relaxed text-[var(--color-text-tertiary)]">
                One daemon. Sessions, memory, skills, and a network of peers — all from the CLI you
                already use.
              </p>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
