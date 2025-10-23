# Robusta Frontend (Next.js)

This frontend is built with Next.js (App Router) + TypeScript + Tailwind CSS (v4). It provides the management console for Spydon.

## Getting Started

Install dependencies (uses npm):

```bash
npm install
```

Start the dev server:

```bash
npm run dev
```

The app will be available at <http://localhost:3000>.

## Available Scripts

- `npm run dev` – start Next.js in development mode with hot reloading.
- `npm run build` – create an optimized production build.
- `npm run start` – run the production build locally.
- `npm run lint` – run ESLint checks.

## Environment Variables

- `NEXT_PUBLIC_API_BASE_URL` – REST API endpoint (defaults to `http://localhost:8080/api/v1`).

## Styling Guidelines

Global styles live in `src/app/globals.css` and follow the rules described in `../../docs/notes/frontend-design.mdc`. Components use Tailwind utility classes together with custom UI primitives in `src/components/ui`.
