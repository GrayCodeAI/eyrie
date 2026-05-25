<!--
  Thanks for your contribution! Please fill out this template so reviewers can
  understand the change quickly. Anything that does not apply can be left in
  place; do not delete unanswered sections — write "n/a".
-->

## Summary

<!--
  One paragraph describing what this PR does and why. Link the related
  issue(s) with `Fixes #N` or `Refs #N` if applicable.
-->

## Changes

<!--
  Bullet list of what changed, grouped by area (client, router, catalog,
  config, storage, conversation, sdk, CI, docs). Reviewers should be able to
  skim this and know what to look at first.
-->

-

## Provider / API impact

<!--
  Does this touch behaviour for Anthropic / OpenAI / Vertex / Azure / OpenAI-
  compatible providers? List which ones, and whether the change is wire-
  compatible with all currently-supported versions. If irrelevant, write "n/a".
-->

## Testing

<!--
  Describe how you tested. Paste output of `make test` and `make lint`. If
  you added new tests, list them. Note any flaky tests you encountered.
-->

```text
$ make test
...
$ make lint
...
```

## Checklist

- [ ] Commits follow [Conventional Commits](https://www.conventionalcommits.org/)
      (`feat:`, `fix:`, `perf:`, `refactor:`, `docs:`, `test:`, etc.)
- [ ] `make build` passes
- [ ] `make lint` passes (no new lint findings, no `nolint:…` without justification)
- [ ] `make test` passes locally with `-race` enabled
- [ ] New or changed code has tests (table-driven where appropriate)
- [ ] Public APIs have godoc comments
- [ ] `CHANGELOG.md` updated under `## [Unreleased]` if user-visible
- [ ] No regression for existing providers (Anthropic, OpenAI, Azure, Vertex,
      OpenAI-compatible)
- [ ] No secrets, tokens, or PII added to the repo
- [ ] No `Co-authored-by:` trailers (this is individual-developer work)
