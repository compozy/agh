#!/usr/bin/env python3
"""Estimate before/after prompt cost for the tokens-perf channel corpus."""

from __future__ import annotations

import argparse
import datetime as dt
import json
import os
import pathlib
import re
from dataclasses import dataclass
from typing import Any

CORPUS_DIR_ENV = "AGH_TOKEN_CORPUS_DIR"
OPEN_CATALOG = "<current-available-skills>"
CLOSE_CATALOG = "</current-available-skills>"
BT = chr(96)
FINAL_CATALOG_LINE = (
    f"If current tool policy denies {BT}agh__skill_view{BT}, "
    f"use {BT}agh skill view <name>{BT} as an operator fallback."
)
NETWORK_CLOSE = "</network-message>"
COMPACT_CATALOG_STATE_TEXT = (
    "Previous catalog remains current; use `agh__skill_view` for full skill/resource instructions."
)
COMPACT_CATALOG_TEXT = "\n".join(
    [
        OPEN_CATALOG,
        f'  <catalog-state unchanged="true">{COMPACT_CATALOG_STATE_TEXT}</catalog-state>',
        CLOSE_CATALOG,
        "",
        FINAL_CATALOG_LINE,
    ]
)
COMPACT_CATALOG_BYTES = len(COMPACT_CATALOG_TEXT.encode())
COMPACT_GUIDANCE_TEXT = (
    f"Full protocol examples were already provided earlier in this session; "
    f"run {BT}agh network --help{BT} for command details."
)
WORK_ID_RE = re.compile(r'\bwork-id="([^"]+)"')


@dataclass
class PromptEvent:
    text: str
    remaining_assistant_calls: int
    timestamp: dt.datetime | None


def parse_args() -> pathlib.Path:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "corpus_dir",
        nargs="?",
        help=f"Directory containing *.jsonl transcripts. Defaults to {CORPUS_DIR_ENV} when set.",
    )
    parser.add_argument(
        "--corpus-dir",
        dest="corpus_dir_override",
        help="Explicit transcript directory override.",
    )
    args = parser.parse_args()
    raw_path = first_non_empty_path(
        args.corpus_dir_override,
        args.corpus_dir,
        os.environ.get(CORPUS_DIR_ENV, ""),
    )
    if raw_path == "":
        parser.error(
            f"corpus directory is required via --corpus-dir, positional argument, or {CORPUS_DIR_ENV}"
        )
    root = pathlib.Path(raw_path).expanduser().resolve()
    if not root.exists():
        parser.error(f"corpus directory does not exist: {root}")
    if not root.is_dir():
        parser.error(f"corpus path is not a directory: {root}")
    return root


def first_non_empty_path(*values: str | None) -> str:
    for value in values:
        if value is None:
            continue
        trimmed = value.strip()
        if trimmed:
            return trimmed
    return ""


def content_text(message: dict[str, Any]) -> str:
    content = message.get("content")
    if isinstance(content, str):
        return content
    if isinstance(content, list):
        return "".join(
            item.get("text", "")
            for item in content
            if isinstance(item, dict) and isinstance(item.get("text"), str)
        )
    return ""


def parse_timestamp(value: object) -> dt.datetime | None:
    if not isinstance(value, str) or not value:
        return None
    try:
        return dt.datetime.fromisoformat(value.replace("Z", "+00:00"))
    except ValueError:
        return None


def is_assistant_call(row: dict[str, Any]) -> bool:
    if row.get("type") != "assistant":
        return False
    message = row.get("message")
    return isinstance(message, dict) and (bool(message.get("usage")) or message.get("role") == "assistant")


def catalog_sections(text: str) -> list[str]:
    sections: list[str] = []
    start = 0
    while True:
        idx = text.find(OPEN_CATALOG, start)
        if idx < 0:
            return sections
        close = text.find(CLOSE_CATALOG, idx)
        if close < 0:
            return sections
        end = close + len(CLOSE_CATALOG)
        final = text.find(FINAL_CATALOG_LINE, end)
        if final >= 0:
            end = final + len(FINAL_CATALOG_LINE)
        sections.append(text[idx:end])
        start = end


def network_guidance(text: str) -> tuple[str, bool]:
    start = text.find("<network-message")
    if start < 0:
        return "", False
    close = text.find(NETWORK_CLOSE, start)
    if close < 0:
        return "", False
    guidance = text[close + len(NETWORK_CLOSE) :]
    has_work_id = bool(WORK_ID_RE.search(text[start:close]))
    return guidance, has_work_id


def compact_guidance_bytes(old_guidance: str) -> int | None:
    marker = "Examples:\n"
    if marker not in old_guidance:
        return None
    prefix = old_guidance.split(marker, 1)[0]
    return len((prefix + COMPACT_GUIDANCE_TEXT + "\n").encode())


def load_network_prompt_events(path: pathlib.Path) -> list[PromptEvent]:
    raw_events: list[tuple[str, str, dt.datetime | None]] = []
    for line in path.read_text(errors="replace").splitlines():
        try:
            row = json.loads(line)
        except json.JSONDecodeError:
            continue
        if row.get("type") == "user":
            text = content_text(row.get("message") or {})
            raw_events.append(("user", text, parse_timestamp(row.get("timestamp"))))
            continue
        if is_assistant_call(row):
            raw_events.append(("assistant", "", parse_timestamp(row.get("timestamp"))))

    remaining_assistant = sum(1 for kind, _, _ in raw_events if kind == "assistant")
    prompts: list[PromptEvent] = []
    for kind, text, timestamp in raw_events:
        if kind == "assistant":
            remaining_assistant -= 1
            continue
        if "<network-message" in text or "<network-body" in text:
            prompts.append(
                PromptEvent(
                    text=text,
                    remaining_assistant_calls=remaining_assistant,
                    timestamp=timestamp,
                )
            )
    return prompts


def main() -> None:
    root = parse_args()
    totals = {
        "network_transcripts": 0,
        "network_prompt_turns_before": 0,
        "network_prompt_turns_after_tasks_006_009": 0,
        "network_messages": 0,
        "network_prompt_direct_bytes_before": 0,
        "network_prompt_direct_bytes_after_tasks_006_009": 0,
        "network_prompt_replay_bytes_before": 0,
        "network_prompt_replay_bytes_after_tasks_006_009": 0,
        "skill_catalog_repeated_blocks": 0,
        "skill_catalog_direct_saved_bytes": 0,
        "skill_catalog_replay_saved_bytes": 0,
        "network_guidance_compacted_prompts": 0,
        "network_guidance_direct_saved_bytes": 0,
        "network_guidance_replay_saved_bytes": 0,
    }
    gaps: list[float] = []
    top_files: list[dict[str, Any]] = []

    for path in sorted(root.glob("*.jsonl")):
        prompts = load_network_prompt_events(path)
        if not prompts:
            continue

        totals["network_transcripts"] += 1
        seen_catalogs: set[str] = set()
        reply_delivered = False
        protocol_delivered = False
        file_direct_saved = 0
        file_replay_saved = 0

        for left, right in zip(prompts, prompts[1:]):
            if left.timestamp is not None and right.timestamp is not None:
                gaps.append((right.timestamp - left.timestamp).total_seconds())

        for prompt in prompts:
            text_bytes = len(prompt.text.encode())
            remaining = prompt.remaining_assistant_calls
            totals["network_prompt_turns_before"] += 1
            totals["network_prompt_turns_after_tasks_006_009"] += 1
            totals["network_messages"] += prompt.text.count("<network-message")
            totals["network_prompt_direct_bytes_before"] += text_bytes
            totals["network_prompt_direct_bytes_after_tasks_006_009"] += text_bytes
            totals["network_prompt_replay_bytes_before"] += text_bytes * remaining
            totals["network_prompt_replay_bytes_after_tasks_006_009"] += text_bytes * remaining

            prompt_direct_saved = 0
            prompt_replay_saved = 0

            for section in catalog_sections(prompt.text):
                if section in seen_catalogs:
                    saved = max(len(section.encode()) - COMPACT_CATALOG_BYTES, 0)
                    totals["skill_catalog_repeated_blocks"] += 1
                    totals["skill_catalog_direct_saved_bytes"] += saved
                    totals["skill_catalog_replay_saved_bytes"] += saved * remaining
                    prompt_direct_saved += saved
                    prompt_replay_saved += saved * remaining
                seen_catalogs.add(section)

            guidance, has_work_id = network_guidance(prompt.text)
            should_compact_guidance = reply_delivered and (not has_work_id or protocol_delivered)
            if should_compact_guidance:
                compact_bytes = compact_guidance_bytes(guidance)
                if compact_bytes is not None:
                    saved = max(len(guidance.encode()) - compact_bytes, 0)
                    totals["network_guidance_compacted_prompts"] += 1
                    totals["network_guidance_direct_saved_bytes"] += saved
                    totals["network_guidance_replay_saved_bytes"] += saved * remaining
                    prompt_direct_saved += saved
                    prompt_replay_saved += saved * remaining

            reply_delivered = True
            if has_work_id:
                protocol_delivered = True

            totals["network_prompt_direct_bytes_after_tasks_006_009"] -= prompt_direct_saved
            totals["network_prompt_replay_bytes_after_tasks_006_009"] -= prompt_replay_saved
            file_direct_saved += prompt_direct_saved
            file_replay_saved += prompt_replay_saved

        top_files.append(
            {
                "file": path.name,
                "prompt_turns": len(prompts),
                "network_messages": sum(prompt.text.count("<network-message") for prompt in prompts),
                "direct_saved_bytes": file_direct_saved,
                "replay_saved_bytes": file_replay_saved,
            }
        )

    direct_before = totals["network_prompt_direct_bytes_before"]
    direct_after = totals["network_prompt_direct_bytes_after_tasks_006_009"]
    replay_before = totals["network_prompt_replay_bytes_before"]
    replay_after = totals["network_prompt_replay_bytes_after_tasks_006_009"]

    result = {
        "corpus": str(root),
        "totals": totals,
        "direct_saved_bytes": direct_before - direct_after,
        "direct_saved_estimated_tokens_bytes_div_4": round((direct_before - direct_after) / 4),
        "direct_reduction_percent": round(((direct_before - direct_after) / direct_before) * 100, 2)
        if direct_before
        else 0,
        "replay_saved_bytes": replay_before - replay_after,
        "replay_saved_estimated_tokens_bytes_div_4": round((replay_before - replay_after) / 4),
        "replay_reduction_percent": round(((replay_before - replay_after) / replay_before) * 100, 2)
        if replay_before
        else 0,
        "gap_seconds": {
            "count": len(gaps),
            "min": min(gaps) if gaps else None,
            "p50": sorted(gaps)[len(gaps) // 2] if gaps else None,
            "p90": sorted(gaps)[int(0.9 * (len(gaps) - 1))] if gaps else None,
            "max": max(gaps) if gaps else None,
        },
        "top_saved_files": sorted(top_files, key=lambda item: item["replay_saved_bytes"], reverse=True)[:5],
    }
    print(json.dumps(result, indent=2, sort_keys=True))


if __name__ == "__main__":
    main()
