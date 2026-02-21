# Dashboard ISO Rework — Sprint 0 (Bootstrap)

Source plan: `docs/llm-skills/dashboard-iso-skill.md`

## Objectives (Week 1)

- Connect execution to a concrete tracker in-repo.
- Inventory local dashboard contracts and key routes.
- Freeze known temporary hacks and mark deprecation path.
- Add reproducible commands for contract/e2e/visual checks.

## Deliverables

- [x] `docs/dashboard-iso/contract-inventory.md` (generated)
- [x] `scripts/dashboard/inventory.sh`
- [x] Make targets:
  - `make test-dashboard-contracts`
  - `make test-dashboard-e2e`
  - `make test-dashboard-visual`
  - `make dashboard-iso-inventory`
- [x] Technical debt note for backend-injected DOM patching

## Next (Sprint 1)

1. Define `schema_version` for dashboard payloads (`/__encore/status` first).
2. Add golden fixture tests for dashboard status payload.
3. Move NATS column rendering from backend script injection to frontend source.
4. Add first Playwright e2e suite for Service Catalog (local only).

## Acceptance checks for Sprint 0

- Commands run locally and are discoverable from `make help`.
- Contract inventory is generated from source, not hand-maintained.
- No behavior change introduced to runtime/dashboard paths.
