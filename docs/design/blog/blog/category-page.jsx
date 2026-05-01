// Blog Category / Archive page — dense list view filtered to one category
function ArchiveRow({ p, i }) {
  return (
    <a
      href="#"
      className="grid items-baseline gap-6 group"
      style={{
        gridTemplateColumns: '88px minmax(0, 1fr) 140px 70px 16px',
        padding: '22px 4px',
        borderTop: i === 0 ? `1px solid ${TOKENS.divider}` : 'none',
        borderBottom: `1px solid ${TOKENS.divider}`,
      }}
    >
      <span
        className="font-mono"
        style={{
          fontSize: 11,
          letterSpacing: '0.06em',
          color: TOKENS.textTertiary,
          textTransform: 'uppercase',
        }}
      >
        {p.date}
      </span>
      <div className="min-w-0">
        <h3
          style={{
            fontFamily: 'Inter, sans-serif',
            fontSize: 19,
            fontWeight: 500,
            letterSpacing: '-0.02em',
            color: TOKENS.textPrimary,
            lineHeight: 1.3,
          }}
        >
          {p.title}
        </h3>
        <p
          className="mt-2"
          style={{
            fontSize: 14,
            lineHeight: 1.6,
            color: TOKENS.textSecondary,
            maxWidth: '64ch',
          }}
        >
          {p.dek}
        </p>
        {p.tags && (
          <div className="mt-3 flex items-center gap-1.5 flex-wrap">
            {p.tags.map(t => (
              <span key={t} className="inline-flex items-center" style={{
                background: TOKENS.elevated,
                borderRadius: 5,
                padding: '2px 6px',
                fontFamily: 'JetBrains Mono, monospace',
                fontSize: 10,
                color: TOKENS.textSecondary,
                letterSpacing: '0.04em',
              }}>{t}</span>
            ))}
          </div>
        )}
      </div>
      <span
        className="font-mono"
        style={{
          fontSize: 11,
          letterSpacing: '0.06em',
          color: TOKENS.textLabel,
          textTransform: 'uppercase',
        }}
      >
        {p.author}
      </span>
      <span className="inline-flex items-center gap-1.5" style={{ color: TOKENS.textTertiary, fontSize: 11, justifySelf: 'end' }}>
        <Clock size={11} color={TOKENS.textTertiary} />
        <span className="font-mono" style={{ letterSpacing: '0.06em' }}>{p.read}</span>
      </span>
      <span style={{ color: TOKENS.textTertiary, justifySelf: 'end' }}>
        <ArrowUpRight size={14} color={TOKENS.textTertiary} />
      </span>
    </a>
  );
}

const PROTOCOL_POSTS = [
  {
    date: 'Apr 28',
    title: 'Why agents need a wire protocol, not another SDK',
    dek: 'Six months of agh-network/v0 in the wild. What shipping a JSON-over-NATS spec teaches you about coordination.',
    author: 'pedronauck', read: '14 min',
    tags: ['agh-network/v0', 'kinds', 'rationale'],
  },
  {
    date: 'Apr 11',
    title: 'Receipts: making delegation auditable',
    dek: 'Every direct kind ends with a receipt. The math, the schema, and the failure modes.',
    author: 'pedronauck', read: '9 min',
    tags: ['receipt', 'direct', 'audit'],
  },
  {
    date: 'Mar 20',
    title: 'whois, greet, say — naming the seven kinds',
    dek: 'Notes on protocol vocabulary. Why direct and not message. Why recipe survived.',
    author: 'pedronauck', read: '7 min',
    tags: ['vocabulary', 'whois', 'greet'],
  },
  {
    date: 'Mar 14',
    title: 'RFC-013 in practice — receipt bundling on the wire',
    dek: 'Bundling small receipts into a single frame, without breaking idempotency or trace ordering.',
    author: 'jpvalente', read: '11 min',
    tags: ['rfc-013', 'receipt', 'wire'],
  },
  {
    date: 'Mar 02',
    title: 'agh-network/v0 is stable',
    dek: 'What stable means for a protocol that is two months old. The compatibility surface from here forward.',
    author: 'team', read: '4 min',
    tags: ['stability', 'v0'],
  },
  {
    date: 'Feb 22',
    title: 'NATS over QUIC — the experiment that did not ship',
    dek: 'A long detour into transport choices, why we tried it, and why JSON-over-NATS is still the answer.',
    author: 'mariarib', read: '13 min',
    tags: ['transport', 'nats', 'quic'],
  },
  {
    date: 'Feb 09',
    title: 'Capability descriptors, version 1',
    dek: 'How agents publish what they can do, and how the runtime resolves the right peer at delegation time.',
    author: 'pedronauck', read: '10 min',
    tags: ['capability', 'whois', 'discovery'],
  },
  {
    date: 'Jan 24',
    title: 'Trace events on the protocol bus',
    dek: 'Why trace is a kind, not a sidecar. The cost we pay and the debugging it buys back.',
    author: 'jpvalente', read: '8 min',
    tags: ['trace', 'observability'],
  },
  {
    date: 'Jan 12',
    title: 'Designing for two operators per agent',
    dek: 'Most agent designs assume one human. Most real deployments do not. What that means for greet and whois.',
    author: 'pedronauck', read: '9 min',
    tags: ['greet', 'multi-tenant'],
  },
];

function BlogCategory() {
  return (
    <div className="site-home" style={{ background: TOKENS.canvas, minHeight: '100%' }}>
      <SiteHeader active="blog" />

      {/* Masthead */}
      <section className="px-4" style={{ paddingTop: 56, paddingBottom: 36, borderBottom: `1px solid ${TOKENS.divider}` }}>
        <div className="mx-auto" style={{ maxWidth: 1200 }}>
          <a href="#" className="inline-flex items-center gap-1.5" style={{ color: TOKENS.textTertiary, fontSize: 13 }}>
            <ArrowLeft size={13} color={TOKENS.textTertiary} />
            <span style={{ fontFamily: 'Inter, sans-serif' }}>All posts</span>
          </a>
          <div className="mt-7 flex items-center gap-3">
            <MonoEyebrow color={TOKENS.accent} tracking={0.08}>CATEGORY</MonoEyebrow>
            <span style={{ width: 36, height: 1, background: TOKENS.divider }} />
            <MonoEyebrow>9 posts · since Jan 2026</MonoEyebrow>
          </div>
          <div className="mt-6 flex items-baseline gap-5 flex-wrap">
            <h1
              style={{
                fontFamily: 'Playfair Display, serif',
                fontSize: 'clamp(2.6rem, 5.4vw, 4.6rem)',
                lineHeight: 0.98,
                letterSpacing: '-0.035em',
                fontWeight: 400,
                color: TOKENS.textPrimary,
              }}
            >
              Protocol.
            </h1>
            <span
              className="inline-flex items-center gap-2"
              style={{
                padding: '6px 12px',
                borderRadius: 9999,
                border: `1px solid ${TOKENS.divider}`,
                background: TOKENS.surface,
                color: TOKENS.textSecondary,
                fontSize: 13,
                fontFamily: 'Inter, sans-serif',
              }}
            >
              <Hash size={13} color={TOKENS.accent} />
              agh-network/v0
            </span>
          </div>
          <p className="mt-6" style={{ fontSize: 18, lineHeight: 1.6, color: TOKENS.textSecondary, maxWidth: '58ch' }}>
            Wire format, kinds, receipts, capabilities. Everything that lives below the runtime and above the transport.
          </p>

          {/* Categories pill row */}
          <div className="mt-9 flex flex-wrap items-center gap-2">
            <CategoryPill label="All" count={47} />
            <CategoryPill label="Protocol" count={9} active />
            <CategoryPill label="Runtime" count={12} />
            <CategoryPill label="Engineering" count={14} />
            <CategoryPill label="Network" count={6} />
            <CategoryPill label="Changelog" count={6} />
          </div>
        </div>
      </section>

      {/* Filter / sort row */}
      <section className="px-4" style={{ background: TOKENS.surface, borderBottom: `1px solid ${TOKENS.divider}` }}>
        <div className="mx-auto flex flex-wrap items-center gap-3" style={{ maxWidth: 1200, paddingTop: 14, paddingBottom: 14 }}>
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
            {[
              { l: 'NEWEST', a: true },
              { l: 'OLDEST', a: false },
              { l: 'POPULAR', a: false },
            ].map(s => (
              <button
                key={s.l}
                style={{
                  height: 22,
                  padding: '0 10px',
                  borderRadius: 5,
                  background: s.a ? TOKENS.elevated : 'transparent',
                  color: s.a ? TOKENS.textPrimary : TOKENS.textTertiary,
                  fontFamily: 'JetBrains Mono, monospace',
                  fontSize: 10,
                  fontWeight: 600,
                  letterSpacing: '0.08em',
                  textTransform: 'uppercase',
                }}
              >
                {s.l}
              </button>
            ))}
          </div>
          <div
            className="inline-flex items-center gap-2 ml-auto"
            style={{
              height: 28,
              borderRadius: 7,
              border: `1px solid ${TOKENS.divider}`,
              background: TOKENS.surfacePanel,
              padding: '0 8px',
              minWidth: 280,
            }}
          >
            <SearchIcon size={12} color={TOKENS.textTertiary} />
            <input
              placeholder="Search within Protocol…"
              className="flex-1 bg-transparent outline-none"
              style={{ fontSize: 13, color: TOKENS.textPrimary, fontFamily: 'Inter, sans-serif' }}
            />
            <span
              className="font-mono"
              style={{
                fontSize: 9,
                color: TOKENS.textTertiary,
                border: `1px solid ${TOKENS.divider}`,
                borderRadius: 4,
                padding: '1px 5px',
                letterSpacing: '0.06em',
                textTransform: 'uppercase',
              }}
            >
              ⌘K
            </span>
          </div>
          <button className="inline-flex items-center gap-1.5" style={{
            height: 28, padding: '0 10px', borderRadius: 9999,
            border: `1px solid ${TOKENS.divider}`, color: TOKENS.textSecondary, fontSize: 12, fontFamily: 'Inter, sans-serif',
          }}>
            <Rss size={11} color={TOKENS.textSecondary} />
            <span className="font-mono" style={{ letterSpacing: '0.06em', textTransform: 'uppercase' }}>RSS · protocol</span>
          </button>
        </div>
      </section>

      {/* Archive list + side rail */}
      <section className="px-4" style={{ paddingTop: 32, paddingBottom: 80 }}>
        <div className="mx-auto grid gap-12" style={{ maxWidth: 1200, gridTemplateColumns: 'minmax(0, 1fr) 280px' }}>
          <div>
            {PROTOCOL_POSTS.map((p, i) => <ArchiveRow key={p.title} p={p} i={i} />)}
            <div className="mt-9 flex items-center justify-between">
              <span style={{ fontSize: 13, color: TOKENS.textTertiary, fontFamily: 'Inter, sans-serif' }}>
                Showing 9 of 9 in <span style={{ color: TOKENS.textSecondary }}>Protocol</span>.
              </span>
              <button style={{
                height: 36, padding: '0 16px', borderRadius: 8,
                border: `1px solid ${TOKENS.divider}`, background: 'transparent',
                color: TOKENS.textPrimary, fontSize: 13, fontFamily: 'Inter, sans-serif', fontWeight: 500,
              }}>Load older posts</button>
            </div>
          </div>

          <aside className="flex flex-col gap-5" style={{ position: 'sticky', top: 80, alignSelf: 'start' }}>
            <div style={{
              background: TOKENS.surface,
              border: `1px solid ${TOKENS.divider}`,
              borderRadius: 12,
              padding: 22,
            }}>
              <MonoEyebrow tracking={0.08}>Spec status</MonoEyebrow>
              <div className="mt-4 flex items-center justify-between">
                <span style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: 14, color: TOKENS.textPrimary }}>
                  agh-network/v0
                </span>
                <span className="inline-flex items-center gap-1.5">
                  <span style={{ width: 8, height: 8, borderRadius: '50%', background: TOKENS.success }} />
                  <MonoEyebrow color={TOKENS.success}>STABLE</MonoEyebrow>
                </span>
              </div>
              <div className="mt-5 flex flex-col gap-2.5">
                {[
                  ['Kinds', '7'],
                  ['Open RFCs', '3'],
                  ['Last update', 'Mar 31'],
                  ['Reference impl', 'Go · TS'],
                ].map(([k, v]) => (
                  <div key={k} className="flex items-center justify-between" style={{ borderBottom: `1px dashed ${TOKENS.divider}`, paddingBottom: 8 }}>
                    <span style={{ fontSize: 13, color: TOKENS.textTertiary, fontFamily: 'Inter, sans-serif' }}>{k}</span>
                    <span className="font-mono" style={{ fontSize: 12, color: TOKENS.textPrimary, letterSpacing: '0.04em' }}>{v}</span>
                  </div>
                ))}
              </div>
            </div>

            <div style={{
              background: TOKENS.surface,
              border: `1px solid ${TOKENS.divider}`,
              borderRadius: 12,
              padding: 22,
            }}>
              <MonoEyebrow tracking={0.08}>Tags</MonoEyebrow>
              <div className="mt-4 flex flex-wrap gap-1.5">
                {[
                  ['kinds', 9], ['receipt', 6], ['direct', 5], ['greet', 4], ['whois', 4],
                  ['trace', 4], ['recipe', 3], ['rfc-013', 2], ['rfc-014', 2], ['transport', 2],
                  ['capability', 3], ['vocabulary', 1],
                ].map(([t, n]) => (
                  <span key={t} className="inline-flex items-center gap-1" style={{
                    background: TOKENS.elevated,
                    borderRadius: 5,
                    padding: '3px 8px',
                    fontFamily: 'JetBrains Mono, monospace',
                    fontSize: 10.5,
                    color: TOKENS.textSecondary,
                    letterSpacing: '0.04em',
                  }}>
                    {t}
                    <span style={{ color: TOKENS.textTertiary }}>· {n}</span>
                  </span>
                ))}
              </div>
            </div>

            <div style={{
              background: TOKENS.canvasDeep,
              border: `1px solid ${TOKENS.divider}`,
              borderRadius: 12,
              padding: 22,
            }}>
              <MonoEyebrow tracking={0.08} color={TOKENS.accent}>Read the spec</MonoEyebrow>
              <p className="mt-3" style={{ fontSize: 13, lineHeight: 1.55, color: TOKENS.textSecondary }}>
                Posts above interpret the protocol. The protocol itself is one document.
              </p>
              <a
                href="#"
                className="mt-4 inline-flex items-center gap-1.5"
                style={{ color: TOKENS.accent, fontSize: 13, fontFamily: 'Inter, sans-serif', fontWeight: 500 }}
              >
                Open agh-network/v0 <ArrowUpRight size={13} color={TOKENS.accent} />
              </a>
            </div>
          </aside>
        </div>
      </section>

      <SiteFooter />
    </div>
  );
}

window.BlogCategory = BlogCategory;
