#!/usr/bin/env python3
"""Smoke tests for bootstrap-qa-env helpers that are not covered by repo gates."""

from __future__ import annotations

import importlib.util
import json
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


if __name__ == "__main__":
    main()
