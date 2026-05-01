// Blog Index page — featured post + grid + sidebar rail
const POSTS = [
  {
    cat: 'PROTOCOL',
    catTone: 'accent',
    title: 'Why agents need a wire protocol, not another SDK',
    dek: 'Six months of agh-network/v0 in the wild — what shipping a JSON-over-NATS spec teaches you about agent coordination, receipts as durability, and why every framework eventually rebuilds the same seven kinds.',
    author: 'pedronauck',
    avatar: 'P',
    date: 'Apr 28, 2026',
    read: '14 min',
    kinds: ['greet', 'whois', 'direct', 'receipt'],
  },
  {
    cat: 'RUNTIME',
    catTone: 'info',
    title: 'Sessions that survive a laptop closing',
    dek: 'Durable session storage in v0.4. The walk-through and the trade-offs.',
    author: 'mariarib',
    date: 'Apr 22, 2026',
    read: '8 min',
  },
  {
    cat: 'ENGINEERING',
    catTone: 'neutral',
    title: 'Replacing Postgres with a single SQLite file',
    dek: 'Why we ripped out the database and what we kept from a year of operating it.',
    author: 'jpvalente',
    date: 'Apr 18, 2026',
    read: '11 min',
  },
  {
    cat: 'NETWORK',
    catTone: 'accent',
    title: 'Receipts: making delegation auditable',
    dek: 'Every direct kind ends with a receipt. Here is the math, the schema, and the failure modes.',
    author: 'pedronauck',
    date: 'Apr 11, 2026',
    read: '9 min',
  },
  {
    cat: 'CHANGELOG',
    catTone: 'success',
    title: 'v0.4.2 — bridges, traces, and the new operator UI',
    dek: 'Slack and Discord bridges land. Trace becomes a first-class kind. Operator UI now ships dark-only.',
    author: 'team',
    date: 'Apr 04, 2026',
    read: '5 min',
  },
  {
    cat: 'ENGINEERING',
    catTone: 'neutral',
    title: 'Building the AGH operator UI in 1200 lines of Tailwind',
    dek: 'The full audit of every component on web/, what we cut, and why we kept the dark warm gray.',
    author: 'mariarib',
    date: 'Mar 27, 2026',
    read: '12 min',
  },
  {
    cat: 'PROTOCOL',
    catTone: 'accent',
    title: 'whois, greet, say — naming the seven kinds',
    dek: 'Notes on protocol vocabulary. Why direct and not message. Why recipe survived.',
    author: 'pedronauck',
    date: 'Mar 20, 2026',
    read: '7 min',
  },
];

function FeaturedPost({ p }) {
  return (
    <article
      className="grid gap-10 lg:grid-cols-[minmax(0,1.05fr)_minmax(0,1fr)] lg:items-center"
      style={{
        background: TOKENS.surface,
        border: `1px solid ${TOKENS.divider}`,
        borderRadius: 12,
        padding: 28,
      }}
    >
      <div className="order-2 lg:order-1">
        <div className="flex items-center gap-3">
          <MonoBadge tone="accent">FEATURED</MonoBadge>
          <span style={{ width: 1, height: 12, background: TOKENS.divider }} />
          <MonoEyebrow>{p.cat}</MonoEyebrow>
          <span style={{ width: 1, height: 12, background: TOKENS.divider }} />
          <MonoEyebrow color={TOKENS.textTertiary}>{p.date}</MonoEyebrow>
        </div>
        <h2
          className="mt-6"
          style={{
            fontFamily: 'Playfair Display, serif',
            fontSize: 'clamp(2rem, 3.4vw, 2.8rem)',
            lineHeight: 1.02,
            letterSpacing: '-0.03em',
            fontWeight: 400,
            color: TOKENS.textPrimary,
            maxWidth: '20ch',
          }}
        >
          {p.title}
        </h2>
        <p className="mt-5" style={{ fontSize: 16, lineHeight: 1.6, color: TOKENS.textSecondary, maxWidth: '54ch' }}>
          {p.dek}
        </p>
        <div className="mt-7 flex flex-wrap items-center gap-2">
          {p.kinds.map(k => <KindChip key={k} kind={k} />)}
        </div>
        <div className="mt-7 flex items-center gap-4">
          <div className="flex items-center gap-2.5">
            <span
              style={{
                width: 28, height: 28, borderRadius: '50%',
                background: TOKENS.elevated,
                color: TOKENS.textPrimary,
                display: 'inline-flex', alignItems: 'center', justifyContent: 'center',
                fontFamily: 'Inter, sans-serif', fontSize: 12, fontWeight: 600,
              }}
            >{p.avatar}</span>
            <span className="font-mono" style={{ fontSize: 11, letterSpacing: '0.06em', color: TOKENS.textLabel, textTransform: 'uppercase' }}>
              {p.author}
            </span>
          </div>
          <span style={{ width: 1, height: 12, background: TOKENS.divider }} />
          <div className="inline-flex items-center gap-1.5" style={{ color: TOKENS.textTertiary, fontSize: 12 }}>
            <Clock size={12} color={TOKENS.textTertiary} />
            <span style={{ fontFamily: 'Inter, sans-serif' }}>{p.read} read</span>
          </div>
          <a
            href="#"
            className="ml-auto inline-flex items-center gap-1.5"
            style={{ color: TOKENS.accent, fontSize: 14, fontWeight: 500 }}
          >
            Read post <ArrowUpRight size={14} color={TOKENS.accent} />
          </a>
        </div>
      </div>

      {/* visual right side — protocol diagram placeholder, made of rules + chips */}
      <div className="order-1 lg:order-2">
        <div
          className="relative"
          style={{
            background: TOKENS.canvasDeep,
            border: `1px solid ${TOKENS.divider}`,
            borderRadius: 12,
            padding: 26,
            minHeight: 340,
          }}
        >
          <div className="flex items-center justify-between">
            <MonoEyebrow tracking={0.08}>agh-network/v0</MonoEyebrow>
            <span className="inline-flex items-center gap-1.5">
              <span style={{ width: 6, height: 6, borderRadius: '50%', background: TOKENS.success }} />
              <MonoEyebrow tracking={0.06} color={TOKENS.success}>LIVE</MonoEyebrow>
            </span>
          </div>
          {/* nodes */}
          <div className="mt-10 grid grid-cols-3 gap-4">
            {[
              { id: 'agent.alpha', role: 'planner' },
              { id: 'agent.bravo', role: 'executor', highlight: true },
              { id: 'agent.charlie', role: 'guardian' },
            ].map(n => (
              <div
                key={n.id}
                style={{
                  background: TOKENS.surface,
                  border: `1px solid ${n.highlight ? TOKENS.accent : TOKENS.divider}`,
                  borderRadius: 8,
                  padding: '10px 12px',
                }}
              >
                <p className="font-mono" style={{ fontSize: 11, color: n.highlight ? TOKENS.accent : TOKENS.textPrimary }}>{n.id}</p>
                <p className="font-mono mt-1" style={{ fontSize: 9.5, letterSpacing: '0.08em', textTransform: 'uppercase', color: TOKENS.textTertiary }}>{n.role}</p>
              </div>
            ))}
          </div>

          {/* wire log */}
          <div
            className="mt-6"
            style={{
              background: TOKENS.surface,
              border: `1px solid ${TOKENS.divider}`,
              borderRadius: 8,
              padding: '10px 12px',
            }}
          >
            <div className="flex items-center justify-between" style={{ borderBottom: `1px solid ${TOKENS.divider}`, paddingBottom: 8 }}>
              <MonoEyebrow tracking={0.06}>WIRE TRACE</MonoEyebrow>
              <MonoEyebrow color={TOKENS.textTertiary}>4 events</MonoEyebrow>
            </div>
            <ul className="mt-3 flex flex-col gap-2">
              {[
                { kind: 'greet', from: 'alpha', to: 'bravo', t: '00:00.041' },
                { kind: 'direct', from: 'alpha', to: 'bravo', t: '00:00.108' },
                { kind: 'receipt', from: 'bravo', to: 'alpha', t: '00:00.382' },
                { kind: 'trace', from: 'bravo', to: '*', t: '00:00.384' },
              ].map((r, i) => (
                <li key={i} className="flex items-center gap-3">
                  <span className="font-mono" style={{ fontSize: 10, color: TOKENS.textTertiary, width: 64 }}>{r.t}</span>
                  <KindChip kind={r.kind} />
                  <span className="font-mono" style={{ fontSize: 11, color: TOKENS.textSecondary }}>
                    {r.from} <span style={{ color: TOKENS.accent }}>→</span> {r.to}
                  </span>
                </li>
              ))}
            </ul>
          </div>
        </div>
      </div>
    </article>
  );
}

function PostCard({ p }) {
  return (
    <article
      className="group flex flex-col"
      style={{
        background: TOKENS.surface,
        border: `1px solid ${TOKENS.divider}`,
        borderRadius: 12,
        padding: 22,
        minHeight: 230,
      }}
    >
      <div className="flex items-center gap-2.5">
        <MonoEyebrow color={TOKENS.accent}>{p.cat}</MonoEyebrow>
        <span style={{ width: 1, height: 10, background: TOKENS.divider }} />
        <MonoEyebrow color={TOKENS.textTertiary}>{p.date}</MonoEyebrow>
      </div>
      <h3
        className="mt-4"
        style={{
          fontFamily: 'Inter, sans-serif',
          fontSize: 20,
          fontWeight: 500,
          letterSpacing: '-0.02em',
          lineHeight: 1.25,
          color: TOKENS.textPrimary,
        }}
      >
        {p.title}
      </h3>
      <p className="mt-3" style={{ fontSize: 14, lineHeight: 1.6, color: TOKENS.textSecondary }}>
        {p.dek}
      </p>
      <div className="mt-auto pt-5 flex items-center justify-between" style={{ borderTop: `1px solid ${TOKENS.divider}`, marginTop: 22, paddingTop: 14 }}>
        <span className="font-mono" style={{ fontSize: 11, letterSpacing: '0.06em', color: TOKENS.textLabel, textTransform: 'uppercase' }}>
          {p.author}
        </span>
        <span className="inline-flex items-center gap-1.5" style={{ color: TOKENS.textTertiary, fontSize: 11 }}>
          <Clock size={11} color={TOKENS.textTertiary} />
          <span className="font-mono" style={{ letterSpacing: '0.04em' }}>{p.read}</span>
        </span>
      </div>
    </article>
  );
}

function ChangelogRail() {
  const items = [
    { tag: 'v0.4.2', date: 'Apr 04', kind: 'release', title: 'Bridges, traces, dark-only UI' },
    { tag: 'v0.4.1', date: 'Mar 22', kind: 'release', title: 'Memory namespaces' },
    { tag: 'v0.4.0', date: 'Mar 02', kind: 'release', title: 'agh-network/v0 stable' },
    { tag: 'v0.3.6', date: 'Feb 18', kind: 'release', title: 'Skill packaging + signing' },
  ];
  return (
    <aside
      style={{
        background: TOKENS.surface,
        border: `1px solid ${TOKENS.divider}`,
        borderRadius: 12,
        padding: 22,
      }}
    >
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <span
            style={{
              width: 6,
              height: 6,
              borderRadius: '50%',
              background: TOKENS.success,
              display: 'inline-block',
            }}
          />
          <MonoEyebrow tracking={0.08}>Changelog</MonoEyebrow>
        </div>
        <a href="#" style={{ fontSize: 12, color: TOKENS.textTertiary, fontFamily: 'Inter, sans-serif' }}>
          all versions →
        </a>
      </div>
      <ul className="mt-5 flex flex-col">
        {items.map((it, i) => (
          <li
            key={it.tag}
            className="flex items-start gap-3 py-3"
            style={{ borderBottom: i < items.length - 1 ? `1px solid ${TOKENS.divider}` : 'none' }}
          >
            <span style={{ width: 56, flexShrink: 0 }}>
              <MonoBadge tone="success">{it.tag}</MonoBadge>
            </span>
            <div className="flex-1 min-w-0">
              <p style={{ fontSize: 13, color: TOKENS.textPrimary, fontFamily: 'Inter, sans-serif', lineHeight: 1.4 }}>
                {it.title}
              </p>
              <p className="font-mono mt-1" style={{ fontSize: 10, color: TOKENS.textTertiary, letterSpacing: '0.06em', textTransform: 'uppercase' }}>
                {it.date} · 2026
              </p>
            </div>
          </li>
        ))}
      </ul>
      <a
        href="#"
        className="mt-4 inline-flex items-center justify-center"
        style={{
          width: '100%',
          height: 32,
          borderRadius: 8,
          border: `1px solid ${TOKENS.divider}`,
          color: TOKENS.textPrimary,
          fontSize: 12,
          fontFamily: 'Inter, sans-serif',
          fontWeight: 500,
        }}
      >
        Open the changelog
      </a>
    </aside>
  );
}

function SubscribeRail() {
  return (
    <aside
      style={{
        background: TOKENS.canvasDeep,
        border: `1px solid ${TOKENS.divider}`,
        borderRadius: 12,
        padding: 22,
      }}
    >
      <MonoEyebrow tracking={0.08} color={TOKENS.accent}>Stay current</MonoEyebrow>
      <h4
        className="mt-3"
        style={{
          fontFamily: 'Inter, sans-serif',
          fontSize: 18,
          fontWeight: 500,
          color: TOKENS.textPrimary,
          letterSpacing: '-0.01em',
          lineHeight: 1.25,
        }}
      >
        One email per release. No marketing.
      </h4>
      <p className="mt-3" style={{ fontSize: 13, color: TOKENS.textSecondary, lineHeight: 1.5 }}>
        Protocol changes, runtime drops, and the occasional engineering note.
      </p>
      <div
        className="mt-5 flex items-center"
        style={{
          background: TOKENS.elevated,
          border: `1px solid ${TOKENS.divider}`,
          borderRadius: 8,
          height: 36,
          padding: '0 4px 0 12px',
        }}
      >
        <input
          placeholder="you@domain.com"
          className="flex-1 bg-transparent outline-none"
          style={{ fontSize: 13, color: TOKENS.textPrimary, fontFamily: 'Inter, sans-serif' }}
        />
        <button
          style={{
            height: 28,
            padding: '0 12px',
            borderRadius: 6,
            background: TOKENS.accent,
            color: '#FFFFFF',
            fontSize: 12,
            fontWeight: 500,
            fontFamily: 'Inter, sans-serif',
          }}
        >Subscribe</button>
      </div>
      <div className="mt-5 flex items-center gap-2">
        <Rss size={12} color={TOKENS.textTertiary} />
        <a href="#" className="font-mono" style={{ fontSize: 11, color: TOKENS.textTertiary, letterSpacing: '0.06em', textTransform: 'uppercase' }}>
          /blog/feed.xml
        </a>
      </div>
    </aside>
  );
}

function BlogIndex() {
  const featured = POSTS[0];
  const grid = POSTS.slice(1);
  return (
    <div className="site-home" style={{ background: TOKENS.canvas, minHeight: '100%' }}>
      <SiteHeader active="blog" />

      {/* Hero */}
      <section className="relative overflow-hidden px-4" style={{ paddingTop: 56, paddingBottom: 56, borderBottom: `1px solid ${TOKENS.divider}` }}>
        <div className="mx-auto" style={{ maxWidth: 1200 }}>
          <div className="flex items-center gap-3">
            <MonoEyebrow color={TOKENS.accent}>BLOG</MonoEyebrow>
            <span style={{ width: 36, height: 1, background: TOKENS.divider }} />
            <MonoEyebrow>Field notes from the runtime</MonoEyebrow>
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
            Notes on running agents, in the open.
          </h1>
          <p className="mt-6" style={{ fontSize: 18, lineHeight: 1.6, color: TOKENS.textSecondary, maxWidth: '58ch' }}>
            Protocol design, runtime engineering, and the occasional release note from the team building <span style={{ color: TOKENS.textPrimary }}>agh-network/v0</span>. Read in any order.
          </p>

          {/* category pills */}
          <div className="mt-9 flex flex-wrap items-center gap-2">
            <CategoryPill label="All" count={47} active />
            <CategoryPill label="Protocol" count={9} />
            <CategoryPill label="Runtime" count={12} />
            <CategoryPill label="Engineering" count={14} />
            <CategoryPill label="Network" count={6} />
            <CategoryPill label="Changelog" count={6} />
            <span style={{ width: 1, height: 16, background: TOKENS.divider, margin: '0 4px' }} />
            <button
              className="inline-flex items-center gap-1.5"
              style={{
                height: 32, padding: '0 12px', borderRadius: 9999,
                color: TOKENS.textTertiary, fontSize: 13, fontFamily: 'Inter, sans-serif',
              }}
            >
              <Rss size={12} color={TOKENS.textTertiary} />
              <span className="font-mono" style={{ fontSize: 11, letterSpacing: '0.06em', textTransform: 'uppercase' }}>RSS</span>
            </button>
          </div>
        </div>
      </section>

      {/* Featured */}
      <section className="px-4" style={{ paddingTop: 48, paddingBottom: 24 }}>
        <div className="mx-auto" style={{ maxWidth: 1200 }}>
          <FeaturedPost p={featured} />
        </div>
      </section>

      {/* Latest grid + sidebar */}
      <section className="px-4" style={{ paddingTop: 32, paddingBottom: 80 }}>
        <div className="mx-auto" style={{ maxWidth: 1200 }}>
          <div className="flex items-baseline justify-between">
            <div className="flex items-center gap-3">
              <MonoEyebrow tracking={0.08}>LATEST</MonoEyebrow>
              <span style={{ width: 36, height: 1, background: TOKENS.divider }} />
              <span style={{ fontSize: 13, color: TOKENS.textTertiary, fontFamily: 'Inter, sans-serif' }}>Newest first</span>
            </div>
            <a href="#" style={{ fontSize: 13, color: TOKENS.textSecondary, fontFamily: 'Inter, sans-serif' }}>
              View archive →
            </a>
          </div>
          <div
            className="mt-6 grid gap-6"
            style={{ gridTemplateColumns: 'minmax(0, 1fr) 320px' }}
          >
            <div className="grid gap-5" style={{ gridTemplateColumns: 'repeat(2, minmax(0, 1fr))' }}>
              {grid.map(p => <PostCard key={p.title} p={p} />)}
            </div>
            <div className="flex flex-col gap-5">
              <ChangelogRail />
              <SubscribeRail />
            </div>
          </div>
        </div>
      </section>

      <SiteFooter />
    </div>
  );
}

window.BlogIndex = BlogIndex;
