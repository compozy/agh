// Session view — the flagship "agent conversation" page, and Settings
const { Icon, PageHeader, Pills, Metric, Section } = window;

function SessionPage({ sessionId }) {
  const [composer, setComposer] = React.useState('');
  const [tab, setTab] = React.useState('thread');
  const msgs = [
    { role: 'system', body: 'Session resumed from checkpoint 8471. 3 prior tool calls replayed.', time: '12:02' },
    { role: 'user', body: 'Find the event mapper that groups tool calls by turn and extract the grouping logic into a pure helper.', time: '12:02' },
    { role: 'agent', body: 'I can see two candidates — `packages/runtime/src/events/map.ts` and `packages/runtime/src/session/stream.ts`. The grouping lives in `stream.ts` inside `onToolCall`. I\'ll extract it into `groupToolCallsByTurn` and point the existing call site at it.', time: '12:03' },
    { role: 'tool', tool: 'shell.safe-run', body: 'rg "onToolCall" packages/runtime -l', output: 'packages/runtime/src/session/stream.ts\npackages/runtime/src/session/replay.ts', time: '12:03' },
    { role: 'tool', tool: 'file.read', body: 'packages/runtime/src/session/stream.ts:84-141', output: '// 57 lines of source — grouping loop over tool.events', time: '12:03' },
    { role: 'agent', body: 'Proposed diff below. The helper takes `ToolEvent[]` and returns `ToolEventGroup[]`. No behavior change — just a move.', time: '12:04' },
    { role: 'diff', body: 'packages/runtime/src/session/stream.ts', additions: 4, removals: 38, time: '12:04' },
    { role: 'agent', body: 'Ready to apply. This touches one file. Approve to continue.', time: '12:04', action: 'approve' },
  ];

  return (
    <>
      <div className="page-header" style={{ gap: 12 }}>
        <span className="dot accent pulse" />
        <div style={{ display: 'flex', flexDirection: 'column', minWidth: 0 }}>
          <div className="flex items-center gap-2">
            <span style={{ fontSize: 14, fontWeight: 500, color: 'var(--color-text-primary)' }}>refactor tokens</span>
            <span className="mono-chip">claude</span>
            <span className="mono-chip">workspace · agh-core</span>
          </div>
          <span className="mono" style={{ fontSize: 10, color: 'var(--color-text-tertiary)', marginTop: 3 }}>
            sess-8471f · started 14m ago · 3 tool calls · resumable
          </span>
        </div>
        <div style={{ marginLeft: 'auto' }} className="flex items-center gap-2">
          <Pills value={tab} onChange={setTab} items={[
            { value: 'thread', label: 'Thread' },
            { value: 'trace', label: 'Trace' },
            { value: 'memory', label: 'Memory' },
            { value: 'settings', label: 'Settings' },
          ]} />
          <button className="btn-icon"><Icon.MoreHorizontal size={14} /></button>
        </div>
      </div>

      <div style={{ flex: 1, display: 'flex', minHeight: 0 }}>
        <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
          <div className="scroll-y" style={{ flex: 1, padding: '20px 28px' }}>
            <div style={{ maxWidth: 820, margin: '0 auto', display: 'flex', flexDirection: 'column', gap: 16 }}>
              {msgs.map((m, i) => <Message key={i} m={m} />)}
            </div>
          </div>
          <Composer value={composer} onChange={setComposer} />
        </div>
        <SessionInspector />
      </div>
    </>
  );
}

function Message({ m }) {
  if (m.role === 'system') {
    return (
      <div className="flex items-center gap-3" style={{ padding: '8px 0', color: 'var(--color-text-tertiary)', fontSize: 12 }}>
        <div style={{ flex: 1, height: 1, background: 'var(--color-divider)' }} />
        <span className="mono">{m.body}</span>
        <div style={{ flex: 1, height: 1, background: 'var(--color-divider)' }} />
      </div>
    );
  }
  if (m.role === 'user') {
    return (
      <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
        <div style={{ maxWidth: '76%', background: 'var(--color-surface)', border: '1px solid var(--color-divider)', borderRadius: 12, borderTopRightRadius: 4, padding: '12px 14px', fontSize: 13.5, lineHeight: 1.6 }}>
          {m.body}
          <div className="mono" style={{ fontSize: 10, color: 'var(--color-text-tertiary)', marginTop: 6, textAlign: 'right' }}>you · {m.time}</div>
        </div>
      </div>
    );
  }
  if (m.role === 'tool') {
    return (
      <div style={{ background: 'var(--color-canvas-deep)', border: '1px solid var(--color-divider)', borderRadius: 10, overflow: 'hidden' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '8px 12px', borderBottom: '1px solid var(--color-divider)', background: 'var(--color-surface-panel)' }}>
          <Icon.Terminal size={12} style={{ color: 'var(--color-accent)' }} />
          <span className="mono" style={{ fontSize: 11, color: 'var(--color-text-primary)' }}>{m.tool}</span>
          <span className="mono-chip success">ok</span>
          <span className="mono" style={{ fontSize: 10, color: 'var(--color-text-tertiary)', marginLeft: 'auto' }}>{m.time} · 212ms</span>
        </div>
        <div style={{ padding: '10px 14px', fontFamily: 'var(--font-mono)', fontSize: 12, color: 'var(--color-accent)' }}>
          <span style={{ color: 'var(--color-text-tertiary)' }}>$ </span>{m.body}
        </div>
        {m.output && (
          <div style={{ padding: '0 14px 10px', fontFamily: 'var(--font-mono)', fontSize: 11.5, color: 'var(--color-text-secondary)', whiteSpace: 'pre-wrap', lineHeight: 1.6 }}>
            {m.output}
          </div>
        )}
      </div>
    );
  }
  if (m.role === 'diff') {
    return (
      <div style={{ background: 'var(--color-canvas-deep)', border: '1px solid var(--color-divider)', borderRadius: 10 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '10px 14px', borderBottom: '1px solid var(--color-divider)' }}>
          <Icon.FileCode size={12} style={{ color: 'var(--color-text-secondary)' }} />
          <span className="mono" style={{ fontSize: 11, color: 'var(--color-text-primary)' }}>{m.body}</span>
          <span className="mono" style={{ fontSize: 10, color: 'var(--color-success)', marginLeft: 'auto' }}>+{m.additions}</span>
          <span className="mono" style={{ fontSize: 10, color: 'var(--color-danger)' }}>−{m.removals}</span>
        </div>
        <div style={{ padding: 14, fontFamily: 'var(--font-mono)', fontSize: 11.5, lineHeight: 1.65 }}>
          <div style={{ color: 'var(--color-text-tertiary)' }}>@@ session/stream.ts @@</div>
          <div style={{ color: 'var(--color-danger)', background: 'rgba(255,69,58,0.06)' }}>-  for (const ev of tool.events) &#123;</div>
          <div style={{ color: 'var(--color-danger)', background: 'rgba(255,69,58,0.06)' }}>-    const key = ev.turnId;</div>
          <div style={{ color: 'var(--color-danger)', background: 'rgba(255,69,58,0.06)' }}>-    groups[key] ??= &#123; turn: key, events: [] &#125;;</div>
          <div style={{ color: 'var(--color-text-tertiary)' }}>   // …</div>
          <div style={{ color: 'var(--color-success)', background: 'rgba(48,209,88,0.06)' }}>+  const groups = groupToolCallsByTurn(tool.events);</div>
        </div>
      </div>
    );
  }
  // agent
  return (
    <div>
      <div className="flex items-center gap-2" style={{ marginBottom: 6 }}>
        <span style={{ width: 16, height: 16, borderRadius: 4, background: 'color-mix(in srgb, #E8572A 20%, transparent)', color: 'var(--color-accent)', fontSize: 10, fontWeight: 700, display: 'inline-flex', alignItems: 'center', justifyContent: 'center', fontFamily: 'var(--font-mono)' }}>C</span>
        <span className="mono" style={{ fontSize: 11, color: 'var(--color-text-secondary)' }}>claude</span>
        <span className="mono" style={{ fontSize: 10, color: 'var(--color-text-tertiary)' }}>· {m.time}</span>
      </div>
      <div style={{ fontSize: 14, lineHeight: 1.65, color: 'var(--color-text-primary)' }}>{m.body}</div>
      {m.action === 'approve' && (
        <div className="flex gap-2" style={{ marginTop: 10 }}>
          <button className="btn btn-primary" style={{ height: 28, padding: '0 12px', fontSize: 12 }}><Icon.Check size={12} />Approve</button>
          <button className="btn btn-ghost" style={{ height: 28, padding: '0 12px', fontSize: 12 }}>Edit</button>
          <button className="btn btn-ghost" style={{ height: 28, padding: '0 12px', fontSize: 12 }}>Reject</button>
        </div>
      )}
    </div>
  );
}

function Composer({ value, onChange }) {
  return (
    <div style={{ borderTop: '1px solid var(--color-divider)', padding: '14px 28px', background: 'var(--color-canvas)' }}>
      <div style={{ maxWidth: 820, margin: '0 auto' }}>
        <div style={{ background: 'var(--color-surface-panel)', border: '1px solid var(--color-divider)', borderRadius: 10, padding: 12, transition: 'border-color 0.15s' }}>
          <textarea
            value={value}
            onChange={e => onChange(e.target.value)}
            placeholder="Send a message — ⌘⏎ to run"
            style={{ width: '100%', background: 'transparent', border: 0, outline: 0, resize: 'none', color: 'var(--color-text-primary)', fontSize: 13.5, lineHeight: 1.6, fontFamily: 'var(--font-sans)', minHeight: 48 }}
          />
          <div className="flex items-center gap-2" style={{ marginTop: 8 }}>
            <button className="mono-chip" style={{ cursor: 'pointer', border: 0 }}><Icon.Plus size={10} />attach</button>
            <button className="mono-chip" style={{ cursor: 'pointer', border: 0 }}><Icon.Wrench size={10} />skills</button>
            <button className="mono-chip" style={{ cursor: 'pointer', border: 0 }}><Icon.Hash size={10} />#coord.core</button>
            <span className="mono" style={{ fontSize: 10, color: 'var(--color-text-tertiary)', marginLeft: 'auto' }}>claude-sonnet · 184k ctx</span>
            <button className="btn btn-primary" style={{ height: 28, padding: '0 12px', fontSize: 12 }}>
              <Icon.Send size={11} />Send
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

function SessionInspector() {
  const [tab, setTab] = React.useState('trace');
  return (
    <div style={{ width: 320, flexShrink: 0, borderLeft: '1px solid var(--color-divider)', background: 'var(--color-canvas)', display: 'flex', flexDirection: 'column', minHeight: 0 }}>
      <div style={{ padding: 12, borderBottom: '1px solid var(--color-divider)' }}>
        <Pills value={tab} onChange={setTab} items={[
          { value: 'trace', label: 'Trace' },
          { value: 'memory', label: 'Memory' },
          { value: 'files', label: 'Files' },
        ]} />
      </div>
      <div className="scroll-y" style={{ flex: 1, padding: '14px 16px' }}>
        {tab === 'trace' && (
          <>
            <div className="eyebrow" style={{ marginBottom: 10 }}>Timeline</div>
            <div style={{ position: 'relative', paddingLeft: 14 }}>
              <div style={{ position: 'absolute', left: 3, top: 6, bottom: 6, width: 1, background: 'var(--color-divider)' }} />
              {[
                { t: '12:02', k: 'start', label: 'Session resumed' },
                { t: '12:03', k: 'user', label: 'Prompt sent' },
                { t: '12:03', k: 'tool', label: 'shell.safe-run · 212ms' },
                { t: '12:03', k: 'tool', label: 'file.read · 18ms' },
                { t: '12:04', k: 'agent', label: 'Diff proposed' },
                { t: '12:04', k: 'approval', label: 'Awaiting approval' },
              ].map((s, i) => (
                <div key={i} style={{ display: 'flex', gap: 10, paddingBottom: 14, position: 'relative' }}>
                  <span style={{
                    position: 'absolute', left: -14, top: 4, width: 8, height: 8, borderRadius: 4,
                    background: s.k === 'approval' ? 'var(--color-accent)' : s.k === 'tool' ? 'var(--color-success)' : 'var(--color-text-tertiary)',
                    boxShadow: s.k === 'approval' ? '0 0 0 3px rgba(232,87,42,0.18)' : 'none',
                  }} />
                  <div style={{ flex: 1 }}>
                    <div style={{ fontSize: 12.5, color: 'var(--color-text-primary)' }}>{s.label}</div>
                    <div className="mono" style={{ fontSize: 10, color: 'var(--color-text-tertiary)', marginTop: 2 }}>{s.t}</div>
                  </div>
                </div>
              ))}
            </div>

            <div className="eyebrow" style={{ marginTop: 16, marginBottom: 10 }}>Usage</div>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
              {[
                { l: 'input', v: '12,481 tok' },
                { l: 'output', v: '2,108 tok' },
                { l: 'tools', v: '3 calls' },
                { l: 'cost', v: '$0.048' },
              ].map(r => (
                <div key={r.l} className="flex items-center" style={{ justifyContent: 'space-between', fontSize: 12 }}>
                  <span className="mono" style={{ color: 'var(--color-text-tertiary)' }}>{r.l}</span>
                  <span className="mono" style={{ color: 'var(--color-text-primary)' }}>{r.v}</span>
                </div>
              ))}
            </div>
          </>
        )}
        {tab === 'memory' && (
          <>
            <div className="eyebrow" style={{ marginBottom: 10 }}>Loaded memories</div>
            {['agh-architecture.md','operator-voice.md','sessions-model.md'].map(m => (
              <div key={m} className="flex items-center gap-2" style={{ padding: '8px 0', borderBottom: '1px solid var(--color-divider)' }}>
                <Icon.FileCode size={12} style={{ color: 'var(--color-text-tertiary)' }} />
                <span className="mono" style={{ fontSize: 11.5, color: 'var(--color-text-primary)', flex: 1 }}>{m}</span>
                <span className="mono" style={{ fontSize: 10, color: 'var(--color-text-tertiary)' }}>ws</span>
              </div>
            ))}
          </>
        )}
        {tab === 'files' && (
          <>
            <div className="eyebrow" style={{ marginBottom: 10 }}>Read this session</div>
            {['stream.ts','replay.ts','map.ts'].map(f => (
              <div key={f} className="flex items-center gap-2" style={{ padding: '8px 0', borderBottom: '1px solid var(--color-divider)' }}>
                <Icon.FileCode size={12} style={{ color: 'var(--color-text-tertiary)' }} />
                <span className="mono" style={{ fontSize: 11.5, color: 'var(--color-text-primary)', flex: 1 }}>{f}</span>
              </div>
            ))}
          </>
        )}
      </div>
    </div>
  );
}

// ====================== SETTINGS ======================
function SettingsPage() {
  const [section, setSection] = React.useState('general');
  const nav = [
    { id: 'general', label: 'General', icon: Icon.Settings },
    { id: 'workspace', label: 'Workspace', icon: Icon.Box },
    { id: 'agents', label: 'Agents', icon: Icon.Bot },
    { id: 'network', label: 'Network', icon: Icon.Network },
    { id: 'providers', label: 'Providers', icon: Icon.Database },
    { id: 'security', label: 'Security', icon: Icon.AlertCircle },
    { id: 'about', label: 'About', icon: Icon.Circle },
  ];

  return (
    <>
      <PageHeader title="Settings" icon={Icon.Settings} />
      <div style={{ flex: 1, display: 'flex', minHeight: 0 }}>
        <div style={{ width: 200, borderRight: '1px solid var(--color-divider)', padding: 12, flexShrink: 0 }}>
          {nav.map(n => (
            <div key={n.id} className={`nav-item ${section === n.id ? 'active' : ''}`}
                 onClick={() => setSection(n.id)}>
              <n.icon size={13} />
              <span>{n.label}</span>
            </div>
          ))}
        </div>
        <div className="scroll-y" style={{ flex: 1, padding: 32 }}>
          <div style={{ maxWidth: 640 }}>
            {section === 'general' && <SettingsGeneral />}
            {section === 'agents' && <SettingsAgents />}
            {section === 'network' && <SettingsNetwork />}
            {section === 'providers' && <SettingsProviders />}
            {section !== 'general' && section !== 'agents' && section !== 'network' && section !== 'providers' && (
              <SettingsPlaceholder section={section} />
            )}
          </div>
        </div>
      </div>
    </>
  );
}

function SettingsGeneral() {
  return (
    <>
      <h1 style={{ margin: '0 0 4px', fontSize: 24, fontWeight: 500, letterSpacing: '-0.02em' }}>General</h1>
      <p style={{ margin: 0, color: 'var(--color-text-secondary)', fontSize: 14 }}>Runtime-wide preferences. Changes apply immediately.</p>

      <div style={{ marginTop: 28, display: 'flex', flexDirection: 'column', gap: 18 }}>
        <Row label="Runtime name" hint="Appears on peer discovery.">
          <input className="input" defaultValue="pedronauck@laptop" style={{ width: 280 }} />
        </Row>
        <Row label="Default workspace" hint="Used when no scope is set.">
          <select className="input" defaultValue="agh-core" style={{ width: 280 }}>
            <option>agh-core</option><option>compozy</option><option>research</option>
          </select>
        </Row>
        <Row label="Auto-resume sessions" hint="Pick up sessions on restart.">
          <Toggle on={true} />
        </Row>
        <Row label="Approval mode" hint="When to interrupt for human review.">
          <Pills value="risky" onChange={()=>{}} items={[
            { value: 'none', label: 'Never' }, { value: 'risky', label: 'Risky only' }, { value: 'always', label: 'Always' }]} />
        </Row>
        <Row label="Telemetry" hint="Send anonymous usage data to Compozy.">
          <Toggle on={false} />
        </Row>
      </div>
    </>
  );
}

function SettingsAgents() {
  const agents = [
    { name: 'claude', label: 'Claude Code', status: 'ready', path: '/usr/local/bin/claude' },
    { name: 'codex', label: 'Codex CLI', status: 'ready', path: '/opt/codex/bin/codex' },
    { name: 'gemini', label: 'Gemini CLI', status: 'missing', path: '—' },
    { name: 'opencode', label: 'OpenCode', status: 'ready', path: '/usr/local/bin/opencode' },
    { name: 'cursor', label: 'Cursor', status: 'not-installed', path: '—' },
  ];
  return (
    <>
      <h1 style={{ margin: '0 0 4px', fontSize: 24, fontWeight: 500, letterSpacing: '-0.02em' }}>Agents</h1>
      <p style={{ margin: 0, color: 'var(--color-text-secondary)', fontSize: 14 }}>CLIs AGH can launch as durable sessions. Paths are auto-detected on boot.</p>
      <div className="card" style={{ marginTop: 28, padding: 0, overflow: 'hidden' }}>
        {agents.map((a, i) => (
          <div key={a.name} style={{ display: 'grid', gridTemplateColumns: '28px 1fr 110px 90px', gap: 14, padding: '14px 18px', alignItems: 'center', borderBottom: i < agents.length - 1 ? '1px solid var(--color-divider)' : 'none' }}>
            <span style={{ width: 24, height: 24, borderRadius: 6, background: 'var(--color-surface-elevated)', display: 'flex', alignItems: 'center', justifyContent: 'center', fontFamily: 'var(--font-mono)', fontSize: 10, fontWeight: 700, color: 'var(--color-accent)' }}>{a.name[0].toUpperCase()}</span>
            <div>
              <div style={{ fontSize: 13, fontWeight: 500 }}>{a.label}</div>
              <span className="mono" style={{ fontSize: 10.5, color: 'var(--color-text-tertiary)' }}>{a.path}</span>
            </div>
            <span className={`mono-chip ${a.status === 'ready' ? 'success' : a.status === 'missing' ? 'warning' : ''}`}>{a.status}</span>
            <button className="btn btn-ghost" style={{ height: 26, fontSize: 11, padding: '0 10px', justifySelf: 'end' }}>
              {a.status === 'ready' ? 'Configure' : 'Install'}
            </button>
          </div>
        ))}
      </div>
    </>
  );
}

function SettingsNetwork() {
  return (
    <>
      <h1 style={{ margin: '0 0 4px', fontSize: 24, fontWeight: 500, letterSpacing: '-0.02em' }}>Network</h1>
      <p style={{ margin: 0, color: 'var(--color-text-secondary)', fontSize: 14 }}>
        <span className="mono" style={{ color: 'var(--color-text-primary)' }}>agh-network/v0</span> runs over NATS. Changes reconnect the bus.
      </p>
      <div style={{ marginTop: 28, display: 'flex', flexDirection: 'column', gap: 18 }}>
        <Row label="NATS URL">
          <input className="input" defaultValue="nats://127.0.0.1:4222" style={{ width: 320, fontFamily: 'var(--font-mono)', fontSize: 12 }} />
        </Row>
        <Row label="Peer ID">
          <input className="input" defaultValue="pedronauck@laptop" style={{ width: 320, fontFamily: 'var(--font-mono)', fontSize: 12 }} />
        </Row>
        <Row label="Advertise on LAN" hint="Discoverable via mDNS on local subnet.">
          <Toggle on={true} />
        </Row>
        <Row label="Accept delegations" hint="Allow peers to assign tasks.">
          <Toggle on={true} />
        </Row>
      </div>
      <div className="card" style={{ marginTop: 24, padding: 16 }}>
        <div className="flex items-center gap-2" style={{ marginBottom: 8 }}>
          <span className="dot success pulse" />
          <span className="mono" style={{ fontSize: 11, color: 'var(--color-text-primary)' }}>connected</span>
          <span className="mono" style={{ fontSize: 10, color: 'var(--color-text-tertiary)', marginLeft: 'auto' }}>uptime 2h 14m</span>
        </div>
        <span style={{ fontSize: 12, color: 'var(--color-text-secondary)' }}>12 peers · 6 channels · 1.28k messages · p95 rtt 24ms</span>
      </div>
    </>
  );
}

function SettingsProviders() {
  const providers = [
    { name: 'Anthropic', env: 'ANTHROPIC_API_KEY', ok: true },
    { name: 'OpenAI', env: 'OPENAI_API_KEY', ok: true },
    { name: 'Google Gemini', env: 'GEMINI_API_KEY', ok: false },
    { name: 'xAI', env: 'XAI_API_KEY', ok: false },
  ];
  return (
    <>
      <h1 style={{ margin: '0 0 4px', fontSize: 24, fontWeight: 500, letterSpacing: '-0.02em' }}>Providers</h1>
      <p style={{ margin: 0, color: 'var(--color-text-secondary)', fontSize: 14 }}>Model providers are resolved by env var at runtime — keys never leave the local machine.</p>
      <div style={{ marginTop: 28, display: 'flex', flexDirection: 'column', gap: 8 }}>
        {providers.map(p => (
          <div key={p.name} className="card" style={{ padding: '14px 18px', display: 'grid', gridTemplateColumns: '1fr 200px 80px 90px', gap: 14, alignItems: 'center' }}>
            <div>
              <div style={{ fontSize: 13, fontWeight: 500 }}>{p.name}</div>
              <span className="mono" style={{ fontSize: 10.5, color: 'var(--color-text-tertiary)' }}>{p.env}</span>
            </div>
            <span className="mono" style={{ fontSize: 11, color: 'var(--color-text-secondary)' }}>{p.ok ? 'sk-ant-…a4f9' : 'not configured'}</span>
            <span className={`mono-chip ${p.ok ? 'success' : ''}`}>{p.ok ? 'active' : 'off'}</span>
            <button className="btn btn-ghost" style={{ height: 26, fontSize: 11, padding: '0 10px', justifySelf: 'end' }}>{p.ok ? 'Edit' : 'Set'}</button>
          </div>
        ))}
      </div>
    </>
  );
}

function SettingsPlaceholder({ section }) {
  return (
    <>
      <h1 style={{ margin: '0 0 4px', fontSize: 24, fontWeight: 500, letterSpacing: '-0.02em', textTransform: 'capitalize' }}>{section}</h1>
      <p style={{ margin: 0, color: 'var(--color-text-secondary)', fontSize: 14 }}>Configuration for this area follows the same grammar as the others above.</p>
    </>
  );
}

function Row({ label, hint, children }) {
  return (
    <div style={{ display: 'grid', gridTemplateColumns: '220px 1fr', gap: 20, alignItems: 'start' }}>
      <div>
        <div style={{ fontSize: 13, fontWeight: 500, color: 'var(--color-text-primary)' }}>{label}</div>
        {hint && <div style={{ fontSize: 12, color: 'var(--color-text-tertiary)', marginTop: 3, lineHeight: 1.5 }}>{hint}</div>}
      </div>
      <div>{children}</div>
    </div>
  );
}

function Toggle({ on }) {
  const [v, setV] = React.useState(on);
  return (
    <button onClick={() => setV(!v)} style={{
      width: 34, height: 20, borderRadius: 10, position: 'relative',
      background: v ? 'var(--color-accent)' : 'var(--color-surface-elevated)',
      border: '1px solid ' + (v ? 'var(--color-accent)' : 'var(--color-divider)'),
      transition: 'all 0.15s', cursor: 'pointer',
    }}>
      <span style={{
        position: 'absolute', top: 1, left: v ? 15 : 1, width: 16, height: 16, borderRadius: 8,
        background: v ? 'var(--color-accent-ink)' : 'var(--color-text-secondary)',
        transition: 'left 0.15s',
      }} />
    </button>
  );
}

Object.assign(window, { SessionPage, SettingsPage });
