// Settings — 10 sections matching the original web app
const { Icon, PageHeader, Pills, Metric, Section, SectionCard, FieldRow, StatGrid, Banner,
        SaveBar, Toggle, SourceBadge, Modal, StatusLine, Tabs } = window;

const SETTINGS_SECTIONS = [
  { slug: 'general', label: 'General', icon: Icon.Settings },
  { slug: 'providers', label: 'Providers', icon: Icon.Database },
  { slug: 'mcp-servers', label: 'MCP Servers', icon: Icon.Server },
  { slug: 'environments', label: 'Environments', icon: Icon.Box },
  { slug: 'memory', label: 'Memory', icon: Icon.Layers },
  { slug: 'skills', label: 'Skills', icon: Icon.Wrench },
  { slug: 'automation', label: 'Automation', icon: Icon.Zap },
  { slug: 'network', label: 'Network', icon: Icon.Network },
  { slug: 'observability', label: 'Observability', icon: Icon.Activity },
  { slug: 'hooks-extensions', label: 'Hooks & Extensions', icon: Icon.Webhook },
];

function SettingsPage() {
  const [slug, setSlug] = React.useState(() => localStorage.getItem('agh:settings-slug') || 'general');
  React.useEffect(() => { localStorage.setItem('agh:settings-slug', slug); }, [slug]);
  const section = SETTINGS_SECTIONS.find(s => s.slug === slug) || SETTINGS_SECTIONS[0];

  const pages = {
    'general': SettingsGeneral,
    'providers': SettingsProviders,
    'mcp-servers': SettingsMcp,
    'environments': SettingsEnvironments,
    'memory': SettingsMemory,
    'skills': SettingsSkills,
    'automation': SettingsAutomation,
    'network': SettingsNetwork,
    'observability': SettingsObservability,
    'hooks-extensions': SettingsHooks,
  };
  const P = pages[slug];

  return (
    <>
      <PageHeader title="Settings" icon={Icon.Settings}
        meta={<StatusLine daemon={true} items={[<span key="v">runtime v0.4.1</span>, <span key="c">config: ~/.agh/config.toml</span>]} />}
      />
      <div style={{ flex: 1, display: 'flex', minHeight: 0 }}>
        <div style={{ width: 220, borderRight: '1px solid var(--color-divider)', padding: 10, flexShrink: 0, background: 'var(--color-canvas-deep)', overflowY: 'auto' }}>
          {SETTINGS_SECTIONS.map(s => (
            <div key={s.slug} className={`nav-item compact ${slug === s.slug ? 'active' : ''}`} onClick={() => setSlug(s.slug)}>
              <s.icon size={13} style={{ color: slug === s.slug ? 'var(--color-text-primary)' : 'var(--color-text-tertiary)' }} />
              <span>{s.label}</span>
            </div>
          ))}
        </div>
        <div className="scroll-y" style={{ flex: 1 }}>
          <div style={{ padding: '28px 32px 4px', maxWidth: 880 }}>
            <h1 style={{ margin: 0, fontSize: 22, fontWeight: 500, letterSpacing: '-0.02em' }}>{section.label}</h1>
            <p style={{ margin: '6px 0 22px', color: 'var(--color-text-secondary)', fontSize: 13 }}>
              {SETTINGS_DESCRIPTIONS[slug]}
            </p>
          </div>
          <div style={{ padding: '0 32px 32px', maxWidth: 880 }}>
            {P && <P />}
          </div>
        </div>
      </div>
    </>
  );
}

const SETTINGS_DESCRIPTIONS = {
  'general': 'Runtime-wide preferences. Changes apply immediately unless noted.',
  'providers': 'Model providers spawned by AGH as ACP subprocesses. Keys resolve from env at runtime.',
  'mcp-servers': 'Model Context Protocol servers exposed to agents. Scope determines which workspaces can see them.',
  'environments': 'Execution profiles — env vars, working directory, approval mode, and secret overlays.',
  'memory': 'How AGH stores, scopes, and compacts memory across sessions and workspaces.',
  'skills': 'Global skill policies. For per-skill config, see Skills in the sidebar.',
  'automation': 'Global automation runner. Cron and event triggers live here.',
  'network': 'agh-network/v0 open agent network protocol configuration and peer discovery.',
  'observability': 'Traces, metrics, and logs. Wire up OTel collectors or ship to stdout.',
  'hooks-extensions': 'Lifecycle hooks fired on session, tool, and bridge events. Drop scripts anywhere on PATH.',
};

// ============== GENERAL ==============
function SettingsGeneral() {
  const [draft, setDraft] = React.useState({
    defaultAgent: 'claude',
    defaultProvider: '',
    defaultEnvironment: 'local',
    permissions: 'approve-reads',
    sessionTimeout: 0,
    autoResume: true,
    telemetry: false,
    theme: 'system',
  });
  const [initial] = React.useState(() => ({ ...draft }));
  const dirty = JSON.stringify(draft) !== JSON.stringify(initial);
  const [saving, setSaving] = React.useState(false);

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      <SectionCard eyebrow="Runtime" note="read-only">
        <StatGrid items={[
          { label: 'UDS socket', value: '~/.agh/agh.sock' },
          { label: 'HTTP bind', value: '127.0.0.1:4747' },
          { label: 'Active sessions', value: '3 / 25 max' },
          { label: 'Concurrent agents', value: '2 / 8 max', tone: 'accent' },
        ]} />
      </SectionCard>

      <SectionCard eyebrow="Defaults" note="applied to new sessions">
        <FieldRow label="Default agent" description="Used when a new session doesn't specify one" tag="CONFIG.TOML">
          <select className="input" value={draft.defaultAgent} onChange={e => setDraft({ ...draft, defaultAgent: e.target.value })} style={{ width: 220 }}>
            <option>claude</option><option>codex</option><option>gemini</option><option>opencode</option>
          </select>
        </FieldRow>
        <FieldRow label="Default provider" description="LLM backend agents spawn against" tag="OPTIONAL">
          <input className="input" placeholder="auto" value={draft.defaultProvider} onChange={e => setDraft({ ...draft, defaultProvider: e.target.value })} style={{ width: 220 }} />
        </FieldRow>
        <FieldRow label="Default environment" description="Execution profile for new workspaces" tag="DEFAULT">
          <input className="input" value={draft.defaultEnvironment} onChange={e => setDraft({ ...draft, defaultEnvironment: e.target.value })} style={{ width: 220, fontFamily: 'var(--font-mono)', fontSize: 12 }} />
        </FieldRow>
      </SectionCard>

      <SectionCard eyebrow="Permissions" note="tool approval policy">
        <div style={{ padding: '10px 0' }}>
          <div className="flex gap-2 mb-2">
            {['deny-all','approve-reads','approve-all'].map(m => (
              <button key={m} className={`pill ${draft.permissions === m ? 'active' : ''}`} onClick={() => setDraft({ ...draft, permissions: m })}>{m}</button>
            ))}
          </div>
          <p style={{ margin: 0, fontSize: 12, color: 'var(--color-text-tertiary)', lineHeight: 1.6 }}>
            {draft.permissions === 'deny-all' && 'All tool calls denied unless explicitly allowed by agent frontmatter.'}
            {draft.permissions === 'approve-reads' && 'Read-only tool calls auto-approved. Writes require confirmation.'}
            {draft.permissions === 'approve-all' && 'All tool calls auto-approved. Agents can lower this individually.'}
          </p>
        </div>
      </SectionCard>

      <SectionCard eyebrow="Session" note="runtime limits">
        <FieldRow label="Session timeout" description="0 disables force-close" tag="SECONDS">
          <input type="number" className="input" value={draft.sessionTimeout}
            onChange={e => setDraft({ ...draft, sessionTimeout: Number(e.target.value) })} style={{ width: 120 }} />
        </FieldRow>
        <FieldRow label="Auto-resume sessions" description="Pick up durable sessions on restart">
          <Toggle on={draft.autoResume} onChange={v => setDraft({ ...draft, autoResume: v })} />
        </FieldRow>
        <FieldRow label="Telemetry" description="Send anonymous usage data to Compozy">
          <Toggle on={draft.telemetry} onChange={v => setDraft({ ...draft, telemetry: v })} />
        </FieldRow>
      </SectionCard>

      <SectionCard eyebrow="Appearance">
        <FieldRow label="Theme" description="System follows OS preference. AGH ships dark-only today.">
          <div className="pills">
            {[{v:'system',l:'System'},{v:'dark',l:'Dark'},{v:'light',l:'Light',d:true}].map(t => (
              <button key={t.v} className={`pill ${draft.theme === t.v ? 'active' : ''}`} disabled={t.d}
                onClick={() => !t.d && setDraft({ ...draft, theme: t.v })} style={t.d ? { opacity: 0.4, cursor: 'not-allowed' } : {}}>{t.l}</button>
            ))}
          </div>
        </FieldRow>
      </SectionCard>

      <SaveBar dirty={dirty} saving={saving} onReset={() => setDraft(initial)}
        onSave={() => { setSaving(true); setTimeout(() => setSaving(false), 700); }}
        lastApplied="just now" />
    </div>
  );
}

// ============== PROVIDERS ==============
function SettingsProviders() {
  const [providers, setProviders] = React.useState([
    { name: 'claude', command: 'npx @agentclientprotocol/claude-agent-acp@latest', model: 'claude-sonnet-4.5', apiKeyEnv: 'ANTHROPIC_API_KEY', keyPresent: true, cmdOk: true, source: 'builtin', default: true },
    { name: 'codex', command: 'codex-cli', model: 'gpt-5-turbo', apiKeyEnv: 'OPENAI_API_KEY', keyPresent: true, cmdOk: true, source: 'overlay' },
    { name: 'gemini', command: 'gemini-cli', model: 'gemini-2.5-pro', apiKeyEnv: 'GEMINI_API_KEY', keyPresent: false, cmdOk: true, source: 'builtin' },
    { name: 'opencode', command: 'opencode', model: '', apiKeyEnv: '', keyPresent: false, cmdOk: true, source: 'builtin' },
    { name: 'kiro', command: 'kiro', model: '', apiKeyEnv: 'XAI_API_KEY', keyPresent: false, cmdOk: false, source: 'overlay' },
  ]);
  const [editor, setEditor] = React.useState({ open: false, mode: 'create', draft: null });

  const counts = {
    total: providers.length,
    installed: providers.filter(p => p.cmdOk && (!p.apiKeyEnv || p.keyPresent)).length,
    missing: providers.filter(p => !p.cmdOk).length,
    unconfigured: providers.filter(p => p.cmdOk && p.apiKeyEnv && !p.keyPresent).length,
  };

  const openCreate = () => setEditor({ open: true, mode: 'create', draft: { name: '', command: '', model: '', apiKeyEnv: '' } });
  const openEdit = p => setEditor({ open: true, mode: 'edit', draft: { ...p } });
  const close = () => setEditor({ open: false, mode: 'create', draft: null });

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      <div className="flex items-center gap-3 mb-2">
        <span className="mono" style={{ fontSize: 11.5, color: 'var(--color-text-secondary)' }}>
          <b style={{ color: 'var(--color-text-primary)' }}>{counts.total}</b> providers
        </span>
        <span className="meta-dot">·</span>
        <span className="mono" style={{ fontSize: 11.5, color: 'var(--color-text-tertiary)' }}>{counts.installed} installed</span>
        <span className="meta-dot">·</span>
        <span className="mono" style={{ fontSize: 11.5, color: 'var(--color-warning)' }}>{counts.missing} binary missing</span>
        <span className="meta-dot">·</span>
        <span className="mono" style={{ fontSize: 11.5, color: 'var(--color-warning)' }}>{counts.unconfigured} unconfigured</span>
        <button className="btn btn-primary" style={{ marginLeft: 'auto' }} onClick={openCreate}><Icon.Plus size={12} />New provider</button>
      </div>

      <div style={{ border: '1px solid var(--color-divider)', borderRadius: 10, overflow: 'hidden' }}>
        <table className="data-table">
          <thead>
            <tr>
              <th>Provider</th><th>Command</th><th>Default model</th><th>API key env</th><th>Source</th><th style={{ textAlign: 'right' }}>Actions</th>
            </tr>
          </thead>
          <tbody>
            {providers.map(p => {
              const state = !p.cmdOk ? 'warning' : (p.apiKeyEnv && !p.keyPresent) ? 'warning' : '';
              return (
                <tr key={p.name}>
                  <td>
                    <div className="flex items-center gap-2">
                      <span className={`dot ${state}`} />
                      <span className="mono" style={{ color: 'var(--color-text-primary)' }}>{p.name}</span>
                      {p.default && <span className="mono-chip accent">DEFAULT</span>}
                    </div>
                  </td>
                  <td className="mono" style={{ fontSize: 11.5, color: 'var(--color-text-secondary)' }}>{p.command || '—'}</td>
                  <td className="mono" style={{ fontSize: 11.5, color: 'var(--color-text-secondary)' }}>{p.model || '—'}</td>
                  <td>
                    <div className="flex items-center gap-2">
                      <span className="mono" style={{ fontSize: 11.5, color: 'var(--color-text-secondary)' }}>{p.apiKeyEnv || '—'}</span>
                      {p.apiKeyEnv && <span className={`mono-chip ${p.keyPresent ? 'success' : 'warning'}`}>{p.keyPresent ? 'SET' : 'MISSING'}</span>}
                    </div>
                  </td>
                  <td><SourceBadge source={p.source} /></td>
                  <td style={{ textAlign: 'right' }}>
                    <button className="btn btn-ghost" style={{ height: 26, padding: '0 10px', fontSize: 12 }} onClick={() => openEdit(p)}>Edit</button>
                    <button className="btn btn-ghost" style={{ height: 26, padding: '0 10px', fontSize: 12, marginLeft: 6 }}
                            disabled={p.source === 'builtin'} title={p.source === 'builtin' ? 'Builtins cannot be deleted' : ''}>Delete</button>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>

      <Modal open={editor.open} onClose={close}
        title={editor.mode === 'create' ? 'New provider' : `Edit provider · ${editor.draft?.name || ''}`}
        description={editor.mode === 'create' ? 'Add a new provider overlay. Saved entries replace any prior overlay for this name.' : 'Saving replaces the entire overlay entry for this provider.'}
        footer={<>
          <span className="mono text-xs" style={{ color: 'var(--color-text-tertiary)', flex: 1 }}>Full PUT — no partial updates</span>
          <button className="btn btn-ghost" onClick={close}>Cancel</button>
          <button className="btn btn-primary" onClick={close}>{editor.mode === 'create' ? 'Create provider' : 'Replace overlay'}</button>
        </>}>
        {editor.draft && <>
          <FieldRow label="Name" description="Lowercase identifier in agent frontmatter and CLI flags" tag={editor.mode === 'edit' ? 'LOCKED' : 'REQUIRED'}>
            <input className="input" disabled={editor.mode === 'edit'} value={editor.draft.name}
              onChange={e => setEditor(s => ({ ...s, draft: { ...s.draft, name: e.target.value } }))}
              placeholder="claude" style={{ width: 240, fontFamily: 'var(--font-mono)', fontSize: 12 }} />
          </FieldRow>
          <FieldRow label="Command" description="Executable used to launch the ACP subprocess" tag="OVERLAY">
            <input className="input" value={editor.draft.command}
              onChange={e => setEditor(s => ({ ...s, draft: { ...s.draft, command: e.target.value } }))}
              placeholder="npx @agentclientprotocol/…" style={{ width: 380, fontFamily: 'var(--font-mono)', fontSize: 12 }} />
          </FieldRow>
          <FieldRow label="Default model" description="Sent to provider when agent doesn't specify one" tag="OPTIONAL">
            <input className="input" value={editor.draft.model}
              onChange={e => setEditor(s => ({ ...s, draft: { ...s.draft, model: e.target.value } }))}
              placeholder="claude-sonnet-4.5" style={{ width: 260, fontFamily: 'var(--font-mono)', fontSize: 12 }} />
          </FieldRow>
          <FieldRow label="API key env" description="Env var the daemon reads before spawning" tag="OPTIONAL">
            <div className="flex items-center gap-2">
              <Icon.Key size={13} style={{ color: 'var(--color-text-tertiary)' }} />
              <input className="input" value={editor.draft.apiKeyEnv}
                onChange={e => setEditor(s => ({ ...s, draft: { ...s.draft, apiKeyEnv: e.target.value } }))}
                placeholder="ANTHROPIC_API_KEY" style={{ width: 240, fontFamily: 'var(--font-mono)', fontSize: 12 }} />
            </div>
          </FieldRow>
        </>}
      </Modal>
    </div>
  );
}

// ============== MCP SERVERS ==============
function SettingsMcp() {
  const [servers] = React.useState([
    { name: 'filesystem', transport: 'stdio', scope: 'workspace', status: 'connected', tools: 8, command: 'npx @mcp/filesystem --root .' },
    { name: 'postgres-prod', transport: 'sse', scope: 'global', status: 'connected', tools: 12, url: 'https://mcp.internal/pg' },
    { name: 'brave-search', transport: 'stdio', scope: 'global', status: 'connected', tools: 2, command: 'npx @mcp/brave-search' },
    { name: 'github', transport: 'sse', scope: 'workspace', status: 'error', tools: 0, url: 'https://mcp.github/api', error: 'auth: GITHUB_TOKEN missing' },
  ]);
  const [sel, setSel] = React.useState(0);
  const [editor, setEditor] = React.useState(false);
  const s = servers[sel];

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      <div className="flex items-center gap-3 mb-2">
        <span className="mono text-xs text-secondary">
          <b className="text-primary">{servers.length}</b> servers · <span className="text-success">{servers.filter(s => s.status === 'connected').length} connected</span> · <span className="text-danger">{servers.filter(s => s.status === 'error').length} errored</span>
        </span>
        <button className="btn btn-primary" style={{ marginLeft: 'auto' }} onClick={() => setEditor(true)}><Icon.Plus size={12} />Add server</button>
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: '280px 1fr', gap: 14 }}>
        <div style={{ border: '1px solid var(--color-divider)', borderRadius: 10, overflow: 'hidden' }}>
          {servers.map((m,i) => (
            <div key={m.name} className={`list-row ${sel === i ? 'active' : ''}`} onClick={() => setSel(i)}
                 style={{ borderRadius: 0, borderBottom: i < servers.length-1 ? '1px solid var(--color-divider)' : 'none' }}>
              <Icon.Server size={14} style={{ color: 'var(--color-text-tertiary)', marginTop: 1 }} />
              <div style={{ flex: 1, minWidth: 0 }}>
                <div className="flex items-center gap-2">
                  <span className={`dot ${m.status === 'connected' ? 'success' : 'danger'}`} />
                  <span className="mono truncate" style={{ fontSize: 12.5, color: 'var(--color-text-primary)' }}>{m.name}</span>
                </div>
                <div className="flex items-center gap-2" style={{ marginTop: 4 }}>
                  <span className="mono-chip">{m.transport}</span>
                  <span className="mono-chip">{m.scope}</span>
                  <span className="mono" style={{ fontSize: 10, color: 'var(--color-text-tertiary)', marginLeft: 'auto' }}>{m.tools} tools</span>
                </div>
              </div>
            </div>
          ))}
        </div>
        <div className="section-card">
          <div className="section-card-head">
            <div className="flex items-center gap-2">
              <span className={`dot ${s.status === 'connected' ? 'success' : 'danger'}`} />
              <span className="mono" style={{ fontSize: 13, color: 'var(--color-text-primary)' }}>{s.name}</span>
              <span className="mono-chip">{s.transport}</span>
              <span className="mono-chip">{s.scope}</span>
            </div>
            <div className="flex items-center gap-1">
              <button className="btn-icon" title="Restart"><Icon.RefreshCw size={13} /></button>
              <button className="btn-icon" title="Edit"><Icon.Edit size={13} /></button>
              <button className="btn-icon" title="Delete"><Icon.Trash size={13} /></button>
            </div>
          </div>
          <div className="section-card-body" style={{ padding: '14px 18px' }}>
            {s.error && <Banner tone="danger">{s.error}</Banner>}
            <div style={{ marginTop: s.error ? 14 : 0 }}>
              <FieldRow label={s.transport === 'stdio' ? 'Command' : 'URL'} tag="CONFIG">
                <span className="mono" style={{ fontSize: 12, color: 'var(--color-text-primary)' }}>{s.command || s.url}</span>
              </FieldRow>
              <FieldRow label="Tools discovered" description={`${s.tools} tool${s.tools === 1 ? '' : 's'} exposed to agents`}>
                <span className="mono" style={{ fontSize: 12, color: 'var(--color-text-primary)' }}>{s.tools}</span>
              </FieldRow>
              <FieldRow label="Timeout" description="Seconds before a call is aborted" tag="DEFAULT 30s">
                <input className="input" type="number" defaultValue={30} style={{ width: 100 }} />
              </FieldRow>
            </div>
            {s.status === 'connected' && (
              <>
                <div className="eyebrow" style={{ marginTop: 20, marginBottom: 10 }}>Exposed tools</div>
                <div className="flex gap-2" style={{ flexWrap: 'wrap' }}>
                  {['fs.read','fs.write','fs.ls','fs.search','fs.stat','fs.mkdir','fs.rm','fs.chmod'].slice(0, s.tools).map(t =>
                    <span key={t} className="mono-chip">{t}</span>
                  )}
                </div>
              </>
            )}
          </div>
        </div>
      </div>

      <Modal open={editor} onClose={() => setEditor(false)} title="Add MCP server"
        description="Model Context Protocol servers expose tools to agents. stdio launches a subprocess; sse connects to a URL."
        footer={<>
          <button className="btn btn-ghost" onClick={() => setEditor(false)}>Cancel</button>
          <button className="btn btn-primary" onClick={() => setEditor(false)}>Add server</button>
        </>}>
        <FieldRow label="Name" tag="REQUIRED"><input className="input" placeholder="my-mcp" style={{ width: 220, fontFamily: 'var(--font-mono)' }} /></FieldRow>
        <FieldRow label="Transport">
          <div className="pills"><button className="pill active">stdio</button><button className="pill">sse</button></div>
        </FieldRow>
        <FieldRow label="Command" description="Executable to launch"><input className="input" placeholder="npx @mcp/…" style={{ width: 380, fontFamily: 'var(--font-mono)', fontSize: 12 }} /></FieldRow>
        <FieldRow label="Scope" description="global available everywhere; workspace scoped to active workspace">
          <div className="pills"><button className="pill">global</button><button className="pill active">workspace</button></div>
        </FieldRow>
      </Modal>
    </div>
  );
}

// ============== ENVIRONMENTS ==============
function SettingsEnvironments() {
  const [envs, setEnvs] = React.useState([
    { name: 'local', cwd: '~/code/agh-core', approval: 'approve-reads', secrets: 4, default: true },
    { name: 'prod', cwd: '~/ops', approval: 'deny-all', secrets: 9, default: false },
    { name: 'sandbox', cwd: '/tmp/sandbox', approval: 'approve-all', secrets: 0, default: false },
  ]);
  const [sel, setSel] = React.useState(0);
  const e = envs[sel];

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      <div className="flex items-center gap-3 mb-2">
        <span className="mono text-xs text-secondary"><b className="text-primary">{envs.length}</b> environments</span>
        <button className="btn btn-primary" style={{ marginLeft: 'auto' }}><Icon.Plus size={12} />Environment</button>
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: '240px 1fr', gap: 14 }}>
        <div style={{ border: '1px solid var(--color-divider)', borderRadius: 10, overflow: 'hidden' }}>
          {envs.map((env, i) => (
            <div key={env.name} className={`list-row ${sel === i ? 'active' : ''}`} onClick={() => setSel(i)}
                 style={{ borderRadius: 0, borderBottom: i < envs.length-1 ? '1px solid var(--color-divider)' : 'none' }}>
              <Icon.Box size={14} style={{ color: 'var(--color-text-tertiary)', marginTop: 1 }} />
              <div style={{ flex: 1, minWidth: 0 }}>
                <div className="flex items-center gap-2">
                  <span className="mono truncate" style={{ fontSize: 12.5, color: 'var(--color-text-primary)' }}>{env.name}</span>
                  {env.default && <span className="mono-chip accent">DEFAULT</span>}
                </div>
                <span className="mono" style={{ fontSize: 10, color: 'var(--color-text-tertiary)' }}>{env.approval} · {env.secrets} secrets</span>
              </div>
            </div>
          ))}
        </div>

        <div className="section-card">
          <div className="section-card-head">
            <div className="flex items-center gap-2">
              <Icon.Box size={14} style={{ color: 'var(--color-text-secondary)' }} />
              <span className="mono" style={{ fontSize: 13, color: 'var(--color-text-primary)' }}>{e.name}</span>
              {e.default && <span className="mono-chip accent">DEFAULT</span>}
            </div>
            <div className="flex items-center gap-1">
              <button className="btn-icon" title="Edit"><Icon.Edit size={13} /></button>
              <button className="btn-icon" title="Delete" disabled={e.default}><Icon.Trash size={13} /></button>
            </div>
          </div>
          <div className="section-card-body" style={{ padding: '14px 18px' }}>
            <FieldRow label="Working directory" description="Resolved when agents spawn in this env" tag="CWD">
              <input className="input" defaultValue={e.cwd} style={{ width: 320, fontFamily: 'var(--font-mono)', fontSize: 12 }} />
            </FieldRow>
            <FieldRow label="Approval mode" description="Overrides runtime default for this env">
              <div className="pills">
                {['deny-all','approve-reads','approve-all'].map(m => (
                  <button key={m} className={`pill ${e.approval === m ? 'active' : ''}`}>{m}</button>
                ))}
              </div>
            </FieldRow>
            <FieldRow label="Inherit parent env" description="Copy host environment vars into sessions"><Toggle on={true} onChange={() => {}} /></FieldRow>

            <div className="eyebrow" style={{ marginTop: 20, marginBottom: 10 }}>Secrets · {e.secrets}</div>
            <div style={{ border: '1px solid var(--color-divider)', borderRadius: 8, overflow: 'hidden' }}>
              {[
                { key: 'ANTHROPIC_API_KEY', value: 'sk-ant-…a4f9', masked: true },
                { key: 'OPENAI_API_KEY', value: 'sk-…93bc', masked: true },
                { key: 'WORKSPACE_ROOT', value: '/Users/pedro/code/agh', masked: false },
                { key: 'LOG_LEVEL', value: 'debug', masked: false },
              ].slice(0, e.secrets).map((s, i, arr) => (
                <div key={s.key} className="flat-row" style={{ gridTemplateColumns: '220px 1fr 80px 60px', borderBottom: i < arr.length-1 ? '1px solid var(--color-divider)' : 'none' }}>
                  <span className="mono" style={{ fontSize: 11, color: 'var(--color-text-primary)' }}>{s.key}</span>
                  <span className="mono" style={{ fontSize: 11, color: 'var(--color-text-secondary)' }}>{s.value}</span>
                  <span className={`mono-chip ${s.masked ? 'warning' : ''}`}>{s.masked ? 'secret' : 'var'}</span>
                  <button className="btn-icon"><Icon.Edit size={12} /></button>
                </div>
              ))}
              {e.secrets === 0 && <div style={{ padding: '16px 14px', fontSize: 12, color: 'var(--color-text-tertiary)', textAlign: 'center' }}>No secrets defined.</div>}
            </div>
            <button className="btn btn-ghost" style={{ marginTop: 10 }}><Icon.Plus size={12} />Add secret</button>
          </div>
        </div>
      </div>
    </div>
  );
}

// ============== MEMORY ==============
function SettingsMemory() {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      <SectionCard eyebrow="Store" note="local SQLite, scoped hierarchy">
        <FieldRow label="Backend" description="Where durable memory chunks live" tag="BUILTIN">
          <span className="mono" style={{ fontSize: 12, color: 'var(--color-text-primary)' }}>sqlite · ~/.agh/memory.db</span>
        </FieldRow>
        <FieldRow label="Embeddings provider" description="Used for semantic recall">
          <select className="input" defaultValue="voyage-2" style={{ width: 220 }}>
            <option>voyage-2</option><option>openai-text-3-small</option><option>local-bge</option>
          </select>
        </FieldRow>
        <FieldRow label="Max chunk size" description="Tokens per stored memory" tag="DEFAULT 1024">
          <input className="input" type="number" defaultValue={1024} style={{ width: 120 }} />
        </FieldRow>
      </SectionCard>

      <SectionCard eyebrow="Scopes" note="global > workspace > session">
        <div style={{ padding: 8 }}>
          {[
            { label: 'Global', count: 14, size: '2.4 MB', retention: 'forever' },
            { label: 'Workspace', count: 47, size: '8.1 MB', retention: '180 days' },
            { label: 'Session', count: 128, size: '12.2 MB', retention: '14 days' },
          ].map(s => (
            <div key={s.label} className="flat-row" style={{ gridTemplateColumns: '120px 1fr 100px 120px 60px' }}>
              <span className="mono-chip">{s.label.toUpperCase()}</span>
              <span className="mono" style={{ fontSize: 11.5, color: 'var(--color-text-secondary)' }}>{s.count} memories · {s.size}</span>
              <span className="mono" style={{ fontSize: 11, color: 'var(--color-text-tertiary)' }}>ret · {s.retention}</span>
              <button className="btn btn-ghost" style={{ height: 26, padding: '0 10px', fontSize: 11 }}>Configure</button>
              <button className="btn-icon"><Icon.Trash size={12} /></button>
            </div>
          ))}
        </div>
      </SectionCard>

      <SectionCard eyebrow="Compaction" note="GC policy">
        <FieldRow label="Auto-compact" description="Merge stale chunks and rebuild index nightly"><Toggle on={true} onChange={() => {}} /></FieldRow>
        <FieldRow label="Size cap per scope" description="Soft cap; oldest chunks evicted first" tag="MB">
          <input className="input" type="number" defaultValue={64} style={{ width: 120 }} />
        </FieldRow>
        <FieldRow label="TTL for session memories" description="0 disables TTL" tag="DAYS">
          <input className="input" type="number" defaultValue={14} style={{ width: 120 }} />
        </FieldRow>
        <div style={{ padding: 12 }}>
          <Banner tone="info">
            Next compaction scheduled for <span className="mono" style={{ color: 'var(--color-text-primary)' }}>02:00 · 6h 24m</span>.
            <button className="btn btn-ghost" style={{ height: 24, padding: '0 8px', fontSize: 11, marginLeft: 10 }}>Run now</button>
          </Banner>
        </div>
      </SectionCard>
    </div>
  );
}

// ============== SKILLS ==============
function SettingsSkills() {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      <SectionCard eyebrow="Registry" note="where AGH resolves skill lookups">
        <FieldRow label="Primary registry" description="Skills here are pulled by name" tag="URL">
          <input className="input" defaultValue="https://skills.compozy.com" style={{ width: 360, fontFamily: 'var(--font-mono)', fontSize: 12 }} />
        </FieldRow>
        <FieldRow label="Mirrors" description="Fallback sources, checked in order">
          <textarea className="input" rows={3} defaultValue={'https://mirror.agh.sh\nhttps://registry.npmjs.org/@agh-skills'}
            style={{ width: 360, fontFamily: 'var(--font-mono)', fontSize: 12, resize: 'vertical' }} />
        </FieldRow>
        <FieldRow label="Auto-update" description="Check for newer versions on session start"><Toggle on={true} onChange={() => {}} /></FieldRow>
      </SectionCard>

      <SectionCard eyebrow="Safety" note="applied to all installed skills">
        <FieldRow label="Sandboxed execution" description="Run skill scripts in isolated workdir"><Toggle on={true} onChange={() => {}} /></FieldRow>
        <FieldRow label="Allow shell skills" description="Permit skills that exec shell commands"><Toggle on={true} onChange={() => {}} /></FieldRow>
        <FieldRow label="Allow network skills" description="Permit skills that make outbound HTTP"><Toggle on={false} onChange={() => {}} /></FieldRow>
        <FieldRow label="Require signed skills" description="Reject skills without a signed manifest"><Toggle on={false} onChange={() => {}} /></FieldRow>
      </SectionCard>

      <SectionCard eyebrow="Developer" note="write your own">
        <FieldRow label="Local skills path" description="AGH autoloads SKILL.md files under this path" tag="PATH">
          <input className="input" defaultValue="~/.agh/skills" style={{ width: 320, fontFamily: 'var(--font-mono)', fontSize: 12 }} />
        </FieldRow>
        <FieldRow label="Watch for changes" description="Reload when a SKILL.md file changes"><Toggle on={true} onChange={() => {}} /></FieldRow>
      </SectionCard>
    </div>
  );
}

// ============== AUTOMATION ==============
function SettingsAutomation() {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      <SectionCard eyebrow="Runner" note="cron + event triggers">
        <StatGrid items={[
          { label: 'Scheduled jobs', value: '4' },
          { label: 'Event triggers', value: '2' },
          { label: 'Runs 24h', value: '182', tone: 'accent' },
          { label: 'Failures 24h', value: '3' },
        ]} />
        <FieldRow label="Enable automation" description="Master switch. Disabling stops schedules and triggers."><Toggle on={true} onChange={() => {}} /></FieldRow>
        <FieldRow label="Concurrency" description="Max parallel automation runs" tag="DEFAULT 4">
          <input className="input" type="number" defaultValue={4} style={{ width: 100 }} />
        </FieldRow>
        <FieldRow label="Timezone" description="Cron expressions evaluate in this zone">
          <select className="input" defaultValue="America/Sao_Paulo" style={{ width: 220 }}>
            <option>UTC</option><option>America/Sao_Paulo</option><option>America/New_York</option><option>Europe/London</option>
          </select>
        </FieldRow>
      </SectionCard>

      <SectionCard eyebrow="Defaults" note="applied to new automations">
        <FieldRow label="Default scope"><div className="pills"><button className="pill">global</button><button className="pill active">workspace</button></div></FieldRow>
        <FieldRow label="Retry policy" description="What to do when a run errors">
          <select className="input" defaultValue="backoff" style={{ width: 200 }}>
            <option value="none">None</option><option value="backoff">Exponential backoff</option><option value="fixed">Fixed 30s</option>
          </select>
        </FieldRow>
        <FieldRow label="Max retries"><input className="input" type="number" defaultValue={3} style={{ width: 100 }} /></FieldRow>
      </SectionCard>

      <SectionCard eyebrow="Notifications" note="when runs need attention">
        <FieldRow label="Notify on failure" description="Route failures to a bridge">
          <select className="input" defaultValue="slack-prod" style={{ width: 220 }}>
            <option>none</option><option>slack-prod</option><option>ops-email</option>
          </select>
        </FieldRow>
        <FieldRow label="Notify on long-running" description="Emit alert if run exceeds" tag="MINUTES">
          <input className="input" type="number" defaultValue={10} style={{ width: 100 }} />
        </FieldRow>
      </SectionCard>
    </div>
  );
}

// ============== NETWORK ==============
function SettingsNetwork() {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      <SectionCard eyebrow="Wire" note="agh-network/v0 over NATS">
        <FieldRow label="Enabled" description="When off, AGH runs local-only"><Toggle on={true} onChange={() => {}} /></FieldRow>
        <FieldRow label="NATS URL" description="Bus agents communicate over" tag="CONNECTION">
          <input className="input" defaultValue="nats://127.0.0.1:4222" style={{ width: 320, fontFamily: 'var(--font-mono)', fontSize: 12 }} />
        </FieldRow>
        <FieldRow label="Peer ID" description="Identifier other peers see">
          <input className="input" defaultValue="pedronauck@laptop" style={{ width: 280, fontFamily: 'var(--font-mono)', fontSize: 12 }} />
        </FieldRow>
      </SectionCard>

      <SectionCard eyebrow="Discovery">
        <FieldRow label="Advertise on LAN" description="Discoverable via mDNS on local subnet"><Toggle on={true} onChange={() => {}} /></FieldRow>
        <FieldRow label="Accept delegations" description="Allow peers to assign tasks"><Toggle on={true} onChange={() => {}} /></FieldRow>
        <FieldRow label="Trusted peers" description="Only auto-approve from these identities">
          <textarea className="input" rows={3} defaultValue={'pedronauck@*\nci-runner-*'}
            style={{ width: 300, fontFamily: 'var(--font-mono)', fontSize: 12, resize: 'vertical' }} />
        </FieldRow>
      </SectionCard>

      <SectionCard eyebrow="Wire kinds" note="agh-network/v0 message shapes">
        <div style={{ padding: 10 }}>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(140px,1fr))', gap: 8 }}>
            {[
              { k: 'greet', desc: 'join wave' },
              { k: 'whois', desc: 'capability probe' },
              { k: 'say', desc: 'broadcast' },
              { k: 'direct', desc: '1:1 delegation' },
              { k: 'recipe', desc: 'advertise capability' },
              { k: 'receipt', desc: 'ack / result' },
              { k: 'trace', desc: 'span chain' },
            ].map(m => (
              <div key={m.k} style={{ padding: '10px 12px', background: 'var(--color-canvas-deep)', border: '1px solid var(--color-divider)', borderRadius: 8 }}>
                <div className="mono text-accent" style={{ fontSize: 12, fontWeight: 600 }}>{m.k}</div>
                <div className="mono" style={{ fontSize: 10, color: 'var(--color-text-tertiary)', marginTop: 2 }}>{m.desc}</div>
              </div>
            ))}
          </div>
        </div>
      </SectionCard>

      <div className="card" style={{ padding: 16 }}>
        <div className="flex items-center gap-2 mb-2">
          <span className="dot success pulse" />
          <span className="mono text-xs text-primary">connected</span>
          <span className="mono text-xs text-tertiary" style={{ marginLeft: 'auto' }}>uptime 2h 14m</span>
        </div>
        <span className="text-secondary text-md">12 peers · 6 channels · 1.28k messages · p95 rtt 24ms</span>
      </div>
    </div>
  );
}

// ============== OBSERVABILITY ==============
function SettingsObservability() {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      <SectionCard eyebrow="Traces" note="OpenTelemetry">
        <FieldRow label="Exporter">
          <div className="pills">
            <button className="pill">stdout</button>
            <button className="pill active">otlp</button>
            <button className="pill">off</button>
          </div>
        </FieldRow>
        <FieldRow label="OTLP endpoint" description="gRPC or HTTP/protobuf" tag="CONNECTION">
          <input className="input" defaultValue="http://localhost:4318/v1/traces" style={{ width: 360, fontFamily: 'var(--font-mono)', fontSize: 12 }} />
        </FieldRow>
        <FieldRow label="Sample rate" description="0.0–1.0; 1.0 samples every span">
          <input className="input" type="number" step="0.1" defaultValue={1.0} style={{ width: 100 }} />
        </FieldRow>
        <FieldRow label="Propagate W3C tracecontext" description="Inject into bridge outbound calls"><Toggle on={true} onChange={() => {}} /></FieldRow>
      </SectionCard>

      <SectionCard eyebrow="Metrics">
        <FieldRow label="Prometheus endpoint" description="Scrape /metrics on this port" tag="PORT">
          <input className="input" type="number" defaultValue={9090} style={{ width: 100 }} />
        </FieldRow>
        <FieldRow label="Include runtime metrics" description="go runtime + process stats"><Toggle on={true} onChange={() => {}} /></FieldRow>
      </SectionCard>

      <SectionCard eyebrow="Logs">
        <FieldRow label="Log level">
          <select className="input" defaultValue="info" style={{ width: 180 }}>
            <option>debug</option><option>info</option><option>warn</option><option>error</option>
          </select>
        </FieldRow>
        <FieldRow label="Structured format">
          <div className="pills"><button className="pill active">json</button><button className="pill">pretty</button></div>
        </FieldRow>
        <FieldRow label="Destination">
          <input className="input" defaultValue="~/.agh/logs/agh.log" style={{ width: 320, fontFamily: 'var(--font-mono)', fontSize: 12 }} />
        </FieldRow>
        <FieldRow label="Rotate" description="Size-based rotation" tag="MB">
          <input className="input" type="number" defaultValue={50} style={{ width: 100 }} />
        </FieldRow>
      </SectionCard>

      <div className="card" style={{ padding: 16 }}>
        <span className="eyebrow">Live trace preview</span>
        <div className="codeblock" style={{ marginTop: 10 }}>
          <div className="codeblock-body" style={{ padding: 12, fontSize: 11 }}>
            <div className="text-tertiary">[12:07:02] INFO  session.stream start sess-8471f</div>
            <div className="text-secondary">[12:07:02] DEBUG tool.call shell.safe-run args=["rg","onToolCall"] span=a8f91</div>
            <div className="text-secondary">[12:07:02] DEBUG tool.result ok 212ms span=a8f91</div>
            <div className="text-tertiary">[12:07:04] INFO  bridge.delivered slack ch=#ops evt=session.done</div>
          </div>
        </div>
      </div>
    </div>
  );
}

// ============== HOOKS ==============
function SettingsHooks() {
  const [hooks, setHooks] = React.useState([
    { event: 'session.before_tool', script: '~/.agh/hooks/guard.sh', enabled: true, last: 'ok · 4ms' },
    { event: 'session.after_approve', script: '~/.agh/hooks/audit.ts', enabled: true, last: 'ok · 18ms' },
    { event: 'bridge.before_deliver', script: '~/.agh/hooks/redact.py', enabled: false, last: '—' },
    { event: 'automation.on_fail', script: '~/.agh/hooks/pageron.sh', enabled: true, last: 'ok · 112ms' },
  ]);
  const [editor, setEditor] = React.useState(false);
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      <div className="flex items-center gap-3 mb-2">
        <span className="mono text-xs text-secondary">
          <b className="text-primary">{hooks.length}</b> hooks configured · {hooks.filter(h => h.enabled).length} enabled
        </span>
        <button className="btn btn-primary" style={{ marginLeft: 'auto' }} onClick={() => setEditor(true)}><Icon.Plus size={12} />Hook</button>
      </div>

      <SectionCard eyebrow="Lifecycle hooks" note="executed in-process; must exit 0 within timeout">
        <div style={{ padding: 6 }}>
          {hooks.map((h, i, arr) => (
            <div key={i} className="flat-row" style={{ gridTemplateColumns: '22px 200px 1fr 110px 80px 40px', borderBottom: i < arr.length-1 ? '1px solid var(--color-divider)' : 'none' }}>
              <Toggle on={h.enabled} onChange={v => setHooks(hs => hs.map((x,j) => j === i ? { ...x, enabled: v } : x))} />
              <span className="mono-chip">{h.event}</span>
              <span className="mono" style={{ fontSize: 11.5, color: 'var(--color-text-secondary)' }}>{h.script}</span>
              <span className="mono" style={{ fontSize: 10.5, color: h.last.includes('ok') ? 'var(--color-success)' : 'var(--color-text-tertiary)' }}>{h.last}</span>
              <button className="btn btn-ghost" style={{ height: 24, padding: '0 8px', fontSize: 11 }}>Edit</button>
              <button className="btn-icon"><Icon.Trash size={12} /></button>
            </div>
          ))}
        </div>
      </SectionCard>

      <SectionCard eyebrow="Extensions" note="daemon plugins">
        <div style={{ padding: 14 }}>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(220px, 1fr))', gap: 10 }}>
            {[
              { name: 'agh-otel', v: '0.3.1', desc: 'OpenTelemetry exporter' },
              { name: 'agh-nats-bridge', v: '1.1.0', desc: 'Proxy agh-network over external NATS' },
              { name: 'agh-slack-rich', v: '0.5.0', desc: 'Rich Slack message renderer' },
            ].map(x => (
              <div key={x.name} style={{ padding: 12, background: 'var(--color-canvas-deep)', border: '1px solid var(--color-divider)', borderRadius: 8 }}>
                <div className="flex items-center gap-2">
                  <Icon.Puzzle size={13} style={{ color: 'var(--color-accent)' }} />
                  <span className="mono text-md text-primary">{x.name}</span>
                  <span className="mono text-xs text-tertiary" style={{ marginLeft: 'auto' }}>v{x.v}</span>
                </div>
                <p style={{ margin: '6px 0 10px', fontSize: 12, color: 'var(--color-text-secondary)' }}>{x.desc}</p>
                <button className="btn btn-ghost" style={{ height: 24, padding: '0 10px', fontSize: 11 }}>Configure</button>
              </div>
            ))}
          </div>
        </div>
      </SectionCard>

      <Modal open={editor} onClose={() => setEditor(false)} title="Add hook"
        description="Hooks fire on lifecycle events. The daemon invokes your script with JSON on stdin."
        footer={<>
          <button className="btn btn-ghost" onClick={() => setEditor(false)}>Cancel</button>
          <button className="btn btn-primary" onClick={() => setEditor(false)}>Add hook</button>
        </>}>
        <FieldRow label="Event" description="Select a lifecycle event" tag="REQUIRED">
          <select className="input" style={{ width: 280, fontFamily: 'var(--font-mono)' }}>
            <option>session.before_tool</option><option>session.after_tool</option>
            <option>session.before_approve</option><option>session.after_approve</option>
            <option>bridge.before_deliver</option><option>automation.on_fail</option>
          </select>
        </FieldRow>
        <FieldRow label="Script path" description="Absolute path to an executable" tag="PATH">
          <input className="input" placeholder="~/.agh/hooks/my-hook.sh" style={{ width: 360, fontFamily: 'var(--font-mono)', fontSize: 12 }} />
        </FieldRow>
        <FieldRow label="Timeout" description="Kill hook after N ms" tag="MS">
          <input className="input" type="number" defaultValue={2000} style={{ width: 120 }} />
        </FieldRow>
        <FieldRow label="Block on fail" description="If hook exits non-zero, abort the event"><Toggle on={true} onChange={() => {}} /></FieldRow>
      </Modal>
    </div>
  );
}

Object.assign(window, { SettingsPage });
