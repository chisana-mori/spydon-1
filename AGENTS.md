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
    *   **Documentation:** Handlers under `features` MUST include standard Swagger annotations. The description MUST be detailed (20-50 words) to facilitate future MCP extraction.

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

## 6. UI Layout Rules
*   **Content Container Margins:** Do not use `container`, `mx-auto`, or `max-w-*` on high-level page wrappers. Let the specific component or global shell handle the width to ensure consistent sidebar alignment.
*   **Page Main Title Styling:** All page main titles (h1 elements in page containers) MUST use `text-2xl font-bold` for consistency. Additional classes like `tracking-tight` are optional for visual enhancement.
    *   ✅ **Correct**: `<h1 className="text-2xl font-bold">设备管理</h1>`
    *   ✅ **Correct**: `<h1 className="text-2xl font-bold tracking-tight">集群管理</h1>`
    *   ❌ **Incorrect**: `<h1 className="text-3xl font-bold">...</h1>` (too large)
    *   ❌ **Incorrect**: `<h1 className="text-xl font-bold">...</h1>` (too small)
*   **Page Title Icon:** All page main titles MUST be preceded by a semantically relevant icon wrapped in a styled container for visual consistency.
    *   **Icon Container**: Use `<div className="p-2 rounded-lg bg-primary/10">` with icon size `h-6 w-6 text-primary`
    *   **Layout Structure**: Wrap icon container and title in `flex items-center gap-3`
    *   **Icon Selection**: Choose icons that visually represent the page content (e.g., `Users` for user management, `Server` for clusters, `Database` for resources)
    *   ✅ **Correct Pattern**:
      ```tsx
      <div className="flex items-center gap-3">
        <div className="p-2 rounded-lg bg-primary/10">
          <Server className="h-6 w-6 text-primary" />
        </div>
        <div>
          <h1 className="text-2xl font-bold">设备管理</h1>
          <p className="text-muted-foreground">...</p>
        </div>
      </div>
      ```

## 7. Frontend Design System

This section is the **SINGLE SOURCE OF TRUTH** for color usage and frontend patterns.

### 7.1. Core Color Philosophy
Our color system conveys meaning instantaneously:
*   **Trust & Information**: Blue/Primary spectrum (Stock, Info)
*   **Action & Alertness**: Semantic colors (Success, Warning, Error)
*   **Domain Specificity**: Consistent mapping of asset classes/features to specific hues.

### 7.2. Semantic Color System
**DO NOT** hardcode hex values. Use these semantic mappings.

**A. Domain/Feature Colors**
| Domain | Color Scale | Tailwind Class | Semantic Meaning |
| :--- | :--- | :--- | :--- |
| **Crypto** | Orange | `text-orange-500` / `bg-orange-500` | Energy, volatility, digital assets |
| **Stock** | Blue | `text-blue-500` / `bg-blue-500` | Stability, traditional markets, trust |
| **ETF** | Green | `text-green-500` / `bg-green-500` | Growth, composition, aggregation |
| **Forex** | Purple | `text-purple-500` / `bg-purple-500` | Global, exchange, luxury/value |
| **Commodity**| Yellow | `text-yellow-500` / `bg-yellow-500` | Gold, resources, raw materials |
| **Index** | Red | `text-red-500` / `bg-red-500` | Market pulse, urgency, heatmaps |

**B. Status & Feedback Colors**
| State | Color Scale | Tailwind Class | Usage Context |
| :--- | :--- | :--- | :--- |
| **Success** | Green | `text-green-600` | Operation completed, system healthy, online |
| **Warning** | Yellow/Amber | `text-yellow-500` | Non-critical issues, pending actions, degrading |
| **Error** | Red | `text-red-500` | Critical failures, offline, danger zones |
| **Info** | Blue | `text-blue-500` | Neutral information, tips, help text |
| **Muted** | Slate/Gray | `text-muted-foreground`| Secondary text, disabled states, placeholders |

### 7.3. Color Usage & Theming Rules
1.  **Subtle Backgrounds**: Use `bg-{color}-50` (light) or `bg-{color}-900/10` (dark).
    *   *Prefer*: `bg-blue-500/10` (Works automatically in both modes via opacity).
    *   *Avoid*: `bg-blue-100` (Too bright in dark mode).
2.  **Icon Containers**:
    *   Pattern: `className="p-2.5 rounded-xl bg-{color}-500/10 text-{color}-500 ring-1 ring-{color}-500/20"`
    *   Icons representing a domain MUST use the domain's primary color.
3.  **Text Hierarchy**:
    *   **Primary**: `text-foreground`
    *   **Secondary**: `text-muted-foreground`
    *   **Interactive**: `text-primary` or `text-blue-600`

```typescript
// Reference Object
export const DESIGN_COLORS = {
  crypto: 'orange-500',
  stock: 'blue-500',
  etf: 'green-500',
  forex: 'purple-500',
  commodity: 'yellow-500',
  index: 'red-500',
  success: 'green-500',
  warning: 'yellow-500',
  error: 'red-500',
  info: 'blue-500'
}
```

### 7.4. Component Patterns

**Dialog Header** (MANDATORY):
```tsx
<DialogHeader className="space-y-3 pb-6">
  <DialogTitle className="text-2xl font-bold flex items-center gap-3">
    <div className="w-8 h-8 bg-primary/10 rounded-lg flex items-center justify-center">
      <Icon className="w-4 h-4 text-primary" />
    </div>
    Dialog Title
  </DialogTitle>
  <DialogDescription className="text-base">Description</DialogDescription>
</DialogHeader>
```

**Card Layout**:
```tsx
// Interactive card
<Card className="border-2 border-dashed border-muted-foreground/20 hover:border-primary/50 transition-colors">

// Information card
<Card className="bg-muted/30 border-blue-200 dark:border-blue-800">
```

**Form Elements** (h-11 height required):
```tsx
<Input className="h-11" />
<Select><SelectTrigger className="h-11" /></Select>
<Button className="h-11" />

// Labels with icons
<Label className="text-sm font-medium flex items-center gap-2">
  <Icon className="w-4 h-4 text-{color}-600" />
  Field Name
</Label>
```

### 7.5. Status Indicators

```tsx
// Animated status dot
<div className="flex items-center gap-1 text-green-600">
  <div className="w-2 h-2 bg-green-500 rounded-full animate-pulse" />
  <span className="text-xs">Connected</span>
</div>

// Status badge
<Badge variant={status === 'active' ? 'default' : 'secondary'}>
  {statusLabel}
</Badge>
```

### 7.6. Table Enhancements

```tsx
<TableRow className="hover:bg-muted/50">
  <TableCell>
    <div className="flex items-center gap-3">
      <div className="w-8 h-8 bg-primary/10 rounded-lg flex items-center justify-center">
        <Icon className="w-4 h-4 text-primary" />
      </div>
      <div>
        <div className="font-medium">Primary Information</div>
        <div className="text-sm text-muted-foreground">Secondary info</div>
      </div>
    </div>
  </TableCell>
</TableRow>
```

### 7.7. Sizing Reference

| Element | Size |
|---------|------|
| Form Elements | `h-11` |
| Large Buttons | `h-12` |
| Icon Containers | `w-8 h-8` (standard), `w-12 h-12` (large) |
| Icons | `w-4 h-4` (small), `w-5 h-5` (medium), `w-6 h-6` (large) |

### 7.8. Spacing Guidelines

| Context | Spacing |
|---------|---------|
| Page Sections | `space-y-6` or `space-y-8` |
| Card Content | `space-y-4` |
| Icon Gaps | `gap-2` or `gap-3` |
| Button Groups | `gap-3` or `gap-4` |
| Grid Layouts | `grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4` |

### 7.9. Responsive Patterns (REQUIRED)

```tsx
// Flex direction change
<div className="flex flex-col sm:flex-row gap-4">

// Grid columns
<div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">

// Action buttons
<div className="flex flex-col sm:flex-row justify-between gap-4 pt-6 border-t">
```

### 7.10. Transitions (MANDATORY)

```tsx
className="transition-colors hover:border-primary/50"
className="transition-all hover:shadow-lg"
className="animate-pulse"  // For loading states
className="animate-spin"   // For spinners
```
