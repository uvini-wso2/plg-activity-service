# PLG Activity Service

A Go HTTP service that retrieves and interprets product activity from Moesif, classifies prospect emails, and decides whether they're worth CS outreach — across multiple SaaS products, for use in the PLG (Product-Led Growth) outreach tool.

**Status:** Asgardeo and APIM are both fully built — activity retrieval, real email classification, validation/classification logic, all tested and verified against real data, **with zero placeholders remaining in the core pipeline** (as of 2026-09-30). Phase 03 (email generation) is built and tested with mocks, blocked only on real Claude API access.

---

## Quick start

```bash
go mod tidy
cp .env.example .env   # then fill in your real keys — see "Environment variables" below
go run ./cmd/server/main.go
```

Server starts on port 8081 by default.

```bash
curl "http://localhost:8081/asgardeo/validate?company_id=<a-real-company-id>&email=<a-real-email>"
```

Run the tests:
```bash
go test ./... -v
```

---

## Public API (exposed to the PLG portal)

Per team decision (2026-09-16, refined 2026-09-29), the public surface is: `/asgardeo/validate`, `/apim/validate`, and the two `/generate-email` endpoints. Raw per-product activity endpoints are internal-only (see below).

### `GET /asgardeo/validate` and `GET /apim/validate`

**UPDATED (2026-09-30):** these now call the **real classification API** internally, given just an email address — no longer accepting `domain`/`category` as separate inputs. This closes out what had been a placeholder since the classification API didn't exist yet.

**Query parameters:**

| Param | Required? | Notes |
|---|---|---|
| `company_id` | One of `company_id`/`user_id` required | Moesif company identifier |
| `user_id` | One of `company_id`/`user_id` required | Moesif user identifier. Not always a UUID. |
| `email` | **Required** | The prospect's email — the service classifies it itself now |

**Classification rules, applied in this exact order** (shared between both products via `internal/validation.Classify`):

1. **Disposable email → Excluded**, unconditionally.
2. **`provider_testing` → Excluded**, tagged `Invalid Email`.
3. **WSO2 domain (`wso2.com`) → Excluded.** Runs *before* the corporate check.
4. **Corporate → Eligible**, unconditionally.
5. **Personal → Eligible only with meaningful activity** — product-specific signal (see below).
6. **Everything else → Monitored.**

**Asgardeo's `productActivity`:** `applicationCreated`, `hasCompletedOnboarding`, `skippedStepNumber`/`skippedStepName`, `onboardingSetupType`.

**APIM's `productActivity`:** `quickStartCompleted`, `quickStartSelectedProduct`, `deploymentModel`, `context` (not yet live in the product), `apiCreated`, `attemptedSourceMethod`, `validationSource`/`validationOutcome` (never fires in real data), `selectedSource`, `gatewayActivated`, `componentDeployed`, `componentTested`, `componentPromoted`, `componentKeyGenerated`, `selfHostedApiInvokedCount` (kept for reference only — **not used for eligibility**, see below), `hasMeaningfulActivity`.

**`hasMeaningfulActivity` for APIM, redefined 2026-09-29:** based on real lifecycle signals (`apiCreated`, `gatewayActivated`, `componentDeployed`, `componentTested`, `componentPromoted`, `componentKeyGenerated`) — any one of these counts. **Explicitly does NOT use `selfHostedApiInvokedCount`** — confirmed with the team that 100% of real accounts checked use SaaS deployment, so a self-hosted-only signal would almost never trigger regardless of real engagement.

### `GET /asgardeo/generate-email` and `GET /apim/generate-email`

Run the same pipeline as their `/validate` counterparts, then feed the result into Claude to produce a personalized outreach email. Skip generation for `Excluded` prospects.

**Status: built and tested with mocks for both products, confirmed to correctly reach the real Claude API call and fail gracefully** — since there's no valid `ANTHROPIC_API_KEY` yet (blocked for interns; may need a shared team key or a Choreo GenAI service connection instead).

---

## Internal/debug endpoints

`GET /asgardeo/events` and `GET /apim/events` return raw per-product activity directly, with no validation logic. **Not meant to be publicly exposed.**

**Gated behind an environment variable, off by default:**

ENABLE_RAW_ACTIVITY_ROUTES=true


---

## The real classification API — confirmed shape and a real gotcha found

**Endpoint (staging):** `POST https://apis-stg.wso2.com/llkq/plg-email-classifier/v1.0/classify-email`

**Confirmed real response shape** (verified twice — via Postman and `curl` — 2026-09-29/30):
```json
{
  "email": "jane@acme.com",
  "domain": "acme.com",
  "category": "corporate",
  "rating": 97,
  "confidence": 0.92,
  "source": "llm",
  "signals": {
    "syntaxValid": true,
    "mxValid": true,
    "disposable": false,
    "freeProvider": false,
    "roleBased": false
  },
  "reasoning": "ACME is a generic company domain name with valid MX records..."
}
```

**Important gotcha, caught before it caused a bug:** the `signals` object uses **camelCase** field names (`syntaxValid`, `mxValid`, `freeProvider`, `roleBased`) — not the snake_case originally assumed from the team's early example responses (`syntax_valid`, etc.). Our `internal/classification` package is built against the confirmed real shape.

**Authentication is still unresolved for real production use.** The only credential tested so far is a `Test-Key` header carrying a JWT explicitly marked `"keytype": "SANDBOX"`, expiring ~10 minutes after issue — clearly a short-lived testing credential, not meant for real, ongoing use. The real production authentication method (header name, credential type, how to obtain it) is still unknown. `Config.AuthHeader`/`AuthValue` are deliberately configurable (not hardcoded to `"Test-Key"`) so switching to the real method is a config change, not a code change.

---

## Design decisions worth knowing

### Why APIM's validation logic reuses Asgardeo's structure

Confirmed directly with the team (2026-09-22): the classification rule structure is shared across products; only the meaningful-activity signal differs.

### Why `isWSO2User` was removed (2026-09-24)

APIM had a direct flag Asgardeo doesn't. Removed for consistency — all products now use the same domain-only WSO2 check.

### Why APIM prefers `company_id` over `user_id`

Per a live, controlled team investigation (2026-09-22): most detailed APIM events are tagged with `company_id`, some lack `user_id` entirely.

**Known accepted limitation:** one real case (`aloyayribedding`, `d2cc7cc8-62ab-4d96-91f8-a2bbada1e988`) proved the reverse can also happen for specific events. Team confirmed `company_id` as the standing default regardless — an accepted tradeoff.

### Why `selfHostedApiInvokedCount` was removed from eligibility (2026-09-29)

See the callout above — confirmed via testing 5 real accounts and 50 real deployment-choice events that ~100% of real APIM usage is SaaS, not self-hosted, making a self-hosted-only signal nearly useless for real decisions. Verified the fix by re-testing the two accounts (`jetbrains`, `unlimitechstore`) that had proven the original gap — both correctly flip to Eligible now.

---

## Confirmed real findings (data-quality investigations)

### The `eventsFound` metric was seriously misleading (RESOLVED)

Real accounts showed thousands of raw events, but almost none were genuine product activity — duplicate events, bot/monitoring traffic, and marketing-tracking pings were all being counted as real Moesif events.

### `company_id`-only lookups can miss real activity (KNOWN, ACCEPTED)

Some real events for some accounts aren't tagged with `company_id` at all. Team decided to accept this tradeoff.

### `QuickStart-Validation` has never fired in real data

Searched with exact match and broad wildcards across a full year of data — zero occurrences. `validationSource`/`validationOutcome` remain built correctly but unverifiable until this flow is actually used.

### The `context` field is a planned feature, not yet shipped

Confirmed directly with the team (2026-09-24) — doesn't exist in production yet.

### Real Moesif pagination cap

`Search()` fetches up to 1000 events per query, most-recent-first. Older lifecycle events can fall outside this window for very high-volume accounts — confirmed happening in practice.

### Classification API's real `signals` shape is camelCase, not snake_case

See "The real classification API" section above.

---

## Confirmed real data details

### Asgardeo
- No authentication/login tracking at all, confirmed across all 4 environments.
- `session_token` is literally the request's IP address.
- Real `action_name` values: `organization_created`, `user_created`, `organization_subscribed`, `Onboarding-Step-Completed`, `Onboarding-Started`, `Onboarding-Skipped`, `Onboarding-Completed`, `Onboarding-Step-Back`.

### APIM (api-platform / "Bijira")
- Real product URLs are `console.bijira.dev`.
- Company name lives at `company.metadata.name`, not `account_name`.
- Does have genuine auth tracking (`Landing-SignIn-Succeeded`).
- Confirmed real `action_name` values: `Landing-SignIn-Succeeded`, `QuickStart-Skipped`, `QuickStart-Selected-Product`, `Component-Created-Start`/`-End`, `QuickStart-Attempted-Source`, `QuickStart-Validation` (never fires), `QuickStart-Selected-Source`, `Gateway-Activated`, `Component-Deployed`, `Component-Tested`, `Component-Promoted`, `Component-Generated-Key`, `API-Invoked` (self-hosted only, no longer used for eligibility).
- Real deployment split confirmed: 100% of 50 real deployment-choice events checked were `"saas"`; zero were `"gateway"`.
- Confirmed real test accounts: `shopwaveorg`, `jetbrains`, `unlimitechstore`, `aloyayribedding`.

---

## Architecture

cmd/server/main.go — HTTP server, route registration (public vs. gated)

internal/moesif/ — Asgardeo integration
internal/apim/ — APIM integration (separate package: real data shape differs)
internal/classification/ — Real email classification API client (2026-09-30)
internal/validation/ — Shared classification rule logic (Classify, Outcome, Tag types,
IsWSO2Domain), used by BOTH products
internal/email/ — Phase 03: email generation (built, blocked on Claude API access)
internal/handler/ — HTTP handlers tying everything together


**Data flow (`/asgardeo/validate` or `/apim/validate`):** `email` → real classification API call → `Search()` (Moesif) → `Normalize()` → `Classify()` (shared package, using the real classification result) → JSON response. **No manually-supplied classification data required anymore.**

---

## Choreo deployment

**Status (2026-09-30):** access granted to WSO2's org in Choreo. Go is natively supported as a build preset — no Docker rework needed. Secrets move from `.env` into Choreo's own encrypted vault, added via the console post-deployment. A `.choreo/component.yaml` declaring the port is still needed. Which specific Choreo project to deploy into is still being confirmed with the team.

---

## Testing notes

~80 tests across all packages, all runnable without any real API key — mocks satisfy every external dependency (Moesif, the classification API, Claude).

---

## Environment variables (`.env`)

| Var | Purpose |
|---|---|
| `ASGARDEO_MOESIF_API_KEY` | Asgardeo's Moesif key |
| `APIM_MOESIF_API_KEY` | APIM's separate Moesif key |
| `MOESIF_BASE_URL` | `https://api.moesif.com` |
| `CLASSIFICATION_API_BASE_URL` | e.g. `https://apis-stg.wso2.com/llkq/plg-email-classifier/v1.0` |
| `CLASSIFICATION_API_AUTH_HEADER` | e.g. `Test-Key` — **sandbox only, real production header unconfirmed** |
| `CLASSIFICATION_API_AUTH_VALUE` | The actual credential value — sandbox tokens expire in ~10 minutes |
| `ANTHROPIC_API_KEY` | Claude API key — pending access |
| `PORT` | Local server port (default `8081`) |
| `ENABLE_RAW_ACTIVITY_ROUTES` | `true` locally to enable `/asgardeo/events`/`/apim/events`. Omit on real deployments. |

**Never commit `.env`.**