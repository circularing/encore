# SKILL: Dashboard ISO (Non-Cloud) - Full Delivery Plan

## 1) Mission

Build a local dashboard that is ISO-equivalent to Encore Dev Dashboard:
- same local features (excluding Cloud Dashboard),
- same perceived design (layout, typography, color, interactions),
- same runtime behavior (latency, live updates, navigation),
- same delivery quality bar (tests, performance, accessibility, observability).

## 2) Scope

### In Scope
- Service Catalog
- API Explorer + local "Call API"
- Infra
- Flow / traces
- DB Explorer
- Snippets
- Navigation, search, filters, UI interactions
- Local real-time WebSocket support
- Local GraphQL proxy required by the dashboard

### Out of Scope
- Everything that depends on Encore Cloud (cloud auth, org, cloud envs, cloud dashboard).

## 3) Non-Negotiable Constraints

- No fragile production DOM patching (no global "best effort" MutationObserver hacks).
- Frontend must be versioned locally in the repo (or official submodule), not only a bundled opaque snapshot.
- Stable, versioned data contract between daemon backend and dashboard frontend.
- Measurable visual parity (not "close enough"): screenshot diff + threshold.
- Performance target: no perceptible lag on Service Catalog.
- Target UI stack: `shadcn/ui` + accessible primitives, with centralized theming/tokens.

## 3.b) Mandatory UI Standard: shadcn/ui

For any new ISO dashboard implementation:
- use `shadcn/ui` as the base component system;
- keep a single token system (colors, typography, spacing, radius, shadows);
- avoid non-justified inline styles;
- centralize variants in component utilities;
- enforce native accessibility defaults (focus states, keyboard, contrast).

Minimum components to use:
- table/data table for Service Catalog and API Explorer;
- sidebar/tabs/navigation-menu for primary navigation;
- dialog/sheet/popover/tooltip for secondary interactions;
- skeleton/spinner/empty for loading/empty states;
- badge/alert/toast for status and errors.

## 4) ISO Definition

A screen is ISO if:
1. Functional: equivalent user story, same outcomes, same useful errors.
2. Visual: equivalent structure, spacing, typography, colors, hover/focus/disabled states.
3. Dynamic: same critical data appearance order, no parasitic pop-in.
4. Robustness: no JS crashes, no render loops, no obvious memory leaks.

## 5) Team Organization (Simplified RACI)

- **Dashboard Tech Lead** (A): architecture, trade-offs, final DoD sign-off.
- **Frontend Lead** (R): UI parity, components, routing, design tokens.
- **Daemon Backend Lead** (R): dashboard APIs, ws events, endpoint performance.
- **DX/Tooling Owner** (R): local build, Make targets, reproducible dev setup.
- **QA Lead** (R): test strategy, e2e, visual regression, perf budgets.
- **Product/UX** (C): screen prioritization and acceptance.
- **Security/Legal** (C): asset/design usage review.

R = Responsible, A = Accountable, C = Consulted.

## 6) Workstreams

### WS1 - Frontend Source of Truth
- Identify the official dashboard frontend repository.
- Integrate it as submodule or monorepo workspace.
- Standardize build (`pnpm`/`npm`) and `dist` artifacts.
- Pin a version (tag/commit SHA).

### WS2 - Data Contracts
- Inventory all payloads used by all pages.
- Add contract versioning (`schema_version`) on daemon side.
- Add compatibility tests (golden JSON fixtures).

### WS3 - Local Dashboard Backend
- Stabilize internal endpoints (`/__encore/*`, ws, graphql proxy).
- Remove presentation hacks.
- Add dashboard endpoint latency metrics.

### WS4 - UI Parity
- Start with Service Catalog (highest visibility/sensitivity).
- Continue with API Explorer, Infra, Flow, DB Explorer, Snippets.
- Tokenize styles to guarantee rendering consistency.
- Map each screen to explicit `shadcn/ui` components (no ad hoc UI).

### WS5 - Quality
- Unit tests (backend + frontend).
- Playwright E2E for critical flows.
- Visual regression (pixel diff on target screens).
- Performance profiling (CPU, long tasks, render counts).

### WS6 - Release & Operations
- Feature flags per page.
- Local canary with internal beta users.
- Rollback runbook.

## 7) Proposed Timeline (8 Weeks)

### Sprint 0 (Week 1) - Alignment/Bootstrap
- Frontend repo connected.
- Full feature and contract mapping completed.
- DoD and performance budgets frozen.

### Sprint 1 (Week 2-3) - Foundations
- Local frontend build + static serving through daemon.
- Versioned contracts + golden tests.
- Stable ws pipeline.

### Sprint 2 (Week 4-5) - Core Parity
- Service Catalog ISO.
- API Explorer ISO (non-cloud).
- Local Call API for HTTP + NATS.

### Sprint 3 (Week 6-7) - Advanced Parity
- Infra + Flow + DB Explorer + Snippets ISO.
- Full visual regression baseline.

### Sprint 4 (Week 8) - Hardening
- Final performance tuning.
- Full regression campaign.
- Documentation + runbook + go/no-go.

## 8) Prioritized Technical Backlog

P0:
- Integrate official dashboard frontend source.
- Remove runtime JS injection patching.
- Expose NATS as native backend contract data.
- Stabilize initial render (no pop-in).

P1:
- Service Catalog UI parity (layout + style + interactions).
- Critical E2E scenarios (navigation, call API, errors, reload).
- Visual baseline tests + threshold.
- `shadcn/ui` component coverage on critical screens >= 90%.

P2:
- Fine-grained render optimization.
- Accessibility hardening (focus, contrast, keyboard).
- Dashboard performance telemetry.

## 9) Acceptance Criteria (DoD)

A screen is done if:
- 0 JS console errors on nominal flow.
- 0 daemon/backend crashes on nominal flow.
- E2E suite is green.
- Visual diff <= agreed threshold (e.g., <= 0.5% changed pixels outside dynamic zones).
- Interaction latency acceptable for local target (no visible lag on key actions).

Project is done if:
- all scoped "ISO non-cloud" screens validated,
- operations and rollback docs delivered,
- team handover completed.

## 10) Detailed QA Plan

### Automated Tests
- Backend unit tests for metadata/encoding contracts.
- Frontend unit tests for critical components.
- Contract tests with golden JSON fixtures.
- Playwright E2E:
  - open Service Catalog
  - switch service
  - open endpoint
  - Call API success + error
  - reload + state continuity

### Manual Tests
- Dark/light verification (if supported).
- Hot responsiveness check (reload, fast navigation).
- Non-regression on projects with and without NATS.
- A11y checks for critical `shadcn/ui` components (visible focus, tab order, labels).

### Performance
- Long-task budget (UI thread).
- Max render count per interaction.
- Max time to show critical columns.

## 11) Risks and Mitigations

- **Risk:** frontend source not public/available.
  - **Mitigation:** obtain official access; otherwise temporary snapshot with explicit technical debt.
- **Risk:** backend/frontend contract drift.
  - **Mitigation:** contract versioning + mandatory CI golden tests.
- **Risk:** visual regressions.
  - **Mitigation:** visual regression CI gate.
- **Risk:** lag on large projects.
  - **Mitigation:** profiling + virtualization/lazy rendering.

## 12) Project Governance

- Daily ritual: 15-minute standup (blockers, risks, KPI).
- Bi-weekly technical review (architecture + debt).
- Weekly product review (UX parity, priorities).
- Release gate: QA + Tech Lead + Product.

## 13) Artifacts to Produce

- Local dashboard architecture ADR.
- UI stack ADR (`shadcn/ui`, theming, design tokens).
- Dashboard contract spec (`schema_version` + changelog).
- Test plan + performance reports.
- Operations/rollback runbook.
- Dashboard contribution guide (frontend + daemon).

## 14) KPIs

- Crash-free sessions (%)
- JS errors / session
- Local TTI per page
- Re-render count per action
- Test pass rate (unit/e2e/visual)
- Average visual delta per screen

## 15) LLM Prompt Template (Team Execution)

Use this template to drive coding agents:

1. Sprint objective:
   - "Implement [feature] with ISO parity on [screen], non-cloud scope."
2. Constraints:
   - "No fragile DOM patching, versioned contracts, mandatory tests."
3. Tasks:
   - "Update files X/Y, add tests A/B, update doc C."
4. Validation:
   - "Provide evidence: tests, perf metrics, before/after captures."
5. Output:
   - "Concise PR, completed DoD checklist, remaining risks."

## 16) Team Standard Commands

```bash
make build-encore
make install-complete
encore run --port 4000
```

Then add:
- `make test-dashboard-contracts`
- `make test-dashboard-e2e`
- `make test-dashboard-visual`

## 17) Immediate Recommended Decision

1. Confirm access to the dashboard frontend source repository.
2. Freeze JS injection hacks as "temporary only".
3. Start Sprint 0 with WS1 + WS2 as top priority.