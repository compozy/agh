// Blog Post Detail page — long-form prose with wire-card embeds
function PostMasthead() {
  return (
    <section className="px-4 relative overflow-hidden" style={{ paddingTop: 56, paddingBottom: 36, borderBottom: `1px solid ${TOKENS.divider}` }}>
      <div className="mx-auto" style={{ maxWidth: 1200 }}>
        <div style={{ maxWidth: 760 }}>
        <a href="#" className="inline-flex items-center gap-1.5" style={{ color: TOKENS.textTertiary, fontSize: 13, fontFamily: 'Inter, sans-serif' }}>
          <ArrowLeft size={13} color={TOKENS.textTertiary} />
          <span>Back to blog</span>
        </a>
        <div className="mt-7 flex flex-wrap items-center gap-3">
          <MonoEyebrow color={TOKENS.accent}>BLOG</MonoEyebrow>
          <span style={{ width: 1, height: 12, background: TOKENS.divider }} />
          <MonoEyebrow>PROTOCOL</MonoEyebrow>
          <span style={{ width: 1, height: 12, background: TOKENS.divider }} />
          <MonoEyebrow>Apr 28, 2026</MonoEyebrow>
          <span style={{ width: 1, height: 12, background: TOKENS.divider }} />
          <span className="inline-flex items-center gap-1.5" style={{ color: TOKENS.textTertiary, fontSize: 11 }}>
            <Clock size={11} color={TOKENS.textTertiary} />
            <span className="font-mono" style={{ letterSpacing: '0.06em', textTransform: 'uppercase' }}>14 min read</span>
          </span>
        </div>
        <h1
          className="mt-7"
          style={{
            fontFamily: 'Playfair Display, serif',
            fontSize: 'clamp(2.4rem, 4.4vw, 3.6rem)',
            lineHeight: 1.0,
            letterSpacing: '-0.035em',
            fontWeight: 400,
            color: TOKENS.textPrimary,
          }}
        >
          Why agents need an open network protocol, not another SDK.
        </h1>
        <p className="mt-6" style={{ fontSize: 19, lineHeight: 1.5, color: TOKENS.textSecondary, maxWidth: '58ch' }}>
          Six months of <span style={{ color: TOKENS.textPrimary }}>agh-network/v0</span> in the wild. What shipping a JSON-over-NATS spec teaches you about agent coordination, receipts as durability, and why every framework eventually rebuilds the same seven kinds.
        </p>

        {/* author row */}
        <div className="mt-9 flex flex-wrap items-center justify-between gap-4">
          <div className="flex items-center gap-3">
            <span
              style={{
                width: 36, height: 36, borderRadius: '50%',
                background: TOKENS.elevated,
                color: TOKENS.textPrimary,
                display: 'inline-flex', alignItems: 'center', justifyContent: 'center',
                fontFamily: 'Inter, sans-serif', fontSize: 14, fontWeight: 600,
              }}
            >P</span>
            <div>
              <p style={{ fontSize: 14, fontWeight: 500, color: TOKENS.textPrimary, fontFamily: 'Inter, sans-serif' }}>Pedro Nauck</p>
              <p className="font-mono" style={{ fontSize: 11, color: TOKENS.textLabel, letterSpacing: '0.06em', textTransform: 'uppercase' }}>
                pedronauck · runtime
              </p>
            </div>
          </div>
          <div className="flex items-center gap-1">
            {[
              { icon: Bookmark, label: 'Save' },
              { icon: Share, label: 'Share' },
              { icon: Copy, label: 'Copy link' },
            ].map(b => (
              <button
                key={b.label}
                aria-label={b.label}
                className="inline-flex h-9 w-9 items-center justify-center"
                style={{ color: TOKENS.textSecondary, borderRadius: 9999 }}
              >
                <b.icon size={14} color={TOKENS.textSecondary} />
              </button>
            ))}
          </div>
        </div>
        </div>
      </div>
    </section>
  );
}

function ProseHeading({ children }) {
  return (
    <h2
      style={{
        marginTop: 64,
        paddingTop: 16,
        borderTop: `1px solid ${TOKENS.divider}`,
        fontFamily: 'Inter, sans-serif',
        fontSize: 'clamp(1.7rem, 3vw, 2.45rem)',
        lineHeight: 1.05,
        letterSpacing: '-0.035em',
        fontWeight: 600,
        color: TOKENS.textPrimary,
      }}
    >
      {children}
    </h2>
  );
}
function Prose({ children }) {
  return (
    <p style={{
      marginTop: 20,
      fontSize: 16,
      lineHeight: 1.8,
      color: TOKENS.textSecondary,
      fontFamily: 'Inter, sans-serif',
      maxWidth: '72ch',
    }}>{children}</p>
  );
}
function Mono({ children }) {
  return (
    <code style={{
      fontFamily: 'JetBrains Mono, ui-monospace, monospace',
      fontSize: '0.9em',
      border: `1px solid ${TOKENS.divider}`,
      borderRadius: 6,
      background: 'rgba(44,44,46,0.78)',
      padding: '0.12rem 0.38rem',
      color: TOKENS.textPrimary,
    }}>{children}</code>
  );
}

function CodeBlock() {
  const lines = [
    { p: '$', t: 'agh net send direct --to bravo --capability deploy.preview' },
    { p: '', t: '' },
    { p: '#', t: 'wire trace' },
    { p: '', t: '00:00.041  greet     alpha → bravo' },
    { p: '', t: '00:00.108  direct    alpha → bravo   intent=deploy.preview' },
    { p: '', t: '00:00.382  receipt   bravo → alpha   status=accepted' },
    { p: '', t: '00:00.384  trace     bravo → *       step=allocate.runner' },
    { p: '', t: '00:01.207  trace     bravo → *       step=apply.manifest' },
    { p: '', t: '00:01.918  receipt   bravo → alpha   status=fulfilled' },
  ];
  return (
    <div
      className="relative"
      style={{
        marginTop: 28,
        background: TOKENS.canvasDeep,
        border: `1px solid ${TOKENS.divider}`,
        borderRadius: 12,
        padding: '18px 20px 20px',
      }}
    >
      <div className="flex items-center justify-between mb-3">
        <MonoEyebrow tracking={0.08}>SHELL · WIRE TRACE</MonoEyebrow>
        <button aria-label="Copy" style={{ color: TOKENS.textTertiary }}>
          <Copy size={13} color={TOKENS.textTertiary} />
        </button>
      </div>
      <pre style={{
        fontFamily: 'JetBrains Mono, ui-monospace, monospace',
        fontSize: 13,
        lineHeight: 1.7,
        color: TOKENS.textPrimary,
        margin: 0,
        whiteSpace: 'pre-wrap',
      }}>
        {lines.map((l, i) => (
          <div key={i}>
            {l.p === '$' && <span style={{ color: TOKENS.accent }}>$ </span>}
            {l.p === '#' && <span style={{ color: TOKENS.textTertiary }}># </span>}
            <span style={{ color: l.p === '#' ? TOKENS.textTertiary : TOKENS.textPrimary }}>{l.t}</span>
          </div>
        ))}
      </pre>
    </div>
  );
}

function Callout() {
  return (
    <aside
      style={{
        marginTop: 28,
        background: TOKENS.surface,
        border: `1px solid ${TOKENS.divider}`,
        borderLeft: `4px solid ${TOKENS.accent}`,
        borderRadius: 12,
        padding: '18px 22px',
      }}
    >
      <div className="flex items-center gap-2">
        <Sparkles size={14} color={TOKENS.accent} />
        <MonoEyebrow tracking={0.08} color={TOKENS.accent}>Engineering note</MonoEyebrow>
      </div>
      <p className="mt-3" style={{ fontSize: 15, lineHeight: 1.6, color: TOKENS.textPrimary, fontFamily: 'Inter, sans-serif' }}>
        Receipts are not a feature. They are the only reason delegation is safe to leave unattended.
      </p>
    </aside>
  );
}

function WireCard() {
  return (
    <div
      style={{
        marginTop: 28,
        background: TOKENS.surface,
        border: `1px solid ${TOKENS.divider}`,
        borderRadius: 6,
        maxWidth: 520,
      }}
    >
      <div
        style={{
          background: TOKENS.canvasDeep,
          borderBottom: `1px solid ${TOKENS.divider}`,
          padding: '6px 10px',
          fontFamily: 'JetBrains Mono, ui-monospace, monospace',
          fontSize: 10.5,
          letterSpacing: '0.06em',
          textTransform: 'uppercase',
          color: TOKENS.textTertiary,
        }}
      >
        kind=receipt · v0
      </div>
      <div style={{ padding: '10px 12px', fontFamily: 'JetBrains Mono, ui-monospace, monospace', fontSize: 11.5, lineHeight: 1.65, color: TOKENS.textSecondary }}>
        <div><span style={{ color: TOKENS.textTertiary }}>id</span>      <span style={{ color: TOKENS.textPrimary }}>rcpt_01HX4Q…</span></div>
        <div><span style={{ color: TOKENS.textTertiary }}>from</span>    <span style={{ color: TOKENS.textPrimary }}>agent.bravo</span></div>
        <div><span style={{ color: TOKENS.textTertiary }}>to</span>      <span style={{ color: TOKENS.textPrimary }}>agent.alpha</span></div>
        <div><span style={{ color: TOKENS.textTertiary }}>status</span>  <span style={{ color: TOKENS.success }}>fulfilled</span></div>
        <div><span style={{ color: TOKENS.textTertiary }}>elapsed</span> <span style={{ color: TOKENS.textPrimary }}>1.918s</span></div>
      </div>
      <div
        style={{
          background: TOKENS.canvasDeep,
          borderTop: `1px solid ${TOKENS.divider}`,
          padding: '6px 10px',
          display: 'flex', alignItems: 'center', gap: 12,
        }}
      >
        <button style={{ fontFamily: 'JetBrains Mono', fontSize: 10.5, color: TOKENS.textTertiary, letterSpacing: '0.06em', textTransform: 'uppercase' }}>Inspect →</button>
        <button style={{ fontFamily: 'JetBrains Mono', fontSize: 10.5, color: TOKENS.textTertiary, letterSpacing: '0.06em', textTransform: 'uppercase' }}>Replay</button>
      </div>
    </div>
  );
}

function PullQuote() {
  return (
    <blockquote
      style={{
        marginTop: 36,
        marginBottom: 12,
        paddingLeft: 24,
        borderLeft: `2px solid ${TOKENS.accent}`,
        fontFamily: 'Playfair Display, serif',
        fontSize: 'clamp(1.5rem, 2.4vw, 1.95rem)',
        lineHeight: 1.25,
        letterSpacing: '-0.02em',
        color: TOKENS.textPrimary,
        fontWeight: 400,
        maxWidth: '40ch',
      }}
    >
      An SDK gives you methods. A protocol gives you a network.
    </blockquote>
  );
}

function TocRail() {
  const items = [
    { id: 'frameworks', label: 'Why frameworks fail at coordination' },
    { id: 'kinds', label: 'The seven kinds, briefly', active: true },
    { id: 'receipts', label: 'Receipts as durability' },
    { id: 'trace', label: 'Trace, the cheap superpower' },
    { id: 'next', label: 'What v1 buys us' },
  ];
  return (
    <aside style={{ position: 'sticky', top: 80 }}>
      <MonoEyebrow tracking={0.08}>On this page</MonoEyebrow>
      <ul className="mt-4 flex flex-col gap-2.5">
        {items.map(it => (
          <li key={it.id}>
            <a
              href={`#${it.id}`}
              style={{
                fontSize: 13,
                color: it.active ? TOKENS.accent : TOKENS.textSecondary,
                fontFamily: 'Inter, sans-serif',
                lineHeight: 1.4,
              }}
            >
              {it.label}
            </a>
          </li>
        ))}
      </ul>

      <div className="mt-9" style={{ borderTop: `1px solid ${TOKENS.divider}`, paddingTop: 18 }}>
        <MonoEyebrow tracking={0.08}>References</MonoEyebrow>
        <ul className="mt-4 flex flex-col gap-3">
          {[
            { label: 'agh-network/v0 spec', mono: true },
            { label: 'RFC-014 — capability versioning', mono: true },
            { label: 'web/CLAUDE.md', mono: true },
          ].map(r => (
            <li key={r.label}>
              <a
                href="#"
                className="inline-flex items-center gap-1.5"
                style={{ fontSize: 12, color: TOKENS.textSecondary, fontFamily: 'JetBrains Mono, monospace', letterSpacing: '0.02em' }}
              >
                {r.label}
                <ArrowUpRight size={11} color={TOKENS.textTertiary} />
              </a>
            </li>
          ))}
        </ul>
      </div>

      <div
        className="mt-9"
        style={{
          background: TOKENS.surface,
          border: `1px solid ${TOKENS.divider}`,
          borderRadius: 12,
          padding: 18,
        }}
      >
        <MonoEyebrow tracking={0.08} color={TOKENS.accent}>Run it locally</MonoEyebrow>
        <p className="mt-3" style={{ fontSize: 13, color: TOKENS.textSecondary, lineHeight: 1.55 }}>
          One binary. macOS or Linux.
        </p>
        <div
          className="mt-3"
          style={{
            background: TOKENS.canvasDeep,
            border: `1px solid ${TOKENS.divider}`,
            borderRadius: 8,
            padding: '8px 10px',
            fontFamily: 'JetBrains Mono, monospace',
            fontSize: 12,
            color: TOKENS.textPrimary,
          }}
        >
          <span style={{ color: TOKENS.accent }}>$ </span>brew install compozy/agh
        </div>
      </div>
    </aside>
  );
}

function ContinueReading() {
  const more = [
    { cat: 'NETWORK', title: 'Receipts: making delegation auditable', date: 'Apr 11', read: '9 min' },
    { cat: 'PROTOCOL', title: 'whois, greet, say — naming the seven kinds', date: 'Mar 20', read: '7 min' },
    { cat: 'RUNTIME', title: 'Sessions that survive a laptop closing', date: 'Apr 22', read: '8 min' },
  ];
  return (
    <section className="px-4" style={{ background: TOKENS.surface, borderTop: `1px solid ${TOKENS.divider}`, paddingTop: 64, paddingBottom: 80 }}>
      <div className="mx-auto" style={{ maxWidth: 1200 }}>
        <div className="flex items-baseline justify-between">
          <div className="flex items-center gap-3">
            <MonoEyebrow tracking={0.08}>Continue reading</MonoEyebrow>
            <span style={{ width: 36, height: 1, background: TOKENS.divider }} />
            <span style={{ fontSize: 13, color: TOKENS.textTertiary, fontFamily: 'Inter, sans-serif' }}>Picked for this post</span>
          </div>
          <a href="#" style={{ fontSize: 13, color: TOKENS.textSecondary, fontFamily: 'Inter, sans-serif' }}>All posts →</a>
        </div>
        <div className="mt-6 grid gap-5" style={{ gridTemplateColumns: 'repeat(3, minmax(0, 1fr))' }}>
          {more.map(p => (
            <article key={p.title} style={{
              background: TOKENS.canvas,
              border: `1px solid ${TOKENS.divider}`,
              borderRadius: 12,
              padding: 22,
            }}>
              <div className="flex items-center gap-2.5">
                <MonoEyebrow color={TOKENS.accent}>{p.cat}</MonoEyebrow>
                <span style={{ width: 1, height: 10, background: TOKENS.divider }} />
                <MonoEyebrow color={TOKENS.textTertiary}>{p.date}</MonoEyebrow>
              </div>
              <h3 className="mt-4" style={{
                fontFamily: 'Inter, sans-serif',
                fontSize: 18,
                fontWeight: 500,
                letterSpacing: '-0.02em',
                lineHeight: 1.25,
                color: TOKENS.textPrimary,
              }}>{p.title}</h3>
              <div className="mt-5 flex items-center justify-between">
                <span className="inline-flex items-center gap-1.5" style={{ color: TOKENS.textTertiary, fontSize: 11 }}>
                  <Clock size={11} color={TOKENS.textTertiary} />
                  <span className="font-mono" style={{ letterSpacing: '0.06em', textTransform: 'uppercase' }}>{p.read}</span>
                </span>
                <span style={{ color: TOKENS.accent }}><ArrowUpRight size={14} color={TOKENS.accent} /></span>
              </div>
            </article>
          ))}
        </div>
      </div>
    </section>
  );
}

function BlogPost() {
  return (
    <div className="site-home" style={{ background: TOKENS.canvas, minHeight: '100%' }}>
      <SiteHeader active="blog" />
      <PostMasthead />

      <section className="px-4" style={{ paddingTop: 32, paddingBottom: 64 }}>
        <div
          className="mx-auto grid gap-12"
          style={{
            maxWidth: 1200,
            gridTemplateColumns: 'minmax(0, 760px) 220px',
          }}
        >
          <article className="site-doc-body">
            <Prose>
              The first version of AGH had no protocol. It had a Go SDK, a TypeScript client, and a list of TODOs about
              cross-process coordination. We thought we'd build a runtime first, sort networking later. We were wrong in
              the same way every framework before us has been wrong: <Mono>localhost</Mono> is a special case, not a starting point.
            </Prose>
            <Prose>
              When you ship an SDK, the network is implicit. When you ship a wire format, the network is the contract. The
              first lets agents talk to your library; the second lets agents talk to each other.
            </Prose>

            <PullQuote />

            <h2 id="frameworks">
              <ProseHeading>Why frameworks fail at coordination</ProseHeading>
            </h2>
            <Prose>
              Every agent framework eventually ships some flavor of "and then call the other agent". Most of them
              hand-roll an in-process bus, a JSON-RPC layer, or a thinly-wrapped queue. The result is correct on one machine
              and a tar pit on two. <Mono>agh-network/v0</Mono> is the smallest spec we could write that survives both.
            </Prose>

            <CodeBlock />

            <h2 id="kinds">
              <ProseHeading>The seven kinds, briefly</ProseHeading>
            </h2>
            <Prose>
              The wire only knows seven kinds. Most coordination problems collapse onto them. <Mono>greet</Mono> announces
              presence; <Mono>whois</Mono> resolves identity; <Mono>say</Mono> is broadcast; <Mono>direct</Mono> is delegated work;
              <Mono>recipe</Mono> carries reusable plans; <Mono>receipt</Mono> closes the loop; <Mono>trace</Mono> emits the steps in between.
            </Prose>

            <Callout />

            <h2 id="receipts">
              <ProseHeading>Receipts as durability</ProseHeading>
            </h2>
            <Prose>
              A direct without a receipt is fire-and-forget. A direct with a receipt is a transaction the operator can
              audit, retry, or roll up into a higher-level capability. The shape is intentionally boring — id, status,
              elapsed, optional payload.
            </Prose>

            <WireCard />

            <h2 id="trace">
              <ProseHeading>Trace, the cheap superpower</ProseHeading>
            </h2>
            <Prose>
              Trace was the last kind we added and the one we now reach for first. It is structured logging at the protocol
              level — every step a delegated agent takes, broadcast on the same bus the operator already listens to. No
              extra exporters. No tail latency. No agreement to argue about.
            </Prose>
            <Prose>
              In practice, trace events are how you debug a runaway loop without ever opening a stack trace. They are also
              how the operator UI builds its replay timeline.
            </Prose>

            <h2 id="next">
              <ProseHeading>What v1 buys us</ProseHeading>
            </h2>
            <Prose>
              v0 was the proof. v1 will tighten capabilities, add receipt bundling, and resolve the open questions around
              cross-tenant <Mono>greet</Mono> filtering. The wire stays seven kinds — that is the part we don't get to change.
            </Prose>
            <Prose>
              If you want to read along, the spec lives at <Mono>agh.network/protocol</Mono>, and the runtime is one
              <Mono> brew install</Mono> away.
            </Prose>

            {/* Author footer */}
            <div className="mt-16 flex items-center justify-between gap-6 flex-wrap" style={{ borderTop: `1px solid ${TOKENS.divider}`, paddingTop: 28 }}>
              <div className="flex items-center gap-3">
                <span
                  style={{
                    width: 44, height: 44, borderRadius: '50%',
                    background: TOKENS.elevated,
                    color: TOKENS.textPrimary,
                    display: 'inline-flex', alignItems: 'center', justifyContent: 'center',
                    fontFamily: 'Inter, sans-serif', fontSize: 16, fontWeight: 600,
                  }}
                >P</span>
                <div>
                  <p style={{ fontSize: 15, fontWeight: 500, color: TOKENS.textPrimary, fontFamily: 'Inter, sans-serif' }}>Pedro Nauck</p>
                  <p style={{ fontSize: 13, color: TOKENS.textSecondary, fontFamily: 'Inter, sans-serif' }}>Working on the runtime and the protocol.</p>
                </div>
              </div>
              <div className="flex items-center gap-2">
                <button style={{
                  height: 36, padding: '0 14px', borderRadius: 9999,
                  background: 'transparent', border: `1px solid ${TOKENS.divider}`,
                  color: TOKENS.textPrimary, fontSize: 13, fontFamily: 'Inter, sans-serif',
                }}>Follow</button>
                <button aria-label="GitHub" className="inline-flex h-9 w-9 items-center justify-center rounded-full" style={{ color: TOKENS.textSecondary }}>
                  <GithubGlyph size={14} color={TOKENS.textSecondary} />
                </button>
              </div>
            </div>
          </article>

          <TocRail />
        </div>
      </section>

      <ContinueReading />
      <SiteFooter />
    </div>
  );
}

window.BlogPost = BlogPost;
