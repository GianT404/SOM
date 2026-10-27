# Graphify

SOM includes a project-scoped Graphify skill for Codex. The checked-in
`graphify-out/graph.json` is a local, AST-derived map of the codebase.
Code extraction runs locally and does not need an API key.

## Install

Install Python 3.10+ and `uv`, then install the official Graphify CLI:

```sh
uv tool install graphifyy
```

The PyPI package is `graphifyy`; its command is `graphify`. The Codex skill and
project instructions are in `.codex/skills/graphify/` and `AGENTS.md`.

## Use

From the repository root, ask Codex a codebase question or invoke `$graphify`.
To query the graph directly:

```sh
graphify query "How does local track rename update storage and the TUI?"
graphify path "RenameDoneMsg" "savePinned"
graphify explain "RenameDoneMsg"
```

After code changes, refresh the graph with:

```sh
graphify update .
```

`graphify-out/graph.json` is versioned so project members share the same graph.
The extraction cache and temporary metadata remain local and are ignored by Git.
