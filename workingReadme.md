# AI Tutor — Cross-Platform Redesign (App Store / Play Store / Web)

**Status:** Stage 1 (tutoring engine + backend API) and the core of Stage 2 (web client) are implemented and test-verified for the narrow MVP slice (see [Section 8](#8-build-stages) current-state note). This document supersedes Section 7 ("Tech Stack —
Recommendation") of the original `README.md` only. Sections 1–6 and 8–13 of
the original README (product definition, learning loop, interaction rules,
age calibration, MVP scope, state machine, testing plan, false-confidence
safeguards, child-safety/data requirements, success criteria, workflow
discipline) are **unchanged and still binding** — they are summarized below
for continuity, but the original README remains the source of truth for the
tutoring *behavior*. This document is the source of truth for *delivery
architecture*: how the same tutoring engine reaches students as an
installable iOS/Android app **and** through a browser, instead of the
original WhatsApp-first plan.

Point engineers to the specific numbered section relevant to their work
rather than the whole document (see [Section 10](#10-document-map)).

---

## 1. Why This Redesign Exists

The original README recommended WhatsApp as the primary delivery channel,
with a plain-HTML web fallback as Plan B, specifically because of the
blueprint's low-data/low-power/phone-only constraint (Blueprint §6) and
because WhatsApp needs no install step.

That reasoning about the *constraint* (weak connectivity, low-end phones,
data cost sensitivity) does not change. What changes is the delivery
requirement: the product now needs to be **downloadable from the Play Store
and App Store**, and **also usable directly in a browser with no install**,
from one shared backend and one shared tutoring engine. Concretely, this
means:

- One **tutoring engine / API backend** — unchanged in behavior from the
  original README's Sections 2–6 — that has no idea whether it's talking to
  a native app or a browser tab.
- One **native mobile client** (iOS + Android) built cross-platform, thin
  enough to install on low-end devices and function on weak connections.
- One **web client** (installable as a Progressive Web App, so a student
  without app-store access still gets an app-like, offline-capable
  experience from a browser with zero install friction).
- WhatsApp is **not discarded** — it drops from "primary interface" to an
  **optional future channel** built later against the same backend API, once
  the app/web clients are stable. This keeps the zero-friction advantage
  available later without making it the thing engineers build first.

---

## 2. Product Behavior Recap (unchanged — see original README for full detail)

These are restated only so this document is self-contained; they are not
redefined here.

- **Core loop (README §2):** DIAGNOSE → TEACH → PRACTICE → VERIFY → LOOP OR
  CLOSE. No stage is skippable.
- **Interaction rules (README §3):** wait for the student, one active
  respondent, exact confirmation count, never mark correct without checking,
  correct ≠ understanding, handle interruptions without losing state,
  multiple learners get separate state.
- **Age-band calibration (README §4):** Primary 4–6, JSS1–3, SSS1–3 each get
  a distinct voice — this is a safety requirement, not a UX nicety.
- **MVP scope (README §5):** one subject (Mathematics recommended), one
  grade band, a small concept set, the full loop proven before expansion.
- **State machine (README §6):** `SELECT_LEARNER → SELECT_SUBJECT →
  SELECT_TOPIC → DIAGNOSE → ASK_QUESTION → WAIT_FOR_ANSWER → CHECK_ANSWER →
  ... → RECORD_RESULT → MOVE_TO_NEXT_LEARNER_OR_QUESTION`, must survive
  interruptions and support resuming an unfinished session.

Everything in this document exists to deliver that engine on the required
platforms — it does not change what the engine does.

---

## 3. System Architecture Overview

```
                    ┌─────────────────────────────┐
                    │        Tutoring Engine        │
                    │  (state machine + diagnostic  │
                    │   loop + answer validation)   │
                    │        — README §2, §6        │
                    └───────────────┬────────────────┘
                                    │
                    ┌───────────────┴────────────────┐
                    │           Backend API            │
                    │   (auth, session state, learner   │
                    │  profiles, progress store, sync)  │
                    └───────┬───────────────┬──────────┘
                            │               │
              ┌─────────────┘               └─────────────┐
              │                                            │
    ┌─────────▼──────────┐                     ┌───────────▼───────────┐
    │   Native Mobile App  │                     │      Web Client        │
    │  (iOS + Android,     │                     │  (browser + installable │
    │   cross-platform      │                     │   PWA, no install       │
    │   framework)          │                     │   required)             │
    └───────────────────────┘                     └─────────────────────────┘
              │                                            │
     App Store / Play Store                        Any browser, any device
```

**Guiding rule for every layer below:** the tutoring engine and the backend
API are platform-agnostic. Neither client is allowed to contain tutoring
logic (diagnosis, answer checking, confirmation counting, state
transitions) — that logic lives once, in the backend, so mobile and web
never drift out of sync with each other or with README §2–§6. Clients are
render + input layers only.

### 3.1 Layer: Tutoring Engine

- Implements the loop and rules in README §2–§3 and the state machine in
  README §6, as a backend service — not as client-side prompt logic.
- Owns the "hybrid" answer-checking approach from the original README §7:
  deterministic validation in code wherever a subject allows it
  (Mathematics first), combined with model-driven diagnostic conversation
  and explanation for open-ended parts. The model never self-reports
  correctness for a checkable answer.
- Exposes its behavior to the Backend API layer only — it has no direct
  knowledge of HTTP, mobile, or web.
- **Engineer note:** this layer is now built and verified — implemented
  test-first (state machine, interaction rules, deterministic Mathematics
  answer checking, AI-provider boundary), proven by the automated tests in
  `tests/`, with no direct knowledge of HTTP, mobile, or web. The original
  manual prompt-testing phase (README §8 / Stage 0 below) was not executed
  as a pre-code gate; it is now a Blueprint-tracked open item to complete
  before any broad/real-world rollout, and does not block further
  development.

### 3.2 Layer: Backend API

- A single API (REST or GraphQL — pick one and use it consistently; do not
  mix) that both clients call identically. No app-specific or
  web-specific endpoints.
- Responsibilities:
  - **Auth** — learner/parent/teacher accounts, session tokens, and (per
    README §10) verifiable parental/guardian consent capture for
    under-18 users before any real student data is collected.
  - **Session state** — the state machine from README §6, persisted so a
    session can be resumed on a different device or after a connectivity
    drop, not just after an app restart.
  - **Learner profile store** — the per-learner data model from README §6
    (grade band, subjects, topics, concepts mastered/needing work, common
    mistakes, diagnostic findings, recent answers, difficulty level,
    session history, last unfinished lesson, spaced re-check schedule).
  - **Sync** — reconciles client-side offline queues (see [3.4](#34-offline-and-low-connectivity-behavior))
    when a device regains connectivity.
  - **Audit log** — per README §9 safeguard 6, stores enough interaction
    detail to investigate any case where the system gave bad feedback.
- **Recommended stack:** Go backend (matches the team's existing
  conventions across other active projects, per the original README §7),
  PostgreSQL for the relational per-learner data model (README §6's data is
  fundamentally relational — learners, sessions, topics, diagnostic
  results, review schedules).
- **Real-time transport:** use WebSockets (or long-polling as a low-data
  fallback) for the live tutoring exchange, since the loop is inherently
  turn-based and low-latency waiting matters (README §3, "wait for the
  student"). Fall back to plain request/response on very poor connections
  rather than dropping the session.

### 3.3 Layer: Native Mobile App (iOS + Android)

- Built with a single cross-platform framework (e.g. Flutter or React
  Native — team should confirm against existing skill sets before Stage 3
  of [Section 8](#8-build-stages)) so iOS and Android ship from one
  codebase instead of two.
- Talks to the Backend API only — no bundled tutoring logic.
- Must satisfy both storefronts' requirements for apps directed at
  children (see [Section 6](#6-app-store--play-store-requirements)) before
  submission.
- Local responsibilities: rendering the current state-machine step,
  capturing input, local caching for offline continuation, push
  notifications for periodic re-checking (README §9 safeguard 4).

### 3.4 Offline and Low-Connectivity Behavior

This preserves the original design constraint (Blueprint §6: text-first,
short exchanges, resumable sessions) inside a native-app model instead of a
WhatsApp model:

- **Text-first, no required media** — same as the original constraint;
  images/audio only where a concept genuinely requires it.
- **Short, resumable exchanges** — one diagnose→teach→practice→verify loop
  should complete in a handful of short round-trips, not a long continuous
  stream, so a dropped connection loses at most one exchange, not a whole
  session.
- **Local session cache** — the client keeps the current state-machine
  step and pending answer locally so the app can reopen mid-session without
  re-hitting the diagnostic stage from scratch (this directly implements
  README §6's "must support resuming an unfinished session" requirement).
- **Queued sync** — answers given while offline queue locally and sync to
  the backend audit/progress store on reconnect; the client never silently
  discards an interaction.

### 3.5 Layer: Web Client (Browser + PWA)

- Same Backend API, same state machine, same rendering rules as the mobile
  app — built as a Progressive Web App so it is:
  - Usable instantly in any browser with **zero install**, preserving the
    original zero-friction goal that motivated the WhatsApp recommendation.
  - **Installable** to a home screen on both desktop and mobile without an
    app-store listing, for students who can't or don't want to install a
    store app.
  - Capable of basic offline caching (service worker) using the same
    resume-without-repeating-diagnosis rule as [3.4](#34-offline-and-low-connectivity-behavior).
- **Recommended stack:** a lightweight framework, not a heavy one — same
  "why" as the original README's fallback reasoning (minimal JS, fast load
  on weak connections, small bundle). Confirm the specific framework choice
  against the team's existing web conventions before Stage 2 of
  [Section 8](#8-build-stages).

### 3.6 Layer: WhatsApp (Deferred, Optional)

- Not built in the MVP. Documented here only so a future engineer doesn't
  accidentally treat this as abandoned or as something to design around
  prematurely.
- When/if built, it becomes a third thin client against the same Backend
  API — never a second implementation of the tutoring engine. Real
  WhatsApp Business API approval steps and per-message costs at scale
  (flagged as unverified in the original README §7) still need confirming
  before that channel is committed to, whenever it's picked up.

---

## 4. Parent / Teacher View

- A separate, simpler client surface (can be a view within the same web
  client, gated by a parent/teacher login) reading from the same Backend
  API's learner-profile and audit endpoints — not a separate data store.
- Implements README §9 safeguard 5: a simple view of gaps found, topics
  practiced, concepts mastered, and areas needing attention.
- Does **not** get its own copy of tutoring logic; it is read-only against
  data the backend already maintains for auditability (README §9
  safeguard 6).

---

## 5. Data Model (shared across all clients)

Owned entirely by the Backend API layer ([3.2](#32-layer-backend-api)); no
client stores its own independent copy of this beyond local offline
caching.

| Entity | Key fields (from README §6 / Blueprint §9–10) |
|---|---|
| **Learner** | identifier, grade band, guardian/consent record, subjects, current difficulty level |
| **Session** | active learner, current state-machine step, current subject/topic, confirmation count, pending question, timestamps |
| **DiagnosticResult** | topic, identified gap, evidence used, whether gap appears closed |
| **ProgressRecord** | concepts mastered, concepts needing work, common mistakes, recent answers, last unfinished lesson |
| **ReviewSchedule** | concepts due for periodic re-checking (README §9 safeguard 4) |
| **AuditLogEntry** | full interaction detail per exchange, for investigating bad feedback (README §9 safeguard 6) |

---

## 6. App Store / Play Store Requirements

These are additional to, not a replacement for, the child-data
requirements already specified in README §10. Confirm all of the following
with legal counsel before submission — this is an engineering checklist for
what needs a real answer, not a legal determination:

- **Apple App Store — Kids Category / age rating:** apps directed at or
  likely to be used by children under 13 have specific data-collection,
  advertising, and third-party-SDK restrictions. Confirm whether this
  product is submitted under the Kids Category or a standard rating with
  disclosed child-directed status.
- **Google Play — Families Policy / Teacher Approved:** similar
  requirements around data handling, ads, and content for apps appealing
  to children; a Data Safety section disclosure is required at submission.
- **Parental/guardian consent flow:** must be implemented as an actual
  gate in the account-creation flow (not just a backend flag) before any
  real student data is collected, per README §10's Nigeria Data Protection
  Act (2023) requirement.
- **No open-ended child-facing chat surface:** the app's UI must not expose
  a general-purpose chat box to a child user — inputs should be scoped to
  the tutoring interaction (answers, confirmations), consistent with README
  §10's "no open-ended, unscoped conversation with a child user."
- **Data controller determination:** README §10 already flags this as
  unresolved — it also determines what the app-store privacy disclosures
  should say, so it needs resolving before either store submission, not
  just before backend build-out.

---

## 7. Tech Stack Summary

| Layer | Recommendation | Rationale |
|---|---|---|
| Tutoring engine + Backend API | Go + PostgreSQL | Matches team convention; concurrency model fits many simultaneous sessions; learner data model is relational (README §7) |
| Real-time transport | WebSockets, with request/response fallback | Matches the inherently turn-based loop (README §3); fallback protects low-connectivity users |
| Native mobile app | Single cross-platform framework (confirm choice at Stage 3) | One codebase for iOS + Android, thin enough for low-end devices |
| Web client | Lightweight framework, PWA | Zero-install browser access, installable without app store, small bundle for weak connections |
| Answer validation | Deterministic/code-based where possible (Mathematics first), model-driven for open-ended diagnosis | README §7's hybrid rule — never let the model self-grade a checkable answer |
| Future channel | WhatsApp Business API (deferred) | Optional third thin client once app/web are proven — not built first |

---

## 8. Build Stages

Each stage has a clear owner-facing deliverable and an explicit "done"
condition, so an engineer can pick up a stage without re-deriving scope.
**Stages are sequential — do not start a stage before the prior stage's
done-condition is met**, per the original README's workflow discipline
(§12): one feature at a time, tested and confirmed working before the next.

> **Current state (mirrors the README.md status line):** Stage 1 is
> implemented and verified for the narrow MVP slice — the tutoring engine
> (state machine, interaction rules, deterministic Mathematics answer
> checking), the AI-provider boundary, a session store (in-memory
> `MemoryStore` for tests, durable file-backed `FileStore` for the server),
> and a thin REST/JSON HTTP API (`ai/`, `session/`, `tutor/`, `api/`) —
> proven by 100 automated tests, with the full loop drivable over the API.
> Sessions survive server restarts via `FileStore`; PostgreSQL is also
> available behind the same Store contract (server `-dsn`). Stage 2's web
> client (`web/` + `cmd/server`)
> drives the full cycle in a browser (happy path verified manually). Not yet
> built: the later client stages (3–8). Stage 0 below is a Blueprint-tracked
> open item that does not block development.

### Stage 0 — Prompt Validation (no application code)
*Unchanged from original README §8.*
> **Status: outstanding — Blueprint-tracked (not a development gate).** The
> engine and web client were built test-first rather than behind this manual
> phase. This real-learner validation is owned under `Blueprint/` and should
> be completed before any broad/real-world rollout; it does not block further
> development.
- **Deliverable:** a strict tutoring-protocol system prompt, manually
  tested across real learners, with documented failures and fixes.
- **Done when:** manual sessions consistently satisfy the Success Criteria
  in [Section 9](#9-success-criteria).
- **Blocks:** broad/real-world rollout only — not Stage 1/2 development.

### Stage 1 — Tutoring Engine + Backend API (headless)
> **Status: done for the narrow MVP slice** (engine + store + callable API).
> The session store is now durable (`FileStore`, survives restarts); PostgreSQL
> remains the planned long-term store. The Stage 0 real-learner item remains
> open (Blueprint-tracked, non-gating).
- **Deliverable:** the state machine (README §6) and interaction rules
  (README §3) implemented as a backend service with a callable API,
  including deterministic Mathematics answer-checking. No UI yet — testable
  via API calls or a bare test harness.
- **Done:** the engine (`session/`, `ai/`), the application layer that owns
  the loop (`tutor/`), a session store (in-memory `MemoryStore` for tests,
  durable `FileStore` for the server), and the HTTP boundary (`api/`: begin,
  input, resume) are implemented in Go and proven by 100 automated tests in
  `tests/`; the full loop runs over the API.
- **Remaining:** the Stage 0 real-learner item (Blueprint-tracked). Durable
  persistence is done: file-backed `FileStore` (default) and an optional
  PostgreSQL store behind the same Store contract (server `-dsn`), per
  Agent.md §20.
- **Done when:** the same failure scenarios collected in Stage 0 (wrong
  answer marked correct, lost learner identity, over-repetition, etc.) no
  longer occur when driven through the API.

### Stage 2 — Web Client (browser, non-installable first pass)
> **Status: core loop + picker done.** A plain HTML/CSS/JS page (`web/`)
> served by the Go server (`cmd/server`) drives the full cycle in a browser.
> The start form is fed by the server-owned `/curriculum` catalog (subject,
> topic, and grade band), and begin captures the learner's grade band
> (README §4/§10). The loop teaches the gap after repeated wrong answers and
> honors stop/quit/exit/enough as a learner exit (README §9).
- **Deliverable:** a minimal browser UI against the Stage 1 API covering
  one subject, one grade band, one learner at a time — proves the full
  client↔backend loop end to end.
- **Done when:** a real student can complete a full diagnose→teach→
  practice→verify cycle in-browser with no engineer intervention.

### Stage 3 — Web Client Hardening (PWA + offline)
- **Deliverable:** service worker, offline queuing, resumable sessions
  ([3.4](#34-offline-and-low-connectivity-behavior)), install-to-home-screen support.
- **Done when:** a session survives a simulated connectivity drop and
  resumes without repeating diagnosis.

### Stage 4 — Native Mobile App (iOS + Android)
- **Deliverable:** cross-platform app consuming the same Stage 1 API,
  reusing the offline/resume behavior proven in Stage 3.
- **Done when:** functionally equivalent to the Stage 3 web client on both
  platforms, on real low-end test devices, not just simulators.

### Stage 5 — Store Compliance and Submission
- **Deliverable:** app-store checklist from [Section 6](#6-app-store--play-store-requirements)
  fully resolved and implemented (consent flow, data safety disclosures,
  age rating).
- **Done when:** both submissions are accepted for review with no
  compliance rejections tied to child-data or child-content handling.

### Stage 6 — Multi-Learner Support (conditional)
- **Deliverable:** README §5's MVP Feature 10 — only built if Stage 0/2
  testing actually showed group tutoring is an important real use case.
- **Done when:** multiple learners in one session maintain fully separate
  state (README §3's "multiple learners, separate state" rule) on both
  clients.

### Stage 7 — Parent/Teacher View
- **Deliverable:** the read-only view from [Section 4](#4-parent--teacher-view).
- **Done when:** a non-subject-expert parent/teacher can identify a
  student's current gaps and mastered concepts from the view alone.

### Stage 8 — WhatsApp Channel (deferred, future)
- **Deliverable:** third client against the Stage 1 API, only after
  WhatsApp Business API access/cost is confirmed (open item carried over
  from original README §7).
- Not scheduled as part of this MVP; listed for future planning only.

---

## 9. Success Criteria

*Unchanged from original README §11 — restated for this document's
completeness. Applies regardless of which client (app or web) a session
runs on.*

The MVP is promising if it can consistently:
1. Wait for the learner.
2. Track whose turn it is, correctly.
3. Check answers accurately (not just plausibly).
4. Avoid confidently approving wrong answers.
5. Identify genuine misconceptions, not just wrong/right.
6. Give explanations targeted at the actual gap, not generic ones.
7. Show evidence of real understanding beyond answer repetition.
8. Maintain separate, non-confused learner records.
9. Recover cleanly from normal interruptions.
10. Help a student genuinely improve on a second attempt at a related
    question.

**Cross-platform addition:** a session started on one client (e.g. web)
and resumed on another (e.g. the native app, once built) must resume at
the correct state-machine step without repeating the diagnostic stage.

---

## 10. Document Map

- New to the project, need the full picture → read the original `README.md`
  in full, then this document in full.
- Building the tutoring protocol / prompt → original README §2, §3, §4, §8,
  §9.
- Building the backend API / state management → this document's
  [Section 3.1–3.2](#3-system-architecture-overview), plus original README
  §6.
- Building the web client → [Section 3.5](#35-layer-web-client-browser-and-pwa),
  [Stage 2–3](#8-build-stages).
- Building the native mobile app → [Section 3.3–3.4](#33-layer-native-mobile-app-ios--android),
  [Stage 4](#8-build-stages).
- App-store / Play Store submission work → [Section 6](#6-app-store--play-store-requirements),
  [Stage 5](#8-build-stages).
- Parent/teacher-facing work → [Section 4](#4-parent--teacher-view),
  [Stage 7](#8-build-stages).
- Anything touching real student data or content safety → original README
  §10 in full, plus this document's [Section 6](#6-app-store--play-store-requirements)
  — read in full, no skimming.
- Pitching, positioning, or fundraising conversations → the attached
  business blueprint document (`Blueprint/AI_Tutor_Blueprint_v2.txt`), not
  either README.
