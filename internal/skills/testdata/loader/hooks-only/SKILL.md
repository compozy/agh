---
name: hooks-only
description: Skill with hook metadata only
metadata:
  agh:
    hooks:
      - event: on_session_created
        command: /bin/sh
        args:
          - -c
          - echo ready
        timeout: 5s
        env:
          HOOK_ENV: enabled
---

body
