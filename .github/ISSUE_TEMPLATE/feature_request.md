---
name: Feature request
about: Suggest a new insight, monitoring capability, or improvement
title: ''
labels: enhancement
assignees: ''
---

## What problem does this solve

Describe the situation where you'd want this — what would you have seen or done differently with it?

## Proposed approach (optional)

If you have an idea of how this should work, describe it here. If it involves a new Insights category, it helps to note which `pg_stat_*` view(s) the data would come from.

## Is this something you'd be interested in implementing yourself?

- [ ] Yes, I'd like to submit a PR for this
- [ ] No, just suggesting it

## A reminder before suggesting an executing/mutating feature

pgscope is deliberately read-only — see the root README's "What pgscope is, and is not" section. Feature requests that involve pgscope executing anything beyond a `SELECT` against a statistics view (killing sessions, applying suggested indexes, changing settings) will be declined regardless of how useful they sound, this is a hard boundary, not a preference.