# Hermes vs AGH — CLI UX, Setup, Config & Packaging

## Executive Summary

- **AGH ships a binary; Hermes ships a product.** Hermes has ~30k LOC of setup/doctor/auth/profiles/plugins/backup/uninstall/completion/banner/tips + cross-platform install scripts + a Homebrew formula. AGH has one Bubble Tea provider/model wizard (`internal/cli/install.go:60`), no `completion`, no `doctor`, no `config set/edit`, no `auth`, no `profile`, no `backup`, no `uninstall`, no install scripts, and no `brews:` / `nfpms:` / `scoops:` stanzas in `.goreleaser.yml`.
- **Config UX is the single biggest gap.** AGH has layered loading (`config.go:284-341`) and `AGH_HOME` override (`home.go:62-75`) but **no user-facing `agh config` commands**. Hermes' `config set KEY VALUE` auto-routes secrets to `.env` vs. settings to `config.yaml`, supports version-check + migrate, and honors a "managed mode" flag (`HERMES_MANAGED=homebrew|nixos`) that blocks mutations on package-manager installs (`hermes_cli/config.py:81-147`).
- **No `doctor`.** Hermes' 1240-line `doctor.py` is its most load-bearing DX asset: it validates Python env, packages, `.env`, config (against provider registry + config version + stale keys — `doctor.py:280-447`), OAuth status, dir layout, SQLite WAL size with auto-checkpoint (`doctor.py:589-612`), CLI symlink integrity (`doctor.py:619-694`), systemd linger. `--fix` auto-remediates where safe.
- **Install/uninstall/packaging absent.** Hermes has `install.sh` (1436 LOC), `install.ps1` (921 LOC), `install.cmd`, dev `setup-hermes.sh`, a full `uninstall.py` that reverses PATH edits + systemd units + launchd plists, and a Homebrew formula. AGH has only binary archives + cosign + SBOM via goreleaser, plus a dev-only `scripts/postinstall.sh` that symlinks skills.
- **Auth, profiles, plugins, backup, completion, banner/tips — all missing.** Each is a discrete Hermes module AGH lacks. Shell completion alone is ~30 lines using Cobra's built-in `GenBashCompletionV2`.

## Capability-by-Capability Gap Analysis

### 1. First-run setup wizard
- **Hermes:** `setup.py` (3224 LOC) — modular sections (Model, Terminal, Agent, Messaging, Tools), curses `prompt_choice` / `prompt_checklist` (`setup.py:212-312`), non-interactive fallback with copy-pasteable `hermes config set ...` commands (`setup.py:172-188`), post-wizard tool-availability matrix with per-tool missing-var hints (`setup.py:343-569`), per-section re-run (`hermes setup model`).
- **AGH:** `install.go` Bubble Tea provider→model→confirm wizard, seeds `~/.agh/agents/general/AGENT.md`. Single-dimension, not re-runnable per section, no non-TTY fallback, no post-install capability summary.
- **Gap:** sectioned re-runnable wizard, non-TTY guidance, capability matrix output, smoke test.

### 2. `doctor` command
- **Hermes:** `doctor.py:164-1240`. Full-stack diagnosis across Python, packages, config files, provider/credential validity, dir structure, SQLite health, CLI symlink, external tools, systemd linger. `--fix` creates `.env`, migrates stale keys, runs WAL checkpoints, repairs broken symlinks, and is skipped in managed mode.
- **AGH:** none.
- **Gap:** entire command. Highest priority — this is the DX safety net for first-time users when anything breaks.

### 3. Install scripts (cross-platform)
- **Hermes:** `install.sh` detects macOS/Linux/Termux/Windows (`install.sh:196-231`), installs `uv`, provisions Python 3.11, creates venv, uses `uv sync --locked` for hash-verified reproducible installs (`setup-hermes.sh:180-195`), symlinks `hermes` into `~/.local/bin`, edits the right shell rc with TTY-safe `/dev/tty` prompt fallback (`install.sh:137-160`). Flags: `--no-venv`, `--skip-setup`, `--branch`, `--dir`, `--hermes-home`. `install.ps1` mirrors this on Windows.
- **AGH:** none. Users download a release archive and place the binary themselves.
- **Gap:** `curl | sh` installer that detects OS/arch, fetches the right goreleaser archive, verifies the cosign signature, symlinks to `~/.local/bin/agh`, edits shell rc, prompts to run `agh install`.

### 4. Packaging
- **Hermes:** Homebrew formula (`packaging/homebrew/hermes-agent.rb`) writes `HERMES_MANAGED=homebrew` into the binary's env wrapper (`hermes-agent.rb:29-38`); `config.py:81-147` reads that and blocks mutations with a friendly "use `brew upgrade hermes-agent`" error. NixOS managed mode via `.managed` marker + `.container-mode` metadata for container exec-in.
- **AGH:** `.goreleaser.yml:22-80` only builds archives + cosign + SBOM. No `brews:`, `nfpms:`, `scoops:`, `dockers:`, or `chocolateys:` stanzas despite goreleaser-pro.
- **Gap:** Homebrew tap (just add `brews:` to goreleaser), nFPM deb/rpm, Scoop, Docker image, plus the `AGH_MANAGED` convention.

### 5. Uninstall
- **Hermes:** `uninstall.py:1-482` — "Keep data" vs "Full" vs "Cancel", discovers named profiles and offers to wipe them, stops gateway, removes systemd units + launchd plists, strips PATH lines from every shell rc, wipes `~/.hermes`. Delegates per-profile teardown via `hermes -p <name> gateway uninstall` (`uninstall.py:231-280`).
- **AGH:** none. Users who bounce leave behind a daemon + socket + data dir.
- **Gap:** `agh uninstall` that stops daemon, removes launchd/systemd unit, removes socket/lock, optionally purges `~/.agh`.

### 6. Auth
- **Hermes:** `auth.py:1-3475` — registry of 20+ providers (Nous/Anthropic/Codex/Qwen/Gemini OAuth + Copilot/GLM/Kimi/Minimax/HF API key), `ProviderConfig` with `auth_type`, env-var priority, base-URL override (`auth.py:92-200`). File-locked `~/.hermes/auth.json`, token refresh with skew. `hermes auth add|list|remove|reset` with credential-pool rotation (round-robin / fill-first / random / least-used — `auth_commands.py:12-32`) and cascading cleanup that suppresses auto-reseed on remove (`auth_commands.py:354-399`).
- **AGH:** providers in `config.toml` via explicit `credential_slots` only. No OAuth, no token store, no `agh auth` tree.
- **Gap:** `agh auth login/logout/status` with locked `~/.agh/auth.json` at mode 0600. Credential pools are a v2 luxury.

### 7. Profiles
- **Hermes:** `profiles.py:1-1085` — named profile = independent `HERMES_HOME` under `~/.hermes/profiles/<name>/` with its own config/env/memory/sessions/logs/gateway. `profile create --clone` copies config+env+SOUL (`profiles.py:53-66`). Creates `~/.local/bin/<name>` wrapper → `hermes -p <name> "$@"` (`profiles.py:227-246`) with collision detection. `-p/--profile` intercepted before any module import so `HERMES_HOME` is set correctly (`main.py:86-147`). Sticky default via `active_profile` file.
- **AGH:** none. Single `~/.agh` via `AGH_HOME`.
- **Gap:** entire profile system. Particularly valuable for separating personal/work/experimental agent setups.

### 8. Shell completion
- **Hermes:** `completion.py:1-315` walks the live argparse tree (no hardcoded command lists) to emit bash/zsh/fish. Includes profile-name completion helpers. Installed via `eval "$(hermes completion bash)"`.
- **AGH:** no `completion` subcommand. Cobra has this built-in.
- **Gap:** ~30 LOC to add `agh completion {bash,zsh,fish,powershell}` + `ValidArgsFunction` on commands taking provider/agent/session IDs.

### 9. Banner / tips / update check
- **Hermes:** ASCII logo + skill count + cached update check (6h TTL, `banner.py:126-150`); `tips.py` shows a random tip per session from 300+ covering slash commands, keybindings, flags, tools, profiles.
- **AGH:** none.
- **Gap:** optional; cheap and high-signal for feature discovery.

### 10. Config discovery & edit
- **Hermes:** default → `~/.hermes/config.yaml` → env → flags. `config set KEY VALUE` auto-routes (secrets → `.env`, settings → `config.yaml`). `config edit`, `path`, `env-path`, `show`, `check`, `migrate`. Managed-mode blocks mutations (`config.py:81-147`).
- **AGH:** 5-layer precedence exists (`config.go:284-341`) but **no `agh config` commands at all** — users hand-edit TOML.
- **Gap:** `agh config show|edit|set|get|path|check`; worth more when paired with doctor.

### 11. Plugin management
- **Hermes:** `plugins.py:1-945` + `plugins_cmd.py:1-1280` — `hermes plugins install|update|remove|list|enable|disable`, `owner/repo` shorthand, path-traversal validation, `plugin.yaml` manifests with `requires_env` prompting (`plugins_cmd.py:152-200`), `.example` file copy, `after-install.md` rendering. Allow-list `plugins.enabled` + deny-list `plugins.disabled`.
- **AGH:** `extension` + `skill` commands cover marketplace install/remove, but no env-var handshake on install and no `enable|disable` toggle.
- **Gap:** add `requires_env` prompt and `extension enable|disable` to existing commands.

### 12. Backup / restore
- **Hermes:** `backup.py:1-655` zips `~/.hermes/` excluding repo/cache/runtime state, uses SQLite `.backup()` for WAL-safe DB snapshots (`backup.py:78-98`). `hermes import` restores.
- **AGH:** none.
- **Gap:** `agh backup create|restore` using the same SQLite-safe pattern for `~/.agh/agh.db` + per-session `events.db`.

### 13. Environment handling / secret hygiene
- **Hermes:** `env_loader.py:1-123` loads `~/.hermes/.env` with user-override-beats-stale-shell logic, **strips non-ASCII from keys ending in `_API_KEY / _TOKEN / _SECRET / _KEY`** (`env_loader.py:15-31`) because PDF-pasted keys sometimes contain Unicode lookalikes that break HTTP headers. Repairs corrupted `.env` files where multiple `KEY=VALUE` pairs got concatenated onto one line (`env_loader.py:47-89`).
- **AGH:** `godotenv` only. No sanitization, no repair.
- **Gap:** port both. Empirical bug-driven fixes — free learnings.

### 14. Update / upgrade
- **Hermes:** `hermes update` runs `git pull` in dev clones but refuses under `is_managed()` and points at `brew upgrade` / `nixos-rebuild switch`. Non-blocking version check in banner.
- **AGH:** no self-update; users re-download from Releases.
- **Gap:** `agh update` for source installs that defers to the package manager when present; add a non-blocking version check on daemon start.

### 15. Timeouts / TTY safety
- **Hermes:** `_require_tty` guard (`main.py:55-69`) prevents interactive TUI commands from spinning at 100% CPU when stdin is piped.
- **AGH:** no equivalent. `agh install` (Bubble Tea) would misbehave under `echo | agh install`.
- **Gap:** TTY guard on every interactive command.

## Patterns worth stealing

1. **Managed-mode convention.** Ship Homebrew/Scoop/nFPM formulas that bake `AGH_MANAGED=<pm>` into the binary's env. `agh config set` and `agh update` detect it and defer to the package manager. ~50 LOC.
2. **Doctor with `--fix`.** Auto-repair broken symlinks, stale config keys, missing `~/.agh`, WAL checkpoints — gated by managed-mode.
3. **Cobra completion.** Just expose `agh completion <shell>` and add `ValidArgsFunction` for IDs.
4. **`curl | sh` installer** with arch detection, cosign verification, shell-rc edit, `~/.local/bin` symlink, final `agh install` launch.
5. **Tips corpus.** One random tip per session; nearly free feature discovery.
6. **Profile wrappers.** `~/.local/bin/<profile>` runs `agh -p <name> "$@"` — instant `work chat` UX.
7. **`.env` credential sanitization.** 10-line ASCII strip on `*_API_KEY|*_TOKEN|*_SECRET`.
8. **SQLite safe-copy for backup** via `VACUUM INTO` or a sidecar `sqlite3 .backup` shell-out.
9. **Config value auto-routing.** `agh config set KEY=VAL` writes secrets to `.env` and dotted paths to `config.toml`.
10. **Thorough uninstaller** that removes PATH edits, wrapper scripts, system services, and data dirs with explicit opt-in.

## Explicitly skip

- **Nous subscription + credential-pool rotation strategies** — Hermes-unique commercial integration.
- **Python venv / uv bootstrapping** — AGH is a Go binary.
- **Curses TUI** — AGH already uses Bubble Tea.
- **Monolithic `cli.py` (10832 LOC)** — its REPL belongs in AGH's web UI / future TUI, not the daemon CLI.
- **Honcho / SOUL.md / personalities / skills-hub taps** — either AGH-unique (marketplace exists) or Hermes-product-specific.
- **Nix / container-mode dispatch, Termux/Android install path** — not AGH's current platforms.
- **Telegram/Discord/WhatsApp gateway setup** — AGH has no messaging bridge concept.

## Key reference paths

- Hermes: `.resources/hermes/hermes_cli/{setup,doctor,auth,auth_commands,profiles,plugins,plugins_cmd,completion,uninstall,backup,env_loader,config,banner,tips,main}.py`
- Hermes install: `.resources/hermes/scripts/{install.sh,install.ps1,install.cmd}`, `.resources/hermes/setup-hermes.sh`
- Hermes Homebrew: `.resources/hermes/packaging/homebrew/hermes-agent.rb`
- AGH CLI: `/Users/pedronauck/Dev/compozy/agh/internal/cli/{root,install}.go`
- AGH config: `/Users/pedronauck/Dev/compozy/agh/internal/config/{config,home,merge}.go`
- AGH release: `/Users/pedronauck/Dev/compozy/agh/.goreleaser.yml`
- AGH scripts: `/Users/pedronauck/Dev/compozy/agh/scripts/postinstall.sh` (dev-only)
