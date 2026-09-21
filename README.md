Set this in your local `.env` to enable them; omit it (or set it to anything else) on any real deployment.

**Important caveat:** `/apim/events` is currently the **only** way to retrieve APIM data at all — there is no APIM equivalent of `/validate` yet. Gating this off removes real functionality until APIM's own validation logic is built (see "Known limitations" below).

### `GET /events` (Asgardeo)

| Param | Required? |
|---|---|
| `company_id` | One of `company_id`/`user_id` required |
| `user_id` | One of `company_id`/`user_id` required |

**Response fields:**

| Field | Meaning |
|---|---|
| `organizationName` | Best-effort, from `company.metadata.account_name` on whichever event carries it |
| `firstSeen` / `lastActivity` | Human-readable, always shown in **Sri Lankan time** (`Asia/Colombo`), regardless of where the prospect actually is — per team decision (2026-09-16), since the CS team operating this is based in Sri Lanka. Format: `"September 14, 2026 2:30 PM"`. |
| `timezone` / `countryName` | The *prospect's own* location, from `request.geo_ip` on the most recent event — separate from the display-time convention above |
| `eventsFound` | Moesif's true total matching event count. `0` means no data exists for this identifier at all. |

**`productActivity` (Asgardeo-specific):**

| Field | Meaning |
|---|---|
| `applicationCreated` | `true` if at least one onboarding step was completed |
| `hasCompletedOnboarding` | `true` only on a genuine `Onboarding-Completed` event — the *entire* wizard finished, not just one step. Distinct from `applicationCreated`. |
| `skippedStepNumber` | Step number of the most recent skip, if any. `null` means never skipped — a pointer, since step `0` is a real, valid step and must not be confused with "no skip." |
| `skippedStepName` | Human-readable name matching `skippedStepNumber`. Omitted entirely if no skip occurred. |
| `onboardingSetupType` | From `metadata.wizard_path` — confirmed real values `"full_setup"` and `"preview"`. |

### `GET /apim/events` (APIM / api-platform, "Bijira")

Same `company_id`/`user_id` parameters as above.

**Response fields:** same parent shape as Asgardeo (`organizationName`, `firstSeen`, `lastActivity`, `timezone`, `countryName`, `eventsFound`), plus:

| Field | Meaning |
|---|---|
| `isWSO2User` | Direct boolean signal from `user.metadata.isWSO2User` — confirms whether this is an internal WSO2 account. Unlike Asgardeo, APIM provides this directly rather than requiring a domain check. |

**`productActivity` (APIM-specific):**

| Field | Meaning |
|---|---|
| `signedIn` | `true` on a real `Landing-SignIn-Succeeded` event — confirmed real auth tracking, which Asgardeo does **not** have |
| `projectCreated` | `true` on `Project-Created-Start` |
| `componentCreated` | `true` on `Component-Created-Start` |
| `quickStartSkipped` | `true` on `QuickStart-Skipped` |
| `hasMeaningfulActivity` | `true` if `eventsFound >= 400`. Confirmed threshold, team decision (2026-09-18, via Slack). **Known open issue:** this currently counts every event Moesif returns, including non-product tracking/telemetry noise (ad pixels, session-recording pings) — not filtered to genuine API usage. Flagged to the team; not yet resolved. |

---

## Confirmed real data details

### Asgardeo

- **Endpoint:** `POST https://api.moesif.com/search/~/search/events`, `Authorization: Bearer <key>`
- Response nested under `hits.hits`/`hits.total` (Elasticsearch-style), not flat
- The meaningful field is `action_name`, not `event_type` (always `"user_action"`)
- **No authentication/login tracking at all** — confirmed across all four environments (Prod/Dev/Staging/Test)
- `session_token` is literally the request's IP address, not a real session ID — no reliable per-session duration is possible from this data (this is also why there's no "average time per day" metric — it was removed by team decision, 2026-09-16)
- Confirmed real `action_name` values include: `organization_created`, `user_created`, `organization_subscribed`, `Onboarding-Step-Completed`, `Onboarding-Started`, `Onboarding-Skipped`, `Onboarding-Completed`, `Onboarding-Step-Back`, plus several marketing/association events not currently used for classification

### APIM (api-platform / "Bijira")

- Same Moesif API mechanics as Asgardeo (same envelope, same pagination), but its own separate credentials and Moesif app
- **Real product URLs are `console.bijira.dev`** — worth confirming with the team whether "Bijira" is the actual internal product name and "APIM" is shorthand
- Company name lives at `company.metadata.name` — **different field name** than Asgardeo's `account_name`
- A large share of raw events are **not product activity at all** — Application Insights telemetry, LinkedIn ad tracking pixels, Microsoft Clarity session recordings all get captured as regular Moesif events
- Confirmed real `action_name` values found so far: `Landing-SignIn-Succeeded`, `QuickStart-Skipped`, `Project-Created-Start`, `Component-Created-Start`, `QuickStart-Selected-Product` — **not yet a complete list**, worth pulling the full set from the dashboard
- Has genuine authentication tracking (`Landing-SignIn-Succeeded`) — a real capability gap versus Asgardeo, not a universal limitation of Moesif itself

---

## Architecture

cmd/server/main.go — HTTP server, route registration (public vs. gated), .env loading

internal/moesif/ — Asgardeo integration
client.go, filter.go Config/Client/Search(), paginated fetch
types.go Raw Moesif response shapes
normalize.go Raw events → Summary (org info, tenure, productActivity)

internal/apim/ — APIM integration (separate package: real data shape differs
client.go, filter.go from Asgardeo's in confirmed ways — different company-name
types.go field, direct isWSO2User flag, real auth events, no
normalize.go unified action_name coverage)

internal/validation/ — Classification logic (Asgardeo only, currently)
types.go EmailClassification, Outcome, Tag constants
classify.go The 5 ordered rules
wso2_domains.go Domain-based internal-account check

internal/email/ — Phase 03: email generation (built, blocked on API access)
types.go, client.go Generator interface + real Anthropic client
template.go Shared outreach email template
summary.go Turns a moesif.Summary into plain text for Claude

internal/handler/ — HTTP handlers tying everything together
events.go, apim_events.go Raw activity endpoints (internal-only)
validate.go Combines Search → Normalize → Classify
generate_email.go Combines the above + calls the email Generator


**Data flow (`/validate`):** query params → `moesif.Search()` → `moesif.Normalize()` → `validation.Classify()` → JSON response (decision + full activity context).

---

## Why one unified service, not separate microservices

This is meant to eventually cover 5 SaaS products, not just Asgardeo. Rather than build 5 separate services, everything lives in one Go service with a consistent parent response shape (org name, tenure, location, event count) across all products, while each product gets its own package for product-specific logic and credentials. Adding a new product means adding a new package (as done for APIM), not rebuilding anything shared.

---

## Testing notes

- 47 tests across all packages, all runnable without any real API key or network access — mocks satisfy each handler's client interface; `client_test.go` files use local `httptest.Server` instances simulating real Moesif response shapes.
- Test data is built from confirmed real shapes, not guesses — including edge cases like onboarding step `0` being valid (not the same as "never skipped"), and geo/wizard-path/action-name data being taken from the correct event in a set.

---

## Environment variables (`.env`)

| Var | Purpose |
|---|---|
| `ASGARDEO_MOESIF_API_KEY` | Asgardeo's Moesif Management API key |
| `APIM_MOESIF_API_KEY` | APIM's separate Moesif Management API key |
| `MOESIF_BASE_URL` | `https://api.moesif.com` (shared across products) |
| `ANTHROPIC_API_KEY` | Claude API key — pending access approval as of this writing |
| `PORT` | Local server port (default `8081`) |
| `ENABLE_RAW_ACTIVITY_ROUTES` | Set to `true` locally to enable `/events` and `/apim/events`. Omit on real deployments. |

**Never commit `.env`** — it's git-ignored at the repo root.