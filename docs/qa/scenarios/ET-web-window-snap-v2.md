---
id: ET-web-window-snap-v2
area: ET
title: Snap v2 — generous zones, gutters, linked seams, split drops, zoom menu
persona: Bruno
journey:
expected: Dragging near a desktop edge arms halves within a 32px band and corner quarters within a 150px reach with a stable (hysteresis) preview; snapped neighbors render with an 8px gutter and grow a draggable seam that resizes both windows live and persists fractions; a snapped or maximized window resizes from its own handles (snapped rewrites fractions in place, maximized detaches to floating); dropping a window onto another window's outer third splits that window's space with both landing snapped; hovering the zoom control opens Move & Resize (halves incl. top/bottom + quarters + restore) and Fill & Arrange (fill, 2-up, grid — disabled without a second window); palette and ⌃⌥ chords stay the keyboard path; agent-written fractions converge live.
entry_points: web desktop windows; zoom traffic light; command palette; agh desktop-state
qa_status: untested
bug_ids:
fix_status:
retest_status:
fix_commits:
evidence:
last_report:
overlaps: ET-web-window-routing-lifecycle; ET-desktop-state-agent-surface; ET-web-command-palette-shortcuts
---

story: As a builder I arrange windows the way macOS/Windows lets me — generous snap targets, breathable gutters, a linked seam, in-place resize, window-relative splits, and a zoom-button menu — and agents converge on the same fractional truth.

qa-impact: Snap UX overhaul (post task_09) changed zone bands/hysteresis, added gutter+seam, resize-in-place fraction rewrite, maximized resize detach, window-relative split drops, top/bottom preset zones, and the zoom hover menu with arrange presets. Flag only; the next QA cycle owns live retesting.
