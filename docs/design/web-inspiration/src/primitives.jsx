// Shared primitives for the app
const { Icon } = window;

function PageHeader({ title, icon: I, count, controls, meta }) {
  return (
    <header className="page-header">
      <div className="page-title">
        {I && <span className="icon"><I size={14} /></span>}
        <span>{title}</span>
        {count !== undefined && <span className="page-count">{count}</span>}
      </div>
      {controls && <div className="flex items-center gap-2">{controls}</div>}
      <div style={{ marginLeft: 'auto' }} className="flex items-center gap-2">{meta}</div>
    </header>
  );
}

function Pills({ items, value, onChange }) {
  return (
    <div className="pills">
      {items.map(it => (
        <button key={it.value}
                className={`pill ${value === it.value ? 'active' : ''}`}
                onClick={() => onChange(it.value)}>
          {it.label}
          {it.badge !== undefined && it.badge > 0 &&
            <span className="pill-badge">{it.badge}</span>}
        </button>
      ))}
    </div>
  );
}

function SearchInput({ value, onChange, placeholder }) {
  return (
    <div className="search-input" style={{ margin: '10px 14px' }}>
      <Icon.Search size={13} style={{ color: 'var(--color-text-tertiary)' }} />
      <input placeholder={placeholder || 'Search…'} value={value || ''} onChange={e => onChange?.(e.target.value)} />
    </div>
  );
}

function Empty({ icon: I = Icon.Box, title, description, action }) {
  return (
    <div className="empty">
      <div className="empty-icon"><I size={20} /></div>
      <h3>{title}</h3>
      {description && <p>{description}</p>}
      {action}
    </div>
  );
}

function Section({ label, children, right }) {
  return (
    <section>
      <div className="section-head">
        <h2>{label}</h2>
        {right}
      </div>
      {children}
    </section>
  );
}

function Metric({ label, value, detail, tone }) {
  return (
    <div className="card" style={{ padding: '14px 16px', display: 'flex', flexDirection: 'column', gap: 6 }}>
      <span className="eyebrow">{label}</span>
      <div style={{ display: 'flex', alignItems: 'baseline', gap: 8 }}>
        <span style={{
          fontFamily: 'var(--font-mono)', fontSize: 22, fontWeight: 500,
          color: tone === 'accent' ? 'var(--color-accent)' : 'var(--color-text-primary)',
          letterSpacing: '-0.02em',
        }}>{value}</span>
        {detail && <span className="mono" style={{ fontSize: 11, color: 'var(--color-text-tertiary)' }}>{detail}</span>}
      </div>
    </div>
  );
}

Object.assign(window, { PageHeader, Pills, SearchInput, Empty, Section, Metric });
