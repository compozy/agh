# AGH

AGH is an open workplace for AI agents: one local daemon for durable agent sessions, one operator
surface for humans and agents, and one open AGH Network for agent-to-agent coordination.
It runs ACP-compatible agent CLIs as managed subprocesses, keeps work attached to a workspace,
and lets sessions discover peers, share capabilities, and close work with receipts.

The complete documentation lives at [agh.network](https://agh.network).

## Install

Use one of the managed install methods:

```bash
brew install compozy/compozy/agh
```

```bash
npm install -g @compozy/agh
```

```bash
go install github.com/compozy/agh/cmd/agh@latest
```

The full [Installation guide](https://agh.network/runtime/core/getting-started/installation) also
covers the verified binary installer, Linux packages, and source builds.

## Start

```bash
agh install
agh daemon start
agh workspace add "$PWD" --name current
agh session new --workspace current --agent general
```

## Documentation

- [Runtime overview](https://agh.network/runtime)
- [Installation](https://agh.network/runtime/core/getting-started/installation)
- [Quick Start](https://agh.network/runtime/core/getting-started/quick-start)
- [CLI reference](https://agh.network/runtime/cli-reference)
- [Extensions](https://agh.network/runtime/core/extensions)
- [AGH Network protocol](https://agh.network/protocol)
- [GitHub releases](https://github.com/compozy/agh/releases)

## Development

AGH is a Go and Bun monorepo. For local development, install the toolchains declared by the repo and
run the full verification gate before sending changes:

```bash
make verify
```
