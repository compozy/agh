// Changelog page — versioned timeline with grouped sections per release.

const RELEASES = [
  {
    tag: 'v0.4.2',
    date: 'Apr 04, 2026',
    title: 'Bridges, traces, dark-only UI',
    status: 'LATEST',
    statusTone: 'accent',
    summary: 'Slack and Discord bridges land. Trace becomes a first-class kind. The operator UI consolidates around dark mode.',
    groups: [
      {
        head: 'ADDED',
        tone: 'success',
        items: [
          'New bridge.slack package with channel, thread, and DM routing.',
          'New bridge.discord package — same surface as Slack, scoped per-guild.',
          'trace is now a wire kind. Every direct emits a step trace by default.',
          'Operator UI: replay timeline view per session.',
        ],
      },
      {
        head: 'CHANGED',
        tone: 'info',
        items: [
          'Operator UI ships dark-only. The light-mode CSS variables were removed.',
          'agh net send now requires --capability for direct kinds.',
          'Receipts include elapsed (ms) instead of started_at + ended_at.',
        ],
      },
      {
        head: 'FIXED',
        tone: 'warning',
        items: [
          'Memory namespaces no longer leak across workspaces under fast switch.',
          'whois resolution honors the local cache TTL.',
        ],
      },
    ],
  },
  {
    tag: 'v0.4.1',
    date: 'Mar 22, 2026',
    title: 'Memory namespaces',
    summary: 'Per-workspace memory isolation, with explicit promotion to global.',
    groups: [
      {
        head: 'ADDED',
        tone: 'success',
        items: [
          'Memory namespaces — workspace, global, ephemeral.',
          'agh memory promote moves a key from workspace to global.',
        ],
      },
      {
        head: 'CHANGED',
        tone: 'info',
        items: [
          'Default memory scope is now workspace, not global.',
        ],
      },
    ],
  },
  {
    tag: 'v0.4.0',
    date: 'Mar 02, 2026',
    title: 'agh-network/v0 stable',
    status: 'STABLE',
    statusTone: 'success',
    summary: 'The wire format freezes at seven kinds. Compatibility from this version forward.',
    groups: [
      {
        head: 'ADDED',
        tone: 'success',
        items: [
          'Stable release of agh-network/v0 — greet, whois, say, direct, recipe, receipt, trace.',
          'Capability descriptors are now part of whois.',
        ],
      },
      {
        head: 'BREAKING',
        tone: 'danger',
        items: [
          'Removed the legacy message kind. Use direct or say.',
          'Receipt schema requires status as one of accepted, fulfilled, rejected.',
        ],
      },
    ],
  },
  {
    tag: 'v0.3.6',
    date: 'Feb 18, 2026',
    title: 'Skill packaging + signing',
    summary: 'Skills can now be distributed as signed bundles. Verification happens at load time.',
    groups: [
      {
        head: 'ADDED',
        tone: 'success',
        items: [
          'agh skill pack and agh skill verify commands.',
          'Trust roots stored in $AGH_HOME/trust.',
        ],
      },
    ],
  },
];

function VersionFilter({ active = '0.4' }) {
  const items = [
    { v: 'all', label: 'All' },
    { v: '0.4', label: '0.4.x' },
    { v: '0.3', label: '0.3.x' },
    { v: '0.2', label: '0.2.x' },
  ];
  return (
    <div
      className="inline-flex items-center"
      style={{
        border: `1px solid ${TOKENS.divider}`,
        background: TOKENS.surfacePanel,
        borderRadius: 8,
        padding: 3,
        gap: 2,
      }}
    >
      {items.map(s => {
        const a = s.v === active;
        return (
          <button
            key={s.v}
            style={{
              height: 22,
              padding: '0 10px',
              borderRadius: 5,
              background: a ? TOKENS.elevated : 'transparent',
              color: a ? TOKENS.textPrimary : TOKENS.textTertiary,
              fontFamily: 'JetBrains Mono, monospace',
              fontSize: 10,
              fontWeight: 600,
              letterSpacing: '0.08em',
              textTransform: 'uppercase',
            }}
          >
            {s.label}
          </button>
        );
      })}
    </div>
  );
}

function ReleaseEntry({ r, last }) {
  return (
    <article
      className="grid gap-10"
      style={{
        gridTemplateColumns: '160px minmax(0, 1fr)',
        paddingTop: 32,
        paddingBottom: 48,
        borderBottom: last ? 'none' : `1px solid ${TOKENS.divider}`,
      }}
    >
      {/* left meta column */}
      <aside style={{ position: 'sticky', top: 96, alignSelf: 'start' }}>
        <MonoEyebrow tracking={0.08} color={TOKENS.textTertiary}>{r.date}</MonoEyebrow>
        <div className="mt-3 flex items-center gap-2">
          <MonoBadge tone="success">{r.tag}</MonoBadge>
          {r.status && <MonoBadge tone={r.statusTone}>{r.status}</MonoBadge>}
        </div>
        <div className="mt-5 flex flex-col gap-2">
          <a href="#" className="inline-flex items-center gap-1.5" style={{ fontSize: 12, color: TOKENS.textSecondary, fontFamily: 'Inter, sans-serif' }}>
            <GithubGlyph size={12} color={TOKENS.textTertiary} />
            <span>Compare on GitHub</span>
          </a>
          <a href="#" className="inline-flex items-center gap-1.5" style={{ fontSize: 12, color: TOKENS.textSecondary, fontFamily: 'Inter, sans-serif' }}>
            <Copy size={12} color={TOKENS.textTertiary} />
            <span>Copy permalink</span>
          </a>
        </div>
      </aside>

      {/* right body */}
      <div>
        <h2
          id={r.tag}
          style={{
            fontFamily: 'Inter, sans-serif',
            fontSize: 'clamp(1.6rem, 2.6vw, 2.1rem)',
            lineHeight: 1.1,
            letterSpacing: '-0.025em',
            fontWeight: 600,
            color: TOKENS.textPrimary,
          }}
        >
          {r.title}
        </h2>
        <p className="mt-4" style={{ fontSize: 16, lineHeight: 1.7, color: TOKENS.textSecondary, maxWidth: '64ch' }}>
          {r.summary}
        </p>

        <div className="mt-7 flex flex-col gap-6">
          {r.groups.map(g => (
            <div key={g.head}>
              <div className="flex items-center gap-2.5">
                <MonoBadge tone={g.tone}>{g.head}</MonoBadge>
                <span style={{ height: 1, flex: 1, background: TOKENS.divider }} />
              </div>
              <ul className="mt-3 flex flex-col gap-2.5" style={{ paddingLeft: 0, listStyle: 'none' }}>
                {g.items.map((it, idx) => (
                  <li key={idx} className="flex items-start gap-3" style={{ fontSize: 14.5, lineHeight: 1.6, color: TOKENS.textSecondary, fontFamily: 'Inter, sans-serif' }}>
                    <span
                      style={{
                        marginTop: 9,
                        width: 4, height: 4, borderRadius: 1,
                        background: TOKENS.accent,
                        flexShrink: 0,
                      }}
                    />
                    <span style={{ maxWidth: '64ch' }}>{renderInline(it)}</span>
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>
      </div>
    </article>
  );
}

// Inline mono detection — wraps tokens that look like commands or kinds.
function renderInline(text) {
  // Tokens to monospace: bare words like "trace", "direct", "agh net send",
  // CLI-style strings starting with "agh ", kebab/slug-with-dot, env vars.
  const monoRe = /(\$AGH_HOME\/trust|agh [a-z .-]+(?: --[a-z-]+)?|bridge\.[a-z]+|memory namespaces?|whois|greet|direct|receipt|recipe|trace|workspace, global, ephemeral|agh-network\/v0|message|status|elapsed|started_at|ended_at|--capability)/g;
  const parts = [];
  let last = 0;
  let m;
  while ((m = monoRe.exec(text)) !== null) {
    if (m.index > last) parts.push(text.slice(last, m.index));
    parts.push(
      <code
        key={m.index}
        style={{
          fontFamily: 'JetBrains Mono, ui-monospace, monospace',
          fontSize: '0.88em',
          border: `1px solid ${TOKENS.divider}`,
          borderRadius: 5,
          background: 'rgba(44,44,46,0.78)',
          padding: '0.06rem 0.34rem',
          color: TOKENS.textPrimary,
        }}
      >{m[0]}</code>
    );
    last = m.index + m[0].length;
  }
  if (last < text.length) parts.push(text.slice(last));
  return parts;
}

function ChangelogTocRail() {
  return (
    <aside style={{ position: 'sticky', top: 80 }}>
      <MonoEyebrow tracking={0.08}>All versions</MonoEyebrow>
      <ul className="mt-4 flex flex-col gap-2.5">
        {RELEASES.map((r, i) => (
          <li key={r.tag}>
            <a
              href={`#${r.tag}`}
              className="flex items-center justify-between"
              style={{
                fontSize: 13,
                color: i === 0 ? TOKENS.accent : TOKENS.textSecondary,
                fontFamily: 'JetBrains Mono, monospace',
                letterSpacing: '0.02em',
              }}
            >
              <span>{r.tag}</span>
              <span style={{ fontSize: 11, color: TOKENS.textTertiary }}>{r.date.slice(0, 6)}</span>
            </a>
          </li>
        ))}
      </ul>

      <div className="mt-9" style={{ borderTop: `1px solid ${TOKENS.divider}`, paddingTop: 18 }}>
        <MonoEyebrow tracking={0.08}>Subscribe</MonoEyebrow>
        <p className="mt-3" style={{ fontSize: 13, color: TOKENS.textSecondary, lineHeight: 1.5 }}>
          One email per release. Nothing else.
        </p>
        <div
          className="mt-3 flex items-center"
          style={{
            background: TOKENS.elevated,
            border: `1px solid ${TOKENS.divider}`,
            borderRadius: 8,
            height: 32,
            padding: '0 4px 0 10px',
          }}
        >
          <input
            placeholder="you@domain.com"
            className="flex-1 bg-transparent outline-none"
            style={{ fontSize: 12.5, color: TOKENS.textPrimary, fontFamily: 'Inter, sans-serif' }}
          />
          <button
            style={{
              height: 24,
              padding: '0 10px',
              borderRadius: 5,
              background: TOKENS.accent,
              color: '#FFFFFF',
              fontSize: 11,
              fontWeight: 500,
              fontFamily: 'Inter, sans-serif',
            }}
          >Join</button>
        </div>
        <div className="mt-4 flex items-center gap-2">
          <Rss size={11} color={TOKENS.textTertiary} />
          <a href="#" className="font-mono" style={{ fontSize: 10.5, color: TOKENS.textTertiary, letterSpacing: '0.06em', textTransform: 'uppercase' }}>
            /changelog/feed.xml
          </a>
        </div>
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
        <MonoEyebrow tracking={0.08} color={TOKENS.accent}>Upgrade</MonoEyebrow>
        <p className="mt-3" style={{ fontSize: 13, color: TOKENS.textSecondary, lineHeight: 1.55 }}>
          Latest is <span style={{ color: TOKENS.textPrimary, fontFamily: 'JetBrains Mono', fontSize: 13 }}>v0.4.2</span>.
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
          <span style={{ color: TOKENS.accent }}>$ </span>brew upgrade agh
        </div>
      </div>
    </aside>
  );
}

function ChangelogPage() {
  return (
    <div className="site-home" style={{ background: TOKENS.canvas, minHeight: '100%' }}>
      <SiteHeader active="changelog" />

      {/* Hero */}
      <section className="px-4" style={{ paddingTop: 56, paddingBottom: 36, borderBottom: `1px solid ${TOKENS.divider}` }}>
        <div className="mx-auto" style={{ maxWidth: 1200 }}>
          <div className="flex items-center gap-3">
            <MonoEyebrow color={TOKENS.accent}>CHANGELOG</MonoEyebrow>
            <span style={{ width: 36, height: 1, background: TOKENS.divider }} />
            <MonoEyebrow>What shipped, when, and why</MonoEyebrow>
          </div>
          <h1
            className="mt-6"
            style={{
              fontFamily: 'Playfair Display, serif',
              fontSize: 'clamp(2.6rem, 5.4vw, 4.6rem)',
              lineHeight: 0.98,
              letterSpacing: '-0.035em',
              fontWeight: 400,
              color: TOKENS.textPrimary,
              maxWidth: '20ch',
            }}
          >
            Every release of AGH, in plain language.
          </h1>
          <p className="mt-6" style={{ fontSize: 18, lineHeight: 1.6, color: TOKENS.textSecondary, maxWidth: '60ch' }}>
            Versioned, dated, and grouped by Added · Changed · Fixed · Breaking. Each release links straight to the diff and the upgrade notes. No marketing copy.
          </p>

          <div className="mt-9 flex flex-wrap items-center gap-4">
            <div className="flex items-center gap-2">
              <span style={{ width: 8, height: 8, borderRadius: '50%', background: TOKENS.success, display: 'inline-block' }} />
              <span className="font-mono" style={{ fontSize: 11, color: TOKENS.textPrimary, letterSpacing: '0.06em', textTransform: 'uppercase' }}>
                Latest · v0.4.2 · shipped Apr 04
              </span>
            </div>
            <span style={{ width: 1, height: 14, background: TOKENS.divider }} />
            <a
              href="#"
              className="inline-flex items-center gap-1.5"
              style={{ fontSize: 13, color: TOKENS.textSecondary, fontFamily: 'Inter, sans-serif' }}
            >
              <GithubGlyph size={13} color={TOKENS.textTertiary} />
              <span>compozy/agh · releases</span>
              <ArrowUpRight size={12} color={TOKENS.textTertiary} />
            </a>
            <a
              href="#"
              className="inline-flex items-center gap-1.5"
              style={{ fontSize: 13, color: TOKENS.textSecondary, fontFamily: 'Inter, sans-serif' }}
            >
              <Rss size={12} color={TOKENS.textTertiary} />
              <span className="font-mono" style={{ fontSize: 11, letterSpacing: '0.06em', textTransform: 'uppercase' }}>changelog/feed.xml</span>
            </a>
          </div>
        </div>
      </section>

      {/* Filter strip */}
      <section className="px-4" style={{ background: TOKENS.surface, borderBottom: `1px solid ${TOKENS.divider}` }}>
        <div className="mx-auto flex flex-wrap items-center gap-3" style={{ maxWidth: 1200, paddingTop: 14, paddingBottom: 14 }}>
          <VersionFilter active="0.4" />
          <div className="flex items-center gap-1.5 ml-2">
            <CategoryPill label="All changes" count={28} active />
            <CategoryPill label="Added" count={14} />
            <CategoryPill label="Changed" count={6} />
            <CategoryPill label="Fixed" count={5} />
            <CategoryPill label="Breaking" count={3} />
          </div>
          <div
            className="inline-flex items-center gap-2 ml-auto"
            style={{
              height: 28,
              borderRadius: 7,
              border: `1px solid ${TOKENS.divider}`,
              background: TOKENS.surfacePanel,
              padding: '0 8px',
              minWidth: 240,
            }}
          >
            <SearchIcon size={12} color={TOKENS.textTertiary} />
            <input
              placeholder="Search by version or keyword…"
              className="flex-1 bg-transparent outline-none"
              style={{ fontSize: 13, color: TOKENS.textPrimary, fontFamily: 'Inter, sans-serif' }}
            />
          </div>
        </div>
      </section>

      {/* Timeline */}
      <section className="px-4" style={{ paddingTop: 24, paddingBottom: 80 }}>
        <div className="mx-auto grid gap-12" style={{ maxWidth: 1200, gridTemplateColumns: 'minmax(0, 1fr) 280px' }}>
          <div>
            {RELEASES.map((r, i) => (
              <ReleaseEntry key={r.tag} r={r} last={i === RELEASES.length - 1} />
            ))}
            <div className="mt-2 flex items-center justify-between">
              <span style={{ fontSize: 13, color: TOKENS.textTertiary, fontFamily: 'Inter, sans-serif' }}>
                Showing 4 of 18 releases since <span style={{ color: TOKENS.textSecondary }}>v0.1.0</span>.
              </span>
              <button style={{
                height: 36, padding: '0 16px', borderRadius: 8,
                border: `1px solid ${TOKENS.divider}`, background: 'transparent',
                color: TOKENS.textPrimary, fontSize: 13, fontFamily: 'Inter, sans-serif', fontWeight: 500,
              }}>Load older releases</button>
            </div>
          </div>
          <ChangelogTocRail />
        </div>
      </section>

      <SiteFooter />
    </div>
  );
}

window.ChangelogPage = ChangelogPage;
