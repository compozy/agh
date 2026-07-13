# Bridge Providers

Each directory with an `extension.toml` is discovered as an in-tree bridge provider. There is no
central provider registry.

## Start here

- Operator setup: `packages/site/content/runtime/core/bridges/setup.mdx`
- Provider author walkthrough: `packages/site/content/runtime/core/bridges/adding-a-bridge.mdx`
- In-repo review checklist: `internal/bridges/ADDING_A_BRIDGE.md`
- CI-safe protocol example: `sdk/examples/telegram-reference`

## Bundled providers

| Directory  | Platform        | Inbound model                                 | Outbound model                           |
| ---------- | --------------- | --------------------------------------------- | ---------------------------------------- |
| `slack`    | Slack           | Signed Events API, commands, and interactions | Message create/edit/delete               |
| `telegram` | Telegram        | Secret-token Bot API webhooks                 | Message create/edit/delete               |
| `discord`  | Discord         | Ed25519 interactions and webhook events       | REST message create/edit/delete          |
| `whatsapp` | WhatsApp        | Meta verification challenge and signed POST   | Cloud API text create                    |
| `teams`    | Microsoft Teams | Bot Framework activities and bearer JWTs      | Activity create/edit/delete              |
| `gchat`    | Google Chat     | Direct, Pub/Sub, or hybrid Google JWTs        | Chat message create/edit/delete          |
| `github`   | GitHub          | Signed issue and review comment webhooks      | Issue/review comment create/edit/delete  |
| `linear`   | Linear          | Signed comment or Agent Session webhooks      | Comments or append-only Agent Activities |

Provider-specific build, configuration, limits, and unsupported operations live in each directory's
README. Public console steps and operator recovery live in the site setup and operations guides.
