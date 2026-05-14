#!/usr/bin/env python3
"""Discover the repository QA contract for qa-execution and agh-qa-bootstrap.

This helper is read-only. It inspects well-known repo manifests and emits a
small JSON contract that downstream QA skills can embed in bootstrap manifests.
"""

from __future__ import annotations

import argparse
import json
from pathlib import Path
from typing import Any


def read_json(path: Path) -> dict[str, Any]:
    if not path.exists():
        return {}
    with path.open("r", encoding="utf-8") as handle:
        data = json.load(handle)
    if not isinstance(data, dict):
        return {}
    return data


def make_targets(makefile: Path) -> list[str]:
    if not makefile.exists():
        return []
    targets: list[str] = []
    for line in makefile.read_text(encoding="utf-8").splitlines():
        if not line or line.startswith(("\t", " ", ".")) or ":" not in line:
            continue
        name = line.split(":", 1)[0].strip()
        if name and all(part for part in name.split()) and not name.startswith("#"):
            targets.append(name)
    return sorted(set(targets))


def script_names(package_json: Path) -> list[str]:
    scripts = read_json(package_json).get("scripts", {})
    if not isinstance(scripts, dict):
        return []
    return sorted(str(name) for name in scripts)


def command_if_target(targets: set[str], target: str) -> str | None:
    if target in targets:
        return f"make {target}"
    return None


def compact_commands(targets: set[str]) -> dict[str, Any]:
    commands: dict[str, Any] = {}
    for key, target in (
        ("verify", "verify"),
        ("codegen", "codegen"),
        ("codegen_check", "codegen-check"),
        ("fmt", "fmt"),
        ("lint_go", "lint"),
        ("test_go", "test"),
        ("build_go", "build"),
        ("boundaries", "boundaries"),
        ("bun_lint", "bun-lint"),
        ("bun_typecheck", "bun-typecheck"),
        ("bun_test", "bun-test"),
        ("web_dev", "web-dev"),
        ("web_build", "web-build"),
        ("web_lint", "web-lint"),
        ("web_typecheck", "web-typecheck"),
        ("web_test", "web-test"),
        ("site_dev", "site-dev"),
        ("site_build", "site-build"),
        ("test_integration", "test-integration"),
        ("test_e2e_runtime", "test-e2e-runtime"),
        ("test_e2e_web", "test-e2e-web"),
        ("test_e2e", "test-e2e"),
    ):
        command = command_if_target(targets, target)
        if command is not None:
            commands[key] = command
    if "verify" in commands:
        commands["canonical_gate"] = commands["verify"]
    return commands


def discover(root: Path) -> dict[str, Any]:
    root = root.resolve()
    targets = set(make_targets(root / "Makefile"))
    package_json = read_json(root / "package.json")
    web_package_json = read_json(root / "web" / "package.json")
    site_package_json = read_json(root / "packages" / "site" / "package.json")

    package_manager = "bun" if (root / "bun.lock").exists() else "unknown"
    languages = []
    if (root / "go.mod").exists():
        languages.append("go")
    if package_json:
        languages.append("typescript")

    commands = compact_commands(targets)
    if "web_dev" in commands:
        commands["web_dev_isolated_daemon"] = (
            "AGH_WEB_API_PROXY_TARGET=$AGH_WEB_API_PROXY_TARGET make web-dev"
        )

    return {
        "schema_version": 1,
        "project_root": str(root),
        "project": {
            "name": package_json.get("name", root.name),
            "package_manager": package_manager,
            "languages": languages,
        },
        "commands": commands,
        "make_targets": sorted(targets),
        "package_scripts": {
            "root": script_names(root / "package.json"),
            "web": script_names(root / "web" / "package.json"),
            "site": script_names(root / "packages" / "site" / "package.json"),
        },
        "surfaces": {
            "runtime": {
                "paths": ["cmd/agh", "internal"],
                "entrypoints": ["HTTP", "SSE", "UDS", "CLI"],
                "verification": [commands.get("test_go"), commands.get("test_e2e_runtime")],
            },
            "web": {
                "paths": ["web"],
                "entrypoints": ["Vite SPA", "Storybook", "Playwright"],
                "verification": [commands.get("bun_typecheck"), commands.get("bun_test"), commands.get("test_e2e_web")],
                "dev_server": commands.get("web_dev"),
                "isolated_daemon_env": "AGH_WEB_API_PROXY_TARGET",
            },
            "site": {
                "paths": ["packages/site"],
                "entrypoints": ["Next.js", "Fumadocs"],
                "verification": [commands.get("site_build"), "bunx turbo run test --filter=./packages/site"],
            },
            "sdk": {
                "paths": ["sdk/typescript"],
                "entrypoints": ["TypeScript SDK"],
                "verification": ["bunx turbo run typecheck --filter=./sdk/typescript", "bunx turbo run test --filter=./sdk/typescript"],
            },
        },
        "qa_contract": {
            "canonical_gate": commands.get("canonical_gate", "make verify"),
            "requires_rtk_prefix": True,
            "isolated_daemon_env": [
                "AGH_HOME",
                "AGH_HTTP_PORT",
                "AGH_UDS_PATH",
                "TMUX_BRIDGE_SOCKET",
                "AGH_WEB_API_PROXY_TARGET",
            ],
            "provider_home_env": ["PROVIDER_HOME", "PROVIDER_CODEX_HOME"],
            "config_writes_parallel_safe": False,
        },
    }


def main() -> int:
    parser = argparse.ArgumentParser(description="Discover the AGH QA project contract.")
    parser.add_argument("--root", default=".", help="Repository root to inspect.")
    args = parser.parse_args()
    contract = discover(Path(args.root))
    print(json.dumps(contract, indent=2, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
