#!/usr/bin/env python3
"""Smoke tests for bootstrap-qa-env helpers that are not covered by repo gates.

Suite: QA bootstrap helper integration.
Invariant: bootstrap resolves and executes its bundled helpers from the current repository layout.
Boundary IN: bootstrap helper path resolution and real workspace initialization script.
Boundary OUT: daemon startup and behavioral QA, owned by downstream QA execution.
"""

from __future__ import annotations

import importlib.util
import json
import os
import sys
import tempfile
import types
from pathlib import Path


def load_bootstrap_module():
    if importlib.util.find_spec("tomllib") is None:
        tomllib_stub = types.ModuleType("tomllib")

        def unavailable_tomllib(*args, **kwargs):
            raise RuntimeError("tomllib is not available in this smoke-test interpreter")

        tomllib_stub.load = unavailable_tomllib
        tomllib_stub.loads = unavailable_tomllib
        sys.modules["tomllib"] = tomllib_stub

    script_path = Path(__file__).with_name("bootstrap-qa-env.py")
    sys.path.insert(0, str(script_path.parent))
    spec = importlib.util.spec_from_file_location("bootstrap_qa_env", script_path)
    if spec is None or spec.loader is None:
        raise RuntimeError(f"failed to load module spec for {script_path}")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def write_discovery_script(repo_root: Path, payload: dict) -> None:
    script_path = repo_root / ".agents" / "skills" / "qa-execution" / "scripts" / "discover-project-contract.py"
    script_path.parent.mkdir(parents=True)
    script_path.write_text(
        "#!/usr/bin/env python3\n"
        "import json\n"
        f"print(json.dumps({json.dumps(payload)}))\n",
        encoding="utf-8",
    )


def main() -> None:
    module = load_bootstrap_module()
    with tempfile.TemporaryDirectory() as raw_dir:
        repo_root = Path(raw_dir)
        if module.discover_project_contract(repo_root) != {}:
            raise AssertionError("missing discovery script should return an empty contract")

        payload = {"schema_version": 1, "surfaces": ["loops"]}
        write_discovery_script(repo_root, payload)
        got = module.discover_project_contract(repo_root)
        if got != payload:
            raise AssertionError(f"discover_project_contract() = {got!r}, want {payload!r}")

    repo_root = Path(__file__).resolve().parents[5]
    playbook = module.load_playbook_via_helper(repo_root, "devtool-oss-launch")
    if playbook.get("playbook_ref") != "devtool-oss-launch":
        raise AssertionError(f"loaded playbook_ref = {playbook.get('playbook_ref')!r}")

    with tempfile.TemporaryDirectory() as raw_dir:
        workspace_info = module.run_init_workspace(repo_root, "path-resolution-smoke", raw_dir)
        workspace_path = Path(workspace_info["WORKSPACE_PATH"])
        if workspace_path.parent != Path(raw_dir):
            raise AssertionError(
                f"run_init_workspace() created {workspace_path}, want a lab directly under {raw_dir}"
            )
        if not (workspace_path / "qa-artifacts" / "qa" / "screenshots").is_dir():
            raise AssertionError("run_init_workspace() did not execute the bundled workspace initializer")

    playbook = module.load_playbook_via_helper(repo_root, "northstar-pay")
    if playbook.get("playbook_ref") != "northstar-pay":
        raise AssertionError("load_playbook_via_helper() returned the wrong playbook")

    with tempfile.TemporaryDirectory() as raw_dir:
        workspace_path = Path(raw_dir)
        summary = module.seed_playbook_workspace(repo_root, workspace_path, "northstar-pay")
        if not Path(summary["playbook_snapshot"]).is_file():
            raise AssertionError("seed_playbook_workspace() did not materialize the playbook")

        qa_root = workspace_path / "qa-artifacts" / "qa"
        qa_root.mkdir(parents=True)
        evidence_paths = module.seed_qa_evidence_contracts(
            repo_root, qa_root, "path-resolution-smoke", playbook
        )
        if not evidence_paths["AUDIT_COMMAND"].is_file():
            raise AssertionError("seed_qa_evidence_contracts() emitted a missing audit command")

    with tempfile.TemporaryDirectory() as raw_dir:
        browser_bin = Path(raw_dir) / "browser-use"
        browser_bin.write_text("#!/bin/sh\nexit 0\n", encoding="utf-8")
        browser_bin.chmod(0o755)
        previous_path = os.environ.get("PATH")
        try:
            os.environ["PATH"] = raw_dir
            mode, blocker = module.detect_browser_mode(Path(raw_dir) / "empty-codex-home")
            if (mode, blocker) != ("browser-use", ""):
                raise AssertionError(
                    f"detect_browser_mode(browser-use CLI) = {(mode, blocker)!r}, "
                    "want ('browser-use', '')"
                )

            browser_bin.unlink()
            mode, blocker = module.detect_browser_mode(Path(raw_dir) / "empty-codex-home")
            if mode != "blocked" or "Neither browser-use" not in blocker:
                raise AssertionError(
                    f"detect_browser_mode(no browser) = {(mode, blocker)!r}, want blocked"
                )
        finally:
            if previous_path is None:
                os.environ.pop("PATH", None)
            else:
                os.environ["PATH"] = previous_path

if __name__ == "__main__":
    main()
