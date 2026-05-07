/* V3 — linear-editorial
 * Heroic editorial masthead, mono uppercase eyebrows + section folios,
 * sidebar without icons (active = accent tick on the gutter), tabular row
 * grid, EditorialQuote pull-quote for failed-row errors, PropertyGrid
 * (hairline cells) replacing the KV table, colophon empty state. */

const { useState, Fragment } = React;
const I = window.AGH_ICONS;
const D = window.AGH_DATA;

/* ── Avatar ──────────────────────────────────────────────────────────── */

function Avatar({ owner, size = 24 }) {
  if (!owner) return null;
  const isAgent = owner.kind === "agent";
  const isSystem = owner.kind === "system";
  const colors = D.avatarColors(owner);
  const initials = D.avatarInitials(owner.label);
  const cls = isSystem
    ? "le-avatar le-avatar--system"
    : isAgent
    ? "le-avatar le-avatar--agent"
    : "le-avatar";
  return (
    <span
      className={cls}
      style={{
        width: size,
        height: size,
        background: isSystem ? undefined : colors.bg,
        color: isSystem ? undefined : colors.fg,
        fontSize: Math.max(9, Math.round(size * 0.42)),
      }}
      title={`${owner.kind} · ${owner.label}`}
      aria-label={`${owner.kind} ${owner.label}`}
    >
      {isSystem ? <I.Workspace size={Math.round(size * 0.6)} stroke={1.5} /> : initials}
    </span>
  );
}

/* ── Sidebar ─────────────────────────────────────────────────────────── */

function NavRow({ item, active, onClick, disabled }) {
  return (
    <button
      type="button"
      className={`le-nav__row${active ? " le-nav__row--active" : ""}`}
      onClick={disabled ? undefined : onClick}
      aria-disabled={disabled || undefined}
      aria-current={active ? "page" : undefined}
    >
      <span style={{ display: "inline-flex", alignItems: "baseline", gap: 4 }}>
        <span>{item.label}</span>
        {item.kbd ? <span className="le-nav__kbd">{item.kbd}</span> : null}
      </span>
      {item.count !== null && item.count !== undefined ? (
        <span className="le-nav__count">{item.count}</span>
      ) : null}
    </button>
  );
}

function Sidebar({ view, setView }) {
  return (
    <aside className="le-side" aria-label="Primary navigation">
      <div className="le-side__head">
        <span className="le-side__brand">agh</span>
        <span className="le-side__alpha">Alpha</span>
      </div>
      <div className="le-side__group">
        <div className="le-side__label">Workspace</div>
        {D.NAV_PRIMARY.map((item) => (
          <NavRow
            key={item.id}
            item={item}
            active={item.id === view}
            disabled={item.id !== "tasks" && item.id !== "jobs"}
            onClick={() => setView(item.id)}
          />
        ))}
      </div>
      <div className="le-side__group">
        <div className="le-side__label">Operations</div>
        {D.NAV_SECONDARY.map((item) => (
          <NavRow key={item.id} item={item} active={false} disabled onClick={() => {}} />
        ))}
      </div>
      <div className="le-side__group">
        {D.NAV_FOOTER.map((item) => (
          <NavRow key={item.id} item={item} active={false} disabled onClick={() => {}} />
        ))}
      </div>
      <div className="le-side__footer">
        <span className="le-side__footer-status">
          <span className="le-status-dot" />
          <span>Connected</span>
        </span>
        <span>v0.41.0</span>
      </div>
    </aside>
  );
}

/* ── Filter bar ──────────────────────────────────────────────────────── */

function FilterBar({ filters, value, onChange, search, onSearch, searchPlaceholder }) {
  return (
    <div className="le-filterbar">
      <div className="le-filterbar__filters">
        {filters.map((f, idx) => (
          <Fragment key={f.value}>
            {idx > 0 ? <span className="le-filterbar__sep">·</span> : null}
            <button
              type="button"
              onClick={() => onChange(f.value)}
              className={`le-filterbar__filter${
                value === f.value ? " le-filterbar__filter--active" : ""
              }`}
            >
              {f.label}
              {f.badge != null ? (
                <span className="le-filterbar__filter-count">{f.badge}</span>
              ) : null}
            </button>
          </Fragment>
        ))}
      </div>
      <div className="le-filterbar__search">
        <I.Search size={13} />
        <input
          placeholder={searchPlaceholder}
          value={search}
          onChange={(e) => onSearch(e.target.value)}
        />
        <span className="le-kbd">⌘K</span>
      </div>
    </div>
  );
}

/* ── Tasks ───────────────────────────────────────────────────────────── */

function PriorityGlyph({ level }) {
  if (!level) return null;
  return (
    <span className={`le-row__priority le-row__priority--${level}`}>
      <I.Priority level={level} size={11} />
      <span style={{ textTransform: "lowercase" }}>{D.PRIORITY_LABEL[level].toLowerCase()}</span>
    </span>
  );
}

function TaskRow({ task, selected, onSelect }) {
  const failedError = task.status === "failed" && task.activeRun?.error;
  const ownerLabel = task.owner.label;

  return (
    <div
      className={`le-row${selected ? " le-row--selected" : ""}`}
      role="button"
      tabIndex={0}
      onClick={() => onSelect(task.id)}
    >
      <span className="le-row__statusicon">
        <I.Status status={task.status} size={14} />
      </span>

      <div className="le-row__main">
        <div className="le-row__title">
          <Avatar owner={task.owner} size={20} />
          <span className="le-row__name">{task.title}</span>
          <span className="le-row__id">{task.id}</span>
        </div>
        <div className="le-row__sub">
          <span>{ownerLabel}</span>
          {task.priority ? <PriorityGlyph level={task.priority} /> : null}
          {task.activeRun ? (
            <span>
              attempt {task.activeRun.attempt}/{task.activeRun.maxAttempts}
            </span>
          ) : null}
          {task.childCount > 0 ? (
            <span>
              {task.childCount} {task.childCount === 1 ? "child" : "children"}
            </span>
          ) : null}
          {task.dependencyCount > 0 ? (
            <span>
              {task.dependencyCount} {task.dependencyCount === 1 ? "dep" : "deps"}
            </span>
          ) : null}
          {task.parentTaskId ? (
            <span>
              parent {task.parentTaskId.slice(-6)}
            </span>
          ) : null}
          {task.approvalState === "pending" ? (
            <span className="le-row__sub--accent">approval pending</span>
          ) : null}
        </div>

        {failedError ? (
          <figure className="le-quote le-quote--danger" style={{ margin: "6px 0 0" }}>
            <blockquote className="le-quote__body">{task.activeRun.error}</blockquote>
            <figcaption className="le-quote__source">
              attempt {task.activeRun.attempt}/{task.activeRun.maxAttempts} · {task.id}
            </figcaption>
          </figure>
        ) : null}

        {task.isBlocked ? (
          <figure className="le-quote le-quote--warning" style={{ margin: "6px 0 0" }}>
            <blockquote className="le-quote__body">{task.blockedReason}</blockquote>
            <figcaption className="le-quote__source">blocked</figcaption>
          </figure>
        ) : null}

        {task.isDraft || task.status === "failed" ? (
          <div className="le-row__sublane">
            {task.isDraft ? (
              <button
                type="button"
                className="le-row__action"
                onClick={(e) => e.stopPropagation()}
              >
                <I.Plus size={11} />
                Publish
              </button>
            ) : null}
            {task.status === "failed" ? (
              <button
                type="button"
                className="le-row__action"
                onClick={(e) => e.stopPropagation()}
              >
                <I.Refresh size={11} />
                Retry
              </button>
            ) : null}
          </div>
        ) : null}
      </div>

      <span className="le-status">
        <span className="le-status__icon">
          <I.Status status={task.status} size={12} />
        </span>
        {D.STATUS_LABEL[task.status]}
      </span>

      <span className="le-row__time" title={task.timestampIso}>
        {task.timestamp}
      </span>
    </div>
  );
}

function TasksView({ selectedId, setSelectedId }) {
  const [lane, setLane] = useState("all");
  const [search, setSearch] = useState("");

  const lanes = [
    { value: "all", label: "All", badge: D.TASK_SUMMARY.open },
    {
      value: "mine",
      label: "Mine",
      badge: D.TASKS.filter(
        (t) => t.owner.kind === "human" && !["completed", "canceled"].includes(t.status)
      ).length,
    },
    { value: "watched", label: "Watched", badge: 2 },
  ];

  const tasks = D.TASKS.filter((t) =>
    lane === "mine" ? t.owner.kind === "human" : lane === "watched" ? false : true
  ).filter((t) => (search ? t.title.toLowerCase().includes(search.toLowerCase()) : true));

  const summary = `${D.TASK_SUMMARY.open} open across ${D.TASK_SUMMARY.workspaces} workspaces, ${D.TASK_SUMMARY.blocked} blocked, ${D.TASK_SUMMARY.failed} failed in the last hour.`;

  return (
    <section className="le-list">
      <header className="le-page-head">
        <p className="le-page-head__eyebrow">
          <span>Workspace · agh-runtime</span>
          <span>Updated 2m ago</span>
        </p>
        <div className="le-page-head__title">
          <h1>Tasks</h1>
          <span className="le-page-head__count">{D.TASK_SUMMARY.total}</span>
        </div>
        <p className="le-page-head__sub">{summary}</p>
        <div className="le-page-head__primary">
          <button type="button" className="le-cta">
            <I.Plus size={13} />
            New task
            <span className="le-cta__kbd">N</span>
          </button>
          <button type="button" className="le-secondary">
            <I.Filter size={13} />
            Filters
          </button>
        </div>
      </header>

      <FilterBar
        filters={lanes}
        value={lane}
        onChange={setLane}
        search={search}
        onSearch={setSearch}
        searchPlaceholder="Filter tasks…"
      />

      <div className="le-list__rows">
        {tasks.map((task) => (
          <TaskRow
            key={task.id}
            task={task}
            selected={task.id === selectedId}
            onSelect={setSelectedId}
          />
        ))}
      </div>
    </section>
  );
}

/* ── Jobs ────────────────────────────────────────────────────────────── */

function JobRow({ job, selected, onSelect }) {
  return (
    <div
      className={`le-row${selected ? " le-row--selected" : ""}`}
      role="button"
      tabIndex={0}
      onClick={() => onSelect(job.id)}
    >
      <span
        className="le-row__statusicon"
        style={{ color: job.enabled ? "var(--color-success)" : "var(--color-text-tertiary)" }}
      >
        <I.Status status={job.enabled ? "ready" : "canceled"} size={14} />
      </span>
      <div className="le-row__main">
        <div className="le-row__title">
          <span className="le-row__name">{job.name}</span>
          <span className="le-row__id">{job.id}</span>
        </div>
        <div className="le-row__sub">
          <span>{job.schedule}</span>
          <span>{job.source}</span>
          <span>{job.scope}</span>
          <span>avg {job.avgDuration}</span>
        </div>
        {job.lastFailure ? (
          <figure className="le-quote le-quote--danger" style={{ margin: "6px 0 0" }}>
            <blockquote className="le-quote__body">{job.lastFailure}</blockquote>
            <figcaption className="le-quote__source">last failure</figcaption>
          </figure>
        ) : null}
      </div>
      <span
        className="le-status"
        style={{ color: job.enabled ? "var(--color-success)" : "var(--color-text-tertiary)" }}
      >
        <span className="le-status__icon">
          <I.Status status={job.enabled ? "ready" : "canceled"} size={12} />
        </span>
        {job.enabled ? "Enabled" : "Disabled"}
      </span>
      <span className="le-row__time">next · {job.nextRun}</span>
    </div>
  );
}

function JobsView({ selectedId, setSelectedId }) {
  const [scope, setScope] = useState("all");
  const [search, setSearch] = useState("");
  const scopes = [
    { value: "all", label: "All", badge: D.JOBS.length },
    { value: "global", label: "Global", badge: D.JOBS.filter((j) => j.scope === "global").length },
    {
      value: "workspace",
      label: "Workspace",
      badge: D.JOBS.filter((j) => j.scope === "workspace").length,
    },
  ];
  const jobs = D.JOBS.filter((j) => (scope === "all" ? true : j.scope === scope)).filter((j) =>
    search ? j.name.toLowerCase().includes(search.toLowerCase()) : true
  );

  const summary = `${D.JOB_SUMMARY.active} active, ${D.JOB_SUMMARY.paused} paused. Next dispatch in 8 minutes.`;

  return (
    <section className="le-list">
      <header className="le-page-head">
        <p className="le-page-head__eyebrow">
          <span>Workspace · agh-runtime</span>
          <span>Cron resolver online</span>
        </p>
        <div className="le-page-head__title">
          <h1>Jobs</h1>
          <span className="le-page-head__count">{D.JOB_SUMMARY.total}</span>
        </div>
        <p className="le-page-head__sub">{summary}</p>
        <div className="le-page-head__primary">
          <button type="button" className="le-cta">
            <I.Plus size={13} />
            New job
            <span className="le-cta__kbd">N</span>
          </button>
          <button type="button" className="le-secondary">
            <I.Activity size={13} />
            History
          </button>
        </div>
      </header>

      <FilterBar
        filters={scopes}
        value={scope}
        onChange={setScope}
        search={search}
        onSearch={setSearch}
        searchPlaceholder="Search jobs…"
      />

      <div className="le-list__rows">
        {jobs.map((job) => (
          <JobRow
            key={job.id}
            job={job}
            selected={job.id === selectedId}
            onSelect={setSelectedId}
          />
        ))}
      </div>
    </section>
  );
}

/* ── Detail ──────────────────────────────────────────────────────────── */

function PropertyValue({ children, mono = false, tone = "neutral" }) {
  const cls = [
    "le-prop__value",
    mono ? "le-prop__value--mono" : null,
    tone !== "neutral" ? `le-prop__value--${tone}` : null,
  ]
    .filter(Boolean)
    .join(" ");
  return <span className={cls}>{children}</span>;
}

function Property({ label, children }) {
  return (
    <div className="le-prop">
      <span className="le-prop__label">{label}</span>
      {children}
    </div>
  );
}

function TaskDetail({ task }) {
  if (!task) return <DetailEmpty />;
  const subissues = D.getSubissuesFor(task);
  const completedKids = subissues.filter((s) => s.status === "completed").length;
  const activity = D.getActivityFor(task);

  return (
    <article className="le-detail">
      <div className="le-detail__head">
        <div className="le-detail__crumb">
          <span>agh-runtime</span>
          <I.ChevronRight size={11} />
          <span>tasks</span>
          <I.ChevronRight size={11} />
          <span>{task.id}</span>
          <span className="le-detail__crumb-actions">
            <button type="button" className="le-iconbtn" title="Copy ID">
              <I.Copy size={12} />
            </button>
            <button type="button" className="le-iconbtn" title="Open in new pane">
              <I.ArrowUpRight size={12} />
            </button>
          </span>
        </div>
        <h2 className="le-detail__title">{task.title}</h2>
        <div className="le-detail__assignees">
          <span className="le-assignee-stack">
            <Avatar owner={task.owner} size={28} />
            {subissues.slice(0, 2).map((s) => (
              <Avatar key={s.id} owner={s.assignee} size={28} />
            ))}
          </span>
          <span className="le-detail__owner-name">{task.owner.label}</span>
          <span style={{ fontFamily: "var(--font-mono)", fontSize: 11, color: "var(--color-text-tertiary)", letterSpacing: "0.06em", textTransform: "uppercase" }}>
            owns
          </span>
        </div>
        <div className="le-propgrid">
          <Property label="Status">
            <PropertyValue tone={D.STATUS_TONE[task.status] === "danger" ? "danger" : D.STATUS_TONE[task.status] === "warning" ? "warning" : "neutral"}>
              <I.Status status={task.status} size={12} />
              {D.STATUS_LABEL[task.status]}
            </PropertyValue>
          </Property>
          <Property label="Priority">
            <PropertyValue>
              {task.priority ? (
                <>
                  <I.Priority level={task.priority} size={11} />
                  {D.PRIORITY_LABEL[task.priority]}
                </>
              ) : (
                "—"
              )}
            </PropertyValue>
          </Property>
          <Property label="Owner">
            <PropertyValue>
              <Avatar owner={task.owner} size={16} />
              {task.owner.label}
            </PropertyValue>
          </Property>
          {task.activeRun ? (
            <Property label="Attempts">
              <PropertyValue mono>
                {task.activeRun.attempt} / {task.activeRun.maxAttempts}
              </PropertyValue>
            </Property>
          ) : null}
          {task.parentTaskId ? (
            <Property label="Parent">
              <PropertyValue mono>{task.parentTaskId}</PropertyValue>
            </Property>
          ) : null}
          {task.childCount > 0 ? (
            <Property label="Children">
              <PropertyValue mono>{task.childCount}</PropertyValue>
            </Property>
          ) : null}
          {task.approvalState ? (
            <Property label="Approval">
              <PropertyValue tone="accent">{task.approvalState}</PropertyValue>
            </Property>
          ) : null}
          <Property label="Created">
            <PropertyValue mono>{task.timestamp}</PropertyValue>
          </Property>
        </div>
      </div>

      <div className="le-detail__body le-detail__body--gap-lg">
        {subissues.length > 0 ? (
          <section className="le-section">
            <div className="le-section__head">
              <span className="le-section__label">Sub-issues</span>
              <span className="le-section__count">
                {completedKids} / {subissues.length} done
              </span>
            </div>
            <div className="le-activity">
              {subissues.map((sub) => (
                <div key={sub.id} className="le-activity__item">
                  <span className="le-activity__avatar">
                    <I.Status status={sub.status} size={14} />
                  </span>
                  <div>
                    <div className="le-activity__msg">
                      <strong>{sub.title}</strong>
                    </div>
                    <div className="le-activity__detail">
                      {sub.id} · {sub.assignee.label}
                    </div>
                  </div>
                  <span className="le-activity__time">
                    <Avatar owner={sub.assignee} size={20} />
                  </span>
                </div>
              ))}
            </div>
          </section>
        ) : null}

        <section className="le-section">
          <div className="le-section__head">
            <span className="le-section__label">Activity</span>
            <span className="le-section__count">{activity.length}</span>
          </div>
          <div className="le-activity">
            {activity.map((event, idx) => (
              <div key={idx} className="le-activity__item">
                <span className="le-activity__avatar">
                  <Avatar owner={event.author} size={28} />
                </span>
                <div>
                  <div className="le-activity__msg">
                    <strong>{event.author.label}</strong>{" "}
                    {event.msg.charAt(0).toLowerCase() + event.msg.slice(1)}
                  </div>
                  {event.detail ? <div className="le-activity__detail">{event.detail}</div> : null}
                </div>
                <span className="le-activity__time">{event.time}</span>
              </div>
            ))}
          </div>
        </section>

        <section className="le-section">
          <div className="le-section__head">
            <span className="le-section__label">Actions</span>
          </div>
          <div style={{ display: "flex", gap: 10, flexWrap: "wrap" }}>
            {task.status === "failed" ? (
              <button className="le-cta">
                <I.Refresh size={13} />
                Retry run
              </button>
            ) : null}
            {task.isDraft ? (
              <button className="le-cta">
                <I.Play size={13} />
                Publish task
              </button>
            ) : null}
            <button className="le-secondary">Open timeline</button>
            <button className="le-link">
              View raw payload
              <I.ArrowUpRight size={12} />
            </button>
          </div>
        </section>
      </div>
    </article>
  );
}

function JobDetail({ job }) {
  if (!job) return <DetailEmpty />;
  return (
    <article className="le-detail">
      <div className="le-detail__head">
        <div className="le-detail__crumb">
          <span>agh-runtime</span>
          <I.ChevronRight size={11} />
          <span>jobs</span>
          <I.ChevronRight size={11} />
          <span>{job.id}</span>
          <span className="le-detail__crumb-actions">
            <button type="button" className="le-iconbtn" title="Copy ID">
              <I.Copy size={12} />
            </button>
          </span>
        </div>
        <h2 className="le-detail__title">{job.name}</h2>
        <div className="le-propgrid">
          <Property label="Status">
            <PropertyValue tone={job.enabled ? "neutral" : "neutral"}>
              <I.Status status={job.enabled ? "ready" : "canceled"} size={12} />
              {job.enabled ? "Enabled" : "Disabled"}
            </PropertyValue>
          </Property>
          <Property label="Cadence">
            <PropertyValue>{job.schedule}</PropertyValue>
          </Property>
          <Property label="Next">
            <PropertyValue mono>{job.nextRun}</PropertyValue>
          </Property>
          <Property label="Last">
            <PropertyValue mono>{job.lastRun}</PropertyValue>
          </Property>
          <Property label="Avg duration">
            <PropertyValue mono>{job.avgDuration}</PropertyValue>
          </Property>
          <Property label="Source">
            <PropertyValue>{job.source}</PropertyValue>
          </Property>
          <Property label="Scope">
            <PropertyValue>{job.scope}</PropertyValue>
          </Property>
          {job.lastFailure ? (
            <Property label="Last failure">
              <PropertyValue tone="danger">{job.lastFailure}</PropertyValue>
            </Property>
          ) : null}
        </div>
      </div>

      <div className="le-detail__body le-detail__body--gap-lg">
        <section className="le-section">
          <div className="le-section__head">
            <span className="le-section__label">Recent runs</span>
            <span className="le-section__count">3</span>
          </div>
          <div className="le-activity">
            {[1, 2, 3].map((n) => (
              <div key={n} className="le-activity__item">
                <span className="le-activity__avatar">
                  <Avatar owner={{ kind: "system", label: "agh" }} size={28} />
                </span>
                <div>
                  <div className="le-activity__msg">
                    <strong>{job.id}</strong> ran successfully
                  </div>
                  <div className="le-activity__detail">
                    duration {job.avgDuration} · 0 retries · workspace agh-runtime
                  </div>
                </div>
                <span className="le-activity__time">
                  {n === 1 ? job.lastRun : `${n - 1}h before`}
                </span>
              </div>
            ))}
          </div>
        </section>

        <section className="le-section">
          <div className="le-section__head">
            <span className="le-section__label">Actions</span>
          </div>
          <div style={{ display: "flex", gap: 10, flexWrap: "wrap" }}>
            <button className="le-cta">
              <I.Play size={13} />
              {job.enabled ? "Pause" : "Enable"}
            </button>
            <button className="le-secondary">Run now</button>
            <button className="le-secondary">Edit schedule</button>
          </div>
        </section>
      </div>
    </article>
  );
}

/* ── Editorial colophon — empty state ────────────────────────────────── */

function DetailEmpty() {
  return (
    <article className="le-detail">
      <div className="le-colophon">
        <p className="le-colophon__eyebrow">The runtime</p>
        <p className="le-colophon__sentence">
          Every contract is a task. <em>Every task is replayable.</em> Every event accounted for, every claim recorded, every retry deliberate.
        </p>
        <p className="le-colophon__footer">agh-runtime · v0.41.0 · uptime 3h 22m</p>
      </div>
    </article>
  );
}

/* ── Variant pin ─────────────────────────────────────────────────────── */

function VariantPin() {
  return (
    <div className="le-variant-pin" aria-label="Design variant">
      <span className="le-variant-pin__label">V3 Editorial</span>
      <a href="../v1-linear-calm/">Calm</a>
      <a href="../v2-linear-panel/">Panel</a>
      <button type="button" className="is-active" aria-current="page">
        Editorial
      </button>
    </div>
  );
}

function App() {
  const [view, setView] = useState("tasks");
  const [taskId, setTaskId] = useState(D.TASKS[2].id); // default = failed task to showcase pull-quote
  const [jobId, setJobId] = useState(D.JOBS[2].id); // default = paused job to showcase quote

  const selectedTask = D.TASKS.find((t) => t.id === taskId);
  const selectedJob = D.JOBS.find((j) => j.id === jobId);

  return (
    <Fragment>
      <VariantPin />
      <div className="le-shell">
        <Sidebar view={view} setView={setView} />
        {view === "tasks" ? (
          <TasksView selectedId={taskId} setSelectedId={setTaskId} />
        ) : (
          <JobsView selectedId={jobId} setSelectedId={setJobId} />
        )}
        {view === "tasks" ? <TaskDetail task={selectedTask} /> : <JobDetail job={selectedJob} />}
      </div>
    </Fragment>
  );
}

ReactDOM.createRoot(document.getElementById("root")).render(<App />);
