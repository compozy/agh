---
name: invalid-hook
description: Skill with invalid hook event
metadata:
  agh:
    hooks:
      - event: on_session_started
        command: /bin/echo
        args:
          - invalid
---

body
