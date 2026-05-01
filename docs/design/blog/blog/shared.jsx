// Shared chrome + primitives for AGH blog screens.
// Mirrors packages/site DESIGN.md tokens exactly.

const TOKENS = {
  canvasDeep: '#0E0E0F',
  canvas: '#141312',
  surface: '#1E1C1B',
  surfacePanel: '#181716',
  elevated: '#2E2C2B',
  divider: '#3C3A39',
  hover: '#353332',
  textPrimary: '#E5E5E7',
  textSecondary: '#8E8E93',
  textTertiary: '#636366',
  textLabel: '#98989D',
  accent: '#E8572A',
  accentInk: '#17110F',
  accentHover: '#D14E25',
  accentTint: '#E8572A26',
  accentDim: '#E8572A59',
  success: '#30D158',
  danger: '#FF453A',
  warning: '#FFD60A',
  info: '#BF5AF2',
};
window.AGH_TOKENS = TOKENS;

// Apply once to <body> (in design canvas). Each screen renders inside its own artboard div,
// which we color directly so artboards float on the canvas's lighter ground.

// ---------- header ----------
function SiteHeader({ active = 'blog' }) {
  const links = [
    { href: '#', label: 'Home', key: 'home' },
    { href: '#', label: 'Runtime', key: 'runtime' },
    { href: '#', label: 'AGH Network', key: 'network' },
    { href: '#', label: 'Blog', key: 'blog' },
    { href: '#', label: 'Changelog', key: 'changelog' },
  ];
  return (
    <header
      className="sticky top-0 z-40 px-4"
      style={{
        background: 'rgba(20,19,18,0.92)',
        borderBottom: `1px solid ${TOKENS.divider}`,
        backdropFilter: 'blur(24px)',
        WebkitBackdropFilter: 'blur(24px)',
      }}
    >
      <div className="mx-auto flex h-14 w-full items-center gap-5" style={{ maxWidth: 1200 }}>
        <a href="#" aria-label="AGH home" className="flex shrink-0 items-center gap-2">
          <span
            style={{
              fontFamily: 'NuixyberNext, Inter, sans-serif',
              fontSize: 26,
              lineHeight: 1,
              color: TOKENS.textPrimary,
              letterSpacing: '-0.01em',
            }}
          >
            agh
          </span>
          <span
            className="font-mono"
            style={{
              fontSize: 10,
              fontWeight: 600,
              letterSpacing: '0.16em',
              color: TOKENS.textLabel,
              border: `1px solid ${TOKENS.divider}`,
              borderRadius: 4,
              padding: '2px 6px',
            }}
          >
            ALPHA
          </span>
        </a>
        <nav className="ml-2 hidden items-center gap-1 md:flex">
          {links.map(l => {
            const isActive = l.key === active;
            return (
              <a
                key={l.key}
                href={l.href}
                className="inline-flex items-center rounded-full px-3 py-1.5 transition-colors"
                style={{
                  fontFamily: 'Inter, sans-serif',
                  fontSize: 14,
                  color: isActive ? TOKENS.accent : TOKENS.textSecondary,
                  background: isActive ? 'rgba(232,87,42,0.12)' : 'transparent',
                }}
              >
                {l.label}
              </a>
            );
          })}
        </nav>
        <div className="ml-auto flex items-center gap-2">
          <div
            className="hidden lg:flex items-center gap-2 rounded-full px-3"
            style={{
              minWidth: 220,
              height: 36,
              background: 'rgba(28,28,30,0.92)',
              border: `1px solid ${TOKENS.divider}`,
              color: TOKENS.textTertiary,
              fontSize: 13,
            }}
          >
            <SearchIcon size={13} color={TOKENS.textTertiary} />
            <span style={{ fontFamily: 'Inter, sans-serif' }}>Search docs and posts…</span>
            <span
              className="ml-auto font-mono"
              style={{
                fontSize: 10,
                color: TOKENS.textTertiary,
                border: `1px solid ${TOKENS.divider}`,
                borderRadius: 4,
                padding: '1px 5px',
                letterSpacing: '0.06em',
              }}
            >
              ⌘K
            </span>
          </div>
          <button
            className="inline-flex h-9 w-9 items-center justify-center rounded-full"
            style={{ color: TOKENS.textSecondary }}
            aria-label="GitHub"
          >
            <GithubGlyph size={16} />
          </button>
        </div>
      </div>
    </header>
  );
}

// ---------- footer ----------
function SiteFooter() {
  const cols = [
    {
      head: 'Runtime',
      items: ['Sessions', 'Memory', 'Skills', 'Workspaces', 'Bridges'],
    },
    {
      head: 'Network',
      items: ['Spec agh-network/v0', 'Kinds', 'Receipts', 'Trace'],
    },
    {
      head: 'Project',
      items: ['Blog', 'Changelog', 'GitHub', 'RFCs'],
    },
  ];
  return (
    <footer
      className="px-4"
      style={{
        background: TOKENS.canvasDeep,
        borderTop: `1px solid ${TOKENS.divider}`,
        paddingTop: 56,
        paddingBottom: 40,
      }}
    >
      <div className="mx-auto" style={{ maxWidth: 1200 }}>
        <div className="grid gap-10 md:grid-cols-[minmax(0,1.4fr)_repeat(3,minmax(0,1fr))]">
          <div>
            <div className="flex items-center gap-2">
              <span
                style={{
                  fontFamily: 'NuixyberNext, Inter, sans-serif',
                  fontSize: 24,
                  color: TOKENS.textPrimary,
                }}
              >
                agh
              </span>
              <span
                className="font-mono"
                style={{
                  fontSize: 10,
                  fontWeight: 600,
                  letterSpacing: '0.16em',
                  color: TOKENS.textLabel,
                  border: `1px solid ${TOKENS.divider}`,
                  borderRadius: 4,
                  padding: '2px 6px',
                }}
              >
                ALPHA
              </span>
            </div>
            <p
              className="mt-4 max-w-[34ch]"
              style={{ fontSize: 13, lineHeight: 1.6, color: TOKENS.textSecondary }}
            >
              Agent operating system. One local binary. One open protocol. Sessions, memory, skills, bridges — everything logged, everything replayable.
            </p>
            <div className="mt-5 flex items-center gap-2">
              <span
                style={{
                  width: 8,
                  height: 8,
                  borderRadius: '50%',
                  background: TOKENS.success,
                  display: 'inline-block',
                }}
              />
              <span
                className="font-mono"
                style={{ fontSize: 11, color: TOKENS.textTertiary, letterSpacing: '0.06em', textTransform: 'uppercase' }}
              >
                Shipped today · v0.4.2
              </span>
            </div>
          </div>
          {cols.map(c => (
            <div key={c.head}>
              <p
                className="font-mono"
                style={{
                  fontSize: 11,
                  fontWeight: 600,
                  letterSpacing: '0.06em',
                  textTransform: 'uppercase',
                  color: TOKENS.textTertiary,
                }}
              >
                {c.head}
              </p>
              <ul className="mt-4 flex flex-col gap-2.5">
                {c.items.map(i => (
                  <li key={i}>
                    <a href="#" style={{ fontSize: 14, color: TOKENS.textSecondary }}>{i}</a>
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>
        <div
          className="mt-12 flex flex-wrap items-center justify-between gap-3"
          style={{ borderTop: `1px solid ${TOKENS.divider}`, paddingTop: 18 }}
        >
          <p
            className="font-mono"
            style={{ fontSize: 11, color: TOKENS.textTertiary, letterSpacing: '0.06em', textTransform: 'uppercase' }}
          >
            agh-network/v0 · MIT · 2026
          </p>
          <p
            className="font-mono"
            style={{ fontSize: 11, color: TOKENS.textTertiary, letterSpacing: '0.06em', textTransform: 'uppercase' }}
          >
            macOS · Linux · single binary
          </p>
        </div>
      </div>
    </footer>
  );
}

// ---------- atoms ----------
function MonoEyebrow({ children, color, size = 11, tracking = 0.06, className = '', style = {} }) {
  return (
    <span
      className={`font-mono ${className}`}
      style={{
        fontSize: size,
        fontWeight: 600,
        textTransform: 'uppercase',
        letterSpacing: `${tracking}em`,
        color: color || TOKENS.textTertiary,
        ...style,
      }}
    >
      {children}
    </span>
  );
}

function KindChip({ kind, label }) {
  const dotMap = {
    say: '#8E8E93',
    greet: '#5BA6FF',
    direct: TOKENS.accent,
    receipt: TOKENS.success,
    recipe: TOKENS.warning,
    trace: '#B892FF',
    whois: '#4FD1C5',
  };
  return (
    <span
      className="inline-flex items-center gap-1.5"
      style={{
        border: `1px solid ${TOKENS.divider}`,
        borderRadius: 3,
        padding: '1px 6px',
        fontFamily: 'JetBrains Mono, ui-monospace, monospace',
        fontSize: 9.5,
        fontWeight: 600,
        textTransform: 'uppercase',
        letterSpacing: '0.08em',
        color: TOKENS.textTertiary,
      }}
    >
      {kind && (
        <span
          style={{
            width: 7,
            height: 7,
            borderRadius: '50%',
            background: dotMap[kind] || TOKENS.textTertiary,
            display: 'inline-block',
          }}
        />
      )}
      {label || kind}
    </span>
  );
}

function MonoBadge({ children, tone = 'neutral' }) {
  const map = {
    neutral: { bg: 'transparent', border: TOKENS.divider, text: TOKENS.textLabel },
    accent: { bg: TOKENS.accentTint, border: 'transparent', text: TOKENS.accent },
    success: { bg: '#30D15826', border: 'transparent', text: TOKENS.success },
    info: { bg: '#BF5AF226', border: 'transparent', text: TOKENS.info },
    warning: { bg: '#FFD60A26', border: 'transparent', text: TOKENS.warning },
    danger: { bg: '#FF453A26', border: 'transparent', text: TOKENS.danger },
  };
  const c = map[tone];
  return (
    <span
      style={{
        background: c.bg,
        border: `1px solid ${c.border === 'transparent' ? 'transparent' : c.border}`,
        borderRadius: 6,
        padding: '2px 6px',
        fontFamily: 'JetBrains Mono, ui-monospace, monospace',
        fontSize: 11,
        fontWeight: 500,
        letterSpacing: '0.06em',
        color: c.text,
      }}
    >
      {children}
    </span>
  );
}

function CategoryPill({ label, count, active }) {
  return (
    <button
      className="inline-flex items-center gap-2"
      style={{
        height: 32,
        padding: '0 14px',
        borderRadius: 9999,
        border: `1px solid ${active ? TOKENS.accent : TOKENS.divider}`,
        background: active ? TOKENS.accentTint : 'transparent',
        color: active ? TOKENS.accent : TOKENS.textSecondary,
        fontFamily: 'Inter, sans-serif',
        fontSize: 13,
        fontWeight: 500,
      }}
    >
      <span>{label}</span>
      {count != null && (
        <span
          className="font-mono"
          style={{
            fontSize: 10,
            color: active ? TOKENS.accent : TOKENS.textTertiary,
            letterSpacing: '0.06em',
          }}
        >
          {String(count).padStart(2, '0')}
        </span>
      )}
    </button>
  );
}

function PrimaryCta({ children }) {
  return (
    <button
      style={{
        height: 44,
        borderRadius: 8,
        background: TOKENS.accent,
        color: '#FFFFFF',
        padding: '0 20px',
        fontFamily: 'Inter, sans-serif',
        fontSize: 14,
        fontWeight: 500,
      }}
    >
      {children}
    </button>
  );
}

function GhostCta({ children }) {
  return (
    <button
      style={{
        height: 44,
        borderRadius: 8,
        background: 'transparent',
        border: `1px solid ${TOKENS.divider}`,
        color: TOKENS.textPrimary,
        padding: '0 20px',
        fontFamily: 'Inter, sans-serif',
        fontSize: 14,
        fontWeight: 500,
      }}
    >
      {children}
    </button>
  );
}

// ---------- icons (lucide-style strokes) ----------
function Icon({ size = 16, color = 'currentColor', children, style = {} }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      stroke={color}
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      style={style}
    >
      {children}
    </svg>
  );
}
const SearchIcon = (p) => <Icon {...p}><circle cx="11" cy="11" r="7" /><line x1="21" y1="21" x2="16.65" y2="16.65" /></Icon>;
const ArrowUpRight = (p) => <Icon {...p}><line x1="7" y1="17" x2="17" y2="7" /><polyline points="7 7 17 7 17 17" /></Icon>;
const ArrowRight = (p) => <Icon {...p}><line x1="5" y1="12" x2="19" y2="12" /><polyline points="12 5 19 12 12 19" /></Icon>;
const ArrowLeft = (p) => <Icon {...p}><line x1="19" y1="12" x2="5" y2="12" /><polyline points="12 19 5 12 12 5" /></Icon>;
const Clock = (p) => <Icon {...p}><circle cx="12" cy="12" r="10" /><polyline points="12 6 12 12 16 14" /></Icon>;
const Network = (p) => <Icon {...p}><rect x="9" y="2" width="6" height="6" rx="1"/><rect x="2" y="16" width="6" height="6" rx="1"/><rect x="16" y="16" width="6" height="6" rx="1"/><path d="M12 8v4M5 16v-2h14v2"/></Icon>;
const Database = (p) => <Icon {...p}><ellipse cx="12" cy="5" rx="9" ry="3"/><path d="M3 5v14a9 3 0 0 0 18 0V5"/><path d="M3 12a9 3 0 0 0 18 0"/></Icon>;
const Activity = (p) => <Icon {...p}><polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/></Icon>;
const FileCode = (p) => <Icon {...p}><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><polyline points="10 12 8 14 10 16"/><polyline points="14 12 16 14 14 16"/></Icon>;
const Plug = (p) => <Icon {...p}><path d="M9 2v6"/><path d="M15 2v6"/><path d="M6 8h12v4a6 6 0 0 1-12 0V8z"/><path d="M12 18v4"/></Icon>;
const Sparkles = (p) => <Icon {...p}><path d="M12 3l1.5 4.5L18 9l-4.5 1.5L12 15l-1.5-4.5L6 9l4.5-1.5z"/><path d="M19 14l.8 2.2L22 17l-2.2.8L19 20l-.8-2.2L16 17l2.2-.8z"/></Icon>;
const Star = (p) => <Icon {...p}><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/></Icon>;
const Rss = (p) => <Icon {...p}><path d="M4 11a9 9 0 0 1 9 9"/><path d="M4 4a16 16 0 0 1 16 16"/><circle cx="5" cy="19" r="1"/></Icon>;
const Copy = (p) => <Icon {...p}><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></Icon>;
const GithubGlyph = (p) => <Icon {...p}><path d="M9 19c-5 1.5-5-2.5-7-3m14 6v-3.87a3.37 3.37 0 0 0-.94-2.61c3.14-.35 6.44-1.54 6.44-7A5.44 5.44 0 0 0 20 4.77 5.07 5.07 0 0 0 19.91 1S18.73.65 16 2.48a13.38 13.38 0 0 0-7 0C6.27.65 5.09 1 5.09 1A5.07 5.07 0 0 0 5 4.77a5.44 5.44 0 0 0-1.5 3.78c0 5.42 3.3 6.61 6.44 7A3.37 3.37 0 0 0 9 18.13V22"/></Icon>;
const Bookmark = (p) => <Icon {...p}><path d="M19 21l-7-5-7 5V5a2 2 0 0 1 2-2h10a2 2 0 0 1 2 2z"/></Icon>;
const Share = (p) => <Icon {...p}><circle cx="18" cy="5" r="3"/><circle cx="6" cy="12" r="3"/><circle cx="18" cy="19" r="3"/><line x1="8.59" y1="13.51" x2="15.42" y2="17.49"/><line x1="15.41" y1="6.51" x2="8.59" y2="10.49"/></Icon>;
const Hash = (p) => <Icon {...p}><line x1="4" y1="9" x2="20" y2="9"/><line x1="4" y1="15" x2="20" y2="15"/><line x1="10" y1="3" x2="8" y2="21"/><line x1="16" y1="3" x2="14" y2="21"/></Icon>;

Object.assign(window, {
  SiteHeader, SiteFooter, MonoEyebrow, KindChip, MonoBadge, CategoryPill,
  PrimaryCta, GhostCta,
  SearchIcon, ArrowUpRight, ArrowRight, ArrowLeft, Clock, Network, Database, Activity,
  FileCode, Plug, Sparkles, Star, Rss, Copy, GithubGlyph, Bookmark, Share, Hash,
});
