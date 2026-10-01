# PLG Activity Service

A Go HTTP service that retrieves and interprets product activity from Moesif, classifies prospect emails, and decides whether they're worth CS outreach — across multiple SaaS products, for use in the PLG (Product-Led Growth) outreach tool.

**Status (2026-10-01):** Asgardeo and APIM are both fully built — activity retrieval, real email classification, validation/classification logic, all tested and verified against real data. **Both products now support company_id-only lookups** — no email required from the caller at all, matching real PLG-insights usage (confirmed with the team, 2026-09-30/10-01). Phase 03 (email generation) is built and tested with mocks for both products, blocked only on real Claude API access.

---

## Quick start

```bash
go mod tidy
cp .env.example .env   # then fill in your real keys — see "Environment variables" below
go run ./cmd/server/main.go
```

Server starts on port 8081 by default.

```bash
curl "http://localhost:8081/asgardeo/validate?company_id=<a-real-company-id>"
```

Run the tests:
```bash
go test ./... -v
```

---

## Public API (exposed to the PLG portal)

Per team decision (2026-09-16, refined 2026-09-29), the public surface is: `/asgardeo/validate`, `/apim/validate`, `/asgardeo/generate-email`, `/apim/generate-email`. Raw per-product activity endpoints are internal-only (see below).

### `GET /asgardeo/validate` and `GET /apim/validate`

**Query parameters:**

| Param | Required? | Notes |
|---|---|---|
| `company_id` | One of `company_id`/`user_id` required | Moesif company identifier |
| `user_id` | One of `company_id`/`user_id` required | Moesif user identifier. Not always a UUID. |
| `email` | **Optional** | If omitted, the service discovers a real email on its own — see below. Real PLG-insights usage gives only `company_id`, confirmed with the team. |

**How email is discovered when not given (2026-09-30/10-01):**

- **Asgardeo:** uses `company.metadata.account_owner_email` — a real, dedicated field confirmed directly on Moesif's company data. Verified against companies with multiple real users tied to them (e.g. `wayfinderenterprise`, which has 2-3 different user emails floating around in its raw events) — confirmed this field correctly identifies the true account owner, not just any collaborator who happened to interact with the account.
- **APIM:** has **no equivalent field** — checked the real company record directly via Moesif's Companies API (`GET /v1/search/~/companies/{id}`) and confirmed nothing resembling an "owner" field exists anywhere in it. Per team decision, falls back to the email tied to the company's chronologically **earliest recorded event** instead. This is a reasonable heuristic, not a confirmed authoritative field like Asgardeo's.

If neither an explicit `email` nor a discoverable one exists, returns a `400` explaining why.

**Classification rules, applied in this exact order** (shared between both products via `internal/validation.Classify`):

1. **Disposable email → Excluded**, unconditionally — even for accounts with strong, confirmed real engagement (verified: `jetbrains`, despite `apiCreated`/`gatewayActivated`/`componentDeployed`/`componentPromoted` all `true`, still correctly Excluded when given a disposable email).
2. **`provider_testing` → Excluded**, tagged `Invalid Email`.
3. **WSO2 domain (`wso2.com`) → Excluded.** Runs *before* the corporate check.
4. **Corporate → Eligible**, unconditionally.
5. **Personal → Eligible only with meaningful activity** — product-specific signal (see below).
6. **Everything else → Monitored.**

**Asgardeo's `productActivity`:**

| Field | Meaning |
|---|---|
| `applicationCreated` | At least one onboarding step completed |
| `hasCompletedOnboarding` | The *entire* wizard finished — distinct from `applicationCreated` |
| `skippedStepName` | Name of the step onboarding was abandoned at, if any. **`skippedStepNumber` removed from the public response (2026-10-01, per team feedback) — kept internally only, since `Classify()` still needs the nil-vs-zero distinction to detect a genuine skip.** |
| `onboardingSetupType` | `"full_setup"` or `"preview"` |

**APIM's `productActivity`:** `quickStartCompleted`, `quickStartSelectedProduct`, `deploymentModel`, `context` (not yet live in the product), `apiCreated`, `attemptedSourceMethod`, `validationSource`/`validationOutcome` (never fires in real data), `selectedSource`, `gatewayActivated`, `componentDeployed`, `componentTested`, `componentPromoted`, `componentKeyGenerated`, `selfHostedApiInvokedCount` (kept for reference only — **not used for eligibility**), `hasMeaningfulActivity`.

**`hasMeaningfulActivity` for APIM (redefined 2026-09-29):** based on real lifecycle signals (`apiCreated`, `gatewayActivated`, `componentDeployed`, `componentTested`, `componentPromoted`, `componentKeyGenerated`) — any one counts. **Explicitly does NOT use `selfHostedApiInvokedCount`** — confirmed 100% of real accounts checked use SaaS deployment, so a self-hosted-only signal would almost never trigger regardless of real engagement.

### `GET /asgardeo/generate-email` and `GET /apim/generate-email`

Run the same pipeline as their `/validate` counterparts, then feed the result into Claude to produce a personalized outreach email. Skip generation for `Excluded` prospects.

**Status: built and tested with mocks for both products, confirmed to correctly reach the real Claude API call and fail gracefully** — blocked on `ANTHROPIC_API_KEY`, which is likely unavailable to interns directly; may need a shared team key or a Choreo GenAI service connection instead.

---

## Internal/debug endpoints

`GET /asgardeo/events` and `GET /apim/events` return raw per-product activity directly, with no validation logic. **Not meant to be publicly exposed.**

**Gated behind an environment variable, off by default:**
```
ENABLE_RAW_ACTIVITY_ROUTES=true
```

---

## The real classification API — confirmed shape and a real gotcha found

**Endpoint (staging):** `POST https://apis-stg.wso2.com/llkq/plg-email-classifier/v1.0/classify-email`

**Confirmed real response shape** (verified via Postman and `curl`, 2026-09-29/30):
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

**Important gotcha, caught before it caused a bug:** `signals` uses **camelCase** field names — not the snake_case originally assumed from early team examples.

**Authentication is still unresolved for real production use.** The only credential tested is a `Test-Key` header carrying a JWT explicitly marked `"keytype": "SANDBOX"`, expiring ~10 minutes after issue. `Config.AuthHeader`/`AuthValue` are deliberately configurable so switching to the real method is a config change, not a code change.

---

## Design decisions worth knowing

### Why APIM's validation logic reuses Asgardeo's structure

Confirmed directly with the team (2026-09-22): the classification rule structure is shared across products; only the meaningful-activity signal differs.

### Why `isWSO2User` was removed (2026-09-24)

APIM had a direct flag Asgardeo doesn't. Removed for consistency — all products now use the same domain-only WSO2 check.

### Why APIM prefers `company_id` over `user_id` for Search()

Per a live, controlled team investigation (2026-09-22): most detailed APIM events are tagged with `company_id`, some lack `user_id` entirely.

**Known accepted limitation:** one real case (`aloyayribedding`, `d2cc7cc8-62ab-4d96-91f8-a2bbada1e988`) proved the reverse can also happen for specific events — its real `Component-Created` activity isn't tagged with `company_id` at all. Team confirmed `company_id` as the standing default regardless. **This surfaced again concretely during company_id-only testing (2026-09-30):** this account's `productActivity` came back entirely `false` (missing its real activity), though the final outcome still happened to be correct since the discovered email was a genuine corporate domain, which overrides the activity check anyway. A personal-email version of this same scenario would produce the wrong outcome.

### Why `selfHostedApiInvokedCount` was removed from eligibility (2026-09-29)

Confirmed via testing 5 real accounts and 50 real deployment-choice events that ~100% of real APIM usage is SaaS, not self-hosted.

### Why Asgardeo and APIM use different company_id-only strategies (2026-09-30/10-01)

See "How email is discovered" above. Asgardeo has a real, dedicated `account_owner_email` field (confirmed via Moesif's Companies API); APIM genuinely does not (also confirmed via the same API, checked directly) — so APIM uses an earliest-event heuristic instead.

---

## Confirmed real findings (data-quality investigations)

### The `eventsFound` metric was seriously misleading (RESOLVED)

Real accounts showed thousands of raw events, but almost none were genuine product activity — duplicate events, bot/monitoring traffic, and marketing-tracking pings were all counted as real Moesif events.

### `company_id`-only lookups can miss real activity (KNOWN, ACCEPTED)

See "Design decisions" above.

### `QuickStart-Validation` has never fired in real data

Searched with exact match and broad wildcards across a full year of data — zero occurrences.

### The `context` field is a planned feature, not yet shipped

Confirmed directly with the team (2026-09-24).

### Real Moesif pagination cap

`Search()` fetches up to 1000 events per query, most-recent-first. Older lifecycle events can fall outside this window for very high-volume accounts.

### Classification API's real `signals` shape is camelCase, not snake_case

See "The real classification API" section above.

### A single Asgardeo company can have multiple real users with different emails

Confirmed via `wayfinderenterprise` (2026-09-30) — found 2-3 distinct real user emails in its raw events, only one of which is the genuine account owner (per `account_owner_email`). The others turned out to be collaborators (confirmed by checking whether the company appeared in that user's `owned_companies` vs. merely `associated_companies` list).

---

## Confirmed real data details

### Asgardeo
- No authentication/login tracking at all, confirmed across all 4 environments.
- `session_token` is literally the request's IP address.
- Real `action_name` values: `organization_created`, `user_created`, `organization_subscribed`, `Onboarding-Step-Completed`, `Onboarding-Started`, `Onboarding-Skipped`, `Onboarding-Completed`, `Onboarding-Step-Back`.
- Confirmed real test companies for the owner-email fallback: `wayfinderenterprise` (multi-user, real owner `anuradhak@wso2.com`), `roadsidecoderytt` (single user, `eon55dude@gmail.com`), `orgsacma` (`pushpendarsingh5809@gmail.com`).

### APIM (api-platform / "Bijira")
- Real product URLs are `console.bijira.dev`.
- Company name lives at `company.metadata.name`, not `account_name`.
- Does have genuine auth tracking (`Landing-SignIn-Succeeded`).
- Confirmed real `action_name` values: `Landing-SignIn-Succeeded`, `QuickStart-Skipped`, `QuickStart-Selected-Product`, `Component-Created-Start`/`-End`, `QuickStart-Attempted-Source`, `QuickStart-Validation` (never fires), `QuickStart-Selected-Source`, `Gateway-Activated`, `Component-Deployed`, `Component-Tested`, `Component-Promoted`, `Component-Generated-Key`, `API-Invoked` (self-hosted only).
- Real deployment split: 100% of 50 real deployment-choice events checked were `"saas"`; zero were `"gateway"`.
- Confirmed via Moesif's Companies API (`GET /v1/search/~/companies/{id}`) that company records have no owner/primary-contact field of any kind.
- Confirmed real test accounts: `shopwaveorg`, `jetbrains`, `unlimitechstore`, `aloyayribedding`.

---

## Architecture

```
cmd/server/main.go              — HTTP server, route registration (public vs. gated)

internal/moesif/                — Asgardeo integration
internal/apim/                  — APIM integration (separate package: real data shape differs)
internal/classification/        — Real email classification API client
internal/validation/            — Shared classification rule logic (Classify, Outcome, Tag types,
                                   IsWSO2Domain), used by BOTH products
internal/email/                 — Phase 03: email generation (built, blocked on Claude API access)
internal/handler/                — HTTP handlers tying everything together
```

**Data flow (`/asgardeo/validate` or `/apim/validate`):** `company_id`/`user_id` → `Search()` (Moesif) → `Normalize()` (also discovers a fallback email if needed) → real classification API call → `Classify()` (shared package) → JSON response.

---

## Choreo deployment

**Status (2026-10-01):** access granted to WSO2's org in Choreo. Go is natively supported as a build preset — no Docker rework needed. Secrets move from `.env` into Choreo's own encrypted vault, added via the console post-deployment. A `.choreo/component.yaml` declaring the port is still needed. Which specific Choreo project to deploy into is still being confirmed with the team.

---

## Testing notes

~85 tests across all packages, all runnable without any real API key — mocks satisfy every external dependency (Moesif, the classification API, Claude).

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