# ADR 0002: Layered Model Catalog Strategy

- Status: Accepted
- Date: 2026-04-09

## Context

Model metadata changes frequently and provider catalogs evolve quickly. Consumers need fast startup and resilient operation even when remote catalog fetches fail.

## Decision

Eyrie uses a layered catalog strategy:

- Embedded provider catalogs as deterministic defaults.
- Optional on-disk cache for fast and offline startup.
- Remote base catalog fetch for updates.
- Optional provider-specific live enrichment (OpenRouter `/models`) when key is configured.

The merge order preserves safety and freshness: embedded baseline + remote updates + provider live overrides.

## Consequences

- Reliable behavior with no network dependency at startup.
- Dynamic providers can expose real-time model lists.
- Consumers can refresh without blocking core startup path.
