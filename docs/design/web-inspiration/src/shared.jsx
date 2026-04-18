// Shared widgets: Modal, FieldRow, SectionCard, StatGrid, Banner, SaveBar, Toggle, SourceBadge, Tabs
const { Icon } = window;

function Modal({ open, onClose, title, description, children, footer, width }) {
  React.useEffect(() => {
    if (!open) return;
    const onKey = e => { if (e.key === 'Escape') onClose?.(); };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [open, onClose]);
  if (!open) return null;
  return (
    <div className="modal-backdrop" onMouseDown={onClose}>
      <div className={`modal ${width === 'wide' ? 'wide' : ''}`} onMouseDown={e => e.stopPropagation()}>
        <div className="modal-head">
          <div style={{ flex: 1, minWidth: 0 }}>
            <h2>{title}</h2>
            {description && <p>{description}</p>}
          </div>
          <button className="btn-icon" onClick={onClose}><Icon.X size={14} /></button>
        </div>
        <div className="modal-body">{children}</div>
        {footer && <div className="modal-foot">{footer}</div>}
      </div>
    </div>
  );
}

function FieldRow({ label, description, tag, children }) {
  return (
    <div className="field-row">
      <div className="field-row-label">
        <span className="field-row-title">{label}</span>
        {description && <span className="field-row-hint">{description}</span>}
        {tag && <span className="field-row-tag">{tag}</span>}
      </div>
      <div>{children}</div>
    </div>
  );
}

function SectionCard({ eyebrow, note, right, children }) {
  return (
    <div className="section-card">
      <div className="section-card-head">
        <div className="flex items-center gap-2">
          <span className="eyebrow">{eyebrow}</span>
          {note && <span className="mono" style={{ fontSize: 10, color: 'var(--color-text-tertiary)' }}>· {note}</span>}
        </div>
        {right}
      </div>
      <div className="section-card-body">{children}</div>
    </div>
  );
}

function StatGrid({ items }) {
  return (
    <div className="stat-grid">
      {items.map(it => (
        <div key={it.label} className="stat-item">
          <span className="lbl">{it.label}</span>
          <span className="val" style={it.tone === 'accent' ? { color: 'var(--color-accent)' } : null}>{it.value}</span>
        </div>
      ))}
    </div>
  );
}

function Banner({ tone = 'info', icon: I, children, action }) {
  const Ico = I || (tone === 'warning' ? Icon.AlertCircle : tone === 'danger' ? Icon.AlertCircle : tone === 'success' ? Icon.CheckCircle : Icon.Info);
  const tc = { info: 'info', warning: 'warning', danger: 'danger', success: 'success' }[tone];
  return (
    <div className={`banner ${tc}`}>
      <Ico size={14} style={{ color: `var(--color-${tc === 'info' ? 'info' : tc})`, flexShrink: 0, marginTop: 1 }} />
      <div style={{ flex: 1 }}>{children}</div>
      {action}
    </div>
  );
}

function SaveBar({ dirty, saving, onSave, onReset, warning, lastApplied }) {
  if (!dirty && !saving && !warning) return null;
  return (
    <div className="save-bar">
      {dirty && <span className="save-bar-dirty-dot" title="unsaved changes" />}
      <span className="mono" style={{ fontSize: 11.5, color: 'var(--color-text-secondary)', flex: 1 }}>
        {saving ? 'Saving…' : dirty ? 'Unsaved changes' : warning ? warning : `Applied ${lastApplied}`}
      </span>
      {dirty && !saving && (
        <>
          <button className="btn btn-ghost" style={{ height: 30 }} onClick={onReset}>Reset</button>
          <button className="btn btn-primary" style={{ height: 30 }} onClick={onSave}>
            <Icon.Save size={12} />Save changes
          </button>
        </>
      )}
    </div>
  );
}

function Toggle({ on, onChange }) {
  return <button className={`toggle ${on ? 'on' : ''}`} onClick={() => onChange?.(!on)} />;
}

function SourceBadge({ source = 'builtin', shadowed }) {
  const map = {
    'builtin': { tone: '', label: 'BUILTIN' },
    'overlay': { tone: 'accent', label: 'OVERLAY' },
    'global': { tone: 'info', label: 'GLOBAL' },
    'workspace': { tone: 'warning', label: 'WORKSPACE' },
    'env': { tone: 'accent', label: 'ENV' },
  };
  const m = map[source] || map.builtin;
  return (
    <span className="flex items-center gap-2">
      <span className={`mono-chip ${m.tone}`}>{m.label}</span>
      {shadowed && shadowed.length > 0 && (
        <span className="mono" style={{ fontSize: 10, color: 'var(--color-text-tertiary)' }}>
          shadows {shadowed.length}
        </span>
      )}
    </span>
  );
}

function Tabs({ items, value, onChange }) {
  return (
    <div className="tabs">
      {items.map(it => (
        <div key={it.value}
             className={`tab ${value === it.value ? 'active' : ''}`}
             onClick={() => onChange(it.value)}>
          {it.icon && <it.icon size={13} />}
          <span>{it.label}</span>
          {it.count !== undefined && <span className="tab-count">{it.count}</span>}
        </div>
      ))}
    </div>
  );
}

function StatusLine({ items = [], daemon = true }) {
  return (
    <div className="flex items-center gap-2" style={{ fontSize: 11.5 }}>
      <span className="flex items-center gap-1.5">
        <span className={`dot ${daemon ? 'success' : 'danger'}`} />
        <span className="mono" style={{ fontSize: 11, color: 'var(--color-text-secondary)' }}>
          {daemon ? 'daemon ok' : 'daemon offline'}
        </span>
      </span>
      {items.map((it, i) => (
        <React.Fragment key={i}>
          <span className="meta-dot">·</span>
          <span className="mono" style={{ fontSize: 11, color: 'var(--color-text-tertiary)' }}>{it}</span>
        </React.Fragment>
      ))}
    </div>
  );
}

function CodeBlock({ lang = 'bash', children, copy = true, head }) {
  const [copied, setCopied] = React.useState(false);
  const text = typeof children === 'string' ? children : '';
  return (
    <div className="codeblock">
      {(head || copy) && (
        <div className="codeblock-head">
          {head || <span className="mono" style={{ fontSize: 10, color: 'var(--color-text-tertiary)' }}>{lang}</span>}
          {copy && (
            <button className="btn-icon" style={{ marginLeft: 'auto' }}
                    onClick={() => { navigator.clipboard?.writeText(text); setCopied(true); setTimeout(() => setCopied(false), 1200); }}>
              {copied ? <Icon.Check size={12} /> : <Icon.Copy size={12} />}
            </button>
          )}
        </div>
      )}
      <div className="codeblock-body">{children}</div>
    </div>
  );
}

Object.assign(window, { Modal, FieldRow, SectionCard, StatGrid, Banner, SaveBar, Toggle, SourceBadge, Tabs, StatusLine, CodeBlock });
