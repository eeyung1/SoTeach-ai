# SoTeach — Free-Diagnosis Acquisition Funnel (product direction)

**Status:** Product direction / idea — **not yet scoped for build.** Business
and product context only (Blueprint level per Agent.md §3's document
hierarchy). Do not treat anything here as an implementation requirement yet;
the binding behavior spec remains `README.md`, with delivery architecture in
`workingReadme.md`.

**Date recorded:** 2026-09-04

---

## 1. The funnel in one line

Give the learner a **free diagnostic that shows the value** (their real gap,
clearly reported), tease the teaching of that gap, then let the user sign up,
choose a plan, and pay to actually learn the concept — and many more.

## 2. The intended flow

1. **User inputs subject + topic** (and age/grade band — see the deferred
   picker gap in workingReadme.md §8, Stage 2).
2. **Diagnosis runs.** The model evaluates what the learner actually knows
   for that topic.
3. **Gap is identified and recorded.**
4. **A short report is generated for the learner / parent / user** stating the
   gap in plain, non-expert language.
5. **A little teaching of that concept** is offered as a sample/teaser.
6. **Conversion point:** an option appears to sign up and learn this concept
   in detail — and many more concepts. This is where the user selects a plan
   and makes payment.

The first, free diagnosis exists to let users **see the value before paying** —
it is the product's demonstration moment, not a loss leader.

## 3. Why this fits SoTeach

- It monetizes the diagnostic-first differentiator: "this tutor first finds
  out what a specific student actually does not understand."
- The gap report for parent/learner is the same data the docs already plan to
  track (`DiagnosticResult` in ai/, and the workingReadme data model) — a
  read-only presentation, not new engine state. It is the same layer that
  becomes the Parent/Teacher view (workingReadme.md §4, Stage 7).

## 4. What this idea depends on

- **Real "model evaluates" diagnosis.** Built: `SubmitDiagnosis` runs an
  AI `Diagnose` behind the `AIProvider` boundary (Groq adapter) and the
  evaluated gap is surfaced as a learner/parent report (with the learner's
  own words recorded verbatim when no provider is configured). Expanding the
  provider to explain/generate/report capabilities stays per Agent.md §38 —
  added only when a concrete requirement exists, with structured, validated
  output (§39-40) and tests that do not need a live model (§43).
- **Account / signup model for a children's product.** The payer is the
  parent/guardian, not the child. Guarded by verifiable consent (README §10;
  workingReadme.md §6) and store-compliance work (Stage 5) before any real
  payment flow. Guardian signup + verifiable consent are built, and a sample
  plan catalog with a guardian plan selection now lands as a **mock checkout**
  (`/guardians/plans`, `/guardians/plan`, `/guardians/me`) — no money is taken.
  Real PSP integration, enforcement of the gate on real collection, and
  pricing sign-off remain the compliance-gated stage.
- **The content library ("and many more concepts").** The server now teaches
  a small deterministic, age-calibrated set — Mathematics (Addition,
  Subtraction, Multiplication, Division) and the first English Language
  topic (Parts of Speech) — still far short of a paid catalog. A paid
  catalog only converts if there is a curriculum-scoped, age-calibrated,
  reliably checkable library behind it — the largest unbuilt dependency
  under this idea, and the reason a content pipeline/AI content matters.
- **A clean free-tier boundary.** The free teaser must never imply mastery.
  Per README §2/§3, reading an explanation is not evidence of understanding —
  only PRACTICE + VERIFY close a concept. The free diagnosis should end on an
  honest boundary ("here is your gap and a sample; subscribe to actually
  close and verify it"), never on a false claim of learning.

## 5. Proposed build sequencing (for when this is picked up)

Not ordered against the current backlog, but each step is independently
valuable and testable before any payments exist:

1. Subject/topic + grade-band picker in the web client — done (server-owned
   catalog; Mathematics topics + English Language, three grade bands).
2. Wire AI-validated diagnosis into the live loop so the gap is evaluated,
   not just recorded — done (Groq adapter + in-browser gap report).
3. A diagnostic gap report view (learner/parent-readable) derived from the
   result — done (read-only presentation of the recorded finding).
4. Guardian/signup + consent gate — done (slice 1). Plan selection — done as
   a **mock checkout** (slice 2, no money). Payment (real PSP, enforcement,
   pricing sign-off) — still the compliance-reviewed stage.

## 6. Open questions to resolve before building

- Exactly what is free vs paid: is only the report free, or report + one
  teaser practice question?
- Where the guardian-consent and payment boundary sits (who is the customer:
  school, parent, or platform — see README §10's data-controller question).
- How "many more concepts" content is produced and checked (curated bank vs
  AI generation, and at what age-calibration quality bar).
- Whether the free diagnosis is gated by anything (rate limit, one free
  diagnostic per topic, per learner) without harming the child experience.
