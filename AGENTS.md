# AGENTS.md

Instructions for AI agents working in this repository (Cursor and similar tools).

## Feature tracking (`feature/gw-*.md`)

On **every feature request** (new capability, enhancement, refactor with user-visible/architectural impact, or hardening follow-up):

1. **Create** a new file `feature/gw-<N>.md` before or with the implementation.
2. Choose `<N>` as the next unused integer after existing `feature/gw-*.md` files (currently through `gw-3`).
3. **Include that file in the same commit(s)** as the feature work. Do not ship code without the matching `feature/gw-*.md`.
4. Do not skip this for “small” features. Only omit it for pure typo/docs-only edits that do not introduce a feature.

### Document shape

Use the same structure as existing entries:

```markdown
# GW-<N>: <short title>

## Problem

<what is wrong or missing>

## Impact

- <bullet risks / user impact>

## Shorthand solution

<concise intended approach>

## Related code

- `<paths>`
```

If the work implements an existing `feature/gw-*.md`, update that file’s status or notes in the same commit rather than inventing a duplicate number.

### Commit checklist

Before committing feature work, verify:

- [ ] `feature/gw-<N>.md` exists (new or intentionally updated)
- [ ] It is staged with the feature changes
- [ ] The commit message reflects the feature; the gw file rides along in that commit
