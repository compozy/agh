# Desktop State

Desktop state is daemon-authoritative, workspace-scoped client state for the AGH OS shell. It is not durable memory and has no native `agh__*` tool in v1. Use the structured CLI or the matching HTTP/UDS API.

## CLI

```bash
agh desktop-state list --workspace <id> -o json
agh desktop-state get --workspace <id> --key desktop -o json
agh desktop-state set --workspace <id> --key desktop --value '{"v":1}' -o json
agh desktop-state set --workspace <id> --key 'win:app:tasks' --file window.json --if-rev 3 -o json
agh desktop-state delete --workspace <id> --key 'win:app:tasks' --if-rev 4
agh desktop-state watch --workspace <id> -o jsonl
```

`list -o json` returns the sorted entry array. `get` and `set` return the canonical entry. `watch -o jsonl` emits one committed `event` frame per line after consuming the snapshot fence over the UDS WebSocket upgrade. Supply `if_rev` for compare-and-swap; omit it for last-writer-wins.

## API And Wire Contract

HTTP and UDS expose identical workspace routes:

- `GET /api/workspaces/{workspace_id}/desktop-state`
- `GET|PUT|DELETE /api/workspaces/{workspace_id}/desktop-state/{key}`
- `POST /api/workspaces/{workspace_id}/desktop-state/apply`
- `GET /api/workspaces/{workspace_id}/desktop-state/stream` (WebSocket)

The public surface has no domain parameter. The daemon fixes the internal domain to `os_shell`. Canonical entries are `{key,value,rev,seq,deleted,updated_at}`; `value` is a JSON object for live entries and `null` only for delete events. Lists and snapshots exclude tombstones. `seq` is the workspace total order and `rev` is the per-key revision.

Send `sub`, `apply`, or `ping` WebSocket frames. The server returns one `snapshot` after `sub`, then ordered `event` frames, correlated `ack` or `error` frames for mutations, and `pong`. An applying connection receives its `ack`; its own mutation is omitted from that connection's event stream by origin id.

Use these `os_shell` key conventions:

- `desktop` for desktop-wide preferences such as focus, rail, wallpaper, dock magnification, and reduced motion.
- `win:<windowId>` for one window's app identity, location, geometry, z-order, minimized/maximized state, and `snap`.
- Include `v` in every value object for client-owned schema evolution.
- Use one atomic `apply` batch when an action changes multiple keys.

### Window snap fractions

`win:*` docs carry `snap: {fx, fy, fw, fh} | null` — normalized fractions (0..1) of the desktop work area. Agents arrange windows by writing fractions; each client renders `work area × fractions` locally and re-derives on viewport resize without writing, so a half stays a half on every screen. Any fraction rect is valid, not just halves/quarters: `{fx:0,fy:0,fw:0.5,fh:1}` is the left half, `{fx:0.5,fy:0,fw:0.5,fh:1}` the right half, `{fx:0.5,fy:0.5,fw:0.5,fh:0.5}` the bottom-right quarter.

Rules:

- Ranges: each origin in 0..1, each span ≥ 0.1 per axis, `fx+fw ≤ 1`, `fy+fh ≤ 1`. Invalid `snap` (out-of-range, sub-minimum, overflow) is salvaged to `null` by clients — the window survives, unsnapped.
- `snap` and `maximized` are mutually exclusive; a doc claiming both keeps `maximized`.
- `snap` travels only inside the whole doc: writing a doc without it (or with `null`) unsnaps the window. `rect` holds the writing client's derived px at commit time (thumbnails/readers); `prevRect` holds the pre-snap rect for restore.
- Clients clamp derived rects to the 280×180 window minimum, so tiny fractions on small viewports render larger than the literal fraction.
- Client rendering insets every derived edge NOT on the work-area boundary by half an 8px gutter, so fraction-adjacent windows (e.g. `fx+fw` of one equals `fx` of the next) render with a visible gap and grow a draggable linked seam. A user resize of a snapped window rewrites its fractions in place (it stays snapped); only dragging the window body away unsnaps it. Payload semantics are unchanged — agents still just write fractions.

```bash
agh desktop-state set --workspace <id> --key 'win:app:tasks' \
  --value '{"v":1,"app":"tasks","instanceKey":null,"location":{"pathname":"/tasks","search":{}},"rect":{"x":10,"y":8,"w":640,"h":480},"prevRect":null,"z":1,"minimized":false,"maximized":false,"snap":{"fx":0,"fy":0,"fw":0.5,"fh":1}}' -o json
```

Deterministic failure codes are `desktop_state_not_found`, `workspace_not_found`, `desktop_state_rev_conflict`, `desktop_state_value_too_large`, `desktop_state_key_quota_exceeded`, `desktop_state_invalid_key`, `desktop_state_invalid_value`, and `desktop_state_slow_consumer`. On slow-consumer eviction, reconnect and replace the local mirror from a fresh snapshot before resuming writes.
