# SoTeach — Engineering Agent Instructions

## 1. Purpose

You are an engineering agent working on **SoTeach**, a diagnostic-first AI tutoring system for Nigerian school students.

Your job is not simply to make the code compile.

Your job is to implement the product specification faithfully, one verified feature at a time, while preserving the tutoring rules, state integrity, safety requirements, and architectural boundaries defined by the project documentation.

The system must first discover what a learner understands, identify the learner's actual knowledge gap, teach only the missing piece, require practice, verify genuine understanding, and then either close the concept or loop back to the remaining gap.

The diagnostic-first behavior is the central product differentiator.

Do not turn SoTeach into a generic chatbot.

---

# 2. Source-of-Truth Hierarchy

The repository contains several specification documents. Use them according to the following hierarchy.

## Level 1 — `README.md`

Use `README.md` as the authoritative source for:

* Core tutoring behavior
* Diagnose → Teach → Practice → Verify → Loop/Close
* Interaction rules
* Age-band calibration
* Required state machine
* Per-learner state
* Reliability requirements
* Child-safety requirements
* MVP success criteria
* Testing philosophy
* Engineering workflow discipline

If an implementation decision conflicts with these behavioral requirements, stop and resolve the conflict before implementing.

---

## Level 2 — `workingReadme.md`

Use `workingReadme.md` as the authoritative source for:

* Current system architecture
* Backend architecture
* Client architecture
* Web/PWA direction
* Native mobile direction
* Offline behavior
* Synchronization
* API boundaries
* Data model
* Build stages
* Cross-platform requirements

This document represents the current architectural direction of the product.

The original README's WhatsApp-first recommendation has been superseded architecturally by the cross-platform redesign.

WhatsApp is currently a deferred/future client, not the MVP's primary implementation target.

---

## Level 3 — `Blueprint/`

The Blueprint documents provide supporting product and business context.

Relevant files include:

* `Blueprint/AI_Tutor_Blueprint_v2.txt`
* `Blueprint/product_overview.txt`
* `Blueprint/sample_lesson.txt`

Use them to understand:

* Why the product exists
* Target users
* Product positioning
* Business assumptions
* Observed tutoring failures
* Product requirements derived from real interaction
* Future scope

Do not introduce implementation requirements from the Blueprint that contradict the current README or working README.

---

## Level 4 — `Blueprint/Engineering_mentor_prompt.md`

Use this document to determine how engineering work should be performed and taught.

The engineering process must be:

* Incremental
* Test-driven
* One concept at a time
* Explicitly verified
* Resistant to premature abstraction
* Focused on understanding rather than blindly generating code

---

# 3. Core Engineering Rule

## TEST FIRST. ALWAYS.

Before implementing a new feature:

1. Identify the exact requirement.
2. Translate the requirement into observable behavior.
3. Write the test.
4. Run the test.
5. Confirm that the test fails for the expected reason.
6. Implement the smallest amount of code necessary.
7. Run the test again.
8. Fix failures.
9. Run the relevant existing tests.
10. Only after everything passes may the feature be considered complete.

Never begin by writing production code and then invent tests afterward.

The test defines the expected behavior.

---

# 4. Build One Feature at a Time

Never implement several unrelated features simultaneously.

For every feature use this cycle:

```text
SPECIFICATION
     ↓
TEST
     ↓
EXPECTED FAILURE
     ↓
MINIMAL IMPLEMENTATION
     ↓
TEST
     ↓
DEBUG
     ↓
REGRESSION TEST
     ↓
VERIFICATION
     ↓
NEXT FEATURE
```

Do not move to the next feature merely because the code compiles.

A feature is complete only when its behavior has been demonstrated by tests.

---

# 5. First-Build Rule

Before adding application functionality, establish the testing foundation.

The first implementation work should establish:

* Project structure
* Test structure
* Test fixtures
* Core domain/state definitions where required by the tests
* Specification-level behavior tests

Do not immediately build:

* Authentication
* UI
* Mobile screens
* WebSockets
* PostgreSQL integration
* LLM integration
* Parent dashboards
* WhatsApp integration

The tutoring behavior must be proven before infrastructure expands around it.

---

# 6. MVP Scope

The initial MVP must remain deliberately narrow.

Start with:

* Mathematics
* One grade band
* A small number of concepts
* One learner at a time unless testing proves multi-learner support necessary
* Full diagnostic learning loop
* Reliable answer checking
* Explicit session state

Do not expand to every subject or every grade.

Do not add:

* Gamification
* Social features
* Animations
* Video systems
* Large institutional dashboards
* Unnecessary infrastructure
* Premature microservices
* Unnecessary abstractions

Scope expansion requires explicit justification.

---

# 7. Required Tutoring Loop

Every learning interaction must follow:

```text
DIAGNOSE
   ↓
TEACH
   ↓
PRACTICE
   ↓
VERIFY
   ↓
LOOP OR CLOSE
```

No stage is optional.

## DIAGNOSE

The tutor must determine what the learner already understands.

Do not assume the learner's problem.

Do not immediately teach the topic.

The system must collect evidence before deciding what needs to be taught.

---

## TEACH

Teach only the identified missing concept or misconception.

Do not unnecessarily reteach material the learner has already demonstrated.

Teaching must be calibrated to the learner's age band.

---

## PRACTICE

The learner must perform an action demonstrating application.

Reading an explanation is not evidence of mastery.

---

## VERIFY

Verification must use a genuinely different but related problem.

Do not simply repeat the same question.

A correct answer alone does not prove understanding.

The system should distinguish:

```text
Correct answer
        ≠
Demonstrated understanding
```

---

## LOOP OR CLOSE

If evidence shows the learner still has a gap:

```text
VERIFY → TEACH
```

but teach the remaining gap rather than restarting the entire lesson unnecessarily.

If understanding is demonstrated:

```text
VERIFY → RECORD_RESULT → MOVE_FORWARD
```

---

# 8. Interaction Rules

These are hard requirements.

## 8.1 Wait for the learner

Never answer a question on the learner's behalf while waiting for the learner.

Never advance to the next question while a response is pending.

---

## 8.2 One active respondent

The session must know whose turn it is.

A learner's response must never accidentally be attributed to another learner.

---

## 8.3 Exact confirmation count

Confirmation state must be explicitly tracked.

Do not rely on the LLM's conversational memory.

Where the specification requires two confirmations, the state machine must enforce exactly two.

Do not accidentally ask three or four.

---

## 8.4 Never mark an unchecked answer correct

For checkable subjects such as Mathematics:

* Validate answers deterministically whenever possible.
* Do not let the LLM decide that a mathematically checkable answer is correct.
* The LLM may explain, diagnose, and converse.
* The deterministic validator owns correctness where deterministic validation is possible.

---

## 8.5 Correct does not automatically mean mastered

A correct answer may result from:

* Guessing
* Memorization
* Copying
* Pattern recognition without understanding

Therefore the system must use transfer/verification questions before recording mastery.

---

## 8.6 Preserve state through interruptions

The system must survive events such as:

* "Wait"
* "Again"
* "What was the question?"
* Learner temporarily leaving
* Parent/teacher interruption
* Connectivity loss
* Session resumption

The system must not lose:

* Active learner
* Current state
* Subject
* Topic
* Pending question
* Confirmation count
* Diagnostic findings
* Progress

---

# 9. Required State Machine

The state machine is explicit and must be represented in code.

```text
SELECT_LEARNER
    ↓
SELECT_SUBJECT
    ↓
SELECT_TOPIC
    ↓
DIAGNOSE
    ↓
ASK_QUESTION
    ↓
WAIT_FOR_ANSWER
    ↓
CHECK_ANSWER
    ↓
HANDLE_INCORRECT_OR_UNCERTAIN_ANSWER
    ↓
ASK_QUESTION
```

When appropriate:

```text
CHECK_ANSWER
    ↓
REQUEST_CONFIRMATION_OR_TRANSFER_CHECK
    ↓
WAIT_FOR_RESPONSE
    ↓
VERIFY_UNDERSTANDING
    ↓
RECORD_RESULT
    ↓
MOVE_TO_NEXT_LEARNER_OR_QUESTION
```

State transitions must be deterministic and testable.

Do not implement state transitions as implicit LLM behavior.

---

# 10. Learner State

The backend must eventually support learner records containing, as appropriate:

* Learner identifier
* Name/identifier
* Grade band
* Subjects
* Topics studied
* Concepts mastered
* Concepts needing work
* Common mistakes
* Diagnostic findings
* Recent answers
* Current difficulty
* Session history
* Last unfinished lesson
* Concepts due for periodic re-checking

Do not duplicate authoritative learner state across clients.

---

# 11. Client/Backend Boundary

The tutoring engine belongs to the backend.

Clients are presentation and input layers.

Neither the web client nor the mobile client may independently implement:

* Diagnosis
* Answer correctness
* Confirmation counting
* Tutoring state transitions
* Mastery decisions
* Learner progress rules

The architecture must be:

```text
             ┌──────────────────────┐
             │    Tutoring Engine   │
             │  State + Diagnosis   │
             │  Validation + Rules  │
             └──────────┬───────────┘
                        │
                 Backend API
                        │
             ┌──────────┴───────────┐
             │                      │
          Web/PWA              Mobile App
```

The same backend behavior must serve all clients.

---

# 12. Answer Validation

Use a hybrid model.

## Deterministic validation

Use code wherever the answer can be reliably checked.

For Mathematics this should be the default.

Examples:

```text
7 + 3 = 10
12 × 4 = 48
3/4 > 1/2
```

The correctness decision should not depend solely on an LLM.

## Model-driven reasoning

Use the model for tasks such as:

* Diagnostic conversation
* Identifying possible misconceptions
* Generating explanations
* Generating appropriately calibrated questions
* Interpreting open-ended reasoning

The model must not override deterministic correctness.

---

# 13. Age Calibration

The system must support distinct age bands.

## Primary 4–6

Use:

* Short sentences
* Concrete examples
* Familiar contexts
* Money, food, objects, everyday situations
* Minimal unexplained terminology

## JSS1–3

Use:

* Proper terminology
* Plain-language explanation when introducing new terminology
* Moderate abstraction

## SSS1–3

Use:

* Full curriculum vocabulary
* Multi-step reasoning
* Exam-style formats where appropriate
* WAEC/NECO/JAMB-style questions where relevant

Never treat age calibration as cosmetic.

It is a product and safety requirement.

---

# 14. Testing Requirements

Tests must cover behavior, not merely implementation details.

At minimum, the test suite must eventually cover:

### State

* Correct initial state
* Valid transitions
* Invalid transitions
* State persistence
* Session resumption

### Turn management

* Correct learner receives the turn
* Wrong learner cannot accidentally answer
* Active respondent is preserved

### Waiting

* No automatic answer while waiting
* No question advancement while awaiting response

### Answer checking

* Correct mathematical answer
* Incorrect mathematical answer
* Boundary cases
* Uncertain/ambiguous answers
* Model cannot override deterministic validation

### Confirmation

* Correct confirmation count
* No accidental extra confirmations
* Confirmation state survives interruption

### Understanding

* Correct repeated answer does not automatically equal mastery
* Transfer question is required
* Genuine understanding can close a concept

### Interruptions

* "Wait"
* "Again"
* Question repetition
* Learner interruption
* Session resume

### Learner separation

* Learner A's state cannot become Learner B's state
* Progress records remain separate

### Regression

Every bug discovered must result in a regression test.

---

# 15. Test Naming

Tests should describe behavior clearly.

Prefer:

```text
TestSessionWaitsForLearnerAnswer
TestIncorrectMathAnswerIsNotMarkedCorrect
TestCorrectAnswerDoesNotImmediatelyRecordMastery
TestConfirmationCountDoesNotExceedTwo
TestLearnerStateRemainsSeparate
TestSessionResumesWithoutRepeatingDiagnosis
```

Avoid vague names such as:

```text
TestSession
TestLogic
TestEverything
TestHandler
```

---

# 16. No Fake Tests

Do not write tests merely to make coverage numbers look good.

A useful test must fail when the required behavior is broken.

Do not weaken assertions simply because implementation is difficult.

Do not mock away the behavior being tested.

Do not make tests pass by changing the specification.

If the specification and implementation disagree, stop and resolve the disagreement.

---

# 17. Failure-First Development

When adding a feature:

### Step 1

Write the test.

### Step 2

Run it.

Expected result:

```text
FAIL
```

The failure should demonstrate that the feature does not yet exist.

### Step 3

Implement the smallest solution.

### Step 4

Run the test.

Expected result:

```text
PASS
```

### Step 5

Run the entire relevant test suite.

### Step 6

Only then continue.

---

# 18. Debugging Discipline

When a test fails:

1. Read the exact failure.
2. Identify the violated invariant.
3. Trace the execution.
4. Determine whether the problem is:

   * Test
   * Implementation
   * Specification misunderstanding
   * State-management bug
5. Fix the smallest correct layer.
6. Re-run the failing test.
7. Re-run regression tests.

Do not blindly rewrite large portions of the codebase.

---

# 19. API Rules

When the API layer is introduced:

* Use one API style consistently.
* Do not mix REST and GraphQL without explicit architectural approval.
* Keep tutoring logic outside HTTP handlers.
* HTTP handlers translate requests into domain operations.
* Domain logic must remain testable without HTTP.
* Clients must consume the same API behavior.

---

# 20. Persistence Rules

PostgreSQL is the planned persistent store.

Persistence must eventually support:

* Learners
* Sessions
* Diagnostic results
* Progress
* Review schedules
* Audit logs

Do not introduce PostgreSQL prematurely if the behavior can first be proven in memory through domain tests.

Prove behavior first.

Persist it second.

---

# 21. Offline and Low-Connectivity Rules

The product is designed for weak connections and low-end devices.

The system should favor:

* Text-first interaction
* Short exchanges
* Small payloads
* Resumable sessions
* Local session state
* Queued synchronization
* No silent loss of learner answers

A connectivity failure must not force the learner to restart diagnosis unnecessarily.

---

# 22. Safety Rules

This product interacts with children.

Treat child safety as a system requirement.

Do not introduce:

* Unscoped child-facing chat
* Unnecessary collection of child data
* Features that bypass guardian consent requirements
* Unreviewed child-facing content mechanisms
* Advertising mechanisms without explicit compliance review

Parental/guardian consent must eventually be an actual account-flow gate where required, not merely a database boolean.

Legal and store-policy questions must not be guessed.

Flag unresolved questions rather than inventing answers.

---

# 23. Security and Privacy

Never log or expose sensitive learner information unnecessarily.

Avoid placing sensitive information into:

* Debug logs
* Test output
* Error messages
* URLs
* Client-side storage without justification

Audit logging must provide enough information to investigate incorrect tutoring behavior while respecting privacy requirements.

---

# 24. Repository Structure

Prefer clear separation of responsibilities.

A likely backend structure is:

```text
SoTeach-ai-main/
├── Agent.md
├── README.md
├── workingReadme.md
├── Blueprint/
│   ├── AI_Tutor_Blueprint_v2.txt
│   ├── Engineering_mentor_prompt.md
│   ├── product_overview.txt
│   └── sample_lesson.txt
│
├── backend/
│   ├── ...
│   └── ...
│
├── tests/
│   ├── ...
│   └── ...
│
├── web/
│   └── ...
│
└── mobile/
    └── ...
```

Do not create every directory immediately.

Create a directory only when its corresponding stage requires it.

The initial structure should remain small.

---

# 25. Build Stages

Follow the project's sequential build strategy.

## Stage 0 — Protocol Validation

Before full application implementation:

* Establish the tutoring protocol
* Test against the sample lesson
* Test correct answers
* Test incorrect answers
* Test interruptions
* Test learner switching
* Test repeated answers
* Test diagnostic behavior
* Record failures
* Improve the protocol

Do not declare Stage 0 complete because the prompt "looks good."

It must demonstrate the required behavior.

---

## Stage 1 — Tutoring Engine + Backend

Build:

* State machine
* Domain models
* Diagnostic behavior
* Answer validation
* Interaction rules
* Session persistence
* API boundary

Do not build UI yet.

---

## Stage 2 — Web Client

Build the smallest browser client capable of exercising the backend.

Prove:

```text
Browser
   ↓
API
   ↓
Tutoring Engine
   ↓
State
   ↓
API
   ↓
Browser
```

---

## Stage 3 — PWA + Offline

Add:

* Service worker
* Local caching
* Offline queue
* Synchronization
* Session resume

---

## Stage 4 — Native Mobile

Build the cross-platform mobile client against the existing backend.

Do not duplicate tutoring logic.

---

## Stage 5 — Store Compliance

Resolve:

* Consent
* Age rating
* Data safety
* Child-facing content requirements
* Privacy requirements
* Store submission requirements

---

## Stage 6 — Multi-Learner

Only implement by default if testing demonstrates that group tutoring is an important MVP use case.

If implemented, every learner must have independent state.

---

## Stage 7 — Parent/Teacher View

Read from backend learner/progress data.

Do not create a second tutoring engine.

---

## Stage 8 — WhatsApp

Deferred until the core system is proven and WhatsApp Business API access and economics are confirmed.

---

# 26. Definition of Done

A feature is NOT done when:

```text
"go test" passes
```

alone.

A feature is done when:

* Specification is identified.
* Test exists.
* Test initially failed for the expected reason.
* Implementation exists.
* Test passes.
* Relevant regression tests pass.
* Edge cases have been considered.
* No unrelated behavior was broken.
* Code belongs in the correct architectural layer.
* The implementation does not violate the tutoring rules.
* The behavior has been explicitly verified.

---

# 27. Agent Behavior

When asked to implement something:

1. Inspect the existing repository.
2. Identify the relevant specification.
3. State the requirement being implemented.
4. Identify the smallest unit of behavior.
5. Write or propose the test first.
6. Run the test.
7. Implement only what is necessary.
8. Run the test again.
9. Run regression tests.
10. Report what changed.
11. Stop.

Do not automatically continue into the next feature.

---

# 28. Do Not Guess

If requirements conflict or are ambiguous:

STOP.

Identify:

```text
Document A says X.
Document B says Y.
```

Then determine which source has authority according to the Source-of-Truth Hierarchy.

If the conflict cannot be resolved from the repository, ask for a decision.

Never silently invent product behavior.

---

# 29. No Premature Abstraction

Do not create:

* Generic frameworks
* Complex interfaces
* Large dependency trees
* Microservices
* Plugin systems
* Over-generalized repositories
* Abstractions for hypothetical future requirements

Build the smallest correct system.

Refactor when real requirements justify it.

---

# 30. No Feature Creep

If a requested change is outside the current build stage:

* Identify it.
* Explain that it is outside the current stage.
* Do not silently implement it.

The project follows:

```text
One feature
    ↓
Test
    ↓
Implementation
    ↓
Verification
    ↓
Next feature
```

---

# 31. Engineering Learning Mode

The developer working on this project is learning software engineering.

Therefore explanations should prioritize understanding.

When teaching an implementation:

1. Explain the problem.
2. Ask what the developer thinks should happen.
3. Introduce one concept at a time.
4. Show the smallest relevant example.
5. Ask prediction questions.
6. Let the developer reason before revealing the answer where practical.
7. Verify understanding.
8. Only then proceed.

Do not dump large amounts of unexplained code.

---

# 32. Preferred Assistance Escalation

When the developer is stuck:

```text
1. Guiding question
2. Small hint
3. Stronger hint
4. Partial solution
5. Complete solution
```

Do not immediately jump to a complete implementation unless explicitly requested or necessary.

---

# 33. Required Verification After Every Feature

After implementation, report:

```text
Feature:
<name>

Specification:
<requirement>

Test added:
<test>

Initial result:
FAIL / expected failure

Implementation:
<short explanation>

Final result:
PASS

Regression tests:
PASS / FAIL

Status:
COMPLETE / BLOCKED
```

If blocked, explain exactly what is blocking progress.

---

# 34. Golden Rule

The most important rule in this repository is:

> **Do not build what the specification merely allows. Build what the tests prove the specification requires.**

And:

> **Never allow an LLM's confidence to replace explicit system state or deterministic validation where correctness can be established by code.**

SoTeach succeeds when it can reliably demonstrate that it understands the learner's actual gap, teaches that gap, verifies understanding, remembers the learner's state, and behaves consistently across interruptions and clients.

Everything else is secondary.

# 35. Mandatory CLI + One-Step-at-a-Time Protocol

This repository is developed interactively through the command line.

The developer will copy and paste the commands provided by the engineering agent into the terminal.

This workflow is mandatory.

## 35.1 Every repository change must be performed through the CLI

When a new:

- Folder
- File
- Source file
- Test file
- Configuration file
- Database migration
- Fixture
- Script

is required, provide the exact CLI command needed to create it.

Do not assume that a folder or file already exists.

If a directory must be created, explicitly provide:

```bash
mkdir -p path/to/directory

# SoTeach — Additional Agent.md Requirements

## 36. AI Integration Contract

AI is a core component of SoTeach.

AI must not be treated as an optional future enhancement or a generic chatbot layer.

The tutoring system must be designed so that AI contributes intelligence while deterministic application code controls state, correctness, persistence, and safety boundaries.

The intended relationship is:

```text
                    ┌─────────────────────┐
                    │        AI           │
                    │                     │
                    │ Diagnose            │
                    │ Explain             │
                    │ Analyze reasoning   │
                    │ Generate practice   │
                    │ Adapt communication │
                    └──────────┬──────────┘
                               │
                               ▼
                    ┌─────────────────────┐
                    │   Tutoring Engine   │
                    │                     │
                    │ State machine       │
                    │ Rules               │
                    │ Turn management     │
                    │ Mastery decisions   │
                    └──────────┬──────────┘
                               │
                 ┌─────────────┴─────────────┐
                 ▼                           ▼
        ┌─────────────────┐        ┌─────────────────┐
        │ Deterministic   │        │   Persistence   │
        │ Validators      │        │                 │
        │                 │        │ Sessions        │
        │ Math correctness│        │ Learner state   │
        │ Exact rules     │        │ Progress        │
        └─────────────────┘        └─────────────────┘
```

The AI must operate inside this architecture.

Do not build SoTeach as:

```text
User → LLM → response
```

Build it as:

```text
User
 ↓
Application
 ↓
Tutoring Engine
 ↓
AI / Validator / State
 ↓
Application
 ↓
User
```

---

# 37. AI Responsibilities

The AI may be responsible for:

* Diagnosing possible misconceptions
* Interpreting learner reasoning
* Generating explanations
* Generating examples
* Generating practice questions
* Generating transfer questions
* Adapting explanation complexity
* Adapting language to age band
* Conversational tutoring
* Suggesting the next pedagogical action

The AI must NOT independently control:

* Current session state
* Current learner
* Turn ownership
* Confirmation count
* Mastery status
* Deterministic mathematical correctness
* Persistence
* Authorization
* Safety gates

The application must validate AI outputs before using them to change authoritative state.

---

# 38. AI Provider Abstraction

The tutoring engine must not directly depend on a specific AI vendor.

Prefer a boundary similar to:

```go
type AIProvider interface {
    Diagnose(...)
    Explain(...)
    GeneratePractice(...)
    GenerateVerification(...)
    AnalyzeReasoning(...)
}
```

The exact interface must be determined from actual requirements before implementation.

Do not create a large generic interface merely for hypothetical providers.

The abstraction exists to keep vendor-specific implementation outside the tutoring domain.

A concrete provider may eventually be:

```text
OpenAI
Anthropic
Google
Local model
Other compatible provider
```

Only implement the provider required by the current stage.

---

# 39. AI Output Must Be Structured

When AI output affects application behavior, prefer structured output.

Do not parse arbitrary conversational text to determine critical state.

For example:

```text
DiagnosticResult
    concept
    knownSkills
    gaps
    misconceptions
    confidence
    recommendedAction
```

And:

```text
TutoringAction
    action
    reason
    targetConcept
    explanation
```

The exact structures must be derived from the specification.

The model may produce natural language for the learner, but internal decisions should use validated structured data.

---

# 40. AI Output Validation

Never blindly trust an AI response.

Before an AI-generated result changes application state:

1. Parse the structured response.
2. Validate required fields.
3. Validate allowed values.
4. Validate consistency with current session state.
5. Reject invalid output.
6. Retry or fall back when appropriate.
7. Never silently convert malformed AI output into a successful tutoring action.

Example:

```text
AI:
recommendedAction = "mark_mastery"

Application:
Does current state contain sufficient verification evidence?

NO

Result:
Reject AI action.
Keep learner in current state.
```

The AI proposes.

The application decides.

---

# 41. AI Failure Is a Normal System Condition

AI failures must be treated as expected operational conditions.

The system must account for:

* Timeout
* Network failure
* Provider unavailable
* Rate limiting
* Invalid structured output
* Malformed response
* Empty response
* Unexpected model behavior
* Provider API changes
* Token/context limits

A provider failure must not corrupt learner state.

For example:

```text
WAIT_FOR_ANSWER
      ↓
AI request
      ↓
TIMEOUT
      ↓
Session remains recoverable
```

Do not advance the tutoring state merely because an AI request failed.

---

# 42. AI Must Not Destroy Deterministic State

AI calls must not directly mutate critical session state.

Prefer:

```text
AI request
    ↓
AI response
    ↓
Validate response
    ↓
Tutoring Engine
    ↓
State transition
```

Avoid:

```text
AI request
    ↓
AI response
    ↓
directly modify session state
```

The tutoring engine remains the authority over state transitions.

---

# 43. Testing AI Without Depending on a Live Provider

Most unit and domain tests must not require a live LLM API.

Tests should be deterministic.

Use a fake or test implementation of the AI provider where appropriate.

Example:

```text
FakeAIProvider
    ↓
returns predetermined diagnostic result
```

This allows tests to verify tutoring behavior independently of:

* Internet connectivity
* API availability
* Provider latency
* API credits
* Model randomness

Live AI integration tests may exist separately, but they must not be required for ordinary unit tests.

---

# 44. AI Integration Test Boundary

AI integration tests should verify things such as:

* Correct request construction
* Correct structured response parsing
* Invalid response handling
* Provider errors
* Timeouts
* Retry behavior where specified
* Model/provider configuration

Domain tests should verify:

* State transitions
* Tutoring behavior
* Mastery logic
* Waiting behavior
* Confirmation counting
* Learner separation

Do not mix every concern into one giant integration test.

---

# 45. Prompt Engineering Is Part of the System

Prompts that influence core tutoring behavior must be treated as engineering artifacts.

Do not casually change critical prompts.

When a prompt is changed:

1. Identify the behavioral requirement affected.
2. Identify the relevant tests.
3. Change the prompt.
4. Run the affected tests.
5. Run regression tests.
6. Confirm no tutoring invariant was weakened.

Where practical, prompts should be versioned.

Example:

```text
prompts/
    diagnostic_v1.txt
    teaching_v1.txt
    verification_v1.txt
```

Do not create this directory until the implementation stage actually requires it.

---

# 46. AI Model Configuration

Do not hard-code:

* API keys
* Secrets
* Tokens
* Provider credentials

into source code.

Use environment/configuration mechanisms.

Never commit credentials.

Tests must be runnable without requiring the developer to expose secrets.

---

# 47. AI Cost and Latency

AI calls have cost and latency.

Do not call an LLM when deterministic code can answer the question reliably.

For example:

```text
2 + 2 = 4
```

does not require an LLM to determine correctness.

The AI should be used where intelligence provides value.

Avoid unnecessary repeated calls during one tutoring turn.

Prefer:

```text
one meaningful AI decision
```

over:

```text
many unnecessary AI calls
```

The exact optimization strategy must be based on measured behavior rather than premature optimization.

---

# 48. AI Should Be Observable

Where appropriate, record enough metadata to debug AI-assisted tutoring behavior without unnecessarily storing sensitive learner content.

Useful metadata may include:

* Provider
* Model
* Prompt/version identifier
* Request type
* Latency
* Success/failure
* Structured-output validation result
* Retry count
* Token usage where available

Do not expose sensitive learner information unnecessarily in logs.

---

# 49. AI Safety Boundary

The AI must not bypass application safety rules.

If the application determines that an action is prohibited:

```text
Application rule
      ↓
BLOCK
```

The model cannot override that decision.

Do not rely on a prompt alone to enforce critical child-safety rules.

Safety-critical controls belong in deterministic application logic wherever possible.

---

# 50. Current Development State Must Be Preserved

When this section was first written, the codebase was pinned at a single RED
test (`TestTutorWaitsForLearnerAnswer`, "tutoring engine not implemented").
That behavior has since been implemented and verified, so this snapshot is
updated rather than appended. If you are unsure where the project stands,
read this section first, then confirm against `README.md` and
`workingReadme.md` (which carry matching status lines).

```text
Go module:
soteach

Go version:
1.22.2

Implementation layout:
ai/       — AIProvider boundary, DiagnosticResult + validation (Agent.md §36-44)
session/  — state machine, interaction rules, per-learner Session + Roster,
            in-memory session store (MemoryStore)
tutor/    — application layer that owns the loop: BeginTopic, SubmitDiagnosis,
            SubmitAnswer, ApplyInput (Agent.md §11, §19)
api/      — thin REST/JSON HTTP boundary over the tutor and store (Agent.md §19)
web/      — embedded Stage 2 web client (plain HTML/CSS/JS)
cmd/server — runnable server: API + static web on one origin
tests/    — 82 behavior tests, all passing via `go test ./tests/`

Verified behavior (narrow MVP slice — Mathematics/Addition, single learner):
- The full server-owned loop runs over HTTP: begin -> diagnose -> practice ->
  verify -> mastery; the client sends only the learner's words and receives a
  prompt plus the authoritative state token back.
- Stage 2 done-condition met: a real learner completes the full cycle in a
  browser (web/ served by cmd/server) with no engineer intervention.
- Wait-for-learner; no advancing while an answer is pending (README §3)
- DIAGNOSE -> TEACH -> PRACTICE -> VERIFY loop; no stage skippable (README §2)
- Deterministic answer checking; uncertain answers distinguished (README §8.4, §9)
- Correct -> transfer question (genuinely different); transfer-correct -> mastery
- Wrong-but-attempted -> fresh question; uncertain -> taught then re-asked
  (README §6, §9)
- Exact confirmation count (max two) tracked as explicit state (README §8.3)
- One active respondent; learner state kept separate (README §3)
- Subject/topic nesting per README §6; mastery per subject+topic, preserved
  across switches; unverified progress discarded on switch
- Session resumption without repeating completed diagnosis — at the domain,
  store, and HTTP layers (README §6)
- Grade-band validation: only Primary 4-6, JSS1-3, SSS1-3 (README §4, §10)
- AI-validated diagnosis: malformed results rejected, provider failure leaves
  the session untouched (Agent.md §40-42)
```

Remaining before this backend is fully closed out:

- Durable persistence: the session store is in-memory (behavior proven per
  Agent.md §20); a real store (PostgreSQL is the planned target) is the next
  backend step, keeping the API, tutor, and web-client behavior unchanged.

Deferred product gaps (noted, not yet built):

- Subject/topic picker and grade-band/age capture in the web client — the UI
  hardcodes Mathematics → Addition and the engine content is not age-
  calibrated yet.
- Wrong answers loop to a fresh question indefinitely: no teach-on-repeated-
  wrong, no attempt cap, and no learner stop/quit affordance until the loop
  closes correctly.

Open, non-code item (Blueprint-tracked, not a development gate):

- Real-learner protocol validation (workingReadme.md §8, Stage 0) is owned
  under `Blueprint/` and should be completed before any broad/real-world
  rollout; it does not block Stage 1/2 development.

The next task:

Make persistence durable (Agent.md §20: swap MemoryStore for a real store with
the API and tutor behavior unchanged) or address one of the deferred product
gaps above. Proceed one feature at a time with the same test-first cycle (§3).

Do not skip directly to:

* Mobile
* Authentication
* Deployment
* Multiple AI providers
* Payments
* Dashboards
* WhatsApp

until the backend is durable. (Stage 2's web client is built — Web UI is no
longer on the skip list.)

---

# 51. Exact CLI Development Protocol

Every repository modification must be executable by the developer through the command line.

When a new file is required, provide the exact path and complete contents.

Example:

```bash
mkdir -p path/to/folder

cat > path/to/file.go <<'EOF'
package example

...
EOF
```

When a file already exists, do not blindly overwrite it.

First inspect it using an appropriate CLI command such as:

```bash
cat path/to/file
```

or:

```bash
sed -n '1,200p' path/to/file
```

Then provide the smallest required modification.

---

# 52. One Command/Operation at a Time

The agent must not give the developer a large batch of commands and ask them to execute everything before reporting back.

Prefer:

```text
STEP 1
↓
Developer executes
↓
Output confirmed
↓
STEP 2
↓
Developer executes
↓
Output confirmed
```

Even if several commands are logically related, keep the development interaction incremental.

If one operation requires several shell lines to perform a single atomic change, those lines may be provided together.

The key rule is:

> One atomic repository change or verification step at a time.

---

# 53. Confirmation Is Mandatory

After giving a CLI step, stop.

Wait for the developer's output.

Do not assume:

```text
file created
```

because the command was provided.

Do not assume:

```text
test passed
```

because the code appears correct.

Do not assume:

```text
compilation succeeded
```

without observed output.

The developer's terminal output is the confirmation.

---

# 54. Never Hide Intermediate Failures

If a test fails, show and investigate the failure.

Do not immediately replace the test.

Do not weaken assertions.

Do not change the expected behavior merely to obtain:

```text
PASS
```

A failure is useful information.

The purpose of the test-first process is to expose incorrect assumptions before they become architecture.

---

# 55. Test Scope Discipline

Every test should have a clear responsibility.

Prefer:

```text
TestTutorWaitsForLearnerAnswer
```

over a test that simultaneously verifies:

```text
state + AI + database + HTTP + UI + authentication
```

Small behavioral tests provide clearer failure signals.

Larger integration tests should be added only when the corresponding integration actually exists.

---

# 56. No Production Dependency Just to Satisfy a Test

Do not introduce a database, AI provider, HTTP server, or external service merely because a test was written.

First determine the smallest domain behavior required.

For example, if the requirement is:

> The tutor must remain waiting until the learner responds.

the first implementation may require only an in-memory state representation.

Do not introduce PostgreSQL simply to test a state transition.

---

# 57. Repository Inspection Before Modification

Before modifying an existing area:

1. Inspect the repository.
2. Inspect the relevant files.
3. Identify existing code.
4. Identify the relevant specification.
5. Determine whether the requested behavior already exists.
6. Write or update the test.
7. Only then modify implementation.

Never overwrite existing work without inspecting it.

---

# 58. Git Discipline

Do not perform destructive Git operations unless explicitly instructed.

Avoid commands such as:

```bash
git reset --hard
git clean -fd
git checkout -- .
```

unless the developer explicitly requests them and understands their consequences.

Prefer observable operations:

```bash
git status
git diff
git log --oneline -5
```

The agent must protect existing work.

---

# 59. Implementation Minimalism

When a test fails, implement the smallest correct change capable of satisfying the specification.

Do not use the first failing test as an excuse to build the entire architecture.

Example:

If the test requires a tutor to remain in:

```text
WAIT_FOR_ANSWER
```

do not simultaneously implement:

* AI integration
* PostgreSQL
* authentication
* REST endpoints
* frontend
* mobile
* deployment

Implement only what the current behavioral requirement demands.

---

# 60. Absolute Development Loop

Every feature follows:

```text
READ SPECIFICATION
        ↓
DEFINE BEHAVIOR
        ↓
WRITE TEST
        ↓
RUN TEST
        ↓
CONFIRM RED
        ↓
IMPLEMENT MINIMUM
        ↓
RUN TEST
        ↓
CONFIRM GREEN
        ↓
RUN REGRESSION
        ↓
CONFIRM REGRESSION
        ↓
STOP
        ↓
WAIT FOR DEVELOPER
        ↓
NEXT FEATURE
```

The agent must not skip a stage because the implementation appears obvious.

The agent must not move forward because the developer "probably" got the expected result.

The agent must wait for actual confirmation.

---

# 61. Final Rule

The SoTeach engineering process is governed by:

> **Specification → Test → Failure → Minimal Implementation → Pass → Regression → Confirmation → Next Step.**

And the execution mechanism is:

> **CLI → Developer executes → Output → Agent verifies → Next step.**

And the architectural principle is:

> **AI provides intelligence. Deterministic software provides control. Tests prove the behavior.**

All three principles must remain intact throughout the project.
"""
