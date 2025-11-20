---
trigger: always_on
---


-----

# Antigravity Project: AGENTS.md

> **Objective:** To build software that defies the "gravity" of technical debt. Keep it light, keep it fast, keep it simple.

## 1\. Core Philosophy: The "Antigravity" Mindset

All generated code must adhere to the **KISS (Keep It Simple, Stupid)** principle. We optimize for readability and maintainability over cleverness.

  * **No Over-Engineering:** Do not implement features "just in case." Implement what is needed *now*.
  * **The Goldilocks Zone:**
      * ❌ **Too Heavy:** Don't put 500+ lines of code in one file/function.
      * ❌ **Too Light:** Don't create a file for a single 3-line helper function.
      * ✅ **Just Right:** Group related logic together.
  * **Explicit is better than Implicit:** Avoid "magic" code. It should be obvious what the code does by reading it.
  * **Flat is better than Nested:** Avoid deep nesting of conditionals or loops (Guard Clauses are your friend).

-----

## 2\. General Architecture & Code Structure

Regardless of the language, follow these structural rules to prevent "Spaghetti Code":

### 2.1. Separation of Concerns

Do not mix logic. Use a simplified 3-layer approach:

1.  **Transport/Handler:** Receives input (HTTP/CLI), validates it, calls the Logic.
2.  **Business Logic (Service):** The brain. Makes decisions.
3.  **Data Access (Repository):** Talks to DB/Files/API.

### 2.2. File Size & Complexity

  * **Files:** Aim for \< 300 lines. If it grows larger, check if responsibilities are leaking.
  * **Functions:** Aim for \< 50 lines. A function should do **one** thing.
  * **Variable Naming:** Be descriptive. `ctx` is fine, `data` is not. `fetchUserData` is better than `getData`.

-----

## 3\. Golang Best Practices

**Style Guide:** Standard Go idioms (Effective Go).

### 3.1. Project Layout

Keep it flat if small, use standard layout if medium/large.

```text
/cmd        # Main entry points
/internal   # Private application code (Business Logic)
/pkg        # Library code ok to use by external apps (Generic tools)
```

### 3.2. Coding Rules

  * **Error Handling:**
      * ❌ Never ignore errors (`_ = func()`).
      * ✅ Handle errors explicitly: `if err != nil { return fmt.Errorf("failed to X: %w", err) }`.
      * Use error wrapping (`%w`) to provide context traces.
  * **Concurrency:**
      * Don't use Channels unless you actually need to communicate data between goroutines. For simple synchronization, use `sync.WaitGroup` or `errgroup`.
      * Avoid global state.
  * **Interfaces:**
      * Define interfaces where they are *used* (consumer side), not where they are implemented.
      * Keep interfaces small (1-3 methods).
  * **Structs:**
      * Use pointer receivers (`func (s *Service)`) generally, unless the struct is a small data container.

**Go Example (Antigravity Style):**

```go
// ✅ GOOD: Flat, explicit error handling, clear flow
func (s *UserService) CreateUser(ctx context.Context, req CreateUserReq) (*User, error) {
    if err := req.Validate(); err != nil {
        return nil, fmt.Errorf("validation failed: %w", err)
    }

    user := &User{Email: req.Email}
    if err := s.repo.Save(ctx, user); err != nil {
        return nil, fmt.Errorf("db save failed: %w", err)
    }

    return user, nil
}
```

-----

## 4\. Node.js (TypeScript) Best Practices

**Style Guide:** Standard Prettier/ESLint configurations. Assume **TypeScript** is the default unless specified otherwise.

### 4.1. Project Layout

```text
/src
  /config     # Environment variables
  /modules    # Feature-based organization (user, auth, product)
     /services
     /controllers
  /utils      # Pure functions
```

### 4.2. Coding Rules

  * **Async/Await:**
      * ❌ No Callbacks. No `then/catch` chains unless absolutely necessary.
      * ✅ Use `async/await`.
      * Wrap top-level logic in `try/catch` blocks at the Controller/Handler level.
  * **Type Safety:**
      * ❌ No `any`.
      * ✅ Define Interfaces/Types for all inputs and outputs.
  * **Modularity:**
      * Use ES Modules (`import`/`export`).
      * Avoid circular dependencies.
  * **Performance:**
      * Use `Promise.all()` for independent concurrent operations.

**Node.js Example (Antigravity Style):**

```typescript
// ✅ GOOD: Typed, Async/Await, Guard Clauses
import { CreateUserDto, User } from './types';

export const createUser = async (dto: CreateUserDto): Promise<User> => {
  // 1. Validation
  if (!dto.email || !dto.email.includes('@')) {
    throw new Error('Invalid email format');
  }

  // 2. Business Logic
  const existingUser = await userRepo.findByEmail(dto.email);
  if (existingUser) {
    throw new Error('User already exists');
  }

  // 3. Execution
  const newUser = await userRepo.save({ ...dto, createdAt: new Date() });
  return newUser;
};
```

-----

## 5\. What to Avoid (Anti-Patterns)

When generating code for Antigravity, **STOP** if you are doing the following:

1.  **The "Manager" / "Util" Trap:** Don't create a `GeneralManager.go` or `CommonUtils.ts` that becomes a trash can for unrelated code. Name files by what they *do* (e.g., `DateFormatter.ts`, `PasswordHasher.go`).
2.  **Global Variables:** Pass dependencies explicitly (Dependency Injection).
3.  **Magic Numbers:** Define constants. `const MAX_RETRIES = 3` is better than `if (count > 3)`.
4.  **Premature Optimization:** Write clean code first. Profile later.

-----

## 6\. Implementation Checklist for Agents

Before outputting code, verify:

  - [ ] Is the file under 300 lines?
  - [ ] Are variable names descriptive?
  - [ ] Is error handling explicit (no silent failures)?
  - [ ] Is logic separated from transport (HTTP/CLI details)?
  - [ ] **Does this code feel "light"? (Simple to read, hard to break)**

-----
