#!/usr/bin/env python3
"""Build or reuse a deterministic AGH local QA lab and emit a canonical manifest."""

from __future__ import annotations

import argparse
from datetime import datetime, timezone
import hashlib
import json
import os
import shutil
import socket
import subprocess
import tempfile
from pathlib import Path
from urllib.parse import urlparse


ALLOWED_PROVIDER_FILES = ("auth.json", "installation_id", "version.json")
MAX_UNIX_SOCKET_PATH_BYTES = 103


def slugify(value: str) -> str:
    cleaned = "".join(ch.lower() if ch.isalnum() else "-" for ch in value.strip())
    while "--" in cleaned:
        cleaned = cleaned.replace("--", "-")
    cleaned = cleaned.strip("-")
    return cleaned or "release-candidate"


def shquote(value: str) -> str:
    if not value:
        return "''"
    safe_chars = set("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_/.:")
    if all(ch in safe_chars for ch in value):
        return value
    return "'" + value.replace("'", "'\\''") + "'"


def pick_free_port() -> int:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        sock.bind(("127.0.0.1", 0))
        return int(sock.getsockname()[1])


def socket_path_length(path: Path) -> int:
    return len(str(path).encode("utf-8"))


def socket_limited_paths(agh_home: Path, provider_home: Path) -> dict[str, Path]:
    return {
        "AGH_UDS_PATH": agh_home / "aghd.sock",
        "TMUX_BRIDGE_SOCKET": agh_home / "tmux-bridge.sock",
        "PROVIDER_DEFAULT_UDS": provider_home / ".agh" / "daemon.sock",
    }


def socket_limit_violations(paths: dict[str, Path]) -> list[str]:
    violations: list[str] = []
    for label, path in paths.items():
        length = socket_path_length(path)
        if length > MAX_UNIX_SOCKET_PATH_BYTES:
            violations.append(f"{label} {path} is {length} bytes; max {MAX_UNIX_SOCKET_PATH_BYTES}")
    return violations


def short_runtime_root(scenario_slug: str) -> Path:
    digest = hashlib.sha256(scenario_slug.encode("utf-8")).hexdigest()[:12]
    return Path(tempfile.gettempdir()) / f"aghqa-{digest}"


def run_init_workspace(repo_root: Path, scenario: str, workspace_root: str) -> dict[str, str]:
    init_script = repo_root / ".agents" / "skills" / "real-scenario-qa" / "scripts" / "init-scenario-workspace.sh"
    proc = subprocess.run(
        [str(init_script), scenario, workspace_root],
        cwd=repo_root,
        check=True,
        capture_output=True,
        text=True,
    )
    result: dict[str, str] = {}
    for line in proc.stdout.splitlines():
        if "=" not in line:
            continue
        key, value = line.split("=", 1)
        result[key.strip()] = value.strip()
    required = {"SCENARIO_SLUG", "WORKSPACE_PATH", "QA_OUTPUT_PATH"}
    missing = sorted(required - set(result))
    if missing:
        raise RuntimeError(f"bootstrap workspace helper omitted required keys: {', '.join(missing)}")
    return result


def ensure_lab_scaffold(workspace_path: Path, qa_output_path: Path) -> None:
    dirs = [
        workspace_path / "company" / "leadership",
        workspace_path / "company" / "planning",
        workspace_path / "product" / "specs",
        workspace_path / "product" / "releases",
        workspace_path / "marketing" / "campaigns",
        workspace_path / "finance" / "reports",
        workspace_path / "ops" / "runbooks",
        workspace_path / "ops" / "logs",
        workspace_path / "reviews" / "findings",
        workspace_path / "knowledge",
        workspace_path / "hooks",
        workspace_path / "extensions",
        workspace_path / "skills",
        workspace_path / ".providers",
        qa_output_path / "qa" / "logs",
        qa_output_path / "qa" / "notes",
        qa_output_path / "qa" / "screenshots",
        qa_output_path / "qa" / "issues",
        qa_output_path / "qa" / "test-cases",
        qa_output_path / "qa" / "test-plans",
    ]
    for path in dirs:
        path.mkdir(parents=True, exist_ok=True)


def load_json(path: Path) -> dict:
    with path.open("r", encoding="utf-8") as handle:
        data = json.load(handle)
    if not isinstance(data, dict):
        raise RuntimeError(f"expected JSON object in {path}")
    return data


def detect_browser_mode(codex_home: Path) -> tuple[str, str]:
    browser_skill_glob = codex_home.glob("plugins/cache/openai-bundled/browser-use/*/skills/browser/SKILL.md")
    if any(browser_skill_glob):
        return "browser-use", ""
    if shutil.which("agent-browser"):
        return "agent-browser", "browser-use skill not found in CODEX_HOME plugin cache"
    return "blocked", "Neither browser-use plugin nor agent-browser CLI is available"


def safe_symlink(target: Path, link: Path) -> None:
    link.parent.mkdir(parents=True, exist_ok=True)
    if link.exists() or link.is_symlink():
        if link.is_symlink() and link.resolve() == target.resolve():
            return
        if link.is_dir() and not link.is_symlink():
            shutil.rmtree(link)
        else:
            link.unlink()
    link.symlink_to(target)


def prepare_provider_home(global_codex_home: Path, provider_home: Path) -> Path:
    provider_home.mkdir(parents=True, exist_ok=True)
    provider_codex_home = provider_home / ".codex"
    provider_codex_home.mkdir(parents=True, exist_ok=True)

    for filename in ALLOWED_PROVIDER_FILES:
        source = global_codex_home / filename
        if source.is_file():
            shutil.copy2(source, provider_codex_home / filename)

    safe_symlink(global_codex_home / "skills", provider_codex_home / "skills")
    safe_symlink(global_codex_home / "plugins" / "cache", provider_codex_home / "plugins" / "cache")

    config_path = provider_codex_home / "config.toml"
    if not config_path.exists():
        config_path.write_text(
            "# Isolated QA provider config.\n"
            "# Keep this minimal so provider launches do not inherit user-global hooks or loop state.\n",
            encoding="utf-8",
        )
    return provider_codex_home


def manifest_health(manifest: dict) -> tuple[bool, list[str]]:
    notes: list[str] = []
    env = manifest.get("env")
    if not isinstance(env, dict):
        return False, ["manifest env block missing"]

    required_paths = {
        "WORKSPACE_PATH": Path(str(env.get("WORKSPACE_PATH", ""))),
        "QA_OUTPUT_PATH": Path(str(env.get("QA_OUTPUT_PATH", ""))),
        "AGH_HOME": Path(str(env.get("AGH_HOME", ""))),
        "PROVIDER_HOME": Path(str(env.get("PROVIDER_HOME", ""))),
        "PROVIDER_CODEX_HOME": Path(str(env.get("PROVIDER_CODEX_HOME", ""))),
    }
    healthy = True
    for label, path in required_paths.items():
        if not str(path):
            healthy = False
            notes.append(f"{label} missing from manifest")
            continue
        if not path.exists():
            healthy = False
            notes.append(f"{label} path missing: {path}")

    proxy_target = str(env.get("AGH_WEB_API_PROXY_TARGET", ""))
    parsed = urlparse(proxy_target)
    if not proxy_target or parsed.scheme not in {"http", "https"} or not parsed.netloc:
        healthy = False
        notes.append("AGH_WEB_API_PROXY_TARGET is missing or not an absolute URL")

    socket_paths = {
        "AGH_UDS_PATH": Path(str(env.get("AGH_UDS_PATH", ""))),
        "TMUX_BRIDGE_SOCKET": Path(str(env.get("TMUX_BRIDGE_SOCKET", ""))),
        "PROVIDER_DEFAULT_UDS": required_paths["PROVIDER_HOME"] / ".agh" / "daemon.sock",
    }
    for note in socket_limit_violations(socket_paths):
        healthy = False
        notes.append(note)
    return healthy, notes


def discover_project_contract(repo_root: Path) -> dict:
    script_path = repo_root / ".agents" / "skills" / "qa-execution" / "scripts" / "discover-project-contract.py"
    proc = subprocess.run(
        ["python3", str(script_path), "--root", str(repo_root)],
        cwd=repo_root,
        check=True,
        capture_output=True,
        text=True,
    )
    return json.loads(proc.stdout)


def write_env_file(env_path: Path, env_map: dict[str, str]) -> None:
    lines = [f"export {key}={shquote(value)}" for key, value in env_map.items()]
    env_path.write_text("\n".join(lines) + "\n", encoding="utf-8")


def fresh_lab_scope(requested_scenario: str) -> str:
    stamp = datetime.now(timezone.utc).strftime("%Y%m%d-%H%M%S-%f")
    return f"{requested_scenario}-{stamp}"


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--scenario", default="release-candidate", help="QA scenario slug or context")
    parser.add_argument("--repo-root", default=".", help="Repository root")
    parser.add_argument(
        "--workspace-root",
        default=os.environ.get("AGH_QA_WORKSPACE_ROOT", str(Path.home() / "dev" / "qa-labs")),
        help="Parent directory for the QA lab workspace",
    )
    parser.add_argument(
        "--reuse-manifest",
        default="",
        help="Reuse an existing bootstrap manifest for the same active QA session or loop continuation",
    )
    args = parser.parse_args()

    repo_root = Path(args.repo_root).resolve()
    global_codex_home = Path(os.environ.get("CODEX_HOME", str(Path.home() / ".codex"))).expanduser().resolve()
    scenario = slugify(args.scenario)

    reused_lab = False
    status_notes: list[str] = []
    existing_manifest: dict | None = None
    reuse_manifest_path = Path(args.reuse_manifest).expanduser().resolve() if args.reuse_manifest.strip() else None
    if reuse_manifest_path is not None and reuse_manifest_path.is_file():
        existing_manifest = load_json(reuse_manifest_path)
        healthy, health_notes = manifest_health(existing_manifest)
        if healthy:
            reused_lab = True
            status_notes.append("Reused existing healthy QA bootstrap manifest from the active QA session")
        else:
            status_notes.extend(health_notes)
            status_notes.append("Rebuilt QA bootstrap after requested manifest health check failed")

    if reused_lab and existing_manifest is not None:
        workspace_path = Path(str(existing_manifest["workspace_path"])).resolve()
        qa_output_path = Path(str(existing_manifest["qa_output_path"])).resolve()
        workspace_info = {
            "SCENARIO_SLUG": str(existing_manifest.get("scenario_slug") or scenario),
            "WORKSPACE_PATH": str(workspace_path),
            "QA_OUTPUT_PATH": str(qa_output_path),
        }
    else:
        workspace_scope = fresh_lab_scope(scenario)
        workspace_info = run_init_workspace(repo_root, workspace_scope, args.workspace_root)
        workspace_path = Path(workspace_info["WORKSPACE_PATH"]).resolve()
        qa_output_path = Path(workspace_info["QA_OUTPUT_PATH"]).resolve()

    qa_root = qa_output_path / "qa"
    ensure_lab_scaffold(workspace_path, qa_output_path)

    manifest_path = qa_root / "bootstrap-manifest.json"
    env_path = qa_root / "bootstrap.env"

    if reused_lab and existing_manifest is not None:
        existing_env = existing_manifest.get("env", {})
        provider_home = Path(str(existing_env.get("PROVIDER_HOME", workspace_path / ".provider-home"))).resolve()
        agh_home = Path(str(existing_env.get("AGH_HOME", workspace_path / ".agh" / "runtime"))).resolve()
    else:
        provider_home = workspace_path / ".provider-home"
        agh_home = workspace_path / ".agh" / "runtime"
        violations = socket_limit_violations(socket_limited_paths(agh_home, provider_home))
        if violations:
            runtime_root = short_runtime_root(workspace_info["SCENARIO_SLUG"])
            provider_home = runtime_root / "provider"
            agh_home = runtime_root / "runtime"
            status_notes.append(
                "Allocated short runtime/provider homes for portable Unix socket paths: " + "; ".join(violations)
            )
            followup_violations = socket_limit_violations(socket_limited_paths(agh_home, provider_home))
            if followup_violations:
                raise RuntimeError(
                    "short QA runtime paths still exceed Unix socket limits: " + "; ".join(followup_violations)
                )

    provider_codex_home = prepare_provider_home(global_codex_home, provider_home)

    env_block: dict[str, str]
    if reused_lab and existing_manifest is not None:
        env_block = {key: str(value) for key, value in existing_manifest.get("env", {}).items()}
    else:
        agh_home.mkdir(parents=True, exist_ok=True)
        env_block = {
            "SCENARIO_SLUG": workspace_info["SCENARIO_SLUG"],
            "WORKSPACE_PATH": str(workspace_path),
            "QA_OUTPUT_PATH": str(qa_output_path),
            "AGH_HOME": str(agh_home),
            "AGH_HTTP_PORT": str(pick_free_port()),
            "AGH_UDS_PATH": str(agh_home / "aghd.sock"),
            "TMUX_BRIDGE_SOCKET": str(agh_home / "tmux-bridge.sock"),
            "AGH_WEB_API_PROXY_TARGET": "",
            "PROVIDER_HOME": str(provider_home),
            "PROVIDER_CODEX_HOME": str(provider_codex_home),
            "BROWSER_MODE": "",
            "BROWSER_BLOCKER": "",
        }

    env_block["SCENARIO_SLUG"] = workspace_info["SCENARIO_SLUG"]
    env_block["WORKSPACE_PATH"] = str(workspace_path)
    env_block["QA_OUTPUT_PATH"] = str(qa_output_path)
    env_block["PROVIDER_HOME"] = str(provider_home)
    env_block["PROVIDER_CODEX_HOME"] = str(provider_codex_home)
    env_block["AGH_WEB_API_PROXY_TARGET"] = f"http://127.0.0.1:{env_block['AGH_HTTP_PORT']}"

    browser_mode, browser_blocker = detect_browser_mode(global_codex_home)
    env_block["BROWSER_MODE"] = browser_mode
    env_block["BROWSER_BLOCKER"] = browser_blocker

    project_contract = discover_project_contract(repo_root)
    manifest = {
        "schema_version": 1,
        "scenario_slug": workspace_info["SCENARIO_SLUG"],
        "workspace_path": str(workspace_path),
        "qa_output_path": str(qa_output_path),
        "manifest_path": str(manifest_path),
        "bootstrap_env_path": str(env_path),
        "status": {
            "reused_lab": reused_lab,
            "health": "healthy" if reused_lab else "fresh",
            "notes": status_notes,
        },
        "env": env_block,
        "browser": {
            "mode": browser_mode,
            "blocker": browser_blocker,
        },
        "paths": {
            "project_root": str(repo_root),
            "qa_root": str(qa_root),
            "provider_home": str(provider_home),
            "provider_codex_home": str(provider_codex_home),
        },
        "project_contract": project_contract,
    }

    manifest_path.write_text(json.dumps(manifest, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    write_env_file(env_path, env_block)

    outputs = {
        "SCENARIO_SLUG": workspace_info["SCENARIO_SLUG"],
        "WORKSPACE_PATH": str(workspace_path),
        "QA_OUTPUT_PATH": str(qa_output_path),
        "BOOTSTRAP_MANIFEST": str(manifest_path),
        "BOOTSTRAP_ENV": str(env_path),
        "AGH_HOME": env_block["AGH_HOME"],
        "AGH_HTTP_PORT": env_block["AGH_HTTP_PORT"],
        "AGH_UDS_PATH": env_block["AGH_UDS_PATH"],
        "TMUX_BRIDGE_SOCKET": env_block["TMUX_BRIDGE_SOCKET"],
        "AGH_WEB_API_PROXY_TARGET": env_block["AGH_WEB_API_PROXY_TARGET"],
        "PROVIDER_HOME": env_block["PROVIDER_HOME"],
        "PROVIDER_CODEX_HOME": env_block["PROVIDER_CODEX_HOME"],
        "BROWSER_MODE": env_block["BROWSER_MODE"],
        "BROWSER_BLOCKER": env_block["BROWSER_BLOCKER"],
        "REUSED_LAB": "true" if reused_lab else "false",
    }
    for key, value in outputs.items():
        print(f"{key}={value}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
