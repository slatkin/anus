# Domain Docs

How the engineering skills should consume this repo's domain documentation when exploring the codebase.

## Code exploration

Use Serena for code exploration and standard Serena workflows when navigating, searching, and understanding the codebase. Prefer Serena over ad hoc file scanning when the task is exploratory rather than an immediate targeted edit.

## Before exploring, read these

- **`CONTEXT.md`** at the repo root, or
- **`CONTEXT-MAP.md`** at the repo root if it exists — it points at one `CONTEXT.md` per context. Read each one relevant to the topic.
- **`docs/adr/`** — read ADRs that touch the area you're about to work in. In multi-context repos, also check `src/<context>/docs/adr/` for context-scoped decisions.

If any of these files don't exist, proceed silently.

## File structure

This repo is configured as a single-context repo:
- root `CONTEXT.md`
- root `docs/adr/`

## Use the glossary's vocabulary

When naming domain concepts, prefer the terms defined in `CONTEXT.md`.

## Flag ADR conflicts

If an output contradicts an existing ADR, surface it explicitly rather than silently overriding.
