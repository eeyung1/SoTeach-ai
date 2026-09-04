# AI Tutor — Diagnostic Learning System for Nigerian Students

**Status:** Stage 1 (tutoring engine + backend API) and the core of Stage 2
(a minimal web client) are implemented and test-verified for the narrow MVP
slice — 100 automated tests green, plus a manually verified full tutoring
cycle in a real browser. The core loop from [Section 2](#2-the-core-learning-loop-this-is-the-product--read-this-section-fully-before-writing-tutor-logic)
(DIAGNOSE → TEACH → PRACTICE → VERIFY → LOOP/CLOSE), the required state
machine ([Section 6](#6-required-state-machine)), the interaction rules from
[Section 3](#3-interaction-rules-derived-directly-from-a-real-observed-tutoring-session)
(including deterministic Mathematics answer-checking), the AI-provider
boundary, a session store, a thin REST/JSON HTTP API, and a plain
HTML/CSS/JS browser client are built in Go + web (`session/`, `ai/`, `tutor/`,
`api/`, `web/`, `cmd/`, `tests/`). A learner picks a grade band, subject, and topic from the server-owned
`/curriculum` catalog in the browser, then is guided through diagnose →
practice → verify → mastery with no engineer intervention. Diagnosis is
AI-evaluated behind the `AIProvider` boundary (a concrete provider still plugs
in later) and surfaced as a learner/parent gap report. Sessions persist durably to a file-backed store
(`FileStore`) and survive server restarts; PostgreSQL remains the planned
long-term store. Not yet built: PWA/offline (Stage 3) and native mobile
(Stage 4). Delivery architecture and build stages live in
`workingReadme.md`.

This README is the single source of truth for the team. It merges three
source documents — the business blueprint, the product overview, and a
real observed sample tutoring session — into one build-ready spec. Point
engineers to the specific numbered section relevant to their work rather
than the whole document.

---

## 1. What This Is

An AI tutoring system for Nigerian school students (Primary 4 – SSS3) that
**diagnoses what a specific student doesn't understand before teaching
anything** — rather than delivering the same generic lesson to every
student who asks about a topic.

**The pitch, precisely**: *"Most tools start teaching immediately. This
tutor first finds out what a specific student actually does not
understand."* Do not pitch or build this as "another AI tutor" — the
diagnostic-first loop is the entire product. If a build decision makes the
diagnostic step optional or skippable, that decision is wrong.

Full business context, competitive landscape, and viability reasoning
lives in the attached blueprint document — engineers building the core
tutoring logic don't need to read that in depth; anyone working on
positioning, pitching, or monetization should.

---

## 2. The Core Learning Loop (this is the product — read this section fully before writing tutor logic)

Every concept a student engages with goes through this loop. No stage is
optional, and no stage may be silently skipped by the model.

1. **DIAGNOSE** — Ask the student to explain what they already know
   *before* teaching anything. Locate the actual gap, not an assumed one.
   Two students stuck on "fractions" may have completely different real
   gaps (one doesn't understand division; another understands division but
   not what a fraction represents) — the loop must find out which.
2. **TEACH** — Explain only the specific missing piece, calibrated to the
   student's age band (see [Section 4](#4-age-band-calibration--non-negotiable)). Do not re-teach what the
   student already demonstrated they know in Stage 1.
3. **PRACTICE** — Give the student a concrete task to apply the concept
   themselves. A concept is not "taught" until the student has done
   something with it, not just read an explanation.
4. **VERIFY** — Check whether the practice attempt reveals a remaining
   misunderstanding, using a genuinely different question, not a repeat of
   the same one. A correct final answer alone is **not** sufficient
   evidence of understanding (see Section 3, Rule 5 — a student can guess,
   memorize, or copy).
5. **LOOP OR CLOSE** — If a gap remains, return to TEACH for that specific
   piece only. If verified, briefly confirm mastery and stop.

---

## 3. Interaction Rules (derived directly from a real observed tutoring session)

These are concrete, tested rules, not theoretical — they come from a real
sample lesson where violating them caused real problems. Treat every rule
below as a hard constraint on the tutor's behavior, not a guideline.

| Rule | What it means | What happens if violated (observed) |
|---|---|---|
| **Wait for the student** | Never reveal or supply an answer while waiting for a response. Never continue to the next question while one is pending. | Tutor answered its own question before the student responded. |
| **One active respondent** | The system must track *whose turn it is* and only accept that learner's answer as the response. | Risk of another learner's input being treated as the active respondent's answer. |
| **Confirmation count is exact** | Exactly two confirmations per answer, tracked as explicit state — not one, not three or four. | Tutor asked for repeated confirmation beyond what was needed. |
| **Never mark correct without checking** | Especially for Mathematics, use deterministic/reliable answer-checking, not the model's own unverified judgment. | `7 + 3` was initially accepted as correct when the answer given was wrong. |
| **Correct ≠ understanding** | A right answer can come from guessing, memorizing, or copying another learner. Require a transfer question (a different but related problem) before treating a concept as mastered. | — |
| **Handle interruptions without losing state** | "Wait," "again," "what was the question," a learner leaving, a parent/teacher interjecting — the system must recover without losing the active learner, current question, confirmation count, or subject. | — |
| **Multiple learners, separate state** | A session may involve several named learners with individual turns and separate progress — never merge or confuse their state. | — |

**This directly requires explicit, code-managed session state — not
reliance on the model's own free-form conversation memory.** The state
machine in [Section 6](#6-required-state-machine) exists specifically because prompting alone was shown
to lose track of confirmation counts and turn order in real testing.

---

## 4. Age-Band Calibration (non-negotiable)

The tutor must not use one voice for every student. Three bands, three
distinct calibrations:

- **Primary 4–6 (ages 9–11):** short sentences, concrete real-world
  analogies (objects, money, food, familiar Nigerian contexts), no
  unexplained abstract terminology.
- **JSS1–3 (ages 12–14):** proper terminology allowed, but every new term
  must be paired with a plain-language explanation the first time it's used.
- **SSS1–3 (ages 15–17):** full curriculum vocabulary, multi-step
  reasoning, exam-style question formats (WAEC/NECO/JAMB) where relevant.

---

## 5. MVP Scope — What to Build First, What to Explicitly Defer

**Build first:**
- One subject (Mathematics is strongly recommended — answers are often
  reliably checkable, and misconceptions can be deliberately tested).
- One grade band.
- A limited, deliberately small set of concepts.
- The full diagnose → teach → practice → verify loop for that narrow
  scope, proven reliable before expanding anything.

**Explicitly do not build first** (per the product overview's own
scoping): every subject, every curriculum topic, animations, video
features, a social platform, gamification, full institutional dashboards,
or heavy infrastructure ahead of proving the tutoring logic itself works.
Multi-learner group session support is listed as MVP Feature 10 —
**conditional**, only if manual testing shows group tutoring is actually
an important real use case, not built by default.

---

## 6. Required State Machine

The system must track explicit state — this is not optional scaffolding,
it is what Section 3's rules depend on to work at all.

```
SELECT_LEARNER
  → SELECT_SUBJECT
    → SELECT_TOPIC
      → DIAGNOSE
        → ASK_QUESTION
          → WAIT_FOR_ANSWER
            → CHECK_ANSWER
              → HANDLE_INCORRECT_OR_UNCERTAIN_ANSWER   (loop back to ASK_QUESTION on the gap)
              → REQUEST_CONFIRMATION_OR_TRANSFER_CHECK
                → WAIT_FOR_RESPONSE
                  → VERIFY_UNDERSTANDING
                    → RECORD_RESULT
                      → MOVE_TO_NEXT_LEARNER_OR_QUESTION
```

State must survive interruptions (Section 3) and must support **resuming
an unfinished session** without repeating the diagnostic stage from
scratch.

### Per-learner data to track
Name/identifier, grade band, subjects, topics studied, concepts mastered,
concepts needing work, common mistakes, diagnostic findings, recent
answers, current difficulty level, session history, last unfinished
lesson, and concepts due for periodic re-checking (see Section 9).

---

## 7. Tech Stack — Recommendation

**Backend: Go.** Matches this team's existing conventions across other
active projects (shared patterns, shared comfort level), and its
concurrency model is a genuine fit for handling many simultaneous tutoring
sessions cleanly — the same reasoning that led to choosing Go for the
logistics platform's order/dispatch engine applies here.

**Database: PostgreSQL.** The per-learner data model in Section 6 is
fundamentally relational (learners, sessions, topics, diagnostic results,
review schedules) — this is not a fit for a document store or
SQLite-at-scale.

**Delivery channel — the decision that most needs team sign-off before
building starts:** given the blueprint's own low-data/low-power/phone-only
design constraint (Section 6 of the blueprint), **WhatsApp is worth
building on as the primary interface, not a custom app.** Reasoning:
- WhatsApp is already near-universal on Nigerian phones, works over weak
  connections, and needs no install step — this satisfies "meet the user
  with zero new adoption friction" more completely than any custom UI can
  (the same principle that separated Moniepoint's real adoption from
  what3words'/Plus Codes' failure to gain traction, per the standalone
  Product Validation Gut Check reference).
- It is naturally text-first and naturally turn-based — which maps almost
  exactly onto the state machine in Section 6.
- **This still needs verifying, not assuming**: WhatsApp's Business API
  has real approval steps and per-message costs at scale — confirm actual
  access requirements and cost before committing fully to this channel. A
  lightweight web fallback (plain HTML, minimal JS, no heavy framework —
  same "why" as crdledger's build) is the reasonable Plan B if WhatsApp
  access turns out to be a blocker.

**LLM integration — hybrid, not pure-model-judgment:** per Section 3's
"never mark correct without checking" rule, use **deterministic answer
validation wherever a subject allows it** (Mathematics answers can and
should be checked in code, not left to the model's own grading judgment),
combined with **model-driven diagnostic conversation and explanation**
for the open-ended parts (understanding what a student's wrong answer
reveals, generating age-calibrated explanations). Do not rely on the
model to self-report whether an answer was correct — verify it.

---

## 8. Testing Plan — Do This Before Writing Any Application Code

This is not a formality — it's the actual next step, and it comes before
any of the architecture above gets built.

> **Status note:** the manual real-learner protocol-validation phase described
> below was not executed as a pre-code gate — the tutoring behavior in
> Sections 2–6 was instead developed incrementally and test-first, and is
> currently proven by the automated tests in `tests/`. The plan below is
> therefore not obsolete: validating the loop with real learners remains an
> open item owned under `Blueprint/` (workingReadme.md §8, Stage 0) and should
> be completed before any broad/real-world rollout; it does not block further
> development. This section's core sequencing intent — prove the loop before
> expanding infrastructure — is what the test-first work has satisfied so far.

1. **Write the tutoring protocol** as a strict system prompt — covering
   turn-taking, waiting behavior, answer checking, confirmation rules,
   diagnostic behavior, teaching behavior, verification behavior,
   interruption handling.
2. **Run manual sessions** with real learners — across correct answers,
   incorrect answers, guesses, "I don't know," repeated questions,
   interruptions, and multiple learners of different ability levels.
3. **Score every session** against: did it wait? Track the right learner?
   Judge answers correctly? Follow the confirmation rule exactly? Actually
   diagnose the real problem? Teach only the missing piece? Show evidence
   of real understanding afterward, not just repetition?
4. **Collect failures deliberately**, not just successes — wrong answers
   marked correct, answers revealed too early, lost learner identity, lost
   question state, over-repetition, missed guessing, misdiagnosed gaps.
5. **Fix the protocol, retest.**
6. **Only then** build the smallest possible interface around the proven
   protocol.

**The single most important thing to learn from this phase**, per the
blueprint's own viability analysis: whether the diagnostic loop holds up
**without a teacher present guiding the session** — the one thing the
strongest real-world evidence for AI tutoring (the Edo State pilot) has
not actually proven, since that pilot always had a teacher in the room.

---

## 9. Known Risk — False Confidence (do not treat as optional polish)

The single biggest product risk: a student can feel confident because the
AI *sounds* confident, even when the answer, the explanation, or the
student's own understanding is wrong.

Required safeguards, not later nice-to-haves:
1. **Reliable, deterministic answer-checking** wherever possible (Section 7).
2. **Explicit uncertainty behavior** — the system must not pretend to know
   something it's uncertain about.
3. **Real verification**, not just repetition (Section 2, Stage 4).
4. **Periodic re-checking** — return to previously "mastered" concepts
   later to confirm the learning actually lasted.
5. **Parent/teacher visibility** — a simple view of gaps found, topics
   practiced, concepts mastered, and areas needing attention, so a human
   who may not have deep subject knowledge themselves can still spot-check.
6. **Auditability** — store enough interaction detail to investigate any
   case where the system gave bad feedback.

---

## 10. Working With Children — Non-Negotiable Requirements

This product's users are minors, most as young as 9. This governs both
design and data handling, and neither is optional or a "later" concern.

**Content and interaction safety:**
- The tutor's outputs must be scoped strictly to curriculum tutoring — no
  open-ended, unscoped conversation with a child user.
- Explanations, examples, and tone must be age-calibrated per Section 4 —
  this is a safety requirement as much as a pedagogy one, not just a UX nicety.
- Any uncertainty or off-topic input from a learner should be handled by
  redirecting to the tutoring task, not by the system engaging with it as
  open conversation.

**Data privacy — confirm with legal counsel before collecting real student
data, the same way BlueMoon's licensing and SplashEvent's escrow custody
questions needed real legal answers, not engineering assumptions:**
- Nigeria's Data Protection Act (2023) has specific provisions for
  processing children's data, generally requiring verifiable parental or
  guardian consent — this needs direct legal confirmation before this
  product collects real student data, not an assumption that "it's just
  an ed-tech app" exempts it.
- **Data minimization**: only collect what Section 6's data model actually
  needs (learning progress, diagnostic results) — no unnecessary personal
  data about the child.
- Whichever party is legally the "data controller" (the school, a parent,
  or the platform operator, depending on which of the blueprint's "who
  pays" models is chosen) needs to be clearly established, since it
  determines who is actually responsible for consent and data-handling
  compliance.
- **Auditability** (Section 9, safeguard 6) also serves this purpose — a
  clear interaction record protects both the student and the platform if
  a concern is ever raised about what the system said to a child.

---

## 11. Success Criteria for the MVP

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
10. Help a student genuinely improve on a second attempt at a related question.

---

## 12. Workflow Discipline

Same discipline used across this team's other active projects:

1. **One feature at a time, tested and confirmed working before the next.**
   Do not batch the state machine, the answer-checking logic, and the
   delivery-channel integration into one untested block.
2. **The testing plan (Section 8) comes before application code.** Do not
   start building the state machine or database schema until manual
   prompt testing has shown real diagnostic signal — building
   infrastructure around an unproven tutoring loop is wasted work if the
   loop itself doesn't hold up.
3. **Never assume a scoping or safety question — confirm it.** Especially
   anything in Section 10; child-data and content-safety questions get a
   real answer, not an engineering guess.
4. **Push back on scope creep** — anything in Section 5's "do not build
   first" list that creeps back in early should be named and questioned,
   not quietly built.

---

## 13. Document Map (for pointing engineers to the right section)

- New to the project, need the full picture → read this whole document.
- Building the tutoring protocol / prompt → Sections 2, 3, 4, 8, 9.
- Building the backend/state management → Sections 6, 7.
- Working on delivery channel (WhatsApp/web) → Section 7.
- Anything touching real student data or content safety → Section 10 (read
  in full, no skimming).
- Pitching, positioning, or fundraising conversations → the attached
  business blueprint document, not this README.
