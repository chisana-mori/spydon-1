# Antigravity: Coding Standards

> **Objective:** Defy technical debt. Keep it light, fast, and simple.

## 1. Philosophy (KISS)
*   **WHAT:** Code must be simple, explicit, and flat.
*   **WHY:** Readability > Cleverness. Maintenance is 90% of the cost.
*   **HOW:**
    *   Files < 300 lines. Functions < 50 lines.
    *   No over-engineering ("YAGNI"). Implement only what is needed *now*.
    *   Logic > Abstraction. Explicit is better than implicit.

## 2. Architecture (Feature-First)
*   **WHAT:** Vertical slicing by feature (`internal/features/{feature}`).
*   **WHY:** High cohesion, low coupling. Isolation makes code safer to change or delete.
*   **HOW:**
    *   **Structure:**
        ```text
        /internal/features/{feature}/
          ├── services/       # Business logic
          ├── transport/      # HTTP/GRPC handlers
          └── models/         # Domain models
        ```
    *   **Layers:** Transport (Validation) → Service (Logic) → Repository (Data).
    *   **Imports:** MUST use `internal/features/{feature}/services`. (Forbidden: `internal/services`).
    *   **Shared:** Common logic goes to `internal/shared`. Export types (`Capitalized`) if used across features.

## 3. Go Rules
*   **WHAT:** Idiomatic Go with explicit error handling and concurrency.
*   **WHY:** Robustness and ease of debugging.
*   **HOW:**
    *   **Errors:** Handle explicitly. `if err != nil { return fmt.Errorf("ctx: %w", err) }`. Never use `_` to ignore errors.
    *   **Concurrency:** Use `errgroup` or `WaitGroup` for synchronization. Avoid channels unless passing data. No global state.
    *   **Handlers:** Keep them thin. 1. Parse Input → 2. Call Service → 3. Return Response (use `httpx`).

## 4. Node/TS Rules
*   **WHAT:** Type-safe, modern TypeScript with standard linting.
*   **WHY:** Compile-time safety prevents runtime crashes.
*   **HOW:**
    *   **Async:** Use `async/await`. No callbacks, no chained `.then`.
    *   **Types:** Strict interfaces. `any` is strictly forbidden.
    *   **Modules:** ES Modules (`import/export`).

## 5. Anti-Patterns
*   **WHAT:** Common pitfalls that degrade code quality.
*   **WHY:** They introduce debt and complexity that is hard to remove later.
*   **HOW:**
    *   ❌ **Manager/Util Dumpsters:** No `utils.go`. Name files by specific intent (e.g., `PasswordHasher.go`).
    *   ❌ **Magic Numbers:** Extract constants (`const MaxRetries = 3`).
    *   ❌ **Premature Optimization:** Write clean code first. Optimize only when profiling proves necessary.

## 6. Pre-Commit Checklist
*   [ ] Is the file under 300 lines?
*   [ ] Are variable names descriptive?
*   [ ] Is logic separated from transport?
*   [ ] Does the code feel "light"?
