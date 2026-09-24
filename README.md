# PLG Activity Service

A Go HTTP service that retrieves and interprets product activity from Moesif, across multiple SaaS products, for use in the PLG (Product-Led Growth) outreach tool. It answers two questions for a CS engineer: *what has this account actually done in the product*, and *is this prospect worth reaching out to*.

**Status:** Asgardeo and APIM both fully built — activity retrieval, validation/classification, tested, and verified against real data. Phase 03 (email generation) is built and tested with mocks, blocked only on real Claude API access.

---

## Quick start

```bash
go mod tidy
cp .env.example .env   # then fill in your real keys — see "Environment variables" below
go run ./cmd/server/main.go
```

Server starts on port 8081 by default.

```bash
curl "http://localhost:8081/validate?company_id=<a-real-company-id>&domain=gmail.com&category=personal"
```

Run the tests:
```bash
go test ./... -v
```

---

## Public API (exposed to the PLG portal)

Per team decision (2026-09-16), only these are meant to be publicly reachable. Raw per-product activity endpoints are internal-only (see below).

### `GET /validate` — Asgardeo

Combines Asgardeo Moesif activity with an email classification result to decide whether a prospect is worth CS outreach.

**Query parameters:**

| Param | Required? | Notes |
|---|---|---|
| `company_id` | One of `company_id`/`user_id` required | Moesif company identifier |
| `user_id` | One of `company_id`/`user_id` required | Moesif user identifier. Not always a UUID. |
| `domain` | **Required** | The email's domain — used for the WSO2-domain exclusion check |
| `category` | **Required** | `corporate`, `personal`, `disposable`, or `provider_testing` |
| `email` | **Optional** | Carried through for context only — never used in any actual rule |

**Classification rules, applied in this exact order:**

1. **Disposable email → Excluded**, unconditionally.
2. **`provider_testing` → Excluded**, tagged `Invalid Email` (confirmed invalid, not treated like personal).
3. **WSO2 domain (`wso2.com`) → Excluded.** Runs *before* the corporate check.
4. **Corporate → Eligible**, unconditionally.
5. **Personal → Eligible only with meaningful activity** — created an application, or started onboarding and stopped at an identifiable step.
6. **Everything else → Monitored.**

**`productActivity` (Asgardeo-specific):**

| Field | Meaning |
|---|---|
| `applicationCreated` | At least one onboarding step completed |
| `hasCompletedOnboarding` | The *entire* onboarding wizard finished (real `Onboarding-Completed` event) — distinct from `applicationCreated` |
| `skippedStepNumber` / `skippedStepName` | The step where onboarding was abandoned. `null` means never skipped — a real, valid step 0 must not be confused with "no skip" |
| `onboardingSetupType` | `"full_setup"` or `"preview"` |

### `GET /apim/validate` — APIM (api-platform / "Bijira")

Same parameter shape as `/validate` above. Reuses the same `Outcome`/`Tag` structure and the same corporate/disposable/provider_testing/WSO2-domain rules — **only the meaningful-activity signal differs per product** (confirmed with the team, 2026-09-22). WSO2-internal detection uses the same domain-based check as Asgardeo (no product-specific flag — see "Design decisions" below).

**`productActivity` (APIM-specific), redefined 2026-09-22/24:**

| Field | Meaning |
|---|---|
| `quickStartCompleted` | `true` when NO `QuickStart-Skipped` event exists (they went through fully) |
| `quickStartSelectedProduct` | They picked a product during quick-start |
| `deploymentModel` | e.g. `"saas"` — from `QuickStart-Selected-Product` metadata. **Confirmed real (2026-09-23).** |
| `context` | A separate metadata value on the same event, alongside `deploymentModel`. **Not yet live in the product (confirmed 2026-09-24)** — this is planned but not yet shipped, which is why no real event has ever carried it. Field name is our best guess, to be confirmed once the feature ships. |
| `apiCreated` | `true` only when **both** `Component-Created-Start` AND `Component-Created-End` are present |
| `attemptedSourceMethod` | e.g. `"sample"`, `"uploaded"` — **confirmed real.** |
| `validationSource` / `validationOutcome` | From `QuickStart-Validation`. **Unverified** — this event has never fired for anyone in a full year of real data. |
| `selectedSource` | e.g. `"sample"`, `"own"` — **confirmed real.** |
| `gatewayActivated`, `componentDeployed`, `componentTested`, `componentPromoted`, `componentKeyGenerated` | Real product-building lifecycle stages — all **confirmed real** against live accounts. |
| `apiInvokedCount` | How many real `API-Invoked` events this account has (within the fetched window — see pagination note below) |
| `hasMeaningfulActivity` | `true` if `apiInvokedCount >= 400`. **Redefined 2026-09-24** — previously compared raw `eventsFound` to 400/500; now counts only genuine API calls. See "Confirmed real findings" below for why this mattered. |

### `GET /generate-email`

Runs the same pipeline as `/validate`, then feeds the result into Claude to produce a personalized outreach email. Skips generation for `Excluded` prospects.

**Status: built and tested with mocks, not yet verified against the real Claude API** — access is pending; as an intern, direct approval is unlikely, so a shared team key may be needed instead.

---

## Internal/debug endpoints

`GET /events` (Asgardeo) and `GET /apim/events` (APIM) return raw per-product activity directly, with no validation logic. **Not meant to be publicly exposed** — both `/validate` endpoints already call the same underlying functions directly and include the full activity summary in their own response.

**Gated behind an environment variable, off by default:**

ENABLE_RAW_ACTIVITY_ROUTES=true

Set locally for testing; omit on real deployments.

**Caveat:** `/apim/events` is currently the only way to retrieve APIM data with no validation applied at all.

---

## Design decisions worth knowing

### Why APIM's validation logic reuses Asgardeo's structure

Confirmed directly with the team (2026-09-22): *"the classification logic is the same for all products."* Only the meaningful-activity signal is product-specific — everything else (`internal/validation.Classify`'s rule types, tags, outcomes) is shared.

### Why `isWSO2User` was removed (2026-09-24)

APIM's raw data has a direct `isWSO2User` flag that Asgardeo doesn't. It was initially used as an *extra* signal alongside domain-checking. Per team decision, this was removed — **all products now use the same domain-only check** (`internal/validation.IsWSO2Domain`), for consistency, even though APIM's direct flag would have been more reliable in theory.

### Why APIM prefers `company_id` over `user_id`

Per a live, controlled investigation by the team (2026-09-22): most of APIM's detailed console-driven events (component created/deployed/tested, quick-start funnel steps) are tagged with `company_id`, and some don't carry a `user_id` at all — so `company_id` is the more complete signal generally.

**Known accepted limitation (2026-09-24):** a real case (`aloyayribedding`, company `d2cc7cc8-62ab-4d96-91f8-a2bbada1e988`) proved some specific events for some accounts carry no `company_id` at all, so `company_id`-only lookups can occasionally miss real activity that `user_id` alone would catch. The team confirmed `company_id` should be used as the standing decision regardless — this is an accepted tradeoff, not an open question.

---

## Confirmed real findings (data-quality investigations)

These were discovered through direct, repeated investigation against real Moesif data — recorded here so the reasoning isn't lost.

### The `eventsFound` metric was seriously misleading (RESOLVED for APIM)

Two real accounts (`apimsaas`, `shopwaveorg`) showed `eventsFound` in the thousands (7988 and 4375 respectively) — but their real `apiInvokedCount` was **8 and 0** respectively. Nearly all of their raw event volume was tracking noise, duplicate events, or bot/monitoring traffic, not genuine product usage. This is why `hasMeaningfulActivity` was redefined to count `API-Invoked` events specifically (2026-09-24) rather than raw event totals.

Confirmed contributing noise sources:
- **Duplicate events**: a single page load can fire two separate logged events at the identical timestamp (e.g. `Portal-Viewed-Home` and `home-page-visit`).
- **Bot/monitoring traffic**: repeated hits from the same datacenter IPs using a `moesif-nodejs` client (not a real browser), plus genuine third-party uptime-monitoring traffic (Site24x7) hitting a test account.
- **Marketing/analytics tracking**: Application Insights telemetry, ad-tracking pixels, Google Analytics collection endpoints (`/g/collect`, `/pixel/collect`) all get logged as regular Moesif events.

### `company_id`-only lookups can miss real activity (KNOWN, ACCEPTED)

See "Design decisions" above — some real events for some accounts simply aren't tagged with `company_id` at all. Team decided to accept this tradeoff and use `company_id` as the standing default anyway.

### `QuickStart-Validation` has never fired in real data

Searched with exact match and broad wildcards, across a full year of data — zero occurrences anywhere in the dataset. `validationSource`/`validationOutcome` remain built exactly per the team's description, but genuinely unverifiable until someone triggers that specific "bring your own API" flow for real.

### The `context` field is a planned feature, not yet shipped

Confirmed directly with the team (2026-09-24) — this metadata value doesn't exist in production yet, which is why it was never found in any real event. Built ahead of time per her description; will need re-confirming against real data once the feature actually ships.

### Real Moesif pagination cap

`Search()` fetches up to 1000 events per query (10 pages × 100), sorted most-recent-first. For very high-volume accounts, older lifecycle events (e.g. an early `Component-Promoted`) can fall outside this window and simply not be counted — not a bug, a known tradeoff, same as Moesif's own guidance for interactive search vs. bulk export.

---

## Confirmed real data details

### Asgardeo

- No authentication/login tracking at all, confirmed across all 4 environments.
- `session_token` is literally the request's IP address, not a real session ID.
- Real `action_name` values: `organization_created`, `user_created`, `organization_subscribed`, `Onboarding-Step-Completed`, `Onboarding-Started`, `Onboarding-Skipped`, `Onboarding-Completed`, `Onboarding-Step-Back`.

### APIM (api-platform / "Bijira")

- Real product URLs are `console.bijira.dev` — worth confirming "Bijira" is the actual internal name.
- Company name lives at `company.metadata.name`, not `account_name` like Asgardeo.
- **Does** have genuine auth tracking (`Landing-SignIn-Succeeded`) — a real capability Asgardeo lacks.
- Confirmed real `action_name` values: `Landing-SignIn-Succeeded`, `QuickStart-Skipped`, `QuickStart-Selected-Product`, `Component-Created-Start`/`-End`, `QuickStart-Attempted-Source`, `QuickStart-Validation` (real name, never fires), `QuickStart-Selected-Source`, `Gateway-Activated`, `Component-Deployed`, `Component-Tested`, `Component-Promoted`, `Component-Generated-Key`, `API-Invoked`.
- Confirmed real test accounts, useful for future verification: `shopwaveorg`, `apimsaas`, `aloyayribedding`, `unlimitechstore`, `jetbrains`.

---

## Architecture

cmd/server/main.go — HTTP server, route registration (public vs. gated)

internal/moesif/ — Asgardeo integration
internal/apim/ — APIM integration (separate package: real data shape
differs from Asgardeo in confirmed, structural ways)
internal/validation/ — Shared classification logic (Classify, Outcome, Tag types,
IsWSO2Domain), used by BOTH products
internal/email/ — Phase 03: email generation (built, blocked on API access)
internal/handler/ — HTTP handlers tying everything together


**Data flow (`/validate` or `/apim/validate`):** query params → `Search()` → `Normalize()` → `Classify()` (shared package) → JSON response.

---

## Testing notes

~70 tests across all packages, all runnable without any real API key. Test data reflects confirmed real shapes, including deliberately tricky edge cases (step 0 vs. never-skipped, WSO2-domain-beats-corporate ordering, `apiCreated` requiring both Start+End events, noise-immune threshold counting).

---

## Environment variables (`.env`)

| Var | Purpose |
|---|---|
| `ASGARDEO_MOESIF_API_KEY` | Asgardeo's Moesif key |
| `APIM_MOESIF_API_KEY` | APIM's separate Moesif key |
| `MOESIF_BASE_URL` | `https://api.moesif.com` |
| `ANTHROPIC_API_KEY` | Claude API key — pending access |
| `PORT` | Local server port (default `8081`) |
| `ENABLE_RAW_ACTIVITY_ROUTES` | `true` locally to enable `/events`/`/apim/events`. Omit on real deployments. |

**Never commit `.env`.**